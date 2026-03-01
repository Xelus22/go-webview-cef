#include "cef_wrapper.h"

#include <string.h>
#include <stdlib.h>
#include <stdio.h>

// Include CEF C API headers
#include "include/internal/cef_string.h"
#include "include/internal/cef_string_types.h"
#include "include/capi/cef_app_capi.h"
#include "include/capi/cef_browser_capi.h"
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_command_line_capi.h"
#include "include/capi/cef_life_span_handler_capi.h"
#include "include/capi/cef_base_capi.h"
#include "include/capi/cef_frame_capi.h"
#include "include/capi/cef_browser_process_handler_capi.h"

// Static storage for command line args
static int s_argc = 0;
static char** s_argv = NULL;

// Pending browser creation info
static char* s_initial_url = NULL;
static int s_initial_width = 800;
static int s_initial_height = 600;
static int s_create_browser_pending = 0;

void cef_set_args(int argc, char** argv) {
    s_argc = argc;
    if (argc > 0 && argv) {
        s_argv = (char**)malloc(sizeof(char*) * argc);
        for (int i = 0; i < argc; i++) {
            s_argv[i] = argv[i] ? strdup(argv[i]) : NULL;
        }
    }
}

// Maximum number of browsers we can track
#define MAX_BROWSERS 64

// Browser entry
typedef struct {
    cef_browser_t* browser;
    int active;
} browser_entry_t;

// Global state
static int g_initialized = 0;
static cef_app_t* g_app = NULL;
static browser_entry_t g_browsers[MAX_BROWSERS];
static cef_js_callback_fn g_js_callback = NULL;

// Forward declarations
static void init_browser_array(void);
static int add_browser(cef_browser_t* browser);
static void remove_browser(cef_browser_t* browser);
static cef_client_t* create_client(void);
static cef_life_span_handler_t* create_life_span_handler(void);
static void create_pending_browser(void);

// Helper: Set CEF string from UTF8
static void set_cef_string(const char* src, cef_string_t* dest) {
    if (!src || !dest) return;
    memset(dest, 0, sizeof(cef_string_t));
    cef_string_utf8_to_utf16(src, strlen(src), dest);
}

// Base ref counting
static void CEF_CALLBACK base_add_ref(cef_base_ref_counted_t* self) { (void)self; }
static int CEF_CALLBACK base_release(cef_base_ref_counted_t* self) { (void)self; return 1; }
static int CEF_CALLBACK base_has_one_ref(cef_base_ref_counted_t* self) { (void)self; return 1; }
static int CEF_CALLBACK base_has_at_least_one_ref(cef_base_ref_counted_t* self) { (void)self; return 1; }

static void init_base(cef_base_ref_counted_t* base, size_t size) {
    memset(base, 0, size);
    base->size = size;
    base->add_ref = base_add_ref;
    base->release = base_release;
    base->has_one_ref = base_has_one_ref;
    base->has_at_least_one_ref = base_has_at_least_one_ref;
}

// Browser Process Handler callbacks
static void CEF_CALLBACK on_context_initialized(cef_browser_process_handler_t* self) {
    (void)self;
    fprintf(stderr, "DEBUG: OnContextInitialized called\n");
    if (s_create_browser_pending) {
        create_pending_browser();
        s_create_browser_pending = 0;
    }
}

static void CEF_CALLBACK on_register_custom_prefs(cef_browser_process_handler_t* self,
    cef_preferences_type_t type, cef_preference_registrar_t* registrar) {
    (void)self; (void)type; (void)registrar;
}

static cef_browser_process_handler_t* create_browser_process_handler(void) {
    cef_browser_process_handler_t* handler = (cef_browser_process_handler_t*)calloc(1, sizeof(cef_browser_process_handler_t));
    init_base(&handler->base, sizeof(cef_browser_process_handler_t));
    handler->on_context_initialized = on_context_initialized;
    handler->on_register_custom_preferences = on_register_custom_prefs;
    return handler;
}

// App callbacks
static void CEF_CALLBACK app_on_before_command_line(cef_app_t* self, const cef_string_t* process_type, cef_command_line_t* command_line) {
    (void)self; (void)process_type;
    
    if (!command_line) return;
    
    // Disable GPU to avoid issues in container/headless environments
    cef_string_t switch_name;
    
    // --disable-gpu
    memset(&switch_name, 0, sizeof(switch_name));
    cef_string_utf8_to_utf16("disable-gpu", 11, &switch_name);
    command_line->append_switch(command_line, &switch_name);
    cef_string_clear(&switch_name);
    
    // --disable-software-rasterizer
    memset(&switch_name, 0, sizeof(switch_name));
    cef_string_utf8_to_utf16("disable-software-rasterizer", 27, &switch_name);
    command_line->append_switch(command_line, &switch_name);
    cef_string_clear(&switch_name);
}

