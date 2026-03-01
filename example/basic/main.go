package main

import (
	"github.com/xelus/go-webview-cef/adapter/webview"
)

func main() {
	// Create webview (handles CEF initialization internally)
	w := webview.New(false)
	defer w.Destroy()

	// Bind a Go function to JavaScript
	w.Bind("add", func(a, b int) int {
		return a + b
	})

	// Set window properties
	w.SetTitle("Go WebView CEF Demo")
	w.SetSize(1024, 768, webview.HintNone)

	// Load simple HTML page
	html := `<html><body style="font-family:sans-serif;text-align:center;padding:40px;">` +
		`<h1>Go + CEF Demo</h1>` +
		`<p>2 + 3 = <span id="result">?</span></p>` +
		`<button onclick="calc()">Calculate</button>` +
		`<script>` +
		`async function calc(){` +
		`document.getElementById('result').textContent=await add(2,3);` +
		`}` +
		`</script>` +
		`</body></html>`

	w.Navigate("data:text/html," + html)

	// Run the application (blocks until window closed)
	w.Run()
}
