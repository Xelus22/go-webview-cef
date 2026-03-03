//go:build linux
// +build linux

package wails

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/xelus/go-webview-cef/internal/cefbindings"
	"github.com/xelus/go-webview-cef/internal/cefshim"
)

var (
	dispatchInitOnce sync.Once
	activeRuntimeMu  sync.RWMutex
	activeRuntime    *Runtime
)

// Window is a single CEF browser window managed by the runtime.
type Window struct {
	runtime *Runtime
	handle  cefbindings.BrowserHandle

	callbacksMu sync.RWMutex
	callbacks   WindowCallbacks
}

// Runtime manages CEF lifecycle and window dispatching for Wails integration.
type Runtime struct {
	mu          sync.RWMutex
	windows     map[cefbindings.BrowserHandle]*Window
	initialized bool
}

// Initialize sets up CEF and returns a runtime instance.
// If exitCode >= 0, the caller must terminate the process with that code (CEF subprocess path).
func Initialize(args []string) (*Runtime, int, error) {
	installDispatchers()

	ret := cefshim.Initialize(args)
	if ret >= 0 {
		return nil, ret, nil
	}
	if ret == -2 {
		return nil, -1, fmt.Errorf("cef initialization failed")
	}

	rt := &Runtime{
		windows:     make(map[cefbindings.BrowserHandle]*Window),
		initialized: true,
	}

	activeRuntimeMu.Lock()
	activeRuntime = rt
	activeRuntimeMu.Unlock()

	return rt, -1, nil
}

func installDispatchers() {
	dispatchInitOnce.Do(func() {
		cefshim.SetMessageDispatch(func(name, payload string, userData uintptr) {
			rt := currentRuntime()
			if rt == nil {
				return
			}
			rt.dispatchMessage(name, payload, userData)
		})
		cefshim.SetLoadDispatch(func(success bool, userData uintptr) {
			rt := currentRuntime()
			if rt == nil {
				return
			}
			rt.dispatchLoad(success, userData)
		})
		cefshim.SetCloseDispatch(func(userData uintptr) {
			rt := currentRuntime()
			if rt == nil {
				return
			}
			rt.dispatchClose(userData)
		})
	})
}

func currentRuntime() *Runtime {
	activeRuntimeMu.RLock()
	defer activeRuntimeMu.RUnlock()
	return activeRuntime
}

// Shutdown tears down all windows and CEF.
func (r *Runtime) Shutdown() {
	if r == nil {
		return
	}

	r.mu.Lock()
	windows := make([]*Window, 0, len(r.windows))
	for _, w := range r.windows {
		windows = append(windows, w)
	}
	r.mu.Unlock()

	for _, w := range windows {
		w.Close()
	}

	cefshim.Shutdown()

	r.mu.Lock()
	r.windows = map[cefbindings.BrowserHandle]*Window{}
	r.initialized = false
	r.mu.Unlock()

	activeRuntimeMu.Lock()
	if activeRuntime == r {
		activeRuntime = nil
	}
	activeRuntimeMu.Unlock()
}

// Run enters the blocking CEF message loop.
func (r *Runtime) Run() {
	if r == nil {
		return
	}
	cefshim.Run()
}

// DoMessageLoopWork processes one CEF message-loop iteration.
func (r *Runtime) DoMessageLoopWork() {
	if r == nil {
		return
	}
	cefshim.DoMessageLoopWork()
}

// CreateWindow creates a CEF window and registers it for callback dispatch.
func (r *Runtime) CreateWindow(opts WindowOptions) (*Window, error) {
	if r == nil || !r.initialized {
		return nil, fmt.Errorf("runtime not initialized")
	}

	bo := cefbindings.DefaultBrowserOptions()
	bo.URL = opts.URL
	bo.Title = opts.Title
	bo.Width = opts.Width
	bo.Height = opts.Height
	bo.Frameless = opts.Frameless
	bo.Chromeless = opts.Chromeless
	bo.Resizable = opts.Resizable
	bo.Fullscreen = opts.Fullscreen
	bo.Maximized = opts.Maximized
	bo.X = opts.X
	bo.Y = opts.Y
	bo.ParentWindow = opts.ParentWindow

	handle := cefshim.CreateBrowser(bo)
	if handle == 0 {
		return nil, fmt.Errorf("create browser returned nil")
	}

	w := &Window{runtime: r, handle: handle}

	r.mu.Lock()
	r.windows[handle] = w
	r.mu.Unlock()

	// Inject bridge with window-specific userData for per-window callback routing
	cefshim.SetMessageCallbackUserData(uintptr(handle))
	cefshim.SetLoadCallbackUserData(uintptr(handle))

	cefshim.InjectBridge(handle)
	return w, nil
}

func (r *Runtime) windowByBrowser(browser cefbindings.BrowserHandle) *Window {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.windows[browser]
}

func (r *Runtime) dispatchMessage(name, payload string, userData uintptr) {
	// Use userData as browser handle
	lookupHandle := cefbindings.BrowserHandle(userData)
	w := r.windowByBrowser(lookupHandle)
	if w == nil {
		return
	}

	w.callbacksMu.RLock()
	cb := w.callbacks.OnMessage
	w.callbacksMu.RUnlock()
	if cb == nil {
		return
	}

	message := payload
	if name == "__cef_post_message" {
		cb(message)
		return
	}
	if message == "" {
		message = name
	}
	cb(message)
}

func (r *Runtime) dispatchLoad(success bool, userData uintptr) {
	// Use userData as browser handle
	lookupHandle := cefbindings.BrowserHandle(userData)
	w := r.windowByBrowser(lookupHandle)
	if w == nil {
		return
	}

	// Keep bridge shims present after navigation transitions.
	cefshim.InjectBridge(w.handle)

	w.callbacksMu.RLock()
	cb := w.callbacks.OnLoad
	w.callbacksMu.RUnlock()
	if cb != nil {
		cb(success)
	}
}

func (r *Runtime) dispatchClose(userData uintptr) {
	// Use userData as browser handle
	lookupHandle := cefbindings.BrowserHandle(userData)
	r.mu.Lock()
	w := r.windows[lookupHandle]
	delete(r.windows, lookupHandle)
	r.mu.Unlock()
	if w == nil {
		return
	}

	w.callbacksMu.RLock()
	cb := w.callbacks.OnClose
	w.callbacksMu.RUnlock()
	if cb != nil {
		cb()
	}
}

func decodeHeaderJSON(raw string) http.Header {
	headers := http.Header{}
	if raw == "" {
		return headers
	}

	var flat map[string]string
	if err := json.Unmarshal([]byte(raw), &flat); err == nil {
		for k, v := range flat {
			headers.Set(k, v)
		}
		return headers
	}

	var multi map[string][]string
	if err := json.Unmarshal([]byte(raw), &multi); err == nil {
		for k, vals := range multi {
			for _, v := range vals {
				headers.Add(k, v)
			}
		}
	}
	return headers
}

func encodeHeaderJSON(headers http.Header) string {
	if len(headers) == 0 {
		return ""
	}

	flat := make(map[string]string, len(headers))
	for k, vals := range headers {
		if len(vals) == 0 {
			continue
		}
		flat[k] = strings.Join(vals, ",")
	}
	if len(flat) == 0 {
		return ""
	}
	data, err := json.Marshal(flat)
	if err != nil {
		return ""
	}
	return string(data)
}
