# Go WebView CEF

A Go binding for the Chromium Embedded Framework (CEF) providing a WebView-compatible API for building desktop applications with web technologies.

## Project Structure

```
/cef-go/
├── /cmd/
│   └── demo/                  # Example application
├── /internal/
│   ├── cef/                   # Go bindings (cgo layer)
│   ├── cef_c/                 # C wrapper layer
│   └── platform/              # OS-specific helpers
├── /adapter/
│   └── webview/               # WebView-compatible API
├── /scripts/
│   ├── download_cef.sh        # Bash CEF downloader
│   ├── download_cef.ps1       # PowerShell CEF downloader
│   └── setup_cef.go           # Go-based CEF setup script
├── /third_party/
│   └── cef/                   # Downloaded CEF binaries
├── go.mod
└── README.md
```

## Prerequisites

- Go 1.21 or later
- C compiler (gcc/clang on Linux/macOS, MinGW or MSVC on Windows)

## Setup

### 1. Download CEF Binaries

**Linux/macOS:**
```bash
./scripts/download_cef.sh
```

**Windows:**
```powershell
.\scripts\download_cef.ps1
```

Or directly with Go:
```bash
go run scripts/setup_cef.go
```

The downloader will:
- Auto-detect your OS and architecture
- Download the appropriate CEF binary from Spotify's CDN
- Extract to `third_party/cef/{platform}_{arch}/`
- Create an environment marker file

### 2. Set Environment Variable

```bash
export CEF_ROOT=$(pwd)/third_party/cef/linux_64  # Adjust for your platform
```

## Supported Platforms

| Platform | Architecture | Status |
|----------|-------------|--------|
| Linux    | x64 (amd64)  | Planned |
| Linux    | ARM64        | Planned |
| macOS    | x64 (amd64)  | Planned |
| macOS    | ARM64 (M1/M2)| Planned |
| Windows  | x64 (amd64)  | Planned |
| Windows  | ARM64        | Planned |

## License

MIT License - See LICENSE file for details.

CEF is Copyright (c) 2008-2023 Marshall A. Greenblatt. Portions Copyright (c)
2006-2009 Google Inc. All rights reserved.
