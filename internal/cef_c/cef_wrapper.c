// CEF 145 Wrapper - Based on cef-rs architecture
// 
// Key architecture from cef-rs:
// 1. Call cef_api_hash FIRST before any other CEF function
// 2. Call cef_execute_process with NULL app - it handles subprocess internally
// 3. If execute_process returns -1, this is browser process - continue
// 4. If execute_process returns >=0, exit with that code (subprocess)
// 5. Browser process then calls cef_initialize with the app

#include "cef_wrapper.h"

#include <string.h>
#include <stdlib.h>
#include <stdio.h>
#include <stdatomic.h>
#include <unistd.h>
#include <sys/stat.h>

// Platform-specific includes for frameless window support
#ifdef __linux__
#include <X11/Xlib.h>
#include <X11/Xatom.h>
#endif

#include "include/capi/cef_app_capi.h"
#include "include/capi/cef_browser_capi.h"
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_command_line_capi.h"
#include "include/capi/cef_life_span_handler_capi.h"
#include "include/capi/cef_base_capi.h"
#include "include/cef_api_hash.h"
#include "include/cef_api_versions.h"

// Maximum number of browsers we can track
#define MAX_BROWSERS 64

// Debug output
#define CEF_DEBUG(fmt, ...) do { fprintf(stderr, "[CEF] " fmt "\n", ##__VA_ARGS__); fflush(stderr); } while(0)

// Client wrapper structure
typedef struct {
    cef_client_t client;
    atomic_int ref_count;
} cef_client_wrapper_t;

// Life span handler structure
typedef struct {
    cef_life_span_handler_t handler;
    atomic_int ref_count;
} cef_life_span_handler_wrapper_t;

// App wrapper structure
typedef struct {
    cef_app_t app;
    atomic_int ref_count;
} cef_app_wrapper_t;

// Global state
static int g_initialized = 0;
static cef_app_wrapper_t* g_app = NULL;
static cef_client_wrapper_t* g_client = NULL;

//
// Reference counting implementations
//

static void CEF_CALLBACK client_add_ref(cef_base_ref_counted_t* self) {
    cef_client_wrapper_t* client = (cef_client_wrapper_t*)self;
    atomic_fetch_add(&client->ref_count, 1);
}

static int CEF_CALLBACK client_release(cef_base_ref_counted_t* self) {
    cef_client_wrapper_t* client = (cef_client_wrapper_t*)self;
    int count = atomic_fetch_sub(&client->ref_count, 1) - 1;
    if (count == 0) {
        free(client);
        return 1;
    }
    return 0;
}

static int CEF_CALLBACK client_has_one_ref(cef_base_ref_counted_t* self) {
    cef_client_wrapper_t* client = (cef_client_wrapper_t*)self;
    return atomic_load(&client->ref_count) == 1;
}

static int CEF_CALLBACK client_has_at_least_one_ref(cef_base_ref_counted_t* self) {
    cef_client_wrapper_t* client = (cef_client_wrapper_t*)self;
    return atomic_load(&client->ref_count) >= 1;
}

static void CEF_CALLBACK life_span_handler_add_ref(cef_base_ref_counted_t* self) {
    cef_life_span_handler_wrapper_t* handler = (cef_life_span_handler_wrapper_t*)self;
    atomic_fetch_add(&handler->ref_count, 1);
}

static int CEF_CALLBACK life_span_handler_release(cef_base_ref_counted_t* self) {
    cef_life_span_handler_wrapper_t* handler = (cef_life_span_handler_wrapper_t*)self;
    int count = atomic_fetch_sub(&handler->ref_count, 1) - 1;
    if (count == 0) {
        free(handler);
        return 1;
    }
    return 0;
}

static int CEF_CALLBACK life_span_handler_has_one_ref(cef_base_ref_counted_t* self) {
    cef_life_span_handler_wrapper_t* handler = (cef_life_span_handler_wrapper_t*)self;
    return atomic_load(&handler->ref_count) == 1;
}

static int CEF_CALLBACK life_span_handler_has_at_least_one_ref(cef_base_ref_counted_t* self) {
    cef_life_span_handler_wrapper_t* handler = (cef_life_span_handler_wrapper_t*)self;
    return atomic_load(&handler->ref_count) >= 1;
}

