//go:build darwin
// +build darwin

package cefshim

/*
#cgo CFLAGS: -I${SRCDIR}/../../runtime -I${SRCDIR}/../../third_party/cef/mac_64
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/cef/mac_64/Release -lcef -framework Cocoa -Wl,-rpath,@executable_path/../Frameworks

#include <stdlib.h>
#include <stdint.h>
#include "cef_runtime.h"

extern void goCEFMessageCallback(void* browser, char* name, char* payload, void* user_data);
extern void goCEFLoadCallback(void* browser, int success, void* user_data);
extern void goCEFCloseCallback(void* browser, void* user_data);
extern int goCEFRequestCallback(void* browser, char* method, char* url, char* request_headers_json,
	unsigned char* request_body, size_t request_body_len,
	int* status_code, char** status_text, char** mime_type, char** response_headers_json,
	unsigned char** response_body, size_t* response_body_len,
	void* user_data);

static void message_callback_wrapper(cef_runtime_browser_t browser, const char* name, const char* payload, void* user_data) {
	goCEFMessageCallback(browser, (char*)name, (char*)payload, user_data);
}

static cef_runtime_message_cb_t get_message_callback() {
	return (cef_runtime_message_cb_t)message_callback_wrapper;
}

static void load_callback_wrapper(cef_runtime_browser_t browser, int success, void* user_data) {
	goCEFLoadCallback(browser, success, user_data);
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

static int request_callback_wrapper(cef_runtime_browser_t browser, const char* method, const char* url, const char* request_headers_json,
	const uint8_t* request_body, size_t request_body_len,
	int* status_code, char** status_text, char** mime_type, char** response_headers_json,
	uint8_t** response_body, size_t* response_body_len,
	void* user_data) {
	return goCEFRequestCallback(browser, (char*)method, (char*)url, (char*)request_headers_json,
		(unsigned char*)request_body, request_body_len,
		status_code, status_text, mime_type, response_headers_json,
		(unsigned char**)response_body, response_body_len,
		user_data);
}

static cef_runtime_request_cb_t get_request_callback() {
	return (cef_runtime_request_cb_t)request_callback_wrapper;
}
*/
import "C"

import (
	"sync"
	"unsafe"

	"github.com/xelus/go-webview-cef/internal/cefbindings"
)

// Request represents an intercepted browser request.
type Request struct {
	Method      string
	URL         string
	HeadersJSON string
	Body        []byte
}

// RequestResponse is the in-process response for a handled request.
type RequestResponse struct {
	StatusCode  int
	StatusText  string
	MimeType    string
	HeadersJSON string
	Body        []byte
}

