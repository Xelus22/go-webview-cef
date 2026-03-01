// Package webview provides a webview-compatible API using CEF as the backend.
// This is designed as a drop-in replacement for github.com/webview/webview
package webview

import (
	"fmt"
	"reflect"
	"runtime"
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
	browser   *cef.Browser
	title     string
	width     int
	height    int
	bindings  map[string]reflect.Value
	mu        sync.RWMutex
	dispatch  chan func()
	terminate chan struct{}
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
	}

	runtime.SetFinalizer(w, (*webview).Destroy)
	return w
}

// Run starts the main loop
func (w *webview) Run() {
	if cef.IsSubprocess() {
		cef.SubprocessEntry()
		return
	}

	// Initialize CEF with browser (will be created in OnContextInitialized)
	cef.InitializeWithBrowser("about:blank", w.width, w.height)

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
}

// Window returns the native window handle
func (w *webview) Window() unsafe.Pointer {
	if w.browser == nil {
		return nil
	}
	return w.browser.NativeHandle()
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

// Navigate loads a URL
func (w *webview) Navigate(url string) {
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