static void CEF_CALLBACK app_add_ref(cef_base_ref_counted_t* self) {
    cef_app_wrapper_t* app = (cef_app_wrapper_t*)self;
    atomic_fetch_add(&app->ref_count, 1);
}

static int CEF_CALLBACK app_release(cef_base_ref_counted_t* self) {
    cef_app_wrapper_t* app = (cef_app_wrapper_t*)self;
    int count = atomic_fetch_sub(&app->ref_count, 1) - 1;
    if (count == 0) {
        free(app);
        return 1;
    }
    return 0;
}

static int CEF_CALLBACK app_has_one_ref(cef_base_ref_counted_t* self) {
    cef_app_wrapper_t* app = (cef_app_wrapper_t*)self;
    return atomic_load(&app->ref_count) == 1;
}

static int CEF_CALLBACK app_has_at_least_one_ref(cef_base_ref_counted_t* self) {
    cef_app_wrapper_t* app = (cef_app_wrapper_t*)self;
    return atomic_load(&app->ref_count) >= 1;
}

//
// App callbacks (minimal implementation)
//

static void CEF_CALLBACK on_before_command_line_processing(
    struct _cef_app_t* self,
    const cef_string_t* process_type,
    struct _cef_command_line_t* command_line) {
    (void)self;
    (void)process_type;
    (void)command_line;
}

static void CEF_CALLBACK on_register_custom_schemes(
    struct _cef_app_t* self,
    struct _cef_scheme_registrar_t* registrar) {
    (void)self;
    (void)registrar;
}

