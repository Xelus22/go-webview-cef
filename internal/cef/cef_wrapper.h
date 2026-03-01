#ifndef CEF_WRAPPER_H
#define CEF_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>

// Forward declarations
typedef struct _cef_browser_t cef_browser_t;

// Callback function type for JS bindings
typedef void (*cef_js_callback_fn)(const char* name, const char* args);

// Set argc/argv before calling cef_wrapper_initialize
void cef_set_args(int argc, char** argv);

// Lifecycle
int cef_wrapper_initialize();
int cef_wrapper_initialize_with_browser(const char* url, int width, int height);
void cef_wrapper_run();
void cef_wrapper_shutdown();

// Browser
cef_browser_t* cef_browser_create(const char* url, int width, int height);
void cef_browser_load_url(cef_browser_t* browser, const char* url);
void cef_browser_execute_js(cef_browser_t* browser, const char* js);
void cef_browser_destroy(cef_browser_t* browser);

// Window handle (platform-specific)
void* cef_get_native_window_handle(cef_browser_t* browser);

// Messaging bridge
void cef_register_js_callback(const char* name, cef_js_callback_fn callback);
void cef_send_message_to_js(cef_browser_t* browser, const char* message);

// Process helpers
int cef_is_subprocess();
void cef_subprocess_entry();

#ifdef __cplusplus
}
#endif

#endif // CEF_WRAPPER_H
