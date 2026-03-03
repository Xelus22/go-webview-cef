//go:build linux
// +build linux

package cefshim

/*
#cgo CFLAGS: -I${SRCDIR}/../../runtime -I${SRCDIR}/../../third_party/cef/linux_64
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/cef/linux_64/Release -lcef -lX11 -Wl,-rpath,'$ORIGIN'

#include <stdlib.h>
#include "cef_runtime.h"

extern void goCEFMessageCallback(char* name, char* payload, void* user_data);
extern void goCEFLoadCallback(int success, void* user_data);
extern void goCEFCloseCallback(void* browser, void* user_data);

static void message_callback_wrapper(const char* name, const char* payload, void* user_data) {
    goCEFMessageCallback((char*)name, (char*)payload, user_data);
}

static cef_runtime_message_cb_t get_message_callback() {
    return (cef_runtime_message_cb_t)message_callback_wrapper;
}

static void load_callback_wrapper(int success, void* user_data) {
    goCEFLoadCallback(success, user_data);
}

static cef_runtime_load_cb_t get_load_callback() {
    return (cef_runtime_load_cb_t)load_callback_wrapper;
}

static void close_callback_wrapper(cef_runtime_browser_t browser, void* user_data) {
    goCEFCloseCallback(browser, user_data);
}

static cef_runtime_close_cb_t get_close_callback() {
    return (cef_runtime_close_cb_t)close_callback_wrapper;
}
*/
import "C"

import (
	"sync"
	"unsafe"

	"github.com/xelus/go-webview-cef/internal/cefbindings"
)

var (
	callbackMu      sync.RWMutex
	messageCallback func(name, payload string, userData uintptr)
	loadCallback    func(success bool, userData uintptr)
	closeCallback   func(userData uintptr)

	argsMu   sync.Mutex
	initArgs []*C.char
)

func cbool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

func toCBrowser(h cefbindings.BrowserHandle) C.cef_runtime_browser_t {
	return C.cef_runtime_browser_t(unsafe.Pointer(uintptr(h)))
}

// SetMessageDispatch sets the Go message dispatcher for JS -> Go callbacks.
func SetMessageDispatch(dispatcher func(name, payload string, userData uintptr)) {
	callbackMu.Lock()
	messageCallback = dispatcher
	callbackMu.Unlock()
}

//export goCEFMessageCallback
func goCEFMessageCallback(name *C.char, payload *C.char, userData unsafe.Pointer) {
	callbackMu.RLock()
	cb := messageCallback
	callbackMu.RUnlock()
	if cb == nil {
		return
	}
	cb(C.GoString(name), C.GoString(payload), uintptr(userData))
}

// SetLoadDispatch sets the Go load dispatcher for load completion callbacks.
func SetLoadDispatch(dispatcher func(success bool, userData uintptr)) {
	callbackMu.Lock()
	loadCallback = dispatcher
	callbackMu.Unlock()
}

//export goCEFLoadCallback
func goCEFLoadCallback(success C.int, userData unsafe.Pointer) {
	callbackMu.RLock()
	cb := loadCallback
	callbackMu.RUnlock()
	if cb == nil {
		return
	}
	cb(success != 0, uintptr(userData))
}

// SetCloseDispatch sets the Go close dispatcher for browser close callbacks.
func SetCloseDispatch(dispatcher func(userData uintptr)) {
	callbackMu.Lock()
	closeCallback = dispatcher
	callbackMu.Unlock()
}

//export goCEFCloseCallback
func goCEFCloseCallback(browser unsafe.Pointer, userData unsafe.Pointer) {
	callbackMu.RLock()
	cb := closeCallback
	callbackMu.RUnlock()
	if cb == nil {
		return
	}
	_ = browser // Not used in this minimal version
	cb(uintptr(userData))
}

// Initialize initializes the CEF runtime and executes subprocesses when needed.
func Initialize(args []string) int {
	argsMu.Lock()
	defer argsMu.Unlock()

	for _, arg := range initArgs {
		if arg != nil {
			C.free(unsafe.Pointer(arg))
		}
	}
	initArgs = initArgs[:0]

	if len(args) == 0 {
		ret := C.cef_runtime_initialize(0, nil)
		return int(ret)
	}

	initArgs = make([]*C.char, len(args))
	for i, arg := range args {
		initArgs[i] = C.CString(arg)
	}

	ret := C.cef_runtime_initialize(C.int(len(initArgs)), (**C.char)(unsafe.Pointer(&initArgs[0])))
	return int(ret)
}

// Run runs the main CEF message loop.
func Run() {
	C.cef_runtime_run()
}

