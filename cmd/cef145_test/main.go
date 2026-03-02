package main

import (
	"fmt"
	"os"

	"github.com/xelus/go-webview-cef/cef"
)

func main() {
	fmt.Println("CEF 145 Test - Starting...")

	// Add flags to disable GPU and avoid GPU process issues
	cef.DisableGPU()
	os.Args = append(os.Args, "--disable-gpu", "--disable-gpu-compositing")

	// Initialize CEF (handles subprocess internally)
	if !cef.Initialize() {
		fmt.Println("Failed to initialize CEF or this is a subprocess!")
		os.Exit(1)
	}
	fmt.Println("CEF initialized (browser process)!")

	fmt.Println("Creating browser...")
	browser := cef.New(cef.Options{
		Title:     "CEF 145 Test",
		URL:       "https://www.google.com",
		Width:     1024,
		Height:    768,
		Resizable: true,
	})
	if browser == nil {
		fmt.Println("Browser creation returned nil!")
		cef.Shutdown()
		os.Exit(1)
	}
	fmt.Println("Browser created successfully!")

	fmt.Println("Running message loop (blocking)...")
	browser.Run()

	fmt.Println("Shutting down...")
	browser.Destroy()
	cef.Shutdown()
	fmt.Println("Done!")
}
