# Go WebView CEF

A cross-platform Go binding for Chromium Embedded Framework (CEF) providing a webview-compatible API for building desktop applications with web technologies.

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## Features

- ✅ **Cross-Platform**: Linux, macOS, Windows support
- ✅ **WebView-Compatible API**: Drop-in replacement for `webview/webview`
- ✅ **CEF Integration**: Full Chromium browser engine
- ✅ **JavaScript Bindings**: Call Go functions from JavaScript
- ✅ **WSL/WSLg Support**: Works on Windows Subsystem for Linux

## Supported Platforms

| Platform | Architecture | Status | Notes |
|----------|-------------|--------|-------|
| Linux    | x64 (amd64)  | ✅ Working | Tested on Ubuntu/Debian |
| Linux    | ARM64        | ⚠️ Ready  | Build tags implemented |
| macOS    | x64 (amd64)  | ⚠️ Ready  | Build tags implemented, needs testing |
| macOS    | ARM64 (M1/M2)| ⚠️ Ready  | Build tags implemented, needs testing |
| Windows  | x64 (amd64)  | ⚠️ Ready  | Build tags implemented, needs testing |
| Windows  | ARM64        | ⚠️ Ready  | Build tags implemented, needs testing |

## Quick Start

### Prerequisites

- Go 1.21 or later
- C compiler (gcc/clang on Linux/macOS, MinGW on Windows)
- For WSL: WSL2 with WSLg support (`xeyes` should work)
- For Linux: X11 or Wayland display server

### Installation

```bash
# Clone the repository
git clone https://github.com/xelus/go-webview-cef.git
cd go-webview-cef

# Download CEF binaries (automatically detects platform)
go run scripts/setup_cef.go

# Build the demo
./run.sh build

# Run the demo
./run.sh run
```

## Usage

### Basic Example

```go
package main

import (
    "os"
    "github.com/xelus/go-webview-cef/adapter/webview"
    "github.com/xelus/go-webview-cef/internal/cef"
)

func main() {
    // Set command line args
    cef.SetArgs(len(os.Args), os.Args)
    
    // Handle subprocess (CEF multi-process model)
    if cef.IsSubprocess() {
        cef.SubprocessEntry()
        return
    }

    // Create webview
    w := webview.New(false)
    defer w.Destroy()

    // Bind a Go function
    w.Bind("add", func(a, b int) int {
        return a + b
    })

    // Set window properties
    w.SetTitle("My App")
    w.SetSize(1024, 768, webview.HintNone)

    // Load HTML
    w.Navigate(`data:text/html,<!DOCTYPE html>
<html>
<body>
    <h1>Go + CEF</h1>
    <button onclick="test()">Call Go</button>
    <script>
        async function test() {
            const result = await add(5, 3);
            alert("5 + 3 = " + result);
        }
    </script>
</body>
</html>`)

    // Run
    w.Run()
}
```

### API Reference

#### WebView Interface

```go
type WebView interface {
    Run()
    Terminate()
    Destroy()
    Window() unsafe.Pointer
    SetTitle(title string)
    SetSize(width, height int, hint int)
    Navigate(url string)
    Init(js string)
    Eval(js string)
    Bind(name string, fn interface{}) error
    Unbind(name string) error
}
```

#### Window Hints

```go
const (
    HintNone  = 0
    HintFixed = 1
    HintMin   = 2
    HintMax   = 3
)
```

## Project Structure

```
.
├── adapter/webview/          # WebView-compatible API
│   ├── webview.go           # Main webview implementation
│   └── types.go             # Type definitions
├── example/                  # Example application
│   └── main.go              # Demo usage
├── internal/cef/             # Go CEF bindings
│   ├── cef_linux.go         # Linux-specific implementation
│   ├── cef_darwin.go        # macOS-specific implementation
│   ├── cef_windows.go       # Windows-specific implementation
│   ├── browser.go           # Browser management
│   ├── cef_wrapper.h        # C wrapper header
│   └── cef_wrapper.c        # C wrapper implementation
├── scripts/
│   └── setup_cef.go         # CEF downloader (cross-platform)
├── third_party/cef/          # Downloaded CEF binaries (gitignored)
├── run.sh                    # Build & run script (Linux/macOS)
├── Makefile                  # Build automation
└── README.md                 # This file
```

## Build Commands

