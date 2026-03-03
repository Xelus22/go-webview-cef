//go:build darwin
// +build darwin

package cefruntime

import "fmt"

// Options defines WebView window and runtime options.
type Options struct {
	Title        string
	URL          string
	Width        int
	Height       int
	Frameless    bool
	Chromeless   bool
	Resizable    bool
	Fullscreen   bool
	Maximized    bool
	X            int
	Y            int
	ParentWindow uintptr
	GPUMode      GPUMode
}

// GPUMode controls runtime GPU flag behavior.
type GPUMode int

const (
	// GPUAuto uses runtime defaults.
	GPUAuto GPUMode = iota
	// GPUEnabled avoids adding disable-gpu flags.
	GPUEnabled
	// GPUDisabled forces software/single-process-safe startup flags.
	GPUDisabled
)

// DefaultOptions returns default window options.
func DefaultOptions() Options {
	return Options{
		Title:     "App",
		Width:     800,
		Height:    600,
		Resizable: true,
		X:         -1,
		Y:         -1,
	}
}

// BrowserHandle is an opaque wrapper around a CEF browser handle.
type BrowserHandle struct {
	raw uintptr
}

func (h BrowserHandle) isZero() bool {
	return h.raw == 0
}

// Runtime defines the Go-owned runtime lifecycle.
type Runtime interface {
	Initialize(args []string) int
	CreateBrowser(opts Options) (BrowserHandle, error)
	SetMessageCallback(userData uintptr)
	SetLoadCallback(userData uintptr)
	SetCloseCallback(userData uintptr)
	SetRequestCallback(userData uintptr)
	InjectBridge(browser BrowserHandle)
	Navigate(browser BrowserHandle, url string)
	Eval(browser BrowserHandle, js string)
	SetTitle(browser BrowserHandle, title string)
	Resize(browser BrowserHandle, width, height int)
	Show(browser BrowserHandle)
	Hide(browser BrowserHandle)
	Close(browser BrowserHandle)
	IsValid(browser BrowserHandle) bool
	RunLoop()
	DoMessageLoopWork()
	Shutdown()
}

// WebView represents a runtime-managed browser window.
type WebView struct {
	// Stub implementation for Darwin
}

// New creates a new WebView instance.
func New(opts Options) *WebView {
	// Stub implementation
	return nil
}

// Run starts the CEF message loop.
func (w *WebView) Run() {
	// Stub implementation
}

// Navigate loads a URL.
func (w *WebView) Navigate(url string) {
	// Stub implementation
}

// Eval executes JavaScript.
func (w *WebView) Eval(js string) {
	// Stub implementation
}

// Bind binds a Go function to JavaScript.
func (w *WebView) Bind(name string, fn interface{}) error {
	return fmt.Errorf("not implemented on Darwin")
}

// Unbind removes an existing binding.
func (w *WebView) Unbind(name string) {
	// Stub implementation
}

// Dispatch queues a function for main-thread execution.
func (w *WebView) Dispatch(f func()) {
	// Stub implementation
}

// Destroy closes the browser.
func (w *WebView) Destroy() {
	// Stub implementation
}

// SetTitle sets the window title.
func (w *WebView) SetTitle(title string) {
	// Stub implementation
}

// SetSize sets the window size.
func (w *WebView) SetSize(width, height int) {
	// Stub implementation
}

// Show shows the window.
func (w *WebView) Show() {
	// Stub implementation
}

// Hide hides the window.
func (w *WebView) Hide() {
	// Stub implementation
}

// IsValid reports whether the browser handle remains valid.
func (w *WebView) IsValid() bool {
	return false
}

// Initialize initializes CEF.
func Initialize() bool {
	return false
}

// Run starts the CEF message loop.
func Run() {
	// Stub implementation
}

// DoMessageLoopWork processes one CEF message-loop iteration.
func DoMessageLoopWork() {
	// Stub implementation
}

// Shutdown shuts down CEF.
func Shutdown() {
	// Stub implementation
}

// DisableGPU enables GPU-disabling command line flags.
func DisableGPU() {
	// Stub implementation
}

// EnableGPU enables GPU-related browser processes.
func EnableGPU() {
	// Stub implementation
}
