//go:build linux || windows || darwin
// +build linux windows darwin

// Package wails provides CEF integration for Wails v3.
// This file contains Window methods shared across all platforms.
package wails

import (
	"github.com/xelus/go-webview-cef/internal/cefshim"
)

// Handle returns the opaque native CEF browser handle.
func (w *Window) Handle() uintptr {
	if w == nil {
		return 0
	}
	return uintptr(w.handle)
}

// SetCallbacks updates callbacks for the window.
func (w *Window) SetCallbacks(callbacks WindowCallbacks) {
	if w == nil {
		return
	}
	w.callbacksMu.Lock()
	w.callbacks = callbacks
	w.callbacksMu.Unlock()
}

// Navigate navigates the browser to url.
func (w *Window) Navigate(url string) {
	if w == nil || w.handle == 0 {
		return
	}
	cefshim.Navigate(w.handle, url)
}

// Eval executes JavaScript in the browser.
func (w *Window) Eval(js string) {
	if w == nil || w.handle == 0 {
		return
	}
	cefshim.Eval(w.handle, js)
}

// SetTitle updates the native window title.
func (w *Window) SetTitle(title string) {
	if w == nil || w.handle == 0 {
		return
	}
	cefshim.SetTitle(w.handle, title)
}

// SetSize resizes the native window.
func (w *Window) SetSize(width, height int) {
	if w == nil || w.handle == 0 {
		return
	}
	cefshim.Resize(w.handle, width, height)
}

// Show shows the native window.
func (w *Window) Show() {
	if w == nil || w.handle == 0 {
		return
	}
	cefshim.Show(w.handle)
}

// Hide hides the native window.
func (w *Window) Hide() {
	if w == nil || w.handle == 0 {
		return
	}
	cefshim.Hide(w.handle)
}

// Close closes the native window.
func (w *Window) Close() {
	if w == nil || w.handle == 0 {
		return
	}
	cefshim.Close(w.handle)
}

// IsValid reports whether the window handle is still valid.
func (w *Window) IsValid() bool {
	if w == nil || w.handle == 0 {
		return false
	}
	return cefshim.IsValid(w.handle)
}
