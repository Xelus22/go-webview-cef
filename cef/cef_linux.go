//go:build linux
// +build linux

// Package cef provides a Go binding to the CEF runtime
// This is the main entry point for CEF WebView functionality
package cef

/*
#cgo CFLAGS: -I${SRCDIR}/../runtime -I${SRCDIR}/../third_party/cef/linux_64
#cgo LDFLAGS: -L${SRCDIR}/../third_party/cef/linux_64/Release -lcef -lX11 -Wl,-rpath,'$ORIGIN'

#include <stdlib.h>
#include "cef_runtime.h"

// Declare runtime functions (implemented in cef_runtime.c)
extern int cef_runtime_initialize(int argc, char** argv);
extern void cef_runtime_run(void);
extern void cef_runtime_shutdown(void);
extern cef_runtime_browser_t cef_runtime_create_browser(const cef_runtime_browser_opts_t* opts);
extern void cef_runtime_navigate(cef_runtime_browser_t browser, const char* url);
extern void cef_runtime_eval(cef_runtime_browser_t browser, const char* js);
extern void cef_runtime_close(cef_runtime_browser_t browser);
extern int cef_runtime_is_valid(cef_runtime_browser_t browser);
extern void cef_runtime_set_title(cef_runtime_browser_t browser, const char* title);
extern void cef_runtime_resize(cef_runtime_browser_t browser, int width, int height);
extern void cef_runtime_show(cef_runtime_browser_t browser);
extern void cef_runtime_hide(cef_runtime_browser_t browser);
extern void cef_runtime_set_message_callback(cef_runtime_message_cb_t callback, void* user_data);
extern void cef_runtime_inject_bridge(cef_runtime_browser_t browser);

// Wrapper function that calls the Go exported function
extern void goMessageCallback(char* name, char* payload, void* user_data);

static void message_callback_wrapper(const char* name, const char* payload, void* user_data) {
    goMessageCallback((char*)name, (char*)payload, user_data);
}

static cef_runtime_message_cb_t get_message_callback() {
    return (cef_runtime_message_cb_t)message_callback_wrapper;
}
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sync"
	"unsafe"
)

// Options for creating a WebView
type Options struct {
	Title      string
	URL        string // Initial URL to load
	Width      int
	Height     int
	Frameless  bool // Remove OS window decorations (no native buttons)
	Chromeless bool // Remove Chrome UI (no URL bar, tabs, etc.)
	Resizable  bool
	Fullscreen bool
	Maximized  bool
	X          int
	Y          int
}

// DefaultOptions returns default WebView options
func DefaultOptions() Options {
	return Options{
		Title:     "App",
		Width:     800,
		Height:    600,
		Resizable: true,
		X:         -1,
		Y:         -1,
	}
}

// WebView represents a CEF WebView window
type WebView struct {
	browser C.cef_runtime_browser_t
	opts    Options

	bindings  map[string]reflect.Value
	mu        sync.RWMutex
	dispatch  chan func()
	terminate chan struct{}
	initJS    []string
	handle    int // Unique handle for C callbacks
}

// Global state
var (
	g_initialized = false
	g_mu          sync.Mutex
	g_nextHandle  = 1
	g_webviews    = make(map[int]*WebView)
	g_webviewsMu  sync.RWMutex
)

// disableGPUFlags stores whether GPU should be disabled
var disableGPUFlags = false

// DisableGPU adds flags to disable GPU rendering (useful for WSL/VMs)
func DisableGPU() {
	disableGPUFlags = true
}

// Initialize initializes CEF and handles subprocess detection
// Returns true if this is the browser process (continue running)
// Returns false if this is a subprocess (will os.Exit)
func Initialize() bool {
	g_mu.Lock()
	defer g_mu.Unlock()

	if g_initialized {
		return true
	}

	// Add GPU disable flags if requested
	if disableGPUFlags {
		os.Args = append(os.Args,
			"--disable-gpu",
			"--disable-gpu-compositing",
			"--disable-software-rasterizer",
			"--disable-features=VizDisplayCompositor,UseSkiaRenderer",
			"--single-process",
		)
	}

	// Convert Go args to C args
	argc := len(os.Args)
	cArgs := make([]*C.char, argc)
	for i, arg := range os.Args {
		cArgs[i] = C.CString(arg)
	}

	// Call CEF initialize
	ret := C.cef_runtime_initialize(C.int(argc), &cArgs[0])

	// Note: We don't free cArgs because CEF keeps references

	// ret >= 0 means subprocess - should exit
	// ret == -1 means browser process - continue
	// ret == -2 means error
	if ret >= 0 {
		os.Exit(int(ret))
	}

	if ret == -2 {
		panic("CEF initialization failed")
	}

	g_initialized = true
	return true
}

// Run starts the CEF message loop (blocks until all windows closed)
func Run() {
	if !g_initialized {
		panic("CEF not initialized")
	}
	C.cef_runtime_run()
}

// Shutdown cleans up CEF resources
func Shutdown() {
	g_mu.Lock()
	defer g_mu.Unlock()

	if !g_initialized {
		return
	}

	C.cef_runtime_shutdown()
	g_initialized = false
}

// New creates a new WebView window
func New(opts Options) *WebView {
	if !g_initialized {
		if !Initialize() {
			return nil
		}
	}

	// Assign unique handle
	g_webviewsMu.Lock()
	handle := g_nextHandle
	g_nextHandle++
	w := &WebView{
		opts:      opts,
		bindings:  make(map[string]reflect.Value),
		dispatch:  make(chan func(), 100),
		terminate: make(chan struct{}),
		handle:    handle,
	}
	g_webviews[handle] = w
	g_webviewsMu.Unlock()

	return w
}

// Run creates the window and starts the message loop
func (w *WebView) Run() {
	if w.browser != nil {
		return // Already running
	}

	// Create browser options
	url := w.opts.URL
	if url == "" {
		url = "about:blank"
	}
	copts := C.cef_runtime_browser_opts_t{
		url:        C.CString(url),
		title:      nil,
		width:      C.int(w.opts.Width),
		height:     C.int(w.opts.Height),
		frameless:  cbool(w.opts.Frameless),
		chromeless: cbool(w.opts.Chromeless),
		resizable:  cbool(w.opts.Resizable),
		fullscreen: cbool(w.opts.Fullscreen),
		maximized:  cbool(w.opts.Maximized),
		x:          C.int(w.opts.X),
		y:          C.int(w.opts.Y),
	}

	if w.opts.Title != "" {
		copts.title = C.CString(w.opts.Title)
	}

	// Create browser
	w.browser = C.cef_runtime_create_browser(&copts)

	// Free C strings
	C.free(unsafe.Pointer(copts.url))
	if copts.title != nil {
		C.free(unsafe.Pointer(copts.title))
	}

	if w.browser == nil {
		panic("Failed to create browser")
	}

	// Set up message callback - pass handle instead of Go pointer
	C.cef_runtime_set_message_callback(
		C.get_message_callback(),
		unsafe.Pointer(uintptr(w.handle)),
	)

	// Inject bridge
	C.cef_runtime_inject_bridge(w.browser)

	// Inject initial JS
	for _, js := range w.initJS {
		w.Eval(js)
	}

	// Run message loop
	Run()
}

// Navigate loads a URL in the WebView
func (w *WebView) Navigate(url string) {
	if w.browser == nil {
		return
	}
	curl := C.CString(url)
	defer C.free(unsafe.Pointer(curl))
	C.cef_runtime_navigate(w.browser, curl)
}

// Eval executes JavaScript in the WebView
func (w *WebView) Eval(js string) {
	if w.browser == nil {
		// Queue for after creation
		w.initJS = append(w.initJS, js)
		return
	}
	cjs := C.CString(js)
	defer C.free(unsafe.Pointer(cjs))
	C.cef_runtime_eval(w.browser, cjs)
}

// Bind binds a Go function to be callable from JavaScript
// The function will be available as window.go.functionName()
func (w *WebView) Bind(name string, fn interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		return fmt.Errorf("bind: %s is not a function", name)
	}

	w.bindings[name] = v

	// Inject JavaScript binding
	w.injectBinding(name, v.Type())

	return nil
}

// Unbind removes a binding
func (w *WebView) Unbind(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.bindings, name)
}

// injectBinding creates JavaScript wrapper for Go function
func (w *WebView) injectBinding(name string, t reflect.Type) {
	// Generate parameter list
	numIn := t.NumIn()
	params := make([]string, numIn)
	for i := 0; i < numIn; i++ {
		params[i] = fmt.Sprintf("p%d", i)
	}
	paramList := joinStrings(params, ", ")

	// Generate JS wrapper
	js := fmt.Sprintf(`
window.%s = function(%s) {
    return new Promise((resolve, reject) => {
        const id = Math.random().toString(36).substr(2, 9);
        window.__goCallbacks = window.__goCallbacks || {};
        window.__goCallbacks[id] = { resolve, reject };
        window.go.invoke("%s", JSON.stringify({ id: id, args: [%s] }));
    });
};
`, name, paramList, name, paramList)

	w.Eval(js)
}

// Dispatch schedules a function to run on the main thread
func (w *WebView) Dispatch(f func()) {
	select {
	case w.dispatch <- f:
	default:
	}
}

// Destroy closes the WebView window
func (w *WebView) Destroy() {
	if w.browser != nil {
		C.cef_runtime_close(w.browser)
		w.browser = nil
	}
}

// SetTitle sets the window title
func (w *WebView) SetTitle(title string) {
	w.opts.Title = title
	if w.browser != nil {
		ctitle := C.CString(title)
		defer C.free(unsafe.Pointer(ctitle))
		C.cef_runtime_set_title(w.browser, ctitle)
	}
}

// SetSize sets the window size
func (w *WebView) SetSize(width, height int) {
	w.opts.Width = width
	w.opts.Height = height
	if w.browser != nil {
		C.cef_runtime_resize(w.browser, C.int(width), C.int(height))
	}
}

// Show shows the window
func (w *WebView) Show() {
	if w.browser != nil {
		C.cef_runtime_show(w.browser)
	}
}

// Hide hides the window
func (w *WebView) Hide() {
	if w.browser != nil {
		C.cef_runtime_hide(w.browser)
	}
}

// IsValid returns true if the browser is still valid
func (w *WebView) IsValid() bool {
	if w.browser == nil {
		return false
	}
	return C.cef_runtime_is_valid(w.browser) != 0
}

// handleMessageFromJS handles messages from JavaScript
func (w *WebView) handleMessageFromJS(name, payload string) {
	w.mu.RLock()
	fn, exists := w.bindings[name]
	w.mu.RUnlock()

	if !exists {
		w.sendError(name, "", fmt.Sprintf("Function %s not found", name))
		return
	}

	// Parse payload
	var data struct {
		ID   string        `json:"id"`
		Args []interface{} `json:"args"`
	}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		w.sendError(name, data.ID, err.Error())
		return
	}

	// Call function
	result, err := w.callFunction(fn, data.Args)
	if err != nil {
		w.sendError(name, data.ID, err.Error())
		return
	}

	// Send result back
	w.sendResult(data.ID, result)
}

// callFunction calls a Go function with the provided arguments
func (w *WebView) callFunction(fn reflect.Value, args []interface{}) (interface{}, error) {
	fnType := fn.Type()
	numIn := fnType.NumIn()

	if len(args) != numIn {
		return nil, fmt.Errorf("expected %d arguments, got %d", numIn, len(args))
	}

	// Convert arguments
	in := make([]reflect.Value, numIn)
	for i := 0; i < numIn; i++ {
		argType := fnType.In(i)
		argValue := reflect.ValueOf(args[i])

		// Try to convert if types don't match
		if argValue.Type() != argType {
			if argValue.Type().ConvertibleTo(argType) {
				argValue = argValue.Convert(argType)
			} else {
				return nil, fmt.Errorf("argument %d: cannot convert %v to %v", i, argValue.Type(), argType)
			}
		}

		in[i] = argValue
	}

	// Call function
	out := fn.Call(in)

	// Handle return values
	if len(out) == 0 {
		return nil, nil
	}

	// If last value is error, handle it
	if len(out) > 0 {
		last := out[len(out)-1]
		if last.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !last.IsNil() {
				return nil, last.Interface().(error)
			}
			out = out[:len(out)-1]
		}
	}

	// Return first value if only one, otherwise return slice
	if len(out) == 1 {
		return out[0].Interface(), nil
	}

	results := make([]interface{}, len(out))
	for i, v := range out {
		results[i] = v.Interface()
	}
	return results, nil
}

// sendResult sends a successful result back to JavaScript
func (w *WebView) sendResult(id string, result interface{}) {
	data, _ := json.Marshal(map[string]interface{}{
		"id":     id,
		"result": result,
	})
	w.Eval(fmt.Sprintf("if (window.__goCallbacks && window.__goCallbacks['%s']) { window.__goCallbacks['%s'].resolve(%s); delete window.__goCallbacks['%s']; }",
		id, id, string(data), id))
}

// sendError sends an error back to JavaScript
func (w *WebView) sendError(name, id, errMsg string) {
	w.Eval(fmt.Sprintf("if (window.__goCallbacks && window.__goCallbacks['%s']) { window.__goCallbacks['%s'].reject(new Error('%s')); delete window.__goCallbacks['%s']; }",
		id, id, errMsg, id))
}

// Helper functions

func cbool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

//export goMessageCallback
func goMessageCallback(name *C.char, payload *C.char, userData unsafe.Pointer) {
	// Convert userData back to handle and look up WebView
	handle := int(uintptr(userData))
	g_webviewsMu.RLock()
	w, exists := g_webviews[handle]
	g_webviewsMu.RUnlock()
	if !exists {
		return
	}
	goName := C.GoString(name)
	goPayload := C.GoString(payload)
	w.handleMessageFromJS(goName, goPayload)
}
