// CEF Runtime Implementation
// Minimal CEF WebView without Views API

#include "cef_runtime.h"

#include <string.h>
#include <stdlib.h>
#include <stdio.h>
#include <stdatomic.h>
#include <unistd.h>
#include <sys/stat.h>

// Platform detection
#ifdef __linux__
#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <X11/Xutil.h>
#endif

// CEF C API headers
#include "include/capi/cef_app_capi.h"
#include "include/capi/cef_browser_capi.h"
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_command_line_capi.h"
#include "include/capi/cef_life_span_handler_capi.h"
#include "include/capi/cef_context_menu_handler_capi.h"
#include "include/capi/cef_display_handler_capi.h"
#include "include/capi/cef_load_handler_capi.h"
#include "include/capi/cef_request_handler_capi.h"
#include "include/capi/cef_process_message_capi.h"
#include "include/capi/cef_frame_capi.h"
#include "include/capi/cef_v8_capi.h"
#include "include/cef_api_hash.h"
#include "include/cef_version.h"

// Debug output
#define CEF_RUNTIME_LOG(level, fmt, ...) do { \
    if (level >= g_log_level) { \
        fprintf(stderr, "[CEF] " fmt "\n", ##__VA_ARGS__); \
        fflush(stderr); \
    } \
} while(0)

#define LOG_VERBOSE 0
#define LOG_INFO    1
#define LOG_WARNING 2
#define LOG_ERROR   3
#define LOG_FATAL   4

// Global state
static int g_initialized = 0;
static int g_log_level = LOG_INFO;
static cef_app_t* g_app = NULL;
static cef_client_t* g_client = NULL;
static cef_runtime_message_cb_t g_message_callback = NULL;
static void* g_message_user_data = NULL;
static cef_runtime_load_cb_t g_load_callback = NULL;
static void* g_load_user_data = NULL;

// Browser tracking
#define MAX_BROWSERS 16
typedef struct {
    cef_browser_t* browser;
    int valid;
} browser_entry_t;
static browser_entry_t g_browsers[MAX_BROWSERS] = {0};
static atomic_int g_browser_count = 0;

// X11 window tracking (Linux only)
#ifdef __linux__
static Window g_x11_parent_window = 0;
static Display* g_x11_display = NULL;
#endif

// Forward declarations
static void inject_bridge_js(cef_browser_t* browser);
static int handle_process_message(cef_browser_t* browser, cef_frame_t* frame,
                                   cef_process_id_t source_process,
                                   cef_process_message_t* message);

//
// Reference counting helpers
//

typedef struct {
    atomic_int ref_count;
} ref_counted_base_t;

static void add_ref_impl(cef_base_ref_counted_t* self) {
    ref_counted_base_t* base = (ref_counted_base_t*)((char*)self - offsetof(cef_client_t, base) + offsetof(ref_counted_base_t, ref_count));
    atomic_fetch_add(&base->ref_count, 1);
}

static int release_impl(cef_base_ref_counted_t* self, size_t struct_size) {
    ref_counted_base_t* base = (ref_counted_base_t*)self;
    int count = atomic_fetch_sub(&base->ref_count, 1) - 1;
    if (count == 0) {
        free(self);
        return 1;
    }
    return 0;
}

static int has_one_ref_impl(cef_base_ref_counted_t* self) {
    ref_counted_base_t* base = (ref_counted_base_t*)self;
    return atomic_load(&base->ref_count) == 1;
}

static int has_at_least_one_ref_impl(cef_base_ref_counted_t* self) {
    ref_counted_base_t* base = (ref_counted_base_t*)self;
    return atomic_load(&base->ref_count) >= 1;
}

//
// App implementation
//

typedef struct {
    cef_app_t app;
    atomic_int ref_count;
} app_wrapper_t;

static void CEF_CALLBACK app_on_before_command_line_processing(
    struct _cef_app_t* self,
    const cef_string_t* process_type,
    struct _cef_command_line_t* command_line) {
    (void)self;
    (void)process_type;

    // Electrobun-style command-line switches for chromeless/frameless mode
    cef_string_t switch_name = {};

    // Disable GPU acceleration for VM compatibility
    cef_string_utf8_to_utf16("disable-gpu", 11, &switch_name);
    command_line->append_switch(command_line, &switch_name);

    cef_string_utf8_to_utf16("disable-gpu-compositing", 23, &switch_name);
    command_line->append_switch(command_line, &switch_name);

    cef_string_utf8_to_utf16("disable-gpu-sandbox", 19, &switch_name);
    command_line->append_switch(command_line, &switch_name);

    cef_string_utf8_to_utf16("enable-software-rasterizer", 26, &switch_name);
    command_line->append_switch(command_line, &switch_name);

    cef_string_utf8_to_utf16("force-software-rasterizer", 25, &switch_name);
    command_line->append_switch(command_line, &switch_name);

    cef_string_utf8_to_utf16("disable-accelerated-2d-canvas", 29, &switch_name);
    command_line->append_switch(command_line, &switch_name);

    cef_string_utf8_to_utf16("disable-dev-shm-usage", 21, &switch_name);
    command_line->append_switch(command_line, &switch_name);

    cef_string_utf8_to_utf16("disable-extensions", 18, &switch_name);
    command_line->append_switch(command_line, &switch_name);

    cef_string_utf8_to_utf16("disable-plugins", 15, &switch_name);
    command_line->append_switch(command_line, &switch_name);

    cef_string_utf8_to_utf16("no-sandbox", 10, &switch_name);
    command_line->append_switch(command_line, &switch_name);

    // Force X11 backend for window embedding compatibility
    cef_string_t switch_value = {};
    cef_string_utf8_to_utf16("ozone-platform", 14, &switch_name);
    cef_string_utf8_to_utf16("x11", 3, &switch_value);
    command_line->append_switch_with_value(command_line, &switch_name, &switch_value);

    // Chromeless/Frameless flags
    cef_string_utf8_to_utf16("hide-frame", 10, &switch_name);
    command_line->append_switch(command_line, &switch_name);

    cef_string_utf8_to_utf16("hide-controls", 13, &switch_name);
    command_line->append_switch(command_line, &switch_name);

    cef_string_utf8_to_utf16("no-first-run", 12, &switch_name);
    command_line->append_switch(command_line, &switch_name);

    cef_string_utf8_to_utf16("no-default-browser-check", 24, &switch_name);
    command_line->append_switch(command_line, &switch_name);

    cef_string_clear(&switch_name);
    cef_string_clear(&switch_value);
}

