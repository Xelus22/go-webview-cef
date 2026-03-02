package main

import (
	"log"

	"github.com/xelus/go-webview-cef/adapter/webview"
	"github.com/xelus/go-webview-cef/cef"
)

func main() {
	cef.DisableGPU()

	app := webview.NewApp(webview.AppConfig{
		Title:      "My CEF App",
		Width:      1200,
		Height:     800,
		Debug:      false,
		Chromeless: true,  // Hide URL bar/tabs/profile Chrome UI.
		Frameless:  false, // Keep native minimize/maximize/close buttons.
	})

	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        }
        .container {
            background: rgba(255, 255, 255, 0.95);
            padding: 50px;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            width: 450px;
            text-align: center;
        }
        h1 { color: #333; margin-bottom: 10px; font-size: 32px; }
        .subtitle { color: #666; font-size: 14px; margin-bottom: 30px; }
        .info-box {
            background: #f8f9fa;
            border-left: 4px solid #667eea;
            padding: 15px;
            margin: 20px 0;
            text-align: left;
            border-radius: 0 8px 8px 0;
        }
        .info-box p { margin: 5px 0; color: #555; font-size: 14px; }
        .info-label { font-weight: 600; color: #667eea; }
        button {
            padding: 14px 28px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: #fff;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-size: 16px;
            font-weight: 600;
            margin: 10px;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        button:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 20px rgba(102, 126, 234, 0.4);
        }
        #output {
            margin-top: 20px;
            padding: 15px;
            background: #f0f0f0;
            border-radius: 8px;
            min-height: 50px;
            font-family: 'Monaco', 'Menlo', monospace;
            font-size: 13px;
            color: #333;
        }
        .badge {
            display: inline-block;
            padding: 4px 12px;
            background: #667eea;
            color: white;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 600;
            margin-bottom: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="badge">CEF 145</div>
        <h1>Chromeless Mode (WebView Layer)</h1>
        <p class="subtitle">No URL bar, no tabs, no Chrome UI!</p>

        <div class="info-box">
            <p><span class="info-label">Runtime:</span> CEF 145.0.26</p>
            <p><span class="info-label">Style:</span> ALLOY (content layer only)</p>
            <p><span class="info-label">Platform:</span> Linux X11</p>
            <p><span class="info-label">Window:</span> Native OS decorations with Chromeless content</p>
        </div>

        <div>
            <button onclick="alert('Test button works!')">Test Button</button>
        </div>

        <div id="output">Chromeless window with native OS buttons</div>
    </div>
</body>
</html>`

	app.LoadHTML(html)

	log.Println("Starting CEF 145 Chromeless App (webview layer)...")
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
