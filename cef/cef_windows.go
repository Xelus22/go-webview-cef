//go:build windows
// +build windows

// Package cef provides a Go binding to the CEF runtime.
package cef

import "github.com/xelus/go-webview-cef/internal/cefruntime"

// Options for creating a WebView.
type Options = cefruntime.Options

// GPUMode controls runtime GPU flag behavior.
type GPUMode = cefruntime.GPUMode

const (
	GPUAuto     = cefruntime.GPUAuto
	GPUEnabled  = cefruntime.GPUEnabled
	GPUDisabled = cefruntime.GPUDisabled
)

// WebView represents a CEF WebView window.
type WebView = cefruntime.WebView

// DefaultOptions returns default WebView options.
func DefaultOptions() Options {
	return cefruntime.DefaultOptions()
}

// DisableGPU adds flags to disable GPU rendering.
func DisableGPU() {
	cefruntime.DisableGPU()
}

// EnableGPU enables GPU-related browser processes.
func EnableGPU() {
	cefruntime.EnableGPU()
}

// Initialize initializes CEF and handles subprocess detection.
func Initialize() bool {
	return cefruntime.Initialize()
}

// Run starts the CEF message loop (blocks until all windows closed).
func Run() {
	cefruntime.Run()
}

// DoMessageLoopWork processes a single CEF message loop iteration.
func DoMessageLoopWork() {
	cefruntime.DoMessageLoopWork()
}

// Shutdown cleans up CEF resources.
func Shutdown() {
	cefruntime.Shutdown()
}

// New creates a new WebView window.
func New(opts Options) *WebView {
	return cefruntime.New(opts)
}
