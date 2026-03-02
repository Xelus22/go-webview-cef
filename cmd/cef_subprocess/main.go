//go:build linux
// +build linux

// cef_subprocess - Minimal CEF subprocess for renderer processes
package main

/*
#cgo CFLAGS: -I${SRCDIR}/../../third_party/cef/linux_64
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/cef/linux_64/Release -lcef -Wl,-rpath,'$ORIGIN'

#include <stdlib.h>
#include "include/capi/cef_app_capi.h"
#include "include/cef_api_hash.h"
*/
import "C"
import (
	"os"
	"runtime"
	"unsafe"
)

func main() {
	// Lock the OS thread since CEF uses thread-local storage
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Get command line args
	argc := len(os.Args)

	// Allocate C array for argv - must keep alive during CEF call
	cArgs := make([]*C.char, argc)
	for i, arg := range os.Args {
		cArgs[i] = C.CString(arg)
		defer C.free(unsafe.Pointer(cArgs[i]))
	}

	// Create main_args on C heap to avoid Go pointer issues
	cArgv := C.malloc(C.size_t(argc) * C.size_t(unsafe.Sizeof(uintptr(0))))
	defer C.free(cArgv)

	// Copy argv pointers to C memory
	cArgvSlice := (*[1 << 30]*C.char)(cArgv)[:argc:argc]
	for i := 0; i < argc; i++ {
		cArgvSlice[i] = cArgs[i]
	}

	// Create main_args
	mainArgs := C.cef_main_args_t{
		argc: C.int(argc),
		argv: (**C.char)(cArgv),
	}

	// Execute subprocess - this handles renderer, GPU, etc.
	exitCode := C.cef_execute_process(&mainArgs, nil, nil)

	// If exitCode >= 0, this is a subprocess and we should exit
	if exitCode >= 0 {
		os.Exit(int(exitCode))
	}

	// Should not reach here for subprocesses
	os.Exit(0)
}
