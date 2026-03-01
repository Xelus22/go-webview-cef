package cef

// #cgo CFLAGS: -I${SRCDIR}/../../third_party/cef/linux_64
// #include <stdlib.h>
// #include "cef_wrapper.h"
import "C"
import (
	"runtime"
	"sync"
	"unsafe"
)

// Browser represents a CEF browser instance
type Browser struct {
	ptr    *C.cef_browser_t
	mu     sync.RWMutex
	closed bool
}

// browserRegistry tracks active browsers
var browserRegistry = &struct {
	sync.RWMutex
	browsers map[uintptr]*Browser
}{browsers: make(map[uintptr]*Browser)}

// NewBrowser creates a new browser window
func NewBrowser(url string, width, height int) *Browser {
	curl := C.CString(url)
	defer C.free(unsafe.Pointer(curl))

	ptr := C.cef_browser_create(curl, C.int(width), C.int(height))
	if ptr == nil {
		return nil
	}

	b := &Browser{ptr: ptr}
	runtime.SetFinalizer(b, (*Browser).Destroy)

	browserRegistry.Lock()
	browserRegistry.browsers[uintptr(unsafe.Pointer(ptr))] = b
	browserRegistry.Unlock()

	return b
}

// Navigate loads a new URL
func (b *Browser) Navigate(url string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed || b.ptr == nil {
		return
	}

	curl := C.CString(url)
	defer C.free(unsafe.Pointer(curl))
	C.cef_browser_load_url(b.ptr, curl)
}

// Eval executes JavaScript in the browser
func (b *Browser) Eval(js string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed || b.ptr == nil {
		return
	}

	cjs := C.CString(js)
	defer C.free(unsafe.Pointer(cjs))
	C.cef_browser_execute_js(b.ptr, cjs)
}

// Destroy closes and cleans up the browser
func (b *Browser) Destroy() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed || b.ptr == nil {
		return
	}

	browserRegistry.Lock()
	delete(browserRegistry.browsers, uintptr(unsafe.Pointer(b.ptr)))
	browserRegistry.Unlock()

	C.cef_browser_destroy(b.ptr)
	b.ptr = nil
	b.closed = true
}

// NativeHandle returns the platform-specific window handle
func (b *Browser) NativeHandle() unsafe.Pointer {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed || b.ptr == nil {
		return nil
	}

	return C.cef_get_native_window_handle(b.ptr)
}