### Linux/WSL
```bash
# Native build
go build -o build/demo ./cmd/demo

# Using the run script
./run.sh build
./run.sh run

# Cross-compile for other platforms
GOOS=darwin GOARCH=amd64 go build -o build/demo ./cmd/demo
GOOS=windows GOARCH=amd64 go build -o build/demo.exe ./cmd/demo
```

### macOS
```bash
# Download CEF for macOS
go run scripts/setup_cef.go

# Build
go build -o build/demo ./cmd/demo

# Run
./build/demo
```

### Windows
```bash
# Download CEF for Windows
go run scripts/setup_cef.go

# Build (requires MinGW)
go build -o build/demo.exe ./cmd/demo

# Run
.\build\demo.exe
```

## CEF Configuration

The implementation uses CEF's C API and includes platform-specific optimizations:

### Platform-Specific Files

Go build constraints automatically select the correct implementation:

- **Linux**: `internal/cef/cef_linux.go`
- **macOS**: `internal/cef/cef_darwin.go`  
- **Windows**: `internal/cef/cef_windows.go`

### CEF Download URLs

All platforms use `.tar.bz2` archives:
- Linux: `cef_binary_*_linux64.tar.bz2`
- macOS: `cef_binary_*_macosx64.tar.bz2`
- Windows: `cef_binary_*_windows64.tar.bz2`

### WSL/WSLg Compatibility

The implementation includes WSL-specific flags for GPU/graphics compatibility:
```
--no-sandbox
--disable-gpu
--disable-gpu-compositing
--use-gl=swiftshader
--single-process
```

## Troubleshooting

### Display Issues (WSL/Linux)

If you see GPU errors or display issues:

```bash
# Check if DISPLAY is set
echo $DISPLAY

# For WSLg, should be something like: :0
# If not set, try:
export DISPLAY=:0

# Run with the script which sets this automatically
./run.sh run
```

### Missing Libraries

If you get library loading errors:

```bash
# Linux/WSL
export LD_LIBRARY_PATH=$PWD/build:$LD_LIBRARY_PATH

# macOS
export DYLD_LIBRARY_PATH=$PWD/build:$DYLD_LIBRARY_PATH

# Windows (PowerShell)
$env:PATH = "$PWD/build;$env:PATH"
```

### CEF Not Found

If CEF binaries are not found:

```bash
# Re-download CEF
rm -rf third_party/cef
go run scripts/setup_cef.go
```

## Architecture

```
┌─────────────────┐
│   App Code      │  Your Go application
├─────────────────┤
│ WebView Adapter │  webview.New(), w.Run()
├─────────────────┤
│  Go CEF Bindings│  Platform-specific (cef_linux.go, etc.)
├─────────────────┤
│  cgo + C Wrapper│  cef_wrapper.c
├─────────────────┤
│   CEF C API     │  cef_initialize(), cef_browser_create()
├─────────────────┤
│   libcef        │  Chromium Embedded Framework
└─────────────────┘
```

## Development Status

- ✅ CEF downloader with auto-platform detection
- ✅ C wrapper with proper initialization flow
- ✅ Go bindings with platform-specific build tags
- ✅ WebView-compatible adapter layer
- ✅ OnContextInitialized callback for browser creation
- ✅ WSL/WSLg compatibility
- ✅ Multi-process model support
- ✅ Cross-platform build system
- ⚠️ macOS testing needed
- ⚠️ Windows testing needed

## Contributing

Contributions are welcome! Areas that need help:

1. **macOS Testing**: Verify build and runtime on macOS
2. **Windows Testing**: Verify build and runtime on Windows
3. **Documentation**: Improve setup instructions
4. **Examples**: Add more usage examples

Please submit issues and pull requests on GitHub.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

```
MIT License

Copyright (c) 2026 Xelus

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## Acknowledgments

- [Chromium Embedded Framework](https://bitbucket.org/chromiumembedded/cef/) - The CEF project
- [Spotify CEF Builds](https://cef-builds.spotifycdn.com/index.html) - Prebuilt CEF binaries
- CEF is Copyright (c) 2008-2023 Marshall A. Greenblatt

## Notes

- CEF binaries are ~150-300MB per platform
- The implementation uses CEF's C API (not C++) for better compatibility
- Single-process mode is used for simplified deployment (not recommended for production)
- For production use, consider implementing proper multi-process handling
- Window management is platform-specific and may need adjustments for each OS
