#include "cef_wrapper.h"

#include <string.h>
#include <stdlib.h>
#include <stdio.h>

#include "include/capi/cef_app_capi.h"
#include "include/capi/cef_browser_capi.h"
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_command_line_capi.h"
#include "include/capi/cef_life_span_handler_capi.h"
#include "include/capi/cef_base_capi.h"

// Maximum number of browsers we can track
#define MAX_BROWSERS 64

// Browser entry in our tracking array
typedef struct {
    cef_browser_t* browser;
    int active;
} browser_entry_t;

// Global state
static int g_initialized = 0;
static cef_app_t* g_app = NULL;
static browser_entry_t g_browsers[MAX_BROWSERS];
static cef_callback_t g_js_callback = NULL;

// Forward declarations
static void initialize_browser_array();
static int add_browser(cef_browser_t* browser);
static void remove_browser(cef_browser_t* browser);
static cef_client_t* create_client();
static cef_life_span_handler_t* create_life_span_handler();

// Base ref counting helpers
static int CEF_CALLBACK add_ref(cef_base_ref_counted_t* self) {
    return ++(self->ref_count);
}

static int CEF_CALLBACK release(cef_base_ref_counted_t* self) {
    int new_count = --(self->ref_count);
    if (new_count == 0) {
        free(self);
    }
    return new_count;
}

static int CEF_CALLBACK has_one_ref(cef_base_ref_counted_t* self) {
    return self->ref_count == 1;
}

static int CEF_CALLBACK has_at_least_one_ref(cef_base_ref_counted_t* self) {
    return self->ref_count >= 1;
}

static void init_base_ref_counted(cef_base_ref_counted_t* base, size_t size) {
    memset(base, 0, size);
    base->size = size;
    base->add_ref = add_ref;
    base->release = release;
    base->has_one_ref = has_one_ref;
    base->has_at_least_one_ref = has_at_least_one_ref;
    base->ref_count = 1;
}

// App callbacks
static void CEF_CALLBACK on_before_command_line_processing(
    struct _cef_app_t* self,
    const cef_string_t* process_type,
    struct _cef_command_line_t* command_line) {
    // No special command line processing needed
}

static void CEF_CALLBACK on_register_custom_schemes(
    struct _cef_app_t* self,
    struct _cef_scheme_registrar_t* registrar) {
    // No custom schemes
}

static struct _cef_resource_bundle_handler_t* CEF_CALLBACK get_resource_bundle_handler(
    struct _cef_app_t* self) {
    return NULL;
}

static struct _cef_browser_process_handler_t* CEF_CALLBACK get_browser_process_handler(
    struct _cef_app_t* self) {
    return NULL;
}

static struct _cef_render_process_handler_t* CEF_CALLBACK get_render_process_handler(
    struct _cef_app_t* self) {
    return NULL;
}

// Client callbacks
static struct _cef_context_menu_handler_t* CEF_CALLBACK get_context_menu_handler(
    struct _cef_client_t* self) {
    return NULL;
}

static struct _cef_dialog_handler_t* CEF_CALLBACK get_dialog_handler(
    struct _cef_client_t* self) {
    return NULL;
}

static struct _cef_display_handler_t* CEF_CALLBACK get_display_handler(
    struct _cef_client_t* self) {
    return NULL;
}

static struct _cef_download_handler_t* CEF_CALLBACK get_download_handler(
    struct _cef_client_t* self) {
    return NULL;
}

static struct _cef_drag_handler_t* CEF_CALLBACK get_drag_handler(
    struct _cef_client_t* self) {
    return NULL;
}

static struct _cef_find_handler_t* CEF_CALLBACK get_find_handler(
    struct _cef_client_t* self) {
    return NULL;
}

static struct _cef_focus_handler_t* CEF_CALLBACK get_focus_handler(
    struct _cef_client_t* self) {
    return NULL;
}

static struct _cef_jsdialog_handler_t* CEF_CALLBACK get_jsdialog_handler(
    struct _cef_client_t* self) {
    return NULL;
}

static struct _cef_keyboard_handler_t* CEF_CALLBACK get_keyboard_handler(
    struct _cef_client_t* self) {
    return NULL;
}

static struct _cef_life_span_handler_t* CEF_CALLBACK get_life_span_handler(
    struct _cef_client_t* self);

static struct _cef_load_handler_t* CEF_CALLBACK get_load_handler(
    struct _cef_client_t* self) {
    return NULL;
}

static struct _cef_render_handler_t* CEF_CALLBACK get_render_handler(
    struct _cef_client_t* self) {
    return NULL;
}

static struct _cef_request_handler_t* CEF_CALLBACK get_request_handler(
    struct _cef_client_t* self) {
    return NULL;
}

static int CEF_CALLBACK on_process_message_received(
    struct _cef_client_t* self,
    struct _cef_browser_t* browser,
    struct _cef_frame_t* frame,
    cef_process_id_t source_process,
    struct _cef_process_message_t* message) {
    return 0;
}

