// CEF 145 Wrapper - Based on cef-rs architecture
// Key insight: execute_process with NULL app handles subprocesses internally

#ifndef CEF_WRAPPER_H
#define CEF_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>

// Opaque handle types
typedef void* cef_browser_handle_t;

//
// Lifecycle functions
//

// Initialize CEF (call this FIRST in main)
// Returns: >=0 if this is a subprocess (exit with this code)
//          -1 if this is the browser process (continue and call cef_run())
int cef_initialize_main(int argc, char** argv);

// Run the message loop (browser process only)
void cef_run_message_loop_wrapper(void);

// Shutdown CEF (browser process only)
void cef_shutdown_wrapper(void);

//
// Browser functions
//

// Create browser (call after cef_initialize_main returns -1)
// Returns opaque handle or NULL
cef_browser_handle_t cef_browser_create_wrapper(const char* url, int width, int height);

// Create browser with flags
// chromeless: 1 = no browser decorations (no URL bar, etc.)
// frameless:  1 = no OS window decorations (no title bar, no borders)
cef_browser_handle_t cef_browser_create_with_flags(const char* url, int width, int height, int chromeless, int frameless);

// Navigate to URL
void cef_browser_load_url_wrapper(cef_browser_handle_t browser, const char* url);

// Execute JavaScript
void cef_browser_execute_js_wrapper(cef_browser_handle_t browser, const char* js);

// Close browser
void cef_browser_destroy_wrapper(cef_browser_handle_t browser);

// Resize browser window
void cef_browser_resize_wrapper(cef_browser_handle_t browser, int width, int height);

#ifdef __cplusplus
}
#endif

#endif // CEF_WRAPPER_H
