package main

import (
	"encoding/base64"

	"github.com/xelus/go-webview-cef/adapter/webview"
)

func main() {
	// Required for constrained/containerized Linux environments.
	// cef.DisableGPU()

	// Create webview (handles CEF initialization internally)
	w := webview.New(false)
	defer w.Destroy()

	// Bind a Go function to JavaScript
	w.Bind("add", func(a, b int) int {
		return a + b
	})
	w.Bind("greet", func(name string) string {
		if name == "" {
			name = "World"
		}
		return "Hello " + name + "!"
	})

	// Set window properties
	w.SetTitle("Go WebView CEF Demo")
	w.SetSize(1024, 768, webview.HintNone)

	// Load simple HTML page
	html := `<html><body style="font-family:sans-serif;text-align:center;padding:40px;">` +
		`<h1>Go + CEF Demo</h1>` +
		`<p>2 + 3 = <span id="sum">?</span></p>` +
		`<button id="btn-calc">Calculate</button>` +
		`<div style="margin-top:20px;">` +
		`<input id="name" placeholder="Enter name" value="Wails" />` +
		`<button id="btn-greet">Greet</button>` +
		`</div>` +
		`<p id="greeting" style="margin-top:16px;color:#2563eb;font-weight:600;"></p>` +
		`<pre id="log" style="margin-top:16px;text-align:left;max-width:700px;margin-left:auto;margin-right:auto;background:#111;color:#0f0;padding:12px;border-radius:8px;min-height:80px;white-space:pre-wrap;"></pre>` +
		`<script>` +
		`function appLog(msg){` +
		`const el=document.getElementById('log');` +
		`el.textContent += msg + '\n';` +
		`console.log(msg);` +
		`}` +
		`async function waitForBridge(timeoutMs){` +
		`const start=Date.now();` +
		`while(Date.now()-start < timeoutMs){` +
		`if(window.go&&window.go.invoke){return;}` +
		`await new Promise(r=>setTimeout(r,25));` +
		`}` +
		`throw new Error('go bridge unavailable');` +
		`}` +
		`async function invokeBinding(name,args){` +
		`await waitForBridge(4000);` +
		`return new Promise((resolve,reject)=>{` +
		`const id=Math.random().toString(36).substr(2,9);` +
		`window.__goCallbacks=window.__goCallbacks||{};` +
		`window.__goCallbacks[id]={resolve,reject};` +
		`window.go.invoke(name,JSON.stringify({id:id,args:args}));` +
		`});` +
		`}` +
		`async function invokeWithTimeout(name,args,timeoutMs){` +
		`return await Promise.race([` +
		`invokeBinding(name,args),` +
		`new Promise((_,reject)=>setTimeout(()=>reject(new Error(name + ' timeout')),timeoutMs))` +
		`]);` +
		`}` +
		`async function calc(){` +
		`try{` +
		`appLog('calc clicked');` +
		`const result=await invokeWithTimeout('add',[2,3],2500);` +
		`appLog('add(2,3) => ' + result);` +
		`document.getElementById('sum').textContent=result;` +
		`}catch(e){appLog('add ERROR: ' + e.message);}` +
		`}` +
		`async function runGreet(){` +
		`try{` +
		`const name=document.getElementById('name').value;` +
		`appLog('greet clicked: ' + name);` +
		`const msg=await invokeWithTimeout('greet',[name],2500);` +
		`appLog('greet(' + name + ') => ' + msg);` +
		`document.getElementById('greeting').textContent=msg;` +
		`}catch(e){appLog('greet ERROR: ' + e.message);}` +
		`}` +
		`window.addEventListener('load', async function(){` +
		`appLog('UI ready');` +
		`document.getElementById('btn-calc').addEventListener('click',calc);` +
		`document.getElementById('btn-greet').addEventListener('click',runGreet);` +
		`appLog('handlers attached');` +
		`try{await calc();await runGreet();}catch(e){appLog('ERROR: ' + e.message);}` +
		`});` +
		`</script>` +
		`</body></html>`

	w.Navigate("data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(html)))

	// Run the application (blocks until window closed)
	w.Run()
}
