//go:build !linux && !darwin && !windows
// +build !linux,!darwin,!windows

package cef

import "fmt"

// GPUMode controls runtime GPU flag behavior.
type GPUMode int

const (
	GPUAuto GPUMode = iota
	GPUEnabled
	GPUDisabled
)

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

// WebView represents a runtime-managed browser window.
type WebView struct{}

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

// New creates a new WebView instance.
func New(opts Options) *WebView {
	return nil
}

// Run starts the CEF message loop.
func (w *WebView) Run() {
}

// Navigate loads a URL.
func (w *WebView) Navigate(url string) {
}

// Eval executes JavaScript.
func (w *WebView) Eval(js string) {
}

// Bind binds a Go function to JavaScript.
func (w *WebView) Bind(name string, fn interface{}) error {
	return fmt.Errorf("CEF not supported on this platform")
}

// Unbind removes an existing binding.
func (w *WebView) Unbind(name string) {
}

// Dispatch queues a function for main-thread execution.
func (w *WebView) Dispatch(f func()) {
}

// Destroy closes the browser.
func (w *WebView) Destroy() {
}

// SetTitle sets the window title.
func (w *WebView) SetTitle(title string) {
}

// SetSize sets the window size.
func (w *WebView) SetSize(width, height int) {
}

// Show shows the window.
func (w *WebView) Show() {
}

// Hide hides the window.
func (w *WebView) Hide() {
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
}

// DoMessageLoopWork processes one CEF message-loop iteration.
func DoMessageLoopWork() {
}

// Shutdown shuts down CEF.
func Shutdown() {
}

// DisableGPU enables GPU-disabling command line flags.
func DisableGPU() {
}

// EnableGPU enables GPU-related browser processes.
func EnableGPU() {
}