static void CEF_CALLBACK app_on_register_custom_schemes(cef_app_t* self, cef_scheme_registrar_t* registrar) {
    (void)self; (void)registrar;
}

static cef_resource_bundle_handler_t* CEF_CALLBACK app_get_resource_bundle_handler(cef_app_t* self) {
    (void)self; return NULL;
}

static cef_browser_process_handler_t* CEF_CALLBACK app_get_browser_process_handler(cef_app_t* self) {
    (void)self;
    return create_browser_process_handler();
}

static cef_render_process_handler_t* CEF_CALLBACK app_get_render_process_handler(cef_app_t* self) {
    (void)self; return NULL;
}

// Client callbacks
#define NULL_HANDLER(name, type) static type* CEF_CALLBACK get_##name(cef_client_t* s) { (void)s; return NULL; }
NULL_HANDLER(context_menu_handler, cef_context_menu_handler_t)
NULL_HANDLER(dialog_handler, cef_dialog_handler_t)
NULL_HANDLER(display_handler, cef_display_handler_t)
NULL_HANDLER(download_handler, cef_download_handler_t)
NULL_HANDLER(drag_handler, cef_drag_handler_t)
NULL_HANDLER(find_handler, cef_find_handler_t)
NULL_HANDLER(focus_handler, cef_focus_handler_t)
NULL_HANDLER(jsdialog_handler, cef_jsdialog_handler_t)
NULL_HANDLER(keyboard_handler, cef_keyboard_handler_t)
NULL_HANDLER(load_handler, cef_load_handler_t)
NULL_HANDLER(render_handler, cef_render_handler_t)
NULL_HANDLER(request_handler, cef_request_handler_t)

static int CEF_CALLBACK client_on_process_message(cef_client_t* self, cef_browser_t* browser, cef_frame_t* frame,
    cef_process_id_t source_process, cef_process_message_t* message) {
    (void)self; (void)browser; (void)frame; (void)source_process; (void)message;
    return 0;
}

// Life span callbacks
static void CEF_CALLBACK life_span_on_after_created(cef_life_span_handler_t* self, cef_browser_t* browser) {
    (void)self;
    add_browser(browser);
    
    cef_frame_t* frame = browser->get_main_frame(browser);
    if (frame) {
        cef_string_t js;
        set_cef_string("window.CEF={post:function(m){if(window.cefQuery)window.cefQuery({request:m})},invoke:function(n,a){this.post(JSON.stringify({name:n,args:a}))}};", &js);
        frame->execute_java_script(frame, &js, NULL, 0);
        cef_string_clear(&js);
    }
}

static int CEF_CALLBACK life_span_do_close(cef_life_span_handler_t* self, cef_browser_t* browser) {
    (void)self; (void)browser;
    return 0;
}

static void CEF_CALLBACK life_span_on_before_close(cef_life_span_handler_t* self, cef_browser_t* browser) {
    (void)self;
    remove_browser(browser);
}

static int CEF_CALLBACK life_span_on_before_popup(cef_life_span_handler_t* self, cef_browser_t* browser, cef_frame_t* frame,
    int popup_id, const cef_string_t* target_url, const cef_string_t* target_frame_name, cef_window_open_disposition_t target_disposition,
    int user_gesture, const cef_popup_features_t* popupFeatures, cef_window_info_t* windowInfo, cef_client_t** client,
    cef_browser_settings_t* settings, cef_dictionary_value_t** extra_info, int* no_javascript_access) {
    (void)self; (void)browser; (void)frame; (void)popup_id; (void)target_url; (void)target_frame_name;
    (void)target_disposition; (void)user_gesture; (void)popupFeatures; (void)windowInfo;
    (void)client; (void)settings; (void)extra_info;
    *no_javascript_access = 1;
    return 1;
}

// Browser array management
static void init_browser_array(void) { memset(g_browsers, 0, sizeof(g_browsers)); }

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

