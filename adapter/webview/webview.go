// Package webview provides a webview-compatible API using CEF as the backend.
// This is designed as a drop-in replacement for github.com/webview/webview
package webview

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/xelus/go-webview-cef/internal/cef"
)

// WebView interface matches the standard webview API
type WebView interface {
	Run()
	Terminate()
	Dispatch(f func())
	Destroy()
	Window() unsafe.Pointer
	SetTitle(title string)
	SetSize(width int, height int, hint int)
	Navigate(url string)
	Init(js string)
	Eval(js string)
	Bind(name string, fn interface{}) error
	Unbind(name string) error
}

// webview implements the WebView interface using CEF
type webview struct {
	browser     *cef.Browser
	title       string
	width       int
	height      int
	url         string
	htmlContent string // HTML to inject after load
	frameless   bool   // Frameless window (no OS decorations)
	bindings    map[string]reflect.Value
	mu          sync.RWMutex
	dispatch    chan func()
	terminate   chan struct{}
}

// New creates a new webview window
// Options: width, height, resizable (optional parameters)
func New(debug bool) WebView {
	w := &webview{
		bindings:  make(map[string]reflect.Value),
		dispatch:  make(chan func(), 100),
		terminate: make(chan struct{}),
		width:     800,
		height:    600,
		url:       "about:blank",
	}

	runtime.SetFinalizer(w, (*webview).Destroy)
	return w
}

// Run starts the main loop
func (w *webview) Run() {
	// Add flags to disable GPU for WSL compatibility
	cef.DisableGPU()

	// Initialize CEF (handles subprocess internally)
	if !cef.Initialize() {
		panic("Failed to initialize CEF")
	}

	// Create browser with frameless support
	if w.frameless {
		w.browser = cef.NewBrowserChromeless(w.url, w.width, w.height, true)
	} else {
		w.browser = cef.NewBrowser(w.url, w.width, w.height)
	}
	if w.browser == nil {
		panic("Failed to create browser")
	}

	// Inject HTML content if provided (more reliable than data URLs)
	if w.htmlContent != "" {
		// Escape quotes for JavaScript
		html := strings.ReplaceAll(w.htmlContent, `"`, `\"`)
		html = strings.ReplaceAll(html, "\n", "")
		w.browser.Eval(`document.open();document.write("` + html + `");document.close();`)
	}

	// Run CEF message loop
	cef.Run()
}

// Terminate signals the webview to close
func (w *webview) Terminate() {
	close(w.terminate)
}

// Dispatch schedules a function on the main thread
func (w *webview) Dispatch(f func()) {
	select {
	case w.dispatch <- f:
	default:
		// Channel full, drop or handle error
	}
}

// Destroy cleans up the webview
func (w *webview) Destroy() {
	if w.browser != nil {
		w.browser.Destroy()
		w.browser = nil
	}
	cef.Shutdown()
}

// Window returns the native window handle
func (w *webview) Window() unsafe.Pointer {
	// Native handle not exposed in simplified API
	return nil
}

// SetTitle sets the window title
func (w *webview) SetTitle(title string) {
	w.title = title
	// CEF doesn't have direct window title API in this minimal binding
	// Would need platform-specific implementation
}

// SetSize sets the window size
func (w *webview) SetSize(width int, height int, hint int) {
	w.width = width
	w.height = height
	// Actual resize would need platform-specific implementation
}

// SetURL sets the initial URL for the browser (must be called before Run)
func (w *webview) SetURL(url string) {
	w.url = url
}

// Navigate loads a URL
func (w *webview) Navigate(url string) {
	w.url = url
	if w.browser != nil {
		w.browser.Navigate(url)
	}
}

// Init injects JavaScript to be available on all pages
func (w *webview) Init(js string) {
	// Store for injection on page load
	// Simplified implementation - inject immediately if browser exists
	w.Eval(js)
}

// Eval executes JavaScript
func (w *webview) Eval(js string) {
	if w.browser != nil {
		w.browser.Eval(js)
	}
}

