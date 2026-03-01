//go:build windows
// +build windows

package cef

// #cgo CFLAGS: -I${SRCDIR}/../../third_party/cef/windows_64/include
// #cgo LDFLAGS: -L${SRCDIR}/../../third_party/cef/windows_64/Release -lcef
// #include <stdlib.h>
// #include "cef_wrapper.h"
import "C"
import "unsafe"

// Initialize initializes the CEF framework
func Initialize() {
	C.cef_wrapper_initialize()
}

// InitializeWithBrowser initializes CEF and queues a browser for creation
func InitializeWithBrowser(url string, width, height int) {
	curl := C.CString(url)
	defer C.free(unsafe.Pointer(curl))
	C.cef_wrapper_initialize_with_browser(curl, C.int(width), C.int(height))
}

// Run starts the CEF message loop
func Run() {
	C.cef_wrapper_run()
}

// Shutdown cleans up CEF resources
func Shutdown() {
	C.cef_wrapper_shutdown()
}

// IsSubprocess returns true if running as a CEF subprocess
func IsSubprocess() bool {
	return C.cef_is_subprocess() != 0
}

// SubprocessEntry is the entry point for CEF subprocesses
func SubprocessEntry() {
	C.cef_subprocess_entry()
}

// SetArgs sets the command line arguments for CEF
func SetArgs(argc int, argv []string) {
	// Windows-specific flags
	winFlags := []string{
		"--no-sandbox",
		"--disable-gpu",
	}

	newArgv := append(argv, winFlags...)
	argc = len(newArgv)

	// Convert Go strings to C strings
	cArgs := make([]*C.char, len(newArgv))
	for i, arg := range newArgv {
		cArgs[i] = C.CString(arg)
	}

	// Call C function to store args (C function copies them)
	C.cef_set_args(C.int(argc), &cArgs[0])

	// Free the temporary C strings
	for _, arg := range cArgs {
		C.free(unsafe.Pointer(arg))
	}
}
