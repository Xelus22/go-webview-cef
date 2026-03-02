// Package webview provides a webview-compatible API using CEF as the backend.
// This is designed as a drop-in replacement for github.com/webview/webview
package webview

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/xelus/go-webview-cef/cef"
)

// Window hints for SetSize
const (
	HintNone  = 0
	HintFixed = 1
	HintMin   = 2
	HintMax   = 3
)

// WindowOptions contains optional settings for NewWindow
// This provides compatibility with webview library usage patterns
type WindowOptions struct {
	Title      string
	Width      int
	Height     int
	Resizable  bool
	MinWidth   int
	MinHeight  int
	MaxWidth   int
	MaxHeight  int
	Fullscreen bool
	Borderless bool
	Center     bool
}

// DialogType represents the type of dialog to show
// Used for alert/confirm/dialog compatibility
type DialogType int

const (
	DialogTypeAlert DialogType = iota
	DialogTypeConfirm
	DialogTypePrompt
	DialogTypeOpenFile
	DialogTypeSaveFile
)

// BindingResult represents the result of a JS binding call
type BindingResult struct {
	ID     string      `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// BindingCall represents a call from JavaScript to Go
type BindingCall struct {
	Name string        `json:"name"`
	ID   string        `json:"id"`
	Args []interface{} `json:"args"`
}

// Event represents a webview event
// Used for event handling compatibility
type Event struct {
	Type string
	Data interface{}
}

// EventType constants for common events
const (
	EventDOMReady     = "domready"
	EventNavigate     = "navigate"
	EventLoadStart    = "loadstart"
	EventLoadEnd      = "loadend"
	EventTitleChanged = "titlechanged"
	EventFocus        = "focus"
	EventBlur         = "blur"
	EventClose        = "close"
)

// SizeHint provides a type-safe way to specify size hints
// This is an alternative to the raw int constants
type SizeHint int

func (h SizeHint) Int() int {
	return int(h)
}

// Predefined size hints using the type
const (
	SizeHintNone  SizeHint = HintNone
	SizeHintFixed SizeHint = HintFixed
	SizeHintMin   SizeHint = HintMin
	SizeHintMax   SizeHint = HintMax
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
	webview     *cef.WebView
	title       string
	width       int
	height      int
	resizable   bool
	fullscreen  bool
	maximized   bool
	x           int
	y           int
	url         string
	htmlContent string
	chromeless  bool
	frameless   bool
	initScripts []string

	// WebView2-compat options mapped when possible and warned when unsupported.
	dataPath                               string
	browserPath                            string
	additionalBrowserArgs                  []string
	language                               string
	targetCompatibleBrowserVersion         string
	allowSingleSignOnUsingOSPrimaryAccount bool
	exclusiveUserDataFolderAccess          bool

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
		resizable: true,
		x:         -1,
		y:         -1,
		url:       "about:blank",
	}

	runtime.SetFinalizer(w, (*webview).Destroy)
	return w
}

// Run starts the main loop
func (w *webview) Run() {
	w.applyWebView2CompatOptions()

	// Create CEF WebView with options
	opts := cef.DefaultOptions()
	opts.Title = w.title
	opts.Width = w.width
	opts.Height = w.height
	opts.Resizable = w.resizable
	opts.Fullscreen = w.fullscreen
	opts.Maximized = w.maximized
	opts.X = w.x
	opts.Y = w.y
	opts.Frameless = w.frameless
	opts.Chromeless = w.chromeless
	if w.htmlContent != "" {
		opts.URL = "about:blank"
	} else {
		opts.URL = w.url
	}

	w.webview = cef.New(opts)

	for _, js := range w.initScripts {
		w.webview.Eval(js)
	}

	// Set up bindings
	for name, fn := range w.bindings {
		w.webview.Bind(name, fn.Interface())
	}

	// Navigate to URL or inject HTML
	if w.htmlContent != "" {
		// Inject HTML via JavaScript
		html := strings.ReplaceAll(w.htmlContent, `"`, `\"`)
		html = strings.ReplaceAll(html, "\n", "")
		w.webview.Eval(`document.open();document.write("` + html + `");document.close();`)
	} else {
		w.webview.Navigate(w.url)
	}

	// Create browser + run message loop.
	w.webview.Run()
}

func (w *webview) applyWebView2CompatOptions() {
	args := make([]string, 0, len(w.additionalBrowserArgs)+1)
	for _, arg := range w.additionalBrowserArgs {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			continue
		}
		args = append(args, arg)
	}
	if w.language != "" {
		args = append(args, "--lang="+w.language)
	}

	for _, arg := range args {
		found := false
		for _, existing := range os.Args {
			if existing == arg {
				found = true
				break
			}
		}
		if !found {
			os.Args = append(os.Args, arg)
		}
	}

	unsupported := make([]string, 0, 5)
	if w.dataPath != "" {
		unsupported = append(unsupported, "DataPath")
	}
	if w.browserPath != "" {
		unsupported = append(unsupported, "BrowserPath")
	}
	if w.targetCompatibleBrowserVersion != "" {
		unsupported = append(unsupported, "TargetCompatibleBrowserVersion")
	}
	if w.allowSingleSignOnUsingOSPrimaryAccount {
		unsupported = append(unsupported, "AllowSingleSignOnUsingOSPrimaryAccount")
	}
	if w.exclusiveUserDataFolderAccess {
		unsupported = append(unsupported, "ExclusiveUserDataFolderAccess")
	}

	if len(unsupported) > 0 {
		log.Printf("webview adapter: WebView2 options not mapped by CEF backend: %s", strings.Join(unsupported, ", "))
	}
}

