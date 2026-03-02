//go:build linux
// +build linux

package cefruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/xelus/go-webview-cef/internal/cefbindings"
	"github.com/xelus/go-webview-cef/internal/cefshim"
)

// Options defines WebView window and runtime options.
type Options struct {
	Title      string
	URL        string
	Width      int
	Height     int
	Frameless  bool
	Chromeless bool
	Resizable  bool
	Fullscreen bool
	Maximized  bool
	X          int
	Y          int
	GPUMode    GPUMode
}

// GPUMode controls runtime GPU flag behavior.
type GPUMode int

const (
	// GPUAuto uses runtime defaults.
	GPUAuto GPUMode = iota
	// GPUEnabled avoids adding disable-gpu flags.
	GPUEnabled
	// GPUDisabled forces software/single-process-safe startup flags.
	GPUDisabled
)

// DefaultOptions returns default window options.
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

// BrowserHandle is an opaque wrapper around a CEF browser handle.
type BrowserHandle struct {
	raw cefbindings.BrowserHandle
}

func (h BrowserHandle) isZero() bool {
	return h.raw == 0
}

// Runtime defines the Go-owned runtime lifecycle.
type Runtime interface {
	Initialize(args []string) int
	CreateBrowser(opts Options) (BrowserHandle, error)
	SetMessageCallback(userData uintptr)
	SetLoadCallback(userData uintptr)
	InjectBridge(browser BrowserHandle)
	Navigate(browser BrowserHandle, url string)
	Eval(browser BrowserHandle, js string)
	SetTitle(browser BrowserHandle, title string)
	Resize(browser BrowserHandle, width, height int)
	Show(browser BrowserHandle)
	Hide(browser BrowserHandle)
	Close(browser BrowserHandle)
	IsValid(browser BrowserHandle) bool
	RunLoop()
	Shutdown()
}

type runtimeImpl struct{}

func (r *runtimeImpl) Initialize(args []string) int {
	return cefshim.Initialize(args)
}

func (r *runtimeImpl) CreateBrowser(opts Options) (BrowserHandle, error) {
	bo := cefbindings.DefaultBrowserOptions()
	bo.URL = opts.URL
	bo.Title = opts.Title
	bo.Width = opts.Width
	bo.Height = opts.Height
	bo.Frameless = opts.Frameless
	bo.Chromeless = opts.Chromeless
	bo.Resizable = opts.Resizable
	bo.Fullscreen = opts.Fullscreen
	bo.Maximized = opts.Maximized
	bo.X = opts.X
	bo.Y = opts.Y

	h := cefshim.CreateBrowser(bo)
	if h == 0 {
		return BrowserHandle{}, fmt.Errorf("create browser returned nil")
	}
	return BrowserHandle{raw: h}, nil
}

func (r *runtimeImpl) SetMessageCallback(userData uintptr) {
	cefshim.SetMessageCallbackUserData(userData)
}

func (r *runtimeImpl) SetLoadCallback(userData uintptr) {
	cefshim.SetLoadCallbackUserData(userData)
}

func (r *runtimeImpl) InjectBridge(browser BrowserHandle) {
	cefshim.InjectBridge(browser.raw)
}

func (r *runtimeImpl) Navigate(browser BrowserHandle, url string) {
	cefshim.Navigate(browser.raw, url)
}

func (r *runtimeImpl) Eval(browser BrowserHandle, js string) {
	cefshim.Eval(browser.raw, js)
}

func (r *runtimeImpl) SetTitle(browser BrowserHandle, title string) {
	cefshim.SetTitle(browser.raw, title)
}

func (r *runtimeImpl) Resize(browser BrowserHandle, width, height int) {
	cefshim.Resize(browser.raw, width, height)
}

func (r *runtimeImpl) Show(browser BrowserHandle) {
	cefshim.Show(browser.raw)
}

func (r *runtimeImpl) Hide(browser BrowserHandle) {
	cefshim.Hide(browser.raw)
}

func (r *runtimeImpl) Close(browser BrowserHandle) {
	cefshim.Close(browser.raw)
}

func (r *runtimeImpl) IsValid(browser BrowserHandle) bool {
	return cefshim.IsValid(browser.raw)
}

func (r *runtimeImpl) RunLoop() {
	cefshim.Run()
}

func (r *runtimeImpl) Shutdown() {
	cefshim.Shutdown()
}

// WebView represents a runtime-managed browser window.
type WebView struct {
	browser BrowserHandle
	opts    Options

	bindings map[string]reflect.Value
	mu       sync.RWMutex

	dispatch  chan func()
	terminate chan struct{}
	initJS    []string
	handle    int
}

var (
	runtimeCore Runtime = &runtimeImpl{}

	initialized bool
	initMu      sync.Mutex

	disableGPUFlags bool

	nextHandle = 1
	viewsMu    sync.RWMutex
	views      = make(map[int]*WebView)

	shimDispatchOnce sync.Once

	globalGPUMode = GPUDisabled
)