// Life span handler callbacks
static void CEF_CALLBACK on_after_created(
    struct _cef_life_span_handler_t* self,
    struct _cef_browser_t* browser) {
    add_browser(browser);
}

static int CEF_CALLBACK do_close(
    struct _cef_life_span_handler_t* self,
    struct _cef_browser_t* browser) {
    return 0; // Allow close
}

static void CEF_CALLBACK on_before_close(
    struct _cef_life_span_handler_t* self,
    struct _cef_browser_t* browser) {
    remove_browser(browser);
}

static void CEF_CALLBACK on_before_popup(
    struct _cef_life_span_handler_t* self,
    struct _cef_browser_t* browser,
    struct _cef_frame_t* frame,
    const cef_string_t* target_url,
    const cef_string_t* target_frame_name,
    cef_window_open_disposition_t target_disposition,
    int user_gesture,
    const struct _cef_popup_features_t* popupFeatures,
    struct _cef_window_info_t* windowInfo,
    struct _cef_client_t** client,
    struct _cef_browser_settings_t* settings,
    struct _cef_dictionary_value_t* extra_info,
    int* no_javascript_access) {
    // Block popups
    *no_javascript_access = 1;
}

// Helper functions
static void initialize_browser_array() {
    memset(g_browsers, 0, sizeof(g_browsers));
}

static int add_browser(cef_browser_t* browser) {
    for (int i = 0; i < MAX_BROWSERS; i++) {
        if (!g_browsers[i].active) {
            g_browsers[i].browser = browser;
            g_browsers[i].active = 1;
            return i;
        }
    }
    return -1;
}

static void remove_browser(cef_browser_t* browser) {
    for (int i = 0; i < MAX_BROWSERS; i++) {
        if (g_browsers[i].active && g_browsers[i].browser == browser) {
            g_browsers[i].active = 0;
            g_browsers[i].browser = NULL;
            break;
        }
    }
}

static cef_life_span_handler_t* create_life_span_handler() {
    cef_life_span_handler_t* handler = (cef_life_span_handler_t*)malloc(sizeof(cef_life_span_handler_t));
    init_base_ref_counted(&handler->base, sizeof(cef_life_span_handler_t));
    handler->on_after_created = on_after_created;
    handler->do_close = do_close;
    handler->on_before_close = on_before_close;
    handler->on_before_popup = on_before_popup;
    return handler;
}

static struct _cef_life_span_handler_t* CEF_CALLBACK get_life_span_handler(
    struct _cef_client_t* self) {
    return create_life_span_handler();
}

static cef_client_t* create_client() {
    cef_client_t* client = (cef_client_t*)malloc(sizeof(cef_client_t));
    init_base_ref_counted(&client->base, sizeof(cef_client_t));
    client->get_context_menu_handler = get_context_menu_handler;
    client->get_dialog_handler = get_dialog_handler;
    client->get_display_handler = get_display_handler;
    client->get_download_handler = get_download_handler;
    client->get_drag_handler = get_drag_handler;
    client->get_find_handler = get_find_handler;
    client->get_focus_handler = get_focus_handler;
    client->get_jsdialog_handler = get_jsdialog_handler;
    client->get_keyboard_handler = get_keyboard_handler;
    client->get_life_span_handler = get_life_span_handler;
    client->get_load_handler = get_load_handler;
    client->get_render_handler = get_render_handler;
    client->get_request_handler = get_request_handler;
    client->on_process_message_received = on_process_message_received;
    return client;
}

// Public API Implementation

void cef_initialize() {
    if (g_initialized) {
        return;
    }

    initialize_browser_array();

    // Create CefApp
    g_app = (cef_app_t*)malloc(sizeof(cef_app_t));
    init_base_ref_counted(&g_app->base, sizeof(cef_app_t));
    g_app->on_before_command_line_processing = on_before_command_line_processing;
    g_app->on_register_custom_schemes = on_register_custom_schemes;
    g_app->get_resource_bundle_handler = get_resource_bundle_handler;
    g_app->get_browser_process_handler = get_browser_process_handler;
    g_app->get_render_process_handler = get_render_process_handler;

    // Initialize CEF main args (platform-specific)
    cef_main_args_t main_args = {};
    #ifdef _WIN32
    main_args.instance = GetModuleHandle(NULL);
    #endif

    // Initialize CEF settings
    cef_settings_t settings = {};
    settings.size = sizeof(cef_settings_t);
    settings.no_sandbox = 1;
    settings.multi_threaded_message_loop = 0;
    settings.external_message_pump = 0;
    settings.windowless_rendering_enabled = 0;

    // Initialize CEF
    cef_initialize(&main_args, &settings, g_app, NULL);

    g_initialized = 1;
}

void cef_run() {
    if (!g_initialized) {
        return;
    }
    cef_run_message_loop();
}