var (
	callbackMu sync.RWMutex

	messageCallback func(browser cefbindings.BrowserHandle, name, payload string, userData uintptr)
	loadCallback    func(browser cefbindings.BrowserHandle, success bool, userData uintptr)
	closeCallback   func(browser cefbindings.BrowserHandle, userData uintptr)
	requestCallback func(browser cefbindings.BrowserHandle, req Request, userData uintptr) (RequestResponse, bool)

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

func fromCBrowser(browser unsafe.Pointer) cefbindings.BrowserHandle {
	return cefbindings.BrowserHandle(uintptr(browser))
}

// SetMessageDispatch sets the Go message dispatcher for JS -> Go callbacks.
func SetMessageDispatch(dispatcher func(browser cefbindings.BrowserHandle, name, payload string, userData uintptr)) {
	callbackMu.Lock()
	messageCallback = dispatcher
	callbackMu.Unlock()
}

//export goCEFMessageCallback
func goCEFMessageCallback(browser unsafe.Pointer, name *C.char, payload *C.char, userData unsafe.Pointer) {
	callbackMu.RLock()
	cb := messageCallback
	callbackMu.RUnlock()
	if cb == nil {
		return
	}
	cb(fromCBrowser(browser), C.GoString(name), C.GoString(payload), uintptr(userData))
}

// SetLoadDispatch sets the Go load dispatcher for load completion callbacks.
func SetLoadDispatch(dispatcher func(browser cefbindings.BrowserHandle, success bool, userData uintptr)) {
	callbackMu.Lock()
	loadCallback = dispatcher
	callbackMu.Unlock()
}

//export goCEFLoadCallback
func goCEFLoadCallback(browser unsafe.Pointer, success C.int, userData unsafe.Pointer) {
	callbackMu.RLock()
	cb := loadCallback
	callbackMu.RUnlock()
	if cb == nil {
		return
	}
	cb(fromCBrowser(browser), success != 0, uintptr(userData))
}

// SetCloseDispatch sets the Go close dispatcher for browser close callbacks.
func SetCloseDispatch(dispatcher func(browser cefbindings.BrowserHandle, userData uintptr)) {
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
	cb(fromCBrowser(browser), uintptr(userData))
}

// SetRequestDispatch sets the Go request dispatcher for in-process request handling.
func SetRequestDispatch(dispatcher func(browser cefbindings.BrowserHandle, req Request, userData uintptr) (RequestResponse, bool)) {
	callbackMu.Lock()
	requestCallback = dispatcher
	callbackMu.Unlock()
}

//export goCEFRequestCallback
func goCEFRequestCallback(
	browser unsafe.Pointer,
	method *C.char,
	url *C.char,
	requestHeadersJSON *C.char,
	requestBody *C.uchar,
	requestBodyLen C.size_t,
	statusCode *C.int,
	statusText **C.char,
	mimeType **C.char,
	responseHeadersJSON **C.char,
	responseBody **C.uchar,
	responseBodyLen *C.size_t,
	userData unsafe.Pointer,
) C.int {
	callbackMu.RLock()
	cb := requestCallback
	callbackMu.RUnlock()
	if cb == nil {
		return 0
	}

	req := Request{
		Method:      C.GoString(method),
		URL:         C.GoString(url),
		HeadersJSON: C.GoString(requestHeadersJSON),
	}

	if requestBody != nil && requestBodyLen > 0 {
		req.Body = C.GoBytes(unsafe.Pointer(requestBody), C.int(requestBodyLen))
	}

	resp, handled := cb(fromCBrowser(browser), req, uintptr(userData))
	if !handled {
		return 0
	}

	if statusCode != nil {
		code := resp.StatusCode
		if code <= 0 {
			code = 200
		}
		*statusCode = C.int(code)
	}
	if statusText != nil && resp.StatusText != "" {
		*statusText = C.CString(resp.StatusText)
	}
	if mimeType != nil && resp.MimeType != "" {
		*mimeType = C.CString(resp.MimeType)
	}
	if responseHeadersJSON != nil && resp.HeadersJSON != "" {
		*responseHeadersJSON = C.CString(resp.HeadersJSON)
	}
	if responseBody != nil && responseBodyLen != nil {
		if len(resp.Body) > 0 {
			*responseBody = (*C.uchar)(C.CBytes(resp.Body))
			*responseBodyLen = C.size_t(len(resp.Body))
		} else {
			*responseBody = nil
			*responseBodyLen = 0
		}
	}

	return 1
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
		url:           curl,
		title:         ctitle,
		width:         C.int(opts.Width),
		height:        C.int(opts.Height),
		frameless:     cbool(opts.Frameless),
		chromeless:    cbool(opts.Chromeless),
		resizable:     cbool(opts.Resizable),
		fullscreen:    cbool(opts.Fullscreen),
		maximized:     cbool(opts.Maximized),
		x:             C.int(opts.X),
		y:             C.int(opts.Y),
		parent_window: C.uintptr_t(opts.ParentWindow),
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

// SetCloseCallbackUserData installs the C close callback wrapper with user data.
func SetCloseCallbackUserData(userData uintptr) {
	C.cef_runtime_set_close_callback(C.get_close_callback(), unsafe.Pointer(userData))
}

// SetRequestCallbackUserData installs the C request callback wrapper with user data.
func SetRequestCallbackUserData(userData uintptr) {
	C.cef_runtime_set_request_callback(C.get_request_callback(), unsafe.Pointer(userData))
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
