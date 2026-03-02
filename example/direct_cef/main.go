package main

import (
	"fmt"
	"log"

	"github.com/xelus/go-webview-cef/cef"
)

func main() {
	// Example using the direct CEF package (lower-level API)

	// Initialize CEF
	cef.Initialize()
	defer cef.Shutdown()

	// Create WebView with options
	w := cef.New(cef.Options{
		Title:     "Direct CEF Example",
		Width:     1024,
		Height:    768,
		Frameless: false,
		Resizable: true,
	})

	// Bind Go functions to JavaScript
	w.Bind("greet", func(name string) string {
		return fmt.Sprintf("Hello, %s! From Go.", name)
	})

	w.Bind("add", func(a, b int) int {
		return a + b
	})

	w.Bind("getInfo", func() map[string]interface{} {
		return map[string]interface{}{
			"app":      "CEF WebView",
			"version":  "1.0.0",
			"platform": "Linux",
		}
	})

	// Load HTML content
	html := `<!DOCTYPE html>
<html>
<head>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
        }
        .container {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            padding: 40px;
            border-radius: 20px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
            max-width: 500px;
            width: 90%;
        }
        h1 { margin-top: 0; }
        button {
            background: #fff;
            color: #667eea;
            border: none;
            padding: 12px 24px;
            border-radius: 8px;
            cursor: pointer;
            font-size: 16px;
            margin: 5px;
            transition: transform 0.2s;
        }
        button:hover {
            transform: translateY(-2px);
        }
        #output {
            margin-top: 20px;
            padding: 15px;
            background: rgba(0, 0, 0, 0.2);
            border-radius: 8px;
            min-height: 60px;
            white-space: pre-wrap;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Direct CEF Package</h1>
        <p>This example uses the low-level cef package directly.</p>
        
        <div>
            <button onclick="testGreet()">Test Greet</button>
            <button onclick="testAdd()">Test Add</button>
            <button onclick="testInfo()">Get Info</button>
        </div>
        
        <div id="output">Click buttons to test Go bindings...</div>
    </div>

    <script>
        async function testGreet() {
            try {
                const result = await greet('World');
                showOutput('greet("World"): ' + result);
            } catch (err) {
                showOutput('Error: ' + err.message);
            }
        }

        async function testAdd() {
            try {
                const result = await add(40, 2);
                showOutput('add(40, 2): ' + result);
            } catch (err) {
                showOutput('Error: ' + err.message);
            }
        }

        async function testInfo() {
            try {
                const result = await getInfo();
                showOutput('getInfo():\n' + JSON.stringify(result, null, 2));
            } catch (err) {
                showOutput('Error: ' + err.message);
            }
        }

        function showOutput(text) {
            document.getElementById('output').textContent = text;
        }
    </script>
</body>
</html>`

	// Navigate to about:blank and inject HTML
	w.Navigate("about:blank")
	w.Eval(`document.open();document.write(` + "`" + html + "`" + `);document.close();`)

	log.Println("Starting Direct CEF Example...")

	// Run the application (blocks until window closed)
	w.Run()
}