// DisableGPU enables GPU-disabling command line flags for environments that need it.
func DisableGPU() {
	disableGPUFlags = true
	globalGPUMode = GPUDisabled
}

// EnableGPU enables GPU-related browser processes.
func EnableGPU() {
	disableGPUFlags = false
	globalGPUMode = GPUEnabled
}

// Initialize initializes CEF and handles subprocess execution semantics.
func Initialize() bool {
	initMu.Lock()
	defer initMu.Unlock()

	if initialized {
		return true
	}

	shimDispatchOnce.Do(func() {
		cefshim.SetMessageDispatch(func(name, payload string, userData uintptr) {
			handle := int(userData)
			viewsMu.RLock()
			w, ok := views[handle]
			viewsMu.RUnlock()
			if !ok {
				return
			}
			w.handleMessageFromJS(name, payload)
		})
		cefshim.SetLoadDispatch(func(success bool, userData uintptr) {
			if !success {
				return
			}
			viewsMu.RLock()
			ws := make([]*WebView, 0, len(views))
			for _, w := range views {
				ws = append(ws, w)
			}
			viewsMu.RUnlock()
			for _, w := range ws {
				w.reinjectBridgeAndBindings()
			}
		})
	})

	alwaysFlags := []string{"--no-sandbox"}
	for _, flag := range alwaysFlags {
		if !containsArg(os.Args, flag) {
			os.Args = append(os.Args, flag)
		}
	}

	shouldDisableGPU := globalGPUMode != GPUEnabled || disableGPUFlags
	if shouldDisableGPU {
		gpuSafeFlags := []string{
			"--disable-gpu",
			"--disable-gpu-compositing",
			"--disable-software-rasterizer",
			"--disable-features=VizDisplayCompositor,UseSkiaRenderer",
			"--single-process",
		}
		for _, flag := range gpuSafeFlags {
			if !containsArg(os.Args, flag) {
				os.Args = append(os.Args, flag)
			}
		}
	}

	ret := runtimeCore.Initialize(os.Args)
	if ret >= 0 {
		os.Exit(ret)
	}
	if ret == -2 {
		panic("CEF initialization failed")
	}

	initialized = true
	return true
}

// Run starts the CEF message loop.
func Run() {
	if !initialized {
		panic("CEF not initialized")
	}
	runtimeCore.RunLoop()
}

// Shutdown shuts down CEF.
func Shutdown() {
	initMu.Lock()
	defer initMu.Unlock()

	if !initialized {
		return
	}
	runtimeCore.Shutdown()
	initialized = false
}

// New creates a new WebView instance.
func New(opts Options) *WebView {
	switch opts.GPUMode {
	case GPUEnabled:
		EnableGPU()
	case GPUDisabled:
		DisableGPU()
	}

	if !initialized {
		if !Initialize() {
			return nil
		}
	}

	viewsMu.Lock()
	handle := nextHandle
	nextHandle++
	w := &WebView{
		opts:      opts,
		bindings:  make(map[string]reflect.Value),
		dispatch:  make(chan func(), 100),
		terminate: make(chan struct{}),
		handle:    handle,
	}
	views[handle] = w
	viewsMu.Unlock()

	return w
}

// Run creates the browser and starts the message loop.
func (w *WebView) Run() {
	if !w.browser.isZero() {
		return
	}

	if w.opts.URL == "" {
		w.opts.URL = "about:blank"
	}

	browser, err := runtimeCore.CreateBrowser(w.opts)
	if err != nil {
		panic(err)
	}
	w.browser = browser

	runtimeCore.SetMessageCallback(uintptr(w.handle))
	runtimeCore.SetLoadCallback(0)
	runtimeCore.InjectBridge(w.browser)

	for _, js := range w.initJS {
		w.Eval(js)
	}

	Run()
}

// Navigate loads a URL.
func (w *WebView) Navigate(url string) {
	if w.browser.isZero() {
		return
	}
	runtimeCore.Navigate(w.browser, url)
}

// Eval executes JavaScript.
func (w *WebView) Eval(js string) {
	if w.browser.isZero() {
		w.initJS = append(w.initJS, js)
		return
	}
	runtimeCore.Eval(w.browser, js)
}

// Bind binds a Go function to window.<name>() in JavaScript.
func (w *WebView) Bind(name string, fn interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		return fmt.Errorf("bind: %s is not a function", name)
	}

	w.bindings[name] = v
	w.injectBinding(name, v.Type())
	return nil
}

// Unbind removes an existing binding.
func (w *WebView) Unbind(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.bindings, name)
}

