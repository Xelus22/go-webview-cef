//go:build linux
// +build linux

package cef

/*
#cgo CFLAGS: -I${SRCDIR}/../../third_party/cef/linux_64 -I${SRCDIR}/../cef_c
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/cef/linux_64/Release -lcef -lX11 -Wl,-rpath,'$ORIGIN'

#include <stdlib.h>
#include "cef_wrapper.h"

// Type definitions to match C header
typedef void* cef_browser_handle_t;

// Explicit declarations
extern void cef_set_args(int argc, char** argv);
extern int cef_initialize_main(int argc, char** argv);
extern void cef_run_message_loop_wrapper(void);
extern void cef_shutdown_wrapper(void);
extern cef_browser_handle_t cef_browser_create_wrapper(const char* url, int width, int height);
extern cef_browser_handle_t cef_browser_create_with_flags(const char* url, int width, int height, int chromeless, int frameless);
extern void cef_browser_load_url_wrapper(cef_browser_handle_t browser, const char* url);
extern void cef_browser_execute_js_wrapper(cef_browser_handle_t browser, const char* js);
extern void cef_browser_destroy_wrapper(cef_browser_handle_t browser);
extern void cef_browser_resize_wrapper(cef_browser_handle_t browser, int width, int height);

#include "cef_wrapper.c"
*/
import "C"
import (
	"os"
	"unsafe"
)

// disableGPUFlags stores whether GPU should be disabled
var disableGPUFlags = false

// DisableGPU adds flags to disable GPU rendering (useful for WSL/VMs)
func DisableGPU() {
	disableGPUFlags = true
}

// Initialize initializes CEF and handles subprocess detection
// Returns: true if this is the browser process (continue running)
//
//	false if this is a subprocess (should exit)
func Initialize() bool {
	// Add GPU disable flags if requested (must be before CEF subprocess spawns)
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

	// Call CEF initialize main
	ret := C.cef_initialize_main(C.int(argc), &cArgs[0])

	// Note: We don't free cArgs because CEF keeps references to them
	// for the lifetime of the process

	// ret >= 0 means subprocess - should exit with that code
	// ret == -1 means browser process - continue
	// ret == -2 means error
	if ret >= 0 {
		// Subprocess - exit
		os.Exit(int(ret))
	}

	return ret == -1
}

// Run starts the CEF message loop
func Run() {
	C.cef_run_message_loop_wrapper()
}

// Shutdown cleans up CEF resources
func Shutdown() {
	C.cef_shutdown_wrapper()
}

// Browser represents a CEF browser instance
type Browser struct {
	ptr C.cef_browser_handle_t
}

// NewBrowser creates a new browser window
func NewBrowser(url string, width, height int) *Browser {
	return NewBrowserChromeless(url, width, height, false)
}

// NewBrowserChromeless creates a new browser window with optional frameless mode
// frameless=true removes OS window decorations (title bar, borders)
func NewBrowserChromeless(url string, width, height int, frameless bool) *Browser {
	curl := C.CString(url)
	defer C.free(unsafe.Pointer(curl))

	var framelessInt C.int = 0
	if frameless {
		framelessInt = 1
	}

	ptr := C.cef_browser_create_with_flags(curl, C.int(width), C.int(height), 0, framelessInt)
	if ptr == nil {
		return nil
	}

	return &Browser{ptr: ptr}
}

// Navigate loads a new URL
func (b *Browser) Navigate(url string) {
	if b.ptr == nil {
		return
	}
	curl := C.CString(url)
	defer C.free(unsafe.Pointer(curl))
	C.cef_browser_load_url_wrapper(b.ptr, curl)
}

// Eval executes JavaScript in the browser
func (b *Browser) Eval(js string) {
	if b.ptr == nil {
		return
	}
	cjs := C.CString(js)
	defer C.free(unsafe.Pointer(cjs))
	C.cef_browser_execute_js_wrapper(b.ptr, cjs)
}

// Destroy closes and cleans up the browser
func (b *Browser) Destroy() {
	if b.ptr == nil {
		return
	}
	C.cef_browser_destroy_wrapper(b.ptr)
	b.ptr = nil
}

// Resize resizes the browser window
func (b *Browser) Resize(width, height int) {
	if b.ptr == nil {
		return
	}
	C.cef_browser_resize_wrapper(b.ptr, C.int(width), C.int(height))
}
