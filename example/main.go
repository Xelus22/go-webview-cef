package main

import (
	"os"

	"github.com/xelus/go-webview-cef/adapter/webview"
	"github.com/xelus/go-webview-cef/internal/cef"
)

func main() {
	// Pass command line args to CEF (required for multi-process model)
	cef.SetArgs(len(os.Args), os.Args)

	// Handle subprocess (renderer, GPU, etc.)
	if cef.IsSubprocess() {
		cef.SubprocessEntry()
		return
	}

	// Create webview
	w := webview.New(false)
	defer w.Destroy()

	// Bind a Go function to JavaScript
	w.Bind("add", func(a, b int) int {
		return a + b
	})

	// Set window properties
	w.SetTitle("Go WebView CEF Demo")
	w.SetSize(1024, 768, webview.HintNone)

	// Load HTML content
	w.Navigate(`data:text/html,<!DOCTYPE html>
<html>
<head>
    <title>CEF Demo</title>
    <style>
        body { font-family: sans-serif; padding: 40px; text-align: center; }
        button { padding: 15px 30px; font-size: 16px; cursor: pointer; }
        #result { margin-top: 20px; font-size: 24px; color: #333; }
    </style>
</head>
<body>
    <h1>Go + CEF WebView Demo</h1>
    <p>This demonstrates Go to JavaScript bindings</p>
    <button onclick="testBinding()">Call Go Function</button>
    <div id="result"></div>
    <script>
        async function testBinding() {
            try {
                const result = await add(5, 3);
                document.getElementById('result').textContent = 
                    '5 + 3 = ' + result;
            } catch (e) {
                document.getElementById('result').textContent = 
                    'Error: ' + e.message;
            }
        }
    </script>
</body>
</html>`)

	// Run the application (blocks until window closed)
	w.Run()
}
