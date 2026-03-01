#ifndef CEF_WRAPPER_H
#define CEF_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>

// Opaque handles
typedef void* cef_browser_t;
typedef void* cef_callback_t;

// Lifecycle
void cef_initialize();
void cef_run();
void cef_shutdown();

// Browser
cef_browser_t cef_browser_create(const char* url, int width, int height);
void cef_browser_load_url(cef_browser_t browser, const char* url);
void cef_browser_execute_js(cef_browser_t browser, const char* js);
void cef_browser_destroy(cef_browser_t browser);

// Window handle (platform-specific)
void* cef_get_native_window_handle(cef_browser_t browser);

// Messaging bridge
void cef_register_js_callback(const char* name, cef_callback_t callback);
void cef_send_message_to_js(cef_browser_t browser, const char* message);

// Process helpers
int cef_is_subprocess();
void cef_subprocess_entry();

#ifdef __cplusplus
}
#endif

#endif // CEF_WRAPPER_H