// DoMessageLoopWork processes a single CEF message loop iteration.
func DoMessageLoopWork() {
	C.cef_runtime_do_message_loop_work()
}

// Shutdown shuts down CEF and releases shim-owned argv memory.
func Shutdown() {
	C.cef_runtime_shutdown()

	argsMu.Lock()
	for _, arg := range initArgs {
		if arg != nil {
			C.free(unsafe.Pointer(arg))
		}
	}
	initArgs = nil
	argsMu.Unlock()
}

// CreateBrowser creates a new browser instance.
func CreateBrowser(opts cefbindings.BrowserOptions) cefbindings.BrowserHandle {
	url := opts.URL
	if url == "" {
		url = "about:blank"
	}

	curl := C.CString(url)
	defer C.free(unsafe.Pointer(curl))

	var ctitle *C.char
	if opts.Title != "" {
		ctitle = C.CString(opts.Title)
		defer C.free(unsafe.Pointer(ctitle))
	}

	copts := C.cef_runtime_browser_opts_t{
		url:        curl,
		title:      ctitle,
		width:      C.int(opts.Width),
		height:     C.int(opts.Height),
		frameless:  cbool(opts.Frameless),
		chromeless: cbool(opts.Chromeless),
		resizable:  cbool(opts.Resizable),
		fullscreen: cbool(opts.Fullscreen),
		maximized:  cbool(opts.Maximized),
		x:          C.int(opts.X),
		y:          C.int(opts.Y),
	}

	handle := C.cef_runtime_create_browser(&copts)
	return cefbindings.BrowserHandle(uintptr(handle))
}

// SetMessageCallbackUserData installs the C callback wrapper with user data.
func SetMessageCallbackUserData(userData uintptr) {
	C.cef_runtime_set_message_callback(C.get_message_callback(), unsafe.Pointer(userData))
}

// SetLoadCallbackUserData installs the C load callback wrapper with user data.
func SetLoadCallbackUserData(userData uintptr) {
	C.cef_runtime_set_load_callback(C.get_load_callback(), unsafe.Pointer(userData))
}

// InjectBridge injects the default JS bridge helpers into the current page.
func InjectBridge(browser cefbindings.BrowserHandle) {
	if browser == 0 {
		return
	}
	C.cef_runtime_inject_bridge(toCBrowser(browser))
}

// Navigate loads a URL into the browser.
func Navigate(browser cefbindings.BrowserHandle, url string) {
	if browser == 0 {
		return
	}
	curl := C.CString(url)
	defer C.free(unsafe.Pointer(curl))
	C.cef_runtime_navigate(toCBrowser(browser), curl)
}

// Eval executes JavaScript in the browser.
func Eval(browser cefbindings.BrowserHandle, js string) {
	if browser == 0 {
		return
	}
	cjs := C.CString(js)
	defer C.free(unsafe.Pointer(cjs))
	C.cef_runtime_eval(toCBrowser(browser), cjs)
}

// Close closes the browser window.
func Close(browser cefbindings.BrowserHandle) {
	if browser == 0 {
		return
	}
	C.cef_runtime_close(toCBrowser(browser))
}

// IsValid reports whether a browser handle remains valid.
func IsValid(browser cefbindings.BrowserHandle) bool {
	if browser == 0 {
		return false
	}
	return C.cef_runtime_is_valid(toCBrowser(browser)) != 0
}

// SetTitle updates the window title.
func SetTitle(browser cefbindings.BrowserHandle, title string) {
	if browser == 0 {
		return
	}
	ctitle := C.CString(title)
	defer C.free(unsafe.Pointer(ctitle))
	C.cef_runtime_set_title(toCBrowser(browser), ctitle)
}

// Resize updates the window content bounds.
func Resize(browser cefbindings.BrowserHandle, width, height int) {
	if browser == 0 {
		return
	}
	C.cef_runtime_resize(toCBrowser(browser), C.int(width), C.int(height))
}

// Show shows the browser window.
func Show(browser cefbindings.BrowserHandle) {
	if browser == 0 {
		return
	}
	C.cef_runtime_show(toCBrowser(browser))
}

// Hide hides the browser window.
func Hide(browser cefbindings.BrowserHandle) {
	if browser == 0 {
		return
	}
	C.cef_runtime_hide(toCBrowser(browser))
}

// SendMessage dispatches a named event to JavaScript.
func SendMessage(browser cefbindings.BrowserHandle, name, payload string) {
	if browser == 0 {
		return
	}
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))

	var cpayload *C.char
	if payload != "" {
		cpayload = C.CString(payload)
		defer C.free(unsafe.Pointer(cpayload))
	}

	C.cef_runtime_send_message(toCBrowser(browser), cname, cpayload)
}