void cef_shutdown() {
    if (!g_initialized) {
        return;
    }

    // Close all browsers
    for (int i = 0; i < MAX_BROWSERS; i++) {
        if (g_browsers[i].active && g_browsers[i].browser) {
            cef_browser_t* browser = (cef_browser_t*)g_browsers[i].browser;
            browser->get_host(browser)->close_browser(browser->get_host(browser), 1);
        }
    }

    cef_shutdown_lib();

    // Release app reference
    if (g_app) {
        release(&g_app->base);
        g_app = NULL;
    }

    g_initialized = 0;
}

cef_browser_t cef_browser_create(const char* url, int width, int height) {
    if (!g_initialized) {
        return NULL;
    }

    // Create window info
    cef_window_info_t window_info = {};
    window_info.width = width;
    window_info.height = height;
    window_info.parent_window = NULL;
    window_info.windowless_rendering_enabled = 0;

    // Create browser settings
    cef_browser_settings_t browser_settings = {};
    browser_settings.size = sizeof(cef_browser_settings_t);

    // Convert URL to CEF string
    cef_string_t cef_url = {};
    cef_string_utf8_to_utf16(url, strlen(url), &cef_url);

    // Create client
    cef_client_t* client = create_client();

    // Create browser
    cef_browser_t* browser = cef_browser_host_create_browser_sync(
        &window_info,
        client,
        &cef_url,
        &browser_settings,
        NULL,
        NULL
    );

    // Clean up
    cef_string_clear(&cef_url);

    return (cef_browser_t)browser;
}

void cef_browser_load_url(cef_browser_t browser, const char* url) {
    if (!browser || !url) return;

    cef_browser_t* b = (cef_browser_t*)browser;
    cef_frame_t* main_frame = b->get_main_frame(b);

    cef_string_t cef_url = {};
    cef_string_utf8_to_utf16(url, strlen(url), &cef_url);

    main_frame->load_url(main_frame, &cef_url);
    cef_string_clear(&cef_url);
}

void cef_browser_execute_js(cef_browser_t browser, const char* js) {
    if (!browser || !js) return;

    cef_browser_t* b = (cef_browser_t*)browser;
    cef_frame_t* main_frame = b->get_main_frame(b);

    cef_string_t cef_js = {};
    cef_string_utf8_to_utf16(js, strlen(js), &cef_js);

    cef_string_t cef_url = {};
    cef_string_set(L"about:blank", wcslen(L"about:blank"), &cef_url, 0);

    main_frame->execute_javaScript(main_frame, &cef_js, &cef_url, 0);

    cef_string_clear(&cef_js);
}

void cef_browser_destroy(cef_browser_t browser) {
    if (!browser) return;

    cef_browser_t* b = (cef_browser_t*)browser;
    cef_browser_host_t* host = b->get_host(b);
    host->close_browser(host, 1);
}

void* cef_get_native_window_handle(cef_browser_t browser) {
    if (!browser) return NULL;

    cef_browser_t* b = (cef_browser_t*)browser;
    cef_browser_host_t* host = b->get_host(b);
    return host->get_window_handle(host);
}

void cef_register_js_callback(const char* name, cef_callback_t callback) {
    (void)name; // Name not used in simple implementation
    g_js_callback = callback;
}

void cef_send_message_to_js(cef_browser_t browser, const char* message) {
    if (!browser || !message) return;

    // Create a JavaScript function call to send the message
    // This is a simple implementation that calls window.onCEFMessage if it exists
    char js_buffer[4096];
    snprintf(js_buffer, sizeof(js_buffer),
        "if (window.onCEFMessage) { window.onCEFMessage('%s'); }",
        message);

    cef_browser_execute_js(browser, js_buffer);
}

int cef_is_subprocess() {
    // Check command line for --type= argument
    cef_command_line_t* cmd_line = cef_command_line_get_global_command_line();
    if (!cmd_line) return 0;

    cef_string_t type_arg = {};
    cef_string_set(L"type", 4, &type_arg, 0);

    int has_type = cmd_line->has_switch(cmd_line, &type_arg);

    cef_string_clear(&type_arg);

    return has_type;
}

void cef_subprocess_entry() {
    // Create minimal app for subprocess
    cef_app_t* app = (cef_app_t*)malloc(sizeof(cef_app_t));
    init_base_ref_counted(&app->base, sizeof(cef_app_t));
    app->on_before_command_line_processing = on_before_command_line_processing;
    app->on_register_custom_schemes = on_register_custom_schemes;
    app->get_resource_bundle_handler = get_resource_bundle_handler;
    app->get_browser_process_handler = get_browser_process_handler;
    app->get_render_process_handler = get_render_process_handler;

    // Initialize main args
    cef_main_args_t main_args = {};
    #ifdef _WIN32
    main_args.instance = GetModuleHandle(NULL);
    #endif

    // Execute subprocess
    int exit_code = cef_execute_process(&main_args, app, NULL);

    // Cleanup
    release(&app->base);

    // Note: In a real implementation, the subprocess would exit here
    // For this wrapper, we just return and let the caller handle it
    (void)exit_code;
}
