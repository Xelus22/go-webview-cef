package main

import (
	"fmt"
	"log"

	"github.com/xelus/go-webview-cef/adapter/webview"
)

func main() {
	// Create app with config (similar to Tauri)
	// Frameless=true removes OS window decorations (title bar, borders)
	// Chromeless=true removes browser UI (no URL bar in CEF by default anyway)
	app := webview.NewApp(webview.AppConfig{
		Title:     "My CEF App",
		Width:     1200,
		Height:    800,
		Debug:     true,
		Frameless: true, // No OS window decorations - pure content
	})

	// Register Go functions
	app.Register("greet", func(name string) string {
		return fmt.Sprintf("Hello, %s!", name)
	})

	app.Register("add", func(a, b float64) float64 {
		return a + b
	})

	// Minimal HTML for testing
	html := `<html><body style="background:#667eea;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;font-family:sans-serif;"><div style="background:#fff;padding:40px;border-radius:16px;box-shadow:0 20px 60px rgba(0,0,0,0.3);width:300px;text-align:center;"><h1>CEF 145</h1><p>It works!</p><button onclick="test()" style="padding:12px 24px;background:#667eea;color:#fff;border:none;border-radius:6px;cursor:pointer;">Test</button><p id="out"></p></div><script>async function test(){var r=await window.go.greet('World');document.getElementById('out').textContent=r;}</script></body></html>`

	app.LoadHTML(html)

	log.Println("Starting CEF 145 App...")
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