// Terminate signals the webview to close
func (w *webview) Terminate() {
	close(w.terminate)
}

// Dispatch schedules a function on the main thread
func (w *webview) Dispatch(f func()) {
	if w.webview != nil {
		w.webview.Dispatch(f)
	}
}

// Destroy cleans up the webview
func (w *webview) Destroy() {
	if w.webview != nil {
		w.webview.Destroy()
		w.webview = nil
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
	if w.webview != nil {
		w.webview.SetTitle(title)
	}
}

// SetSize sets the window size
func (w *webview) SetSize(width int, height int, hint int) {
	w.width = width
	w.height = height
	if w.webview != nil {
		w.webview.SetSize(width, height)
	}
	_ = hint // hint not implemented
}

// SetURL sets the initial URL for the browser (must be called before Run)
func (w *webview) SetURL(url string) {
	w.url = url
}

// Navigate loads a URL
func (w *webview) Navigate(url string) {
	w.url = url
	if w.webview != nil {
		w.webview.Navigate(url)
	}
}

// Init injects JavaScript to be available on all pages
func (w *webview) Init(js string) {
	if w.webview == nil {
		w.initScripts = append(w.initScripts, js)
		return
	}
	w.webview.Eval(js)
}

// Eval executes JavaScript
func (w *webview) Eval(js string) {
	if w.webview == nil {
		w.initScripts = append(w.initScripts, js)
		return
	}
	w.webview.Eval(js)
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

	if w.webview != nil {
		return w.webview.Bind(name, fn)
	}

	return nil
}

// Unbind removes a binding
func (w *webview) Unbind(name string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.bindings, name)
	if w.webview != nil {
		w.webview.Unbind(name)
	}
	return nil
}

//
// Tauri/Wails-style App API
//

// AppConfig holds configuration for the app
type AppConfig struct {
	Title      string
	Width      int
	Height     int
	Resizable  bool
	Fullscreen bool
	Maximized  bool
	X          int
	Y          int
	Debug      bool
	Chromeless bool // Remove browser decorations (no URL bar, etc.)
	Frameless  bool // Remove OS window decorations (no title bar, borders)

	// WebView2-compat environment options.
	DataPath                               string
	BrowserPath                            string
	AdditionalBrowserArgs                  []string
	Language                               string
	TargetCompatibleBrowserVersion         string
	AllowSingleSignOnUsingOSPrimaryAccount bool
	ExclusiveUserDataFolderAccess          bool
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
	if !config.Resizable {
		// Keep parity with previous defaults.
		config.Resizable = true
	}
	if config.X == 0 {
		config.X = -1
	}
	if config.Y == 0 {
		config.Y = -1
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
	w := New(a.config.Debug)
	a.webview = w

	// Set properties
	w.SetTitle(a.config.Title)
	w.SetSize(a.config.Width, a.config.Height, HintNone)

	// Bind all registered functions
	for name, fn := range a.bindings {
		if err := w.Bind(name, fn); err != nil {
			return fmt.Errorf("failed to bind %s: %w", name, err)
		}
	}

	// Inject bindings bridge
	a.injectBindingsBridge()

	// Set chromeless and frameless mode on webview
	if wv, ok := w.(*webview); ok {
		wv.resizable = a.config.Resizable
		wv.fullscreen = a.config.Fullscreen
		wv.maximized = a.config.Maximized
		wv.x = a.config.X
		wv.y = a.config.Y
		wv.frameless = a.config.Frameless
		wv.chromeless = a.config.Chromeless
		wv.dataPath = a.config.DataPath
		wv.browserPath = a.config.BrowserPath
		wv.additionalBrowserArgs = append([]string(nil), a.config.AdditionalBrowserArgs...)
		wv.language = a.config.Language
		wv.targetCompatibleBrowserVersion = a.config.TargetCompatibleBrowserVersion
		wv.allowSingleSignOnUsingOSPrimaryAccount = a.config.AllowSingleSignOnUsingOSPrimaryAccount
		wv.exclusiveUserDataFolderAccess = a.config.ExclusiveUserDataFolderAccess
	}

	// Load HTML if provided
	if a.html != "" {
		if wv, ok := w.(*webview); ok {
			wv.htmlContent = a.html
		}
	}

	// Run the webview
	w.Run()

	return nil
}

// injectBindingsBridge injects JavaScript to set up window.go namespace
func (a *App) injectBindingsBridge() {
	if a.webview == nil {
		return
	}
	// Create the bindings bridge that exposes window.go.* functions
	bridge := `
		window.go = window.go || {};
		window.__goInvoke = async function(name, args) {
			const fn = window[name];
			if (!fn) throw new Error('Function ' + name + ' not found');
			return await fn.apply(null, args);
		};
	`
	a.webview.Init(bridge)
}