// Bind binds a Go function to be callable from JavaScript
func (w *webview) Bind(name string, fn interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		return fmt.Errorf("bind: %s is not a function", name)
	}

	w.bindings[name] = v

	// Inject JavaScript binding
	w.injectBinding(name, v.Type())

	return nil
}

// Unbind removes a binding
func (w *webview) Unbind(name string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.bindings, name)
	return nil
}

// injectBinding creates JavaScript wrapper for Go function
func (w *webview) injectBinding(name string, t reflect.Type) {
	// Generate JS wrapper that calls window.external.invoke
	js := fmt.Sprintf(`
        window.%s = function(...args) {
            return new Promise((resolve, reject) => {
                const id = Math.random().toString(36).substr(2, 9);
                window.__goCallbacks[id] = { resolve, reject };
                window.external.invoke(JSON.stringify({
                    name: "%s",
                    id: id,
                    args: args
                }));
            });
        };
    `, name, name)

	w.Init(js)
	w.Init(`window.__goCallbacks = window.__goCallbacks || {};`)
}

// handleBindingCall processes a JavaScript binding call
func (w *webview) handleBindingCall(message string) {
	// Parse message and call appropriate binding
	// This would be called from the C layer when JS invokes
}

//
// Tauri/Wails-style App API
//

// AppConfig holds configuration for the app
type AppConfig struct {
	Title      string
	Width      int
	Height     int
	Debug      bool
	Chromeless bool // Remove browser decorations (no URL bar, etc.)
	Frameless  bool // Remove OS window decorations (no title bar, borders)
}

// App provides a high-level API similar to Tauri/Wails
type App struct {
	config   AppConfig
	bindings map[string]interface{}
	webview  WebView
	html     string
}

// NewApp creates a new Tauri-style app
func NewApp(config AppConfig) *App {
	if config.Width == 0 {
		config.Width = 800
	}
	if config.Height == 0 {
		config.Height = 600
	}

	return &App{
		config:   config,
		bindings: make(map[string]interface{}),
	}
}

// Register binds a Go function to be callable from JavaScript
func (a *App) Register(name string, fn interface{}) {
	a.bindings[name] = fn
}

// LoadHTML sets the HTML content for the app
func (a *App) LoadHTML(html string) {
	a.html = html
}

// Run starts the application
func (a *App) Run() error {
	// Create webview
	a.webview = New(a.config.Debug)

	// Set properties
	a.webview.SetTitle(a.config.Title)
	a.webview.SetSize(a.config.Width, a.config.Height, HintNone)

	// Bind all registered functions
	for name, fn := range a.bindings {
		if err := a.webview.Bind(name, fn); err != nil {
			return fmt.Errorf("failed to bind %s: %w", name, err)
		}
	}

	// Inject bindings bridge
	a.injectBindingsBridge()

	// Load HTML if provided - set URL before Run so browser is created with it
	if a.html != "" {
		// Type assert to set URL and frameless mode before browser creation
		if wv, ok := a.webview.(*webview); ok {
			// Use about:blank first, then inject HTML via JavaScript for better compatibility
			wv.SetURL("about:blank")
			// Store HTML for injection after load
			wv.htmlContent = a.html
			// Set frameless mode
			wv.frameless = a.config.Frameless
		}
	}

	// Run the webview (chromeless mode set via DisableGPU flags if needed)
	if a.config.Chromeless {
		cef.DisableGPU() // Helps with chromeless rendering
	}
	a.webview.Run()

	return nil
}

// injectBindingsBridge injects JavaScript to set up window.go namespace
func (a *App) injectBindingsBridge() {
	// Create the bindings bridge that exposes window.go.* functions
	bridge := `
		window.go = {};
		window.__goInvoke = async function(name, args) {
			const fn = window[name];
			if (!fn) throw new Error('Function ' + name + ' not found');
			return await fn.apply(null, args);
		};
	`
	a.webview.Init(bridge)
}