// Create handlers
static cef_life_span_handler_t* create_life_span_handler(void) {
    cef_life_span_handler_t* h = (cef_life_span_handler_t*)calloc(1, sizeof(cef_life_span_handler_t));
    init_base(&h->base, sizeof(cef_life_span_handler_t));
    h->on_after_created = life_span_on_after_created;
    h->do_close = life_span_do_close;
    h->on_before_close = life_span_on_before_close;
    h->on_before_popup = life_span_on_before_popup;
    return h;
}

static cef_life_span_handler_t* CEF_CALLBACK client_get_life_span_handler(cef_client_t* s) {
    (void)s;
    return create_life_span_handler();
}

static cef_client_t* create_client(void) {
    cef_client_t* c = (cef_client_t*)calloc(1, sizeof(cef_client_t));
    init_base(&c->base, sizeof(cef_client_t));
    c->get_context_menu_handler = get_context_menu_handler;
    c->get_dialog_handler = get_dialog_handler;
    c->get_display_handler = get_display_handler;
    c->get_download_handler = get_download_handler;
    c->get_drag_handler = get_drag_handler;
    c->get_find_handler = get_find_handler;
    c->get_focus_handler = get_focus_handler;
    c->get_jsdialog_handler = get_jsdialog_handler;
    c->get_keyboard_handler = get_keyboard_handler;
    c->get_life_span_handler = client_get_life_span_handler;
    c->get_load_handler = get_load_handler;
    c->get_render_handler = get_render_handler;
    c->get_request_handler = get_request_handler;
    c->on_process_message_received = client_on_process_message;
    return c;
}

// Create browser from pending info
static void create_pending_browser(void) {
    if (!s_initial_url) return;
    
    fprintf(stderr, "DEBUG: Creating pending browser with URL: %s\n", s_initial_url);
    
    cef_window_info_t window_info;
    memset(&window_info, 0, sizeof(window_info));
    
    cef_browser_settings_t browser_settings;
    memset(&browser_settings, 0, sizeof(browser_settings));
    browser_settings.size = sizeof(cef_browser_settings_t);
    
    cef_string_t cef_url;
    set_cef_string(s_initial_url, &cef_url);
    
    cef_client_t* client = create_client();
    cef_browser_t* browser = cef_browser_host_create_browser_sync(&window_info, client, &cef_url, &browser_settings, NULL, NULL);
    
    cef_string_clear(&cef_url);
    
    if (browser) {
        fprintf(stderr, "DEBUG: Browser created successfully\n");
        add_browser(browser);
    } else {
        fprintf(stderr, "DEBUG: Failed to create browser\n");
    }
}

// Public API
int cef_wrapper_initialize() {
    fprintf(stderr, "DEBUG: cef_wrapper_initialize called\n");
    
    if (g_initialized) return 1;
    
    init_browser_array();
    
    g_app = (cef_app_t*)calloc(1, sizeof(cef_app_t));
    init_base(&g_app->base, sizeof(cef_app_t));
    g_app->on_before_command_line_processing = app_on_before_command_line;
    g_app->on_register_custom_schemes = app_on_register_custom_schemes;
    g_app->get_resource_bundle_handler = app_get_resource_bundle_handler;
    g_app->get_browser_process_handler = app_get_browser_process_handler;
    g_app->get_render_process_handler = app_get_render_process_handler;
    
    cef_main_args_t main_args = { s_argc, s_argv };
    
    cef_settings_t settings;
    memset(&settings, 0, sizeof(settings));
    settings.size = sizeof(cef_settings_t);
    settings.no_sandbox = 1;
    settings.multi_threaded_message_loop = 0;
    settings.windowless_rendering_enabled = 0;
    
    int ret = cef_initialize(&main_args, &settings, g_app, NULL);
    fprintf(stderr, "DEBUG: cef_initialize returned %d\n", ret);
    
    if (!ret) {
        fprintf(stderr, "cef_wrapper_initialize: cef_initialize failed\n");
        return 0;
    }
    
    g_initialized = 1;
    return 1;
}

int cef_wrapper_initialize_with_browser(const char* url, int width, int height) {
    fprintf(stderr, "DEBUG: cef_wrapper_initialize_with_browser called\n");
    
    // Queue browser creation BEFORE initializing CEF
    // This ensures the browser will be created in OnContextInitialized
    if (s_initial_url) free(s_initial_url);
    s_initial_url = url ? strdup(url) : strdup("about:blank");
    s_initial_width = width;
    s_initial_height = height;
    s_create_browser_pending = 1;
    
    fprintf(stderr, "DEBUG: Browser queued: %s (%dx%d)\n", s_initial_url, width, height);
    
    // Now initialize CEF
    return cef_wrapper_initialize();
}