func (w *WebView) injectBinding(name string, t reflect.Type) {
	numIn := t.NumIn()
	params := make([]string, numIn)
	for i := 0; i < numIn; i++ {
		params[i] = fmt.Sprintf("p%d", i)
	}
	paramList := strings.Join(params, ", ")

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

func (w *WebView) reinjectBridgeAndBindings() {
	if w.browser.isZero() {
		return
	}
	runtimeCore.InjectBridge(w.browser)
	w.mu.RLock()
	defer w.mu.RUnlock()
	for name, fn := range w.bindings {
		w.injectBinding(name, fn.Type())
	}
}

// Dispatch queues a function for main-thread execution.
func (w *WebView) Dispatch(f func()) {
	select {
	case w.dispatch <- f:
	default:
	}
}

// Destroy closes the browser.
func (w *WebView) Destroy() {
	if !w.browser.isZero() {
		runtimeCore.Close(w.browser)
		w.browser = BrowserHandle{}
	}
	viewsMu.Lock()
	delete(views, w.handle)
	viewsMu.Unlock()
}

// SetTitle sets the window title.
func (w *WebView) SetTitle(title string) {
	w.opts.Title = title
	if !w.browser.isZero() {
		runtimeCore.SetTitle(w.browser, title)
	}
}

// SetSize sets the window size.
func (w *WebView) SetSize(width, height int) {
	w.opts.Width = width
	w.opts.Height = height
	if !w.browser.isZero() {
		runtimeCore.Resize(w.browser, width, height)
	}
}

// Show shows the window.
func (w *WebView) Show() {
	if !w.browser.isZero() {
		runtimeCore.Show(w.browser)
	}
}

// Hide hides the window.
func (w *WebView) Hide() {
	if !w.browser.isZero() {
		runtimeCore.Hide(w.browser)
	}
}

// IsValid reports whether the browser handle remains valid.
func (w *WebView) IsValid() bool {
	if w.browser.isZero() {
		return false
	}
	return runtimeCore.IsValid(w.browser)
}

func (w *WebView) handleMessageFromJS(name, payload string) {
	w.mu.RLock()
	fn, exists := w.bindings[name]
	w.mu.RUnlock()
	if !exists {
		w.sendError("", fmt.Sprintf("Function %s not found", name))
		return
	}

	var data struct {
		ID   string        `json:"id"`
		Args []interface{} `json:"args"`
	}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		w.sendError(data.ID, err.Error())
		return
	}

	result, err := w.callFunction(fn, data.Args)
	if err != nil {
		w.sendError(data.ID, err.Error())
		return
	}

	w.sendResult(data.ID, result)
}

func (w *WebView) callFunction(fn reflect.Value, args []interface{}) (interface{}, error) {
	fnType := fn.Type()
	numIn := fnType.NumIn()
	if len(args) != numIn {
		return nil, fmt.Errorf("expected %d arguments, got %d", numIn, len(args))
	}

	in := make([]reflect.Value, numIn)
	for i := 0; i < numIn; i++ {
		argType := fnType.In(i)
		if args[i] == nil {
			in[i] = reflect.Zero(argType)
			continue
		}

		argValue := reflect.ValueOf(args[i])
		if !argValue.IsValid() {
			in[i] = reflect.Zero(argType)
			continue
		}
		if argValue.Type() != argType {
			if argValue.Type().ConvertibleTo(argType) {
				argValue = argValue.Convert(argType)
			} else {
				return nil, fmt.Errorf("argument %d: cannot convert %v to %v", i, argValue.Type(), argType)
			}
		}
		in[i] = argValue
	}

	out := fn.Call(in)
	if len(out) == 0 {
		return nil, nil
	}

	if last := out[len(out)-1]; last.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		if !last.IsNil() {
			return nil, last.Interface().(error)
		}
		out = out[:len(out)-1]
	}

	if len(out) == 1 {
		return out[0].Interface(), nil
	}

	results := make([]interface{}, len(out))
	for i, v := range out {
		results[i] = v.Interface()
	}
	return results, nil
}

func (w *WebView) sendResult(id string, result interface{}) {
	data, err := json.Marshal(result)
	if err != nil {
		data = []byte("null")
	}
	w.Eval(fmt.Sprintf("if (window.__goCallbacks && window.__goCallbacks['%s']) { window.__goCallbacks['%s'].resolve(%s); delete window.__goCallbacks['%s']; }", id, id, string(data), id))
}

func (w *WebView) sendError(id, errMsg string) {
	errMsg = strings.ReplaceAll(errMsg, "'", "\\'")
	w.Eval(fmt.Sprintf("if (window.__goCallbacks && window.__goCallbacks['%s']) { window.__goCallbacks['%s'].reject(new Error('%s')); delete window.__goCallbacks['%s']; }", id, id, errMsg, id))
}

func containsArg(args []string, needle string) bool {
	for _, arg := range args {
		if arg == needle {
			return true
		}
	}
	return false
}
