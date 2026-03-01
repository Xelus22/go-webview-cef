# Cross-Platform Support Status

## Current Status

| Platform | Build | Run | Notes |
|----------|-------|-----|-------|
| Linux    | ✅    | ✅  | Fully working on WSLg |
| macOS    | ⚠️    | ❓  | Needs testing, paths may differ |
| Windows  | ❌    | ❌  | Needs significant changes |

## Required Changes

### 1. CEF Downloader (`scripts/setup_cef.go`)

**Windows:**
- Change extension from `.tar.bz2` to `.zip`
- Add ZIP extraction support
- Example URL: `cef_binary_132.3.1+g144febe+chromium-132.0.6834.83_windows64.zip`

**macOS:**
- Should work with current `.tar.bz2` format
- May need to handle `.dmg` for some versions

### 2. cgo Flags (`internal/cef/cef.go`)

**Current:**
```go
// #cgo darwin LDFLAGS: -L${SRCDIR}/../../third_party/cef/linux_64/Release -lcef -framework Cocoa -framework Foundation
// #cgo windows LDFLAGS: -L${SRCDIR}/../../third_party/cef/linux_64/Release -lcef
```

**Needed Changes:**
- **macOS**: Path should be `third_party/cef/macos_64/` or similar
- **Windows**: Path should be `third_party/cef/windows_64/`, needs `-lcef_dll_wrapper` and `-lcef`

### 3. Build Script (`run.sh`)

**Windows:**
- Create `run.ps1` PowerShell script
- Use `set PATH` instead of `export LD_LIBRARY_PATH`
- Copy `.dll` files instead of `.so`
- Handle `.exe` extension

**macOS:**
- Create app bundle structure
- Handle `.dylib` or framework

### 4. C Wrapper (`cef_wrapper.c`)

**Platform-specific code needed:**

```c
#ifdef _WIN32
    // Windows-specific window creation
    #include <windows.h>
#elif __APPLE__
    // macOS-specific
    #include <Cocoa/Cocoa.h>
#else
    // Linux X11
    #include <X11/Xlib.h>
#endif
```

## Implementation Priority

1. **macOS** (Easiest)
   - Test current code
   - Adjust paths in cgo flags
   - Should work with minor tweaks

2. **Windows** (More work)
   - Fix downloader for .zip
   - Update cgo flags for MSVC
   - Create PowerShell build script
   - Handle Windows-specific window creation

## Quick Test Commands

### macOS
```bash
# Download CEF
go run scripts/setup_cef.go

# Build (should work with minor path fixes)
go build -o build/demo ./cmd/demo

# Copy frameworks
cp -r third_party/cef/macos_64/Release/Chromium\ Embedded\ Framework.framework build/

# Run
./build/demo
```

### Windows (PowerShell)
```powershell
# Download CEF
go run scripts/setup_cef.go

# Build (requires MSVC)
go build -o build/demo.exe ./cmd/demo

# Copy DLLs
Copy-Item third_party/cef/windows_64/Release/libcef.dll build/
Copy-Item third_party/cef/windows_64/Release/*.bin build/

# Run
.\build\demo.exe
```

## Recommendation

**Short term**: Use the Linux version (works great in WSL/WSLg on Windows)

**Long term**: For native Windows/macOS support:
1. Fork this project
2. Implement platform-specific changes
3. Test on each platform

The architecture is sound - just needs platform-specific plumbing.
