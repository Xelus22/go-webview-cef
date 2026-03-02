// CEF Runtime - Minimal CEF WebView runtime boundary
// This provides a clean C API for Go to call without exposing CEF structs
//
// Architecture:
// - Only CefBrowserHost::CreateBrowser is used (NO Views API)
// - Native window with WS_OVERLAPPEDWINDOW (OS window buttons)
// - Subprocess support via settings.browser_subprocess_path
// - Proper cache path configuration

#ifndef CEF_RUNTIME_H
#define CEF_RUNTIME_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>

// Opaque handle to browser instance
typedef void* cef_runtime_browser_t;

// Callback types for IPC
typedef void (*cef_runtime_message_cb_t)(const char* name, const char* payload, void* user_data);
typedef void (*cef_runtime_load_cb_t)(int success, void* user_data);

//
// Lifecycle functions
//

// Initialize CEF runtime (call FIRST in main)
// Returns: >=0 if subprocess (exit with this code)
//          -1 if browser process (continue and call cef_run())
int cef_runtime_initialize(int argc, char** argv);

// Run the CEF message loop (browser process only)
void cef_runtime_run(void);

// Shutdown CEF runtime
void cef_runtime_shutdown(void);

//
// Browser functions
//

// Browser creation options
typedef struct {
    const char* url;           // Initial URL (required)
    const char* title;         // Window title (NULL for default)
    int width;                 // Window width (default: 800)
    int height;                // Window height (default: 600)
    int frameless;             // 1 = no OS decorations (default: 0)
    int chromeless;            // 1 = no Chrome UI like URL bar, tabs (default: 0)
    int resizable;             // 1 = resizable (default: 1)
    int fullscreen;            // 1 = fullscreen (default: 0)
    int maximized;             // 1 = maximized (default: 0)
    int x;                     // Window X position (default: centered)
    int y;                     // Window Y position (default: centered)
} cef_runtime_browser_opts_t;

// Default options helper
#define CEF_RUNTIME_BROWSER_OPTS_DEFAULT { \
    .url = "about:blank", \
    .title = NULL, \
    .width = 800, \
    .height = 600, \
    .frameless = 0, \
    .chromeless = 0, \
    .resizable = 1, \
    .fullscreen = 0, \
    .maximized = 0, \
    .x = -1, \
    .y = -1 \
}

// Create a browser window
// Returns opaque handle or NULL on failure
cef_runtime_browser_t cef_runtime_create_browser(const cef_runtime_browser_opts_t* opts);

// Navigate to URL
void cef_runtime_navigate(cef_runtime_browser_t browser, const char* url);

// Execute JavaScript
void cef_runtime_eval(cef_runtime_browser_t browser, const char* js);

// Set window title
void cef_runtime_set_title(cef_runtime_browser_t browser, const char* title);

// Resize window
void cef_runtime_resize(cef_runtime_browser_t browser, int width, int height);

// Show/hide window
void cef_runtime_show(cef_runtime_browser_t browser);
void cef_runtime_hide(cef_runtime_browser_t browser);

// Close and destroy browser
void cef_runtime_close(cef_runtime_browser_t browser);

// Check if browser is still valid
int cef_runtime_is_valid(cef_runtime_browser_t browser);

//
// IPC functions (JS ↔ Go bridge)
//

// Register callback for messages from JavaScript
// Called when JS calls: window.go.invoke(name, payload)
void cef_runtime_set_message_callback(cef_runtime_message_cb_t callback, void* user_data);

// Send message to JavaScript
// Calls: window.__go_dispatch(name, payload)
void cef_runtime_send_message(cef_runtime_browser_t browser, const char* name, const char* payload);

// Inject JavaScript that sets up window.go namespace
// Call after browser creation to enable JS bindings
void cef_runtime_inject_bridge(cef_runtime_browser_t browser);

//
// Utility functions
//

// Get CEF version string
const char* cef_runtime_version(void);

// Set log level (0=verbose, 1=info, 2=warning, 3=error, 4=fatal)
void cef_runtime_set_log_level(int level);

// Enable/disable developer tools
void cef_runtime_set_dev_tools(cef_runtime_browser_t browser, int enabled);

//
// Event callbacks
//

// Set callback for when page finishes loading
void cef_runtime_set_load_callback(cef_runtime_load_cb_t callback, void* user_data);

//
// Process message handler for IPC
// Uses CefMessageRouter for JS->Go communication
//

// Called from render process to send message to browser process
void cef_runtime_post_message(const char* message);

#ifdef __cplusplus
}
#endif

#endif // CEF_RUNTIME_H
