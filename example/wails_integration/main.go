//go:build linux
// +build linux

// Example demonstrating Wails v3 integration with CEF backend.
// This shows how the Wails fork would use the go-webview-cef integration package.
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/xelus/go-webview-cef/integration/wails"
)

func main() {
	// Initialize CEF runtime
	rt, exitCode, err := wails.Initialize([]string{"wails-cef-example"})
	if err != nil {
		log.Fatalf("Failed to initialize CEF: %v", err)
	}
	if exitCode >= 0 {
		// This is a subprocess, exit with the code
		return
	}
	defer rt.Shutdown()

	// Create a window with Wails-compatible options
	window, err := rt.CreateWindow(wails.WindowOptions{
		Title:      "Wails v3 CEF Integration Example",
		URL:        "wails://localhost/",
		Width:      1024,
		Height:     768,
		Resizable:  true,
		Chromeless: false,
	})
	if err != nil {
		log.Fatalf("Failed to create window: %v", err)
	}

	// Set up callbacks for Wails-style communication
	window.SetCallbacks(wails.WindowCallbacks{
		OnMessage: func(message string) {
			fmt.Printf("Received message from JS: %s\n", message)

			// Handle Wails messages
			if strings.HasPrefix(message, "wails:") {
				handleWailsMessage(window, message)
			}
		},
		OnLoad: func(success bool) {
			if success {
				fmt.Println("Page loaded successfully")
				// Inject Wails runtime-ready signal
				window.Eval(`window.dispatchEvent(new CustomEvent('wails:runtime:ready'));`)
			} else {
				fmt.Println("Page failed to load")
			}
		},
		OnClose: func() {
			fmt.Println("Window closing")
		},
		OnRequest: func(req wails.Request) (wails.Response, bool) {
			// Handle wails://localhost/ requests (in-process asset serving)
			if strings.HasPrefix(req.URL, "wails://localhost/") {
				return handleAssetRequest(req)
			}
			return wails.Response{}, false // Not handled, let CEF handle it
		},
	})

	// Show the window
	window.Show()

	// Run the message loop
	rt.Run()
}

func handleWailsMessage(window *wails.Window, message string) {
	// Example: handle Wails messages
	switch message {
	case "wails:runtime:ready":
		fmt.Println("Wails runtime is ready")
		// Emit event back to frontend
		window.Eval(`window.dispatchEvent(new CustomEvent('wails:event', { detail: 'backend-ready' }));`)
	default:
		fmt.Printf("Unknown Wails message: %s\n", message)
	}
}

func handleAssetRequest(req wails.Request) (wails.Response, bool) {
	path := strings.TrimPrefix(req.URL, "wails://localhost/")

	// Simple in-memory asset server for demonstration
	assets := map[string]string{
		"": `<!DOCTYPE html>
<html>
<head>
    <title>Wails CEF Test</title>
    <style>
        body { font-family: sans-serif; padding: 20px; }
        button { padding: 10px 20px; margin: 5px; }
    </style>
</head>
<body>
    <h1>Wails v3 CEF Backend</h1>
    <p>This demonstrates the CEF integration for Wails v3.</p>
    <button onclick="sendMessage()">Send Message to Go</button>
    <div id="output"></div>
    <script>
        // Wails-style IPC using chrome.webview.postMessage
        function sendMessage() {
            if (window.chrome && window.chrome.webview) {
                window.chrome.webview.postMessage('Hello from JavaScript!');
                document.getElementById('output').innerHTML += '<p>Message sent!</p>';
            } else {
                document.getElementById('output').innerHTML += '<p>CEF bridge not available</p>';
            }
        }
        
        // Listen for backend events
        window.addEventListener('wails:event', function(e) {
            document.getElementById('output').innerHTML += '<p>Received: ' + e.detail + '</p>';
        });
    </script>
</body>
</html>`,
	}

	content, ok := assets[path]
	if !ok {
		return wails.Response{
			StatusCode: 404,
			StatusText: "Not Found",
			Headers:    http.Header{"Content-Type": []string{"text/plain"}},
			Body:       []byte("Not found: " + path),
		}, true
	}

	return wails.Response{
		StatusCode: 200,
		StatusText: "OK",
		Headers:    http.Header{"Content-Type": []string{"text/html"}},
		Body:       []byte(content),
	}, true
}