static void CEF_CALLBACK app_on_register_custom_schemes(
    struct _cef_app_t* self,
    struct _cef_scheme_registrar_t* registrar) {
    (void)self;
    (void)registrar;
}

static struct _cef_resource_bundle_handler_t* CEF_CALLBACK app_get_resource_bundle_handler(
    struct _cef_app_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_browser_process_handler_t* CEF_CALLBACK app_get_browser_process_handler(
    struct _cef_app_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_render_process_handler_t* CEF_CALLBACK app_get_render_process_handler(
    struct _cef_app_t* self) {
    (void)self;
    return NULL;
}

static void CEF_CALLBACK app_add_ref(cef_base_ref_counted_t* self) {
    app_wrapper_t* app = (app_wrapper_t*)((char*)self - offsetof(app_wrapper_t, app.base));
    atomic_fetch_add(&app->ref_count, 1);
}

static int CEF_CALLBACK app_release(cef_base_ref_counted_t* self) {
    app_wrapper_t* app = (app_wrapper_t*)((char*)self - offsetof(app_wrapper_t, app.base));
    int count = atomic_fetch_sub(&app->ref_count, 1) - 1;
    if (count == 0) {
        free(app);
        return 1;
    }
    return 0;
}

static int CEF_CALLBACK app_has_one_ref(cef_base_ref_counted_t* self) {
    app_wrapper_t* app = (app_wrapper_t*)((char*)self - offsetof(app_wrapper_t, app.base));
    return atomic_load(&app->ref_count) == 1;
}

static int CEF_CALLBACK app_has_at_least_one_ref(cef_base_ref_counted_t* self) {
    app_wrapper_t* app = (app_wrapper_t*)((char*)self - offsetof(app_wrapper_t, app.base));
    return atomic_load(&app->ref_count) >= 1;
}

static app_wrapper_t* create_app(void) {
    app_wrapper_t* app = (app_wrapper_t*)calloc(1, sizeof(app_wrapper_t));
    if (!app) return NULL;

    app->app.base.size = sizeof(cef_app_t);
    app->app.base.add_ref = app_add_ref;
    app->app.base.release = app_release;
    app->app.base.has_one_ref = app_has_one_ref;
    app->app.base.has_at_least_one_ref = app_has_at_least_one_ref;

    app->app.on_before_command_line_processing = app_on_before_command_line_processing;
    app->app.on_register_custom_schemes = app_on_register_custom_schemes;
    app->app.get_resource_bundle_handler = app_get_resource_bundle_handler;
    app->app.get_browser_process_handler = app_get_browser_process_handler;
    app->app.get_render_process_handler = app_get_render_process_handler;

    atomic_store(&app->ref_count, 1);
    return app;
}

//
// Context Menu Handler - Disable right-click menu
//

typedef struct {
    cef_context_menu_handler_t handler;
    atomic_int ref_count;
} context_menu_handler_t;

static void CEF_CALLBACK cmh_add_ref(cef_base_ref_counted_t* self) {
    context_menu_handler_t* handler = (context_menu_handler_t*)((char*)self - offsetof(context_menu_handler_t, handler.base));
    atomic_fetch_add(&handler->ref_count, 1);
}

static int CEF_CALLBACK cmh_release(cef_base_ref_counted_t* self) {
    context_menu_handler_t* handler = (context_menu_handler_t*)((char*)self - offsetof(context_menu_handler_t, handler.base));
    int count = atomic_fetch_sub(&handler->ref_count, 1) - 1;
    if (count == 0) {
        free(handler);
        return 1;
    }
    return 0;
}

static int CEF_CALLBACK cmh_has_one_ref(cef_base_ref_counted_t* self) {
    context_menu_handler_t* handler = (context_menu_handler_t*)((char*)self - offsetof(context_menu_handler_t, handler.base));
    return atomic_load(&handler->ref_count) == 1;
}

static int CEF_CALLBACK cmh_has_at_least_one_ref(cef_base_ref_counted_t* self) {
    context_menu_handler_t* handler = (context_menu_handler_t*)((char*)self - offsetof(context_menu_handler_t, handler.base));
    return atomic_load(&handler->ref_count) >= 1;
}

static void CEF_CALLBACK cmh_on_before_context_menu(
    struct _cef_context_menu_handler_t* self,
    struct _cef_browser_t* browser,
    struct _cef_frame_t* frame,
    struct _cef_context_menu_params_t* params,
    struct _cef_menu_model_t* model) {
    (void)self;
    (void)browser;
    (void)frame;
    (void)params;
    model->clear(model);
}

static context_menu_handler_t* create_context_menu_handler(void) {
    context_menu_handler_t* handler = (context_menu_handler_t*)calloc(1, sizeof(context_menu_handler_t));
    if (!handler) return NULL;

    handler->handler.base.size = sizeof(cef_context_menu_handler_t);
    handler->handler.base.add_ref = cmh_add_ref;
    handler->handler.base.release = cmh_release;
    handler->handler.base.has_one_ref = cmh_has_one_ref;
    handler->handler.base.has_at_least_one_ref = cmh_has_at_least_one_ref;
    handler->handler.on_before_context_menu = cmh_on_before_context_menu;

    atomic_store(&handler->ref_count, 1);
    return handler;
}

//
// Life Span Handler
//

typedef struct {
    cef_life_span_handler_t handler;
    atomic_int ref_count;
} life_span_handler_t;

static void CEF_CALLBACK lsh_add_ref(cef_base_ref_counted_t* self) {
    life_span_handler_t* handler = (life_span_handler_t*)((char*)self - offsetof(life_span_handler_t, handler.base));
    atomic_fetch_add(&handler->ref_count, 1);
}

static int CEF_CALLBACK lsh_release(cef_base_ref_counted_t* self) {
    life_span_handler_t* handler = (life_span_handler_t*)((char*)self - offsetof(life_span_handler_t, handler.base));
    int count = atomic_fetch_sub(&handler->ref_count, 1) - 1;
    if (count == 0) {
        free(handler);
        return 1;
    }
    return 0;
}

static int CEF_CALLBACK lsh_has_one_ref(cef_base_ref_counted_t* self) {
    life_span_handler_t* handler = (life_span_handler_t*)((char*)self - offsetof(life_span_handler_t, handler.base));
    return atomic_load(&handler->ref_count) == 1;
}

static int CEF_CALLBACK lsh_has_at_least_one_ref(cef_base_ref_counted_t* self) {
    life_span_handler_t* handler = (life_span_handler_t*)((char*)self - offsetof(life_span_handler_t, handler.base));
    return atomic_load(&handler->ref_count) >= 1;
}

static void CEF_CALLBACK lsh_on_after_created(
    struct _cef_life_span_handler_t* self,
    struct _cef_browser_t* browser) {
    (void)self;
    CEF_RUNTIME_LOG(LOG_INFO, "Browser created");

    // Store browser reference
    for (int i = 0; i < MAX_BROWSERS; i++) {
        if (!g_browsers[i].valid) {
            g_browsers[i].browser = browser;
            g_browsers[i].valid = 1;
            browser->base.add_ref(&browser->base);
            atomic_fetch_add(&g_browser_count, 1);
            break;
        }
    }

    // Inject bridge JavaScript
    inject_bridge_js(browser);
}

static int CEF_CALLBACK lsh_do_close(
    struct _cef_life_span_handler_t* self,
    struct _cef_browser_t* browser) {
    (void)self;
    (void)browser;
    return 0;
}

static void CEF_CALLBACK lsh_on_before_close(
    struct _cef_life_span_handler_t* self,
    struct _cef_browser_t* browser) {
    (void)self;
    CEF_RUNTIME_LOG(LOG_INFO, "Browser closing");

    // Remove from tracking
    for (int i = 0; i < MAX_BROWSERS; i++) {
        if (g_browsers[i].valid && g_browsers[i].browser == browser) {
            g_browsers[i].browser->base.release(&g_browsers[i].browser->base);
            g_browsers[i].valid = 0;
            g_browsers[i].browser = NULL;
            int remaining = atomic_fetch_sub(&g_browser_count, 1) - 1;
            if (remaining <= 0) {
                CEF_RUNTIME_LOG(LOG_INFO, "Last browser closed, quitting message loop");
                cef_quit_message_loop();
            }
            break;
        }
    }
}

static int CEF_CALLBACK lsh_on_before_popup(
    struct _cef_life_span_handler_t* self,
    struct _cef_browser_t* browser,
    struct _cef_frame_t* frame,
    int popup_id,
    const cef_string_t* target_url,
    const cef_string_t* target_frame_name,
    cef_window_open_disposition_t target_disposition,
    int user_gesture,
    const struct _cef_popup_features_t* popupFeatures,
    struct _cef_window_info_t* windowInfo,
    struct _cef_client_t** client,
    struct _cef_browser_settings_t* settings,
    struct _cef_dictionary_value_t** extra_info,
    int* no_javascript_access) {
    (void)self;
    (void)browser;
    (void)frame;
    (void)popup_id;
    (void)target_url;
    (void)target_frame_name;
    (void)target_disposition;
    (void)user_gesture;
    (void)popupFeatures;
    (void)windowInfo;
    (void)client;
    (void)settings;
    (void)extra_info;
    *no_javascript_access = 1;
    return 1; // Block popups
}

static life_span_handler_t* create_life_span_handler(void) {
    life_span_handler_t* handler = (life_span_handler_t*)calloc(1, sizeof(life_span_handler_t));
    if (!handler) return NULL;

    handler->handler.base.size = sizeof(cef_life_span_handler_t);
    handler->handler.base.add_ref = lsh_add_ref;
    handler->handler.base.release = lsh_release;
    handler->handler.base.has_one_ref = lsh_has_one_ref;
    handler->handler.base.has_at_least_one_ref = lsh_has_at_least_one_ref;

    handler->handler.on_after_created = lsh_on_after_created;
    handler->handler.do_close = lsh_do_close;
    handler->handler.on_before_close = lsh_on_before_close;
    handler->handler.on_before_popup = lsh_on_before_popup;

    atomic_store(&handler->ref_count, 1);
    return handler;
}

//
// Load Handler
//

typedef struct {
    cef_load_handler_t handler;
    atomic_int ref_count;
} load_handler_t;

static void CEF_CALLBACK lh_add_ref(cef_base_ref_counted_t* self) {
    load_handler_t* handler = (load_handler_t*)((char*)self - offsetof(load_handler_t, handler.base));
    atomic_fetch_add(&handler->ref_count, 1);
}

static int CEF_CALLBACK lh_release(cef_base_ref_counted_t* self) {
    load_handler_t* handler = (load_handler_t*)((char*)self - offsetof(load_handler_t, handler.base));
    int count = atomic_fetch_sub(&handler->ref_count, 1) - 1;
    if (count == 0) {
        free(handler);
        return 1;
    }
    return 0;
}

static int CEF_CALLBACK lh_has_one_ref(cef_base_ref_counted_t* self) {
    load_handler_t* handler = (load_handler_t*)((char*)self - offsetof(load_handler_t, handler.base));
    return atomic_load(&handler->ref_count) == 1;
}

static int CEF_CALLBACK lh_has_at_least_one_ref(cef_base_ref_counted_t* self) {
    load_handler_t* handler = (load_handler_t*)((char*)self - offsetof(load_handler_t, handler.base));
    return atomic_load(&handler->ref_count) >= 1;
}

static void CEF_CALLBACK lh_on_loading_state_change(
    struct _cef_load_handler_t* self,
    struct _cef_browser_t* browser,
    int isLoading,
    int canGoBack,
    int canGoForward) {
    (void)self;
    (void)browser;
    (void)canGoBack;
    (void)canGoForward;

    if (!isLoading && g_load_callback) {
        g_load_callback(1, g_load_user_data);
    }
}

static void CEF_CALLBACK lh_on_load_start(
    struct _cef_load_handler_t* self,
    struct _cef_browser_t* browser,
    struct _cef_frame_t* frame,
    cef_transition_type_t transition_type) {
    (void)self;
    (void)browser;
    (void)frame;
    (void)transition_type;
}

static void CEF_CALLBACK lh_on_load_end(
    struct _cef_load_handler_t* self,
    struct _cef_browser_t* browser,
    struct _cef_frame_t* frame,
    int httpStatusCode) {
    (void)self;
    (void)browser;
    (void)frame;
    (void)httpStatusCode;
}

static void CEF_CALLBACK lh_on_load_error(
    struct _cef_load_handler_t* self,
    struct _cef_browser_t* browser,
    struct _cef_frame_t* frame,
    cef_errorcode_t errorCode,
    const cef_string_t* errorText,
    const cef_string_t* failedUrl) {
    (void)self;
    (void)browser;
    (void)frame;
    (void)errorCode;
    (void)errorText;
    (void)failedUrl;

    if (g_load_callback) {
        g_load_callback(0, g_load_user_data);
    }
}

static load_handler_t* create_load_handler(void) {
    load_handler_t* handler = (load_handler_t*)calloc(1, sizeof(load_handler_t));
    if (!handler) return NULL;

    handler->handler.base.size = sizeof(cef_load_handler_t);
    handler->handler.base.add_ref = lh_add_ref;
    handler->handler.base.release = lh_release;
    handler->handler.base.has_one_ref = lh_has_one_ref;
    handler->handler.base.has_at_least_one_ref = lh_has_at_least_one_ref;

    handler->handler.on_loading_state_change = lh_on_loading_state_change;
    handler->handler.on_load_start = lh_on_load_start;
    handler->handler.on_load_end = lh_on_load_end;
    handler->handler.on_load_error = lh_on_load_error;

    atomic_store(&handler->ref_count, 1);
    return handler;
}

//
// Client implementation
//

typedef struct {
    cef_client_t client;
    atomic_int ref_count;
} client_wrapper_t;

static void CEF_CALLBACK client_add_ref(cef_base_ref_counted_t* self) {
    client_wrapper_t* client = (client_wrapper_t*)((char*)self - offsetof(client_wrapper_t, client.base));
    atomic_fetch_add(&client->ref_count, 1);
}

static int CEF_CALLBACK client_release(cef_base_ref_counted_t* self) {
    client_wrapper_t* client = (client_wrapper_t*)((char*)self - offsetof(client_wrapper_t, client.base));
    int count = atomic_fetch_sub(&client->ref_count, 1) - 1;
    if (count == 0) {
        free(client);
        return 1;
    }
    return 0;
}

static int CEF_CALLBACK client_has_one_ref(cef_base_ref_counted_t* self) {
    client_wrapper_t* client = (client_wrapper_t*)((char*)self - offsetof(client_wrapper_t, client.base));
    return atomic_load(&client->ref_count) == 1;
}

static int CEF_CALLBACK client_has_at_least_one_ref(cef_base_ref_counted_t* self) {
    client_wrapper_t* client = (client_wrapper_t*)((char*)self - offsetof(client_wrapper_t, client.base));
    return atomic_load(&client->ref_count) >= 1;
}

static struct _cef_context_menu_handler_t* CEF_CALLBACK client_get_context_menu_handler(
    struct _cef_client_t* self) {
    (void)self;
    context_menu_handler_t* handler = create_context_menu_handler();
    if (handler) {
        handler->handler.base.add_ref(&handler->handler.base);
        return &handler->handler;
    }
    return NULL;
}

static struct _cef_life_span_handler_t* CEF_CALLBACK client_get_life_span_handler(
    struct _cef_client_t* self) {
    (void)self;
    life_span_handler_t* handler = create_life_span_handler();
    if (handler) {
        handler->handler.base.add_ref(&handler->handler.base);
        return &handler->handler;
    }
    return NULL;
}

static struct _cef_load_handler_t* CEF_CALLBACK client_get_load_handler(
    struct _cef_client_t* self) {
    (void)self;
    load_handler_t* handler = create_load_handler();
    if (handler) {
        handler->handler.base.add_ref(&handler->handler.base);
        return &handler->handler;
    }
    return NULL;
}

static int CEF_CALLBACK client_on_process_message_received(
    struct _cef_client_t* self,
    struct _cef_browser_t* browser,
    struct _cef_frame_t* frame,
    cef_process_id_t source_process,
    struct _cef_process_message_t* message) {
    (void)self;
    return handle_process_message(browser, frame, source_process, message);
}

static client_wrapper_t* create_client(void) {
    client_wrapper_t* client = (client_wrapper_t*)calloc(1, sizeof(client_wrapper_t));
    if (!client) return NULL;

    client->client.base.size = sizeof(cef_client_t);
    client->client.base.add_ref = client_add_ref;
    client->client.base.release = client_release;
    client->client.base.has_one_ref = client_has_one_ref;
    client->client.base.has_at_least_one_ref = client_has_at_least_one_ref;

    client->client.get_context_menu_handler = client_get_context_menu_handler;
    client->client.get_life_span_handler = client_get_life_span_handler;
    client->client.get_load_handler = client_get_load_handler;
    client->client.on_process_message_received = client_on_process_message_received;

    atomic_store(&client->ref_count, 1);
    return client;
}

//
// IPC: Process message handler
//

static int handle_process_message(cef_browser_t* browser, cef_frame_t* frame,
                                   cef_process_id_t source_process,
                                   cef_process_message_t* message) {
    (void)browser;
    (void)frame;
    (void)source_process;

    cef_string_userfree_t name = message->get_name(message);
    if (!name) return 0;

    // Check if this is our IPC message
    if (cef_string_utf16_cmp(name, u"go_invoke") == 0) {
        cef_list_value_t* args = message->get_argument_list(message);
        if (args && args->get_size(args) >= 2) {
            cef_string_t name_str = {};
            cef_string_t payload_str = {};

            cef_string_userfree_t name_val = args->get_string(args, 0);
            cef_string_userfree_t payload_val = args->get_string(args, 1);

            if (name_val && payload_val) {
                // Convert to UTF-8
                char name_utf8[256] = {0};
                char payload_utf8[4096] = {0};

                cef_string_utf8_t name_utf8_str = {};
                cef_string_utf8_t payload_utf8_str = {};

                cef_string_utf16_to_utf8(name_val->str, name_val->length, &name_utf8_str);
                cef_string_utf16_to_utf8(payload_val->str, payload_val->length, &payload_utf8_str);

                if (name_utf8_str.str) {
                    strncpy(name_utf8, name_utf8_str.str, sizeof(name_utf8) - 1);
                    cef_string_utf8_clear(&name_utf8_str);
                }
                if (payload_utf8_str.str) {
                    strncpy(payload_utf8, payload_utf8_str.str, sizeof(payload_utf8) - 1);
                    cef_string_utf8_clear(&payload_utf8_str);
                }

                if (g_message_callback) {
                    g_message_callback(name_utf8, payload_utf8, g_message_user_data);
                }
            }

            if (name_val) cef_string_userfree_free(name_val);
            if (payload_val) cef_string_userfree_free(payload_val);
        }
        cef_string_userfree_free(name);
        return 1;
    }

    cef_string_userfree_free(name);
    return 0;
}

//
// Bridge JavaScript injection
//

static void inject_bridge_js(cef_browser_t* browser) {
    const char* bridge_js =
        "window.go = window.go || {};"
        "window.__goCallbacks = window.__goCallbacks || {};"
        "window.go.invoke = function(name, payload) {"
        "  if (typeof payload === 'object') payload = JSON.stringify(payload);"
        "  if (typeof cef !== 'undefined' && cef.postMessage) {"
        "    cef.postMessage(JSON.stringify({type: 'go_invoke', name: name, payload: payload}));"
        "  }"
        "};"
        "window.__goDispatch = function(name, payload) {"
        "  if (window.__goHandlers && window.__goHandlers[name]) {"
        "    window.__goHandlers[name](payload);"
        "  }"
        "};";

    cef_frame_t* main_frame = browser->get_main_frame(browser);
    if (main_frame) {
        cef_string_t js = {};
        cef_string_utf8_to_utf16(bridge_js, strlen(bridge_js), &js);
        cef_string_t url = {};
        cef_string_from_ascii("about:blank", 11, &url);
        main_frame->execute_java_script(main_frame, &js, &url, 0);
        cef_string_clear(&js);
        cef_string_clear(&url);
        main_frame->base.release(&main_frame->base);
    }
}

//
// Public API Implementation
//

int cef_runtime_initialize(int argc, char** argv) {
    CEF_RUNTIME_LOG(LOG_INFO, "Initializing CEF runtime");

    // Step 1: API hash check (required for CEF 145)
    cef_api_hash(CEF_API_VERSION, 0);

    // Step 2: Create main_args
    cef_main_args_t main_args = {};
    main_args.argc = argc;
    main_args.argv = argv;

    // Step 3: Execute process (handles subprocess detection)
    int exit_code = cef_execute_process(&main_args, NULL, NULL);
    if (exit_code >= 0) {
        return exit_code; // Subprocess
    }

    // Step 4: Browser process - create app
    app_wrapper_t* app = create_app();
    if (!app) {
        CEF_RUNTIME_LOG(LOG_ERROR, "Failed to create app");
        return -2;
    }

    // Add refs for CEF
    app->app.base.add_ref(&app->app.base);
    app->app.base.add_ref(&app->app.base);
    g_app = &app->app;

    // Create client
    client_wrapper_t* client = create_client();
    if (!client) {
        CEF_RUNTIME_LOG(LOG_ERROR, "Failed to create client");
        app->app.base.release(&app->app.base);
        g_app = NULL;
        return -2;
    }
    g_client = &client->client;

    // Step 5: Configure settings (Electrobun-style)
    cef_settings_t settings = {};
    settings.size = sizeof(cef_settings_t);
    settings.no_sandbox = 1;
    settings.multi_threaded_message_loop = 0;
    settings.external_message_pump = 0;
    settings.windowless_rendering_enabled = 1;  // Required for OSR/transparent windows
    settings.log_severity = LOGSEVERITY_ERROR;  // Change to WARNING for more logs

    // Linux-specific: disable sandbox for embedded scenarios
    settings.command_line_args_disabled = 0;

    // Set paths
    char cwd[4096];
    if (getcwd(cwd, sizeof(cwd)) != NULL) {
        char path_str[4096];
        static cef_string_t cache_path = {};
        static cef_string_t root_cache_path = {};
        static cef_string_t resources_path = {};
        static cef_string_t locales_path = {};

        // Cache path
        snprintf(path_str, sizeof(path_str), "%s/cef_cache", cwd);
        mkdir(path_str, 0755);
        cef_string_utf8_to_utf16(path_str, strlen(path_str), &cache_path);
        settings.cache_path = cache_path;
        settings.root_cache_path = root_cache_path;

        // Resources path
        snprintf(path_str, sizeof(path_str), "%s", cwd);
        cef_string_utf8_to_utf16(path_str, strlen(path_str), &resources_path);
        settings.resources_dir_path = resources_path;

        // Locales path
        snprintf(path_str, sizeof(path_str), "%s/locales", cwd);
        cef_string_utf8_to_utf16(path_str, strlen(path_str), &locales_path);
        settings.locales_dir_path = locales_path;

        // Subprocess path (for renderer process)
        static cef_string_t subprocess_path = {};
        snprintf(path_str, sizeof(path_str), "%s/cef_subprocess", cwd);
        cef_string_utf8_to_utf16(path_str, strlen(path_str), &subprocess_path);
        settings.browser_subprocess_path = subprocess_path;
    }

    // Step 6: Initialize CEF
    int result = cef_initialize(&main_args, &settings, g_app, NULL);
    if (!result) {
        CEF_RUNTIME_LOG(LOG_ERROR, "CEF initialization failed");
        g_app->base.release(&g_app->base);
        g_app = NULL;
        return -2;
    }

    g_initialized = 1;
    CEF_RUNTIME_LOG(LOG_INFO, "CEF initialized successfully");
    return -1;
}

void cef_runtime_run(void) {
    if (!g_initialized) {
        CEF_RUNTIME_LOG(LOG_ERROR, "CEF not initialized");
        return;
    }
    CEF_RUNTIME_LOG(LOG_INFO, "Running message loop");
    cef_run_message_loop();
}

void cef_runtime_shutdown(void) {
    if (!g_initialized) return;

    CEF_RUNTIME_LOG(LOG_INFO, "Shutting down CEF");

    // Close all browsers
    for (int i = 0; i < MAX_BROWSERS; i++) {
        if (g_browsers[i].valid && g_browsers[i].browser) {
            cef_browser_host_t* host = g_browsers[i].browser->get_host(g_browsers[i].browser);
            if (host) {
                host->close_browser(host, 1);
                host->base.release(&host->base);
            }
        }
    }

    cef_shutdown();

    if (g_app) {
        g_app->base.release(&g_app->base);
        g_app = NULL;
    }

    g_initialized = 0;
}

cef_runtime_browser_t cef_runtime_create_browser(const cef_runtime_browser_opts_t* opts) {
    if (!g_initialized || !opts) {
        CEF_RUNTIME_LOG(LOG_ERROR, "Cannot create browser: not initialized or null opts");
        return NULL;
    }

    const char* url = opts->url ? opts->url : "about:blank";
    int width = opts->width > 0 ? opts->width : 800;
    int height = opts->height > 0 ? opts->height : 600;

    CEF_RUNTIME_LOG(LOG_INFO, "Creating browser: %s %dx%d (frameless=%d)",
                    url, width, height, opts->frameless);

    // Create window info (Electrobun-style)
    cef_window_info_t window_info = {};
    window_info.size = sizeof(cef_window_info_t);

    // Use ALLOY runtime style for chromeless mode (hides Chrome UI like URL bar, tabs)
    // Use CHROME runtime style for normal mode
    // This is the key setting for true chromeless mode!
    if (opts->chromeless) {
        window_info.runtime_style = CEF_RUNTIME_STYLE_ALLOY;
        CEF_RUNTIME_LOG(LOG_INFO, "Using ALLOY runtime style (chromeless mode)");
    } else {
        window_info.runtime_style = CEF_RUNTIME_STYLE_CHROME;
        CEF_RUNTIME_LOG(LOG_INFO, "Using CHROME runtime style");
    }

    // Set window title
    const char* title = opts->title ? opts->title : "App";
    cef_string_utf8_to_utf16(title, strlen(title), &window_info.window_name);

#ifdef __linux__
    // Create an X11 parent window only for frameless mode.
    // In normal framed mode we let CEF/WM own the top-level window so that
    // native maximize/close behaviors and resize propagation work reliably.
    Window parent_window = 0;
    Display* display = NULL;
    
    if (opts->frameless) {
        display = XOpenDisplay(NULL);
        if (display) {
            int screen = DefaultScreen(display);
            Window root = RootWindow(display, screen);
            
            // Create a window
            XSetWindowAttributes attrs;
            attrs.event_mask = ExposureMask | KeyPressMask | KeyReleaseMask |
                              ButtonPressMask | ButtonReleaseMask | PointerMotionMask |
                              FocusChangeMask | StructureNotifyMask;
            
            unsigned long attr_mask = CWEventMask;
            
            parent_window = XCreateWindow(
                display, root,
                opts->x >= 0 ? opts->x : 100, 
                opts->y >= 0 ? opts->y : 100, 
                width, height, 0,
                CopyFromParent, InputOutput, CopyFromParent,
                attr_mask, &attrs
            );
            
            if (parent_window) {
                // Set window attributes
                XSetWindowAttributes win_attrs;
                win_attrs.background_pixel = WhitePixel(display, screen);
                win_attrs.border_pixel = BlackPixel(display, screen);
                win_attrs.colormap = DefaultColormap(display, screen);
                XChangeWindowAttributes(display, parent_window, CWBackPixel | CWBorderPixel | CWColormap, &win_attrs);
                
                // Set window title
                XStoreName(display, parent_window, title);
                
                // Set WM_CLASS
                XClassHint class_hint;
                class_hint.res_name = (char*)title;
                class_hint.res_class = (char*)title;
                XSetClassHint(display, parent_window, &class_hint);
                
                // Set window protocols for close button
                Atom wmDelete = XInternAtom(display, "WM_DELETE_WINDOW", False);
                XSetWMProtocols(display, parent_window, &wmDelete, 1);
                
                // For frameless mode, remove window decorations
                Atom wmHints = XInternAtom(display, "_MOTIF_WM_HINTS", False);
                struct {
                    unsigned long flags;
                    unsigned long functions;
                    unsigned long decorations;
                    long inputMode;
                    unsigned long status;
                } hints = { 2, 0, 0, 0, 0 };  // MWM_HINTS_DECORATIONS = 2, no decorations
                
                XChangeProperty(display, parent_window, wmHints, wmHints, 32,
                               PropModeReplace, (unsigned char*)&hints, 5);
                
                // Set window type
                Atom wmWindowType = XInternAtom(display, "_NET_WM_WINDOW_TYPE", False);
                Atom wmWindowTypeNormal = XInternAtom(display, "_NET_WM_WINDOW_TYPE_NORMAL", False);
                XChangeProperty(display, parent_window, wmWindowType, XA_ATOM, 32,
                               PropModeReplace, (unsigned char*)&wmWindowTypeNormal, 1);
                
                // Set size hints
                XSizeHints* sizeHints = XAllocSizeHints();
                if (sizeHints) {
                    sizeHints->flags = PPosition | PSize;
                    sizeHints->x = opts->x >= 0 ? opts->x : 100;
                    sizeHints->y = opts->y >= 0 ? opts->y : 100;
                    sizeHints->width = width;
                    sizeHints->height = height;
                    XSetWMNormalHints(display, parent_window, sizeHints);
                    XFree(sizeHints);
                }
                
                // DON'T map the window yet - wait for CEF to be created
                XFlush(display);
                
                CEF_RUNTIME_LOG(LOG_INFO, "Created X11 parent window: %lu (not mapped yet)", parent_window);
            }
        }
    } else {
        CEF_RUNTIME_LOG(LOG_INFO, "Using CEF-managed native window (framed mode)");
    }
    
    // Set parent window for embedding
    if (parent_window) {
        window_info.parent_window = parent_window;
        window_info.bounds.x = 0;
        window_info.bounds.y = 0;
        window_info.bounds.width = width;
        window_info.bounds.height = height;
    } else
#endif
    {
        // Fallback: no parent window
        window_info.bounds.x = opts->x >= 0 ? opts->x : 100;
        window_info.bounds.y = opts->y >= 0 ? opts->y : 100;
        window_info.bounds.width = width;
        window_info.bounds.height = height;
    }

    // Create browser settings
    cef_browser_settings_t browser_settings = {};
    browser_settings.size = sizeof(cef_browser_settings_t);

    // Convert URL
    cef_string_t cef_url = {};
    cef_string_utf8_to_utf16(url, strlen(url), &cef_url);

    // Add client ref
    g_client->base.add_ref(&g_client->base);

    // Create browser
    cef_browser_t* browser = cef_browser_host_create_browser_sync(
        &window_info,
        g_client,
        &cef_url,
        &browser_settings,
        NULL,
        NULL
    );

    cef_string_clear(&cef_url);
    cef_string_clear(&window_info.window_name);

    if (!browser) {
        CEF_RUNTIME_LOG(LOG_ERROR, "Browser creation failed");
        g_client->base.release(&g_client->base);
        return NULL;
    }

#ifdef __linux__
    // Store X11 window and display globally, then map the window
    if (parent_window && display) {
        g_x11_parent_window = parent_window;
        g_x11_display = display;
        
        // Now that CEF is created, map the parent window
        CEF_RUNTIME_LOG(LOG_INFO, "Mapping X11 parent window now that CEF is ready");
        XMapWindow(display, parent_window);
        XRaiseWindow(display, parent_window);
        XSetInputFocus(display, parent_window, RevertToParent, CurrentTime);
        XFlush(display);
        CEF_RUNTIME_LOG(LOG_INFO, "X11 parent window mapped and raised");
    }
#endif

    CEF_RUNTIME_LOG(LOG_INFO, "Browser created successfully");
    return (cef_runtime_browser_t)browser;
}

void cef_runtime_navigate(cef_runtime_browser_t browser, const char* url) {
    if (!browser || !url) return;

    cef_browser_t* b = (cef_browser_t*)browser;
    cef_frame_t* main_frame = b->get_main_frame(b);
    if (!main_frame) return;

    cef_string_t cef_url = {};
    cef_string_utf8_to_utf16(url, strlen(url), &cef_url);
    main_frame->load_url(main_frame, &cef_url);
    cef_string_clear(&cef_url);
    main_frame->base.release(&main_frame->base);
}

void cef_runtime_eval(cef_runtime_browser_t browser, const char* js) {
    if (!browser || !js) return;

    cef_browser_t* b = (cef_browser_t*)browser;
    cef_frame_t* main_frame = b->get_main_frame(b);
    if (!main_frame) return;

    cef_string_t cef_js = {};
    cef_string_utf8_to_utf16(js, strlen(js), &cef_js);
    cef_string_t cef_url = {};
    cef_string_from_ascii("about:blank", 11, &cef_url);

    main_frame->execute_java_script(main_frame, &cef_js, &cef_url, 0);

    cef_string_clear(&cef_js);
    cef_string_clear(&cef_url);
    main_frame->base.release(&main_frame->base);
}

void cef_runtime_set_title(cef_runtime_browser_t browser, const char* title) {
    (void)browser;
    (void)title;
    // Platform-specific implementation needed
    CEF_RUNTIME_LOG(LOG_WARNING, "set_title not implemented");
}

void cef_runtime_resize(cef_runtime_browser_t browser, int width, int height) {
    (void)browser;
    (void)width;
    (void)height;
    // Platform-specific implementation needed
    CEF_RUNTIME_LOG(LOG_WARNING, "resize not implemented");
}

void cef_runtime_show(cef_runtime_browser_t browser) {
    (void)browser;
    // Platform-specific implementation needed
}

void cef_runtime_hide(cef_runtime_browser_t browser) {
    (void)browser;
    // Platform-specific implementation needed
}

void cef_runtime_close(cef_runtime_browser_t browser) {
    if (!browser) return;

    cef_browser_t* b = (cef_browser_t*)browser;
    cef_browser_host_t* host = b->get_host(b);
    if (host) {
        host->close_browser(host, 1);
        host->base.release(&host->base);
    }
}

int cef_runtime_is_valid(cef_runtime_browser_t browser) {
    if (!browser) return 0;

    cef_browser_t* b = (cef_browser_t*)browser;
    for (int i = 0; i < MAX_BROWSERS; i++) {
        if (g_browsers[i].valid && g_browsers[i].browser == b) {
            return 1;
        }
    }
    return 0;
}

void cef_runtime_set_message_callback(cef_runtime_message_cb_t callback, void* user_data) {
    g_message_callback = callback;
    g_message_user_data = user_data;
}

void cef_runtime_send_message(cef_runtime_browser_t browser, const char* name, const char* payload) {
    if (!browser || !name) return;

    // Build JavaScript to dispatch message
    char js[4096];
    if (payload) {
        snprintf(js, sizeof(js),
            "if (window.__goDispatch) window.__goDispatch('%s', '%s');",
            name, payload);
    } else {
        snprintf(js, sizeof(js),
            "if (window.__goDispatch) window.__goDispatch('%s', null);",
            name);
    }

    cef_runtime_eval(browser, js);
}

void cef_runtime_inject_bridge(cef_runtime_browser_t browser) {
    if (!browser) return;
    inject_bridge_js((cef_browser_t*)browser);
}

const char* cef_runtime_version(void) {
    // Return CEF version string
    return CEF_VERSION;
}

void cef_runtime_set_log_level(int level) {
    g_log_level = level;
}

void cef_runtime_set_dev_tools(cef_runtime_browser_t browser, int enabled) {
    (void)browser;
    (void)enabled;
    // DevTools not implemented in minimal runtime
    // Can be added later using cef_browser_host_show_dev_tools
}

void cef_runtime_set_load_callback(cef_runtime_load_cb_t callback, void* user_data) {
    g_load_callback = callback;
    g_load_user_data = user_data;
}

void cef_runtime_post_message(const char* message) {
    (void)message;
    // Used by render process - not implemented in this minimal version
}
