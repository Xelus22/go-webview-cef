//go:build linux || windows || darwin
// +build linux windows darwin

// Package wails provides CEF integration for Wails v3.
// This file contains shared types used across all platforms.
package wails

import "net/http"

// WindowOptions configures a CEF-backed window for Wails integration.
type WindowOptions struct {
	URL          string
	Title        string
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
}

// Request represents an intercepted in-process resource request.
type Request struct {
	Method  string
	URL     string
	Headers http.Header
	Body    []byte
}

// Response represents a response for an intercepted request.
type Response struct {
	StatusCode int
	StatusText string
	Headers    http.Header
	MimeType   string
	Body       []byte
}

// WindowCallbacks are callbacks bound to a specific window.
type WindowCallbacks struct {
	OnMessage func(message string)
	OnLoad    func(success bool)
	OnClose   func()
	OnRequest func(req Request) (Response, bool)
}