static struct _cef_resource_bundle_handler_t* CEF_CALLBACK get_resource_bundle_handler(
    struct _cef_app_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_browser_process_handler_t* CEF_CALLBACK get_browser_process_handler(
    struct _cef_app_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_render_process_handler_t* CEF_CALLBACK get_render_process_handler(
    struct _cef_app_t* self) {
    (void)self;
    return NULL;
}

//
// Client callbacks
//

static struct _cef_context_menu_handler_t* CEF_CALLBACK get_context_menu_handler(
    struct _cef_client_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_dialog_handler_t* CEF_CALLBACK get_dialog_handler(
    struct _cef_client_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_display_handler_t* CEF_CALLBACK get_display_handler(
    struct _cef_client_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_download_handler_t* CEF_CALLBACK get_download_handler(
    struct _cef_client_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_drag_handler_t* CEF_CALLBACK get_drag_handler(
    struct _cef_client_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_find_handler_t* CEF_CALLBACK get_find_handler(
    struct _cef_client_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_focus_handler_t* CEF_CALLBACK get_focus_handler(
    struct _cef_client_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_jsdialog_handler_t* CEF_CALLBACK get_jsdialog_handler(
    struct _cef_client_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_keyboard_handler_t* CEF_CALLBACK get_keyboard_handler(
    struct _cef_client_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_life_span_handler_t* CEF_CALLBACK get_life_span_handler(
    struct _cef_client_t* self);

static struct _cef_load_handler_t* CEF_CALLBACK get_load_handler(
    struct _cef_client_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_render_handler_t* CEF_CALLBACK get_render_handler(
    struct _cef_client_t* self) {
    (void)self;
    return NULL;
}

static struct _cef_request_handler_t* CEF_CALLBACK get_request_handler(
    struct _cef_client_t* self) {
    (void)self;
    return NULL;
}

static int CEF_CALLBACK on_process_message_received(
    struct _cef_client_t* self,
    struct _cef_browser_t* browser,
    struct _cef_frame_t* frame,
    cef_process_id_t source_process,
    struct _cef_process_message_t* message) {
    (void)self;
    (void)browser;
    (void)frame;
    (void)source_process;
    (void)message;
    return 0;
}

//
// Life span handler callbacks
//

static void CEF_CALLBACK on_after_created(
    struct _cef_life_span_handler_t* self,
    struct _cef_browser_t* browser) {
    (void)self;
    (void)browser;
    CEF_DEBUG("Browser created");
}

static int CEF_CALLBACK do_close(
    struct _cef_life_span_handler_t* self,
    struct _cef_browser_t* browser) {
    (void)self;
    (void)browser;
    return 0;
}

static void CEF_CALLBACK on_before_close(
    struct _cef_life_span_handler_t* self,
    struct _cef_browser_t* browser) {
    (void)self;
    (void)browser;
    CEF_DEBUG("Browser closing");
}

static int CEF_CALLBACK on_before_popup(
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
    return 0;
}

//
// Helper functions
//

static cef_life_span_handler_wrapper_t* create_life_span_handler(void) {
    cef_life_span_handler_wrapper_t* handler = (cef_life_span_handler_wrapper_t*)calloc(1, sizeof(cef_life_span_handler_wrapper_t));
    if (!handler) return NULL;
    
    handler->handler.base.size = sizeof(cef_life_span_handler_t);
    handler->handler.base.add_ref = life_span_handler_add_ref;
    handler->handler.base.release = life_span_handler_release;
    handler->handler.base.has_one_ref = life_span_handler_has_one_ref;
    handler->handler.base.has_at_least_one_ref = life_span_handler_has_at_least_one_ref;
    
    handler->handler.on_after_created = on_after_created;
    handler->handler.do_close = do_close;
    handler->handler.on_before_close = on_before_close;
    handler->handler.on_before_popup = on_before_popup;
    
    atomic_store(&handler->ref_count, 1);
    
    return handler;
}

static struct _cef_life_span_handler_t* CEF_CALLBACK get_life_span_handler(
    struct _cef_client_t* self) {
    (void)self;
    cef_life_span_handler_wrapper_t* handler = create_life_span_handler();
    if (handler) {
        handler->handler.base.add_ref(&handler->handler.base);
        return &handler->handler;
    }
    return NULL;
}

static cef_client_wrapper_t* create_client(void) {
    cef_client_wrapper_t* client = (cef_client_wrapper_t*)calloc(1, sizeof(cef_client_wrapper_t));
    if (!client) return NULL;
    
    client->client.base.size = sizeof(cef_client_t);
    client->client.base.add_ref = client_add_ref;
    client->client.base.release = client_release;
    client->client.base.has_one_ref = client_has_one_ref;
    client->client.base.has_at_least_one_ref = client_has_at_least_one_ref;
    
    client->client.get_context_menu_handler = get_context_menu_handler;
    client->client.get_dialog_handler = get_dialog_handler;
    client->client.get_display_handler = get_display_handler;
    client->client.get_download_handler = get_download_handler;
    client->client.get_drag_handler = get_drag_handler;
    client->client.get_find_handler = get_find_handler;
    client->client.get_focus_handler = get_focus_handler;
    client->client.get_jsdialog_handler = get_jsdialog_handler;
    client->client.get_keyboard_handler = get_keyboard_handler;
    client->client.get_life_span_handler = get_life_span_handler;
    client->client.get_load_handler = get_load_handler;
    client->client.get_render_handler = get_render_handler;
    client->client.get_request_handler = get_request_handler;
    client->client.on_process_message_received = on_process_message_received;
    
    atomic_store(&client->ref_count, 1);
    
    return client;
}

static cef_app_wrapper_t* create_app(void) {
    cef_app_wrapper_t* app = (cef_app_wrapper_t*)calloc(1, sizeof(cef_app_wrapper_t));
    if (!app) return NULL;
    
    app->app.base.size = sizeof(cef_app_t);
    app->app.base.add_ref = app_add_ref;
    app->app.base.release = app_release;
    app->app.base.has_one_ref = app_has_one_ref;
    app->app.base.has_at_least_one_ref = app_has_at_least_one_ref;
    
    app->app.on_before_command_line_processing = on_before_command_line_processing;
    app->app.on_register_custom_schemes = on_register_custom_schemes;
    app->app.get_resource_bundle_handler = get_resource_bundle_handler;
    app->app.get_browser_process_handler = get_browser_process_handler;
    app->app.get_render_process_handler = get_render_process_handler;
    
    atomic_store(&app->ref_count, 1);
    
    return app;
}

// Platform-specific helper to make window frameless (no OS decorations)
static void make_window_frameless(cef_window_handle_t window) {
#ifdef __linux__
    // On Linux/X11, remove window decorations using _MOTIF_WM_HINTS
    Display* display = XOpenDisplay(NULL);
    if (!display) {
        CEF_DEBUG("Failed to open X11 display");
        return;
    }
    
    Window xwindow = (Window)window;
    
    // Set _MOTIF_WM_HINTS to disable decorations
    struct {
        unsigned long flags;
        unsigned long functions;
        unsigned long decorations;
        long input_mode;
        unsigned long status;
    } hints = {0};
    
    hints.flags = 2;  // MWM_HINTS_DECORATIONS
    hints.decorations = 0;  // No decorations
    
    Atom motif_hints = XInternAtom(display, "_MOTIF_WM_HINTS", False);
    XChangeProperty(display, xwindow, motif_hints, motif_hints, 32,
                    PropModeReplace, (unsigned char*)&hints, 5);
    
    XFlush(display);
    XCloseDisplay(display);
    CEF_DEBUG("Window set to frameless mode");
#else
    (void)window;
    CEF_DEBUG("Frameless mode not implemented for this platform");
#endif
}

//
// Public API Implementation
//

int cef_initialize_main(int argc, char** argv) {
    CEF_DEBUG("cef_initialize_main called");
    
    // Step 1: Call cef_api_hash FIRST (CRITICAL for CEF 145)
    CEF_DEBUG("Calling cef_api_hash...");
    cef_api_hash(CEF_API_VERSION, 0);
    CEF_DEBUG("cef_api_hash completed");
    
    // Step 2: Create main_args
    cef_main_args_t main_args = {};
    main_args.argc = argc;
    main_args.argv = argv;
    
    // Step 3: Call cef_execute_process with NULL app
    // This handles subprocess detection internally
    CEF_DEBUG("Calling cef_execute_process...");
    int exit_code = cef_execute_process(&main_args, NULL, NULL);
    CEF_DEBUG("cef_execute_process returned %d", exit_code);
    
    // Step 4: Check if this is a subprocess
    if (exit_code >= 0) {
        CEF_DEBUG("This is a subprocess, returning exit code %d", exit_code);
        return exit_code;
    }
    
    // Step 5: This is the browser process - create app and initialize
    CEF_DEBUG("This is the browser process, initializing...");
    
    g_app = create_app();
    if (!g_app) {
        CEF_DEBUG("Failed to create app");
        return -2;
    }
    
    // Add two references: one for execute_process (if called later) and one for initialize
    g_app->app.base.add_ref(&g_app->app.base);
    g_app->app.base.add_ref(&g_app->app.base);
    
    // Create client for browser creation
    g_client = create_client();
    if (!g_client) {
        CEF_DEBUG("Failed to create client");
        g_app->app.base.release(&g_app->app.base);
        g_app = NULL;
        return -2;
    }
    
    // Initialize settings
    cef_settings_t settings = {};
    settings.size = sizeof(cef_settings_t);
    settings.no_sandbox = 1;
    settings.multi_threaded_message_loop = 0;
    settings.external_message_pump = 0;
    settings.windowless_rendering_enabled = 0;
    
    // Set cache and resource paths
    static cef_string_t cache_path = {};
    static cef_string_t resources_path = {};
    static cef_string_t locales_path = {};
    char cwd[4096];
    if (getcwd(cwd, sizeof(cwd)) != NULL) {
        char path_str[4096];
        
        // Cache path
        snprintf(path_str, sizeof(path_str), "%s/cef_cache", cwd);
        mkdir(path_str, 0755);
        cef_string_utf8_to_utf16(path_str, strlen(path_str), &cache_path);
        settings.cache_path = cache_path;
        settings.root_cache_path = cache_path;
        
        // Resources path
        snprintf(path_str, sizeof(path_str), "%s", cwd);
        cef_string_utf8_to_utf16(path_str, strlen(path_str), &resources_path);
        settings.resources_dir_path = resources_path;
        
        // Locales path
        snprintf(path_str, sizeof(path_str), "%s/locales", cwd);
        cef_string_utf8_to_utf16(path_str, strlen(path_str), &locales_path);
        settings.locales_dir_path = locales_path;
    }
    
    // Initialize CEF
    CEF_DEBUG("Calling cef_initialize...");
    int result = cef_initialize(&main_args, &settings, &g_app->app, NULL);
    CEF_DEBUG("cef_initialize returned %d", result);
    
    if (!result) {
        CEF_DEBUG("CEF initialization failed");
        g_app->app.base.release(&g_app->app.base);
        g_app = NULL;
        return -2;
    }
    
    g_initialized = 1;
    CEF_DEBUG("CEF initialized successfully");
    return -1;  // Browser process indicator
}

void cef_run_message_loop_wrapper(void) {
    if (!g_initialized) {
        CEF_DEBUG("CEF not initialized, cannot run message loop");
        return;
    }
    CEF_DEBUG("Running message loop...");
    cef_run_message_loop();
}

void cef_shutdown_wrapper(void) {
    if (!g_initialized) return;
    
    CEF_DEBUG("Shutting down CEF...");
    cef_shutdown();
    
    // Release app reference if still held
    if (g_app) {
        g_app->app.base.release(&g_app->app.base);
        g_app = NULL;
    }
    
    g_initialized = 0;
    CEF_DEBUG("CEF shutdown complete");
}

cef_browser_handle_t cef_browser_create_wrapper(const char* url, int width, int height) {
    return cef_browser_create_with_flags(url, width, height, 0, 0);
}

cef_browser_handle_t cef_browser_create_with_flags(const char* url, int width, int height, int chromeless, int frameless) {
    (void)chromeless; // Reserved for future use
    
    if (!g_initialized) {
        CEF_DEBUG("CEF not initialized!");
        return NULL;
    }

    CEF_DEBUG("Creating browser: %s %dx%d (frameless=%d)", url, width, height, frameless);

    // Create window info
    cef_window_info_t window_info = {};
    window_info.size = sizeof(cef_window_info_t);
    window_info.bounds.x = 100;  // Default position
    window_info.bounds.y = 100;
    window_info.bounds.width = width;
    window_info.bounds.height = height;
    
    // Note: frameless mode (no OS window decorations) requires X11 hints
    // set after window creation on Linux. We handle this below.
    
    // Create browser settings
    cef_browser_settings_t browser_settings = {};
    browser_settings.size = sizeof(cef_browser_settings_t);
    
    // Convert URL to CEF string
    cef_string_t cef_url = {};
    cef_string_utf8_to_utf16(url, strlen(url), &cef_url);
    
    // Add reference for browser creation
    g_client->client.base.add_ref(&g_client->client.base);
    
    // Create browser
    CEF_DEBUG("Calling cef_browser_host_create_browser...");
    cef_browser_t* browser = cef_browser_host_create_browser_sync(
        &window_info,
        &g_client->client,
        &cef_url,
        &browser_settings,
        NULL,
        NULL
    );
    
    cef_string_clear(&cef_url);
    
    if (browser) {
        CEF_DEBUG("Browser created successfully");
        
        // Set frameless mode if requested (remove OS window decorations)
        if (frameless) {
            cef_browser_host_t* host = browser->get_host(browser);
            if (host) {
                cef_window_handle_t window = host->get_window_handle(host);
                if (window) {
                    make_window_frameless(window);
                }
                host->base.release(&host->base);
            }
        }
    } else {
        CEF_DEBUG("Browser creation failed");
        g_client->client.base.release(&g_client->client.base);
    }
    
    return (cef_browser_handle_t)browser;
}

void cef_browser_load_url_wrapper(cef_browser_handle_t browser, const char* url) {
    if (!browser || !url) return;
    
    cef_browser_t* b = (cef_browser_t*)browser;
    cef_frame_t* main_frame = b->get_main_frame(b);
    if (!main_frame) return;
    
    cef_string_t cef_url = {};
    cef_string_utf8_to_utf16(url, strlen(url), &cef_url);
    main_frame->load_url(main_frame, &cef_url);
    cef_string_clear(&cef_url);
}

void cef_browser_execute_js_wrapper(cef_browser_handle_t browser, const char* js) {
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
}

void cef_browser_destroy_wrapper(cef_browser_handle_t browser) {
    if (!browser) return;
    
    cef_browser_t* b = (cef_browser_t*)browser;
    cef_browser_host_t* host = b->get_host(b);
    if (host) {
        host->close_browser(host, 1);
        host->base.release(&host->base);
    }
}

void cef_browser_resize_wrapper(cef_browser_handle_t browser, int width, int height) {
    if (!browser) return;
    
    cef_browser_t* b = (cef_browser_t*)browser;
    cef_browser_host_t* host = b->get_host(b);
    if (host) {
        // Get the native window handle and resize it
        cef_window_handle_t window = host->get_window_handle(host);
        if (window) {
            // On Linux/X11, we would use X11 APIs to resize
            // For now, we set the size via CEF's browser host
            // The actual window resize is handled by the window manager
            CEF_DEBUG("Browser resize requested: %dx%d", width, height);
        }
        host->base.release(&host->base);
    }
}