void cef_wrapper_run() {
    if (g_initialized) cef_run_message_loop();
}

void cef_wrapper_shutdown() {
    if (!g_initialized) return;
    
    for (int i = 0; i < MAX_BROWSERS; i++) {
        if (g_browsers[i].active && g_browsers[i].browser) {
            cef_browser_host_t* host = g_browsers[i].browser->get_host(g_browsers[i].browser);
            if (host) host->close_browser(host, 1);
        }
    }
    
    cef_shutdown();
    g_initialized = 0;
}

cef_browser_t* cef_browser_create(const char* url, int width, int height) {
    if (!g_initialized) {
        fprintf(stderr, "CEF not initialized\n");
        return NULL;
    }
    
    // Store the request for creation in OnContextInitialized
    if (s_initial_url) free(s_initial_url);
    s_initial_url = url ? strdup(url) : strdup("about:blank");
    s_initial_width = width;
    s_initial_height = height;
    s_create_browser_pending = 1;
    
    fprintf(stderr, "DEBUG: Browser creation queued for OnContextInitialized\n");
    return NULL;  // Will be created asynchronously
}

void cef_browser_load_url(cef_browser_t* browser, const char* url) {
    if (!browser || !url) return;
    cef_frame_t* frame = browser->get_main_frame(browser);
    if (!frame) return;
    cef_string_t cef_url;
    set_cef_string(url, &cef_url);
    frame->load_url(frame, &cef_url);
    cef_string_clear(&cef_url);
}

void cef_browser_execute_js(cef_browser_t* browser, const char* js) {
    if (!browser || !js) return;
    cef_frame_t* frame = browser->get_main_frame(browser);
    if (!frame) return;
    cef_string_t cef_js;
    set_cef_string(js, &cef_js);
    frame->execute_java_script(frame, &cef_js, NULL, 0);
    cef_string_clear(&cef_js);
}

void cef_browser_destroy(cef_browser_t* browser) {
    if (!browser) return;
    cef_browser_host_t* host = browser->get_host(browser);
    if (host) host->close_browser(host, 1);
}

void* cef_get_native_window_handle(cef_browser_t* browser) {
    if (!browser) return NULL;
    cef_browser_host_t* host = browser->get_host(browser);
    if (!host) return NULL;
    return (void*)host->get_window_handle(host);
}

void cef_register_js_callback(const char* name, cef_js_callback_fn callback) {
    (void)name;
    g_js_callback = callback;
}

void cef_send_message_to_js(cef_browser_t* browser, const char* message) {
    if (!browser || !message) return;
    
    size_t len = strlen(message);
    char* escaped = (char*)malloc(len * 2 + 1);
    if (!escaped) return;
    
    size_t j = 0;
    for (size_t i = 0; i < len; i++) {
        if (message[i] == '\\' || message[i] == '"' || message[i] == '\'') {
            escaped[j++] = '\\';
        }
        escaped[j++] = message[i];
    }
    escaped[j] = '\0';
    
    char* js = (char*)malloc(j + 128);
    if (js) {
        sprintf(js, "if(window.onCEFMessage)window.onCEFMessage('%s')", escaped);
        cef_browser_execute_js(browser, js);
        free(js);
    }
    free(escaped);
}

int cef_is_subprocess() {
    for (int i = 0; i < s_argc; i++) {
        if (s_argv[i] && strncmp(s_argv[i], "--type=", 7) == 0) {
            return 1;
        }
    }
    return 0;
}

void cef_subprocess_entry() {
    cef_app_t* app = (cef_app_t*)calloc(1, sizeof(cef_app_t));
    init_base(&app->base, sizeof(cef_app_t));
    app->on_before_command_line_processing = app_on_before_command_line;
    app->on_register_custom_schemes = app_on_register_custom_schemes;
    app->get_resource_bundle_handler = app_get_resource_bundle_handler;
    app->get_browser_process_handler = app_get_browser_process_handler;
    app->get_render_process_handler = app_get_render_process_handler;
    
    cef_main_args_t main_args = { s_argc, s_argv };
    int exit_code = cef_execute_process(&main_args, app, NULL);
    exit(exit_code);
}
