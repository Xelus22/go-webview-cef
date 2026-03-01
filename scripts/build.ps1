#!/usr/bin/env pwsh
param(
    [string]$OutDir = "",
    [string]$CefRoot = ""
)

$ErrorActionPreference = "Stop"

# Detect platform
$OS = go env GOOS
$ARCH = go env GOARCH

if (-not $OutDir) {
    $OutDir = "build/${OS}_${ARCH}"
}
if (-not $CefRoot) {
    $CefRoot = "third_party/cef/${OS}_${ARCH}"
}

Write-Host "Building for ${OS}_${ARCH}..."

# Ensure CEF is downloaded
if (-not (Test-Path $CefRoot)) {
    Write-Error "CEF not found. Run: go run scripts/setup_cef.go"
    exit 1
}

# Convert to absolute path
$CefRoot = (Resolve-Path $CefRoot).Path

# Create output directory
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

# Platform-specific build flags
$env:CGO_CFLAGS = "-I${CefRoot}/include"

switch ($OS) {
    "windows" {
        $env:CGO_LDFLAGS = "-L${CefRoot}/Release -lcef"
        
        # Build main executable
        Write-Host "Building main executable..."
        go build -o "$OutDir/demo.exe" ./cmd/demo
        
        # Copy CEF runtime files
        Write-Host "Copying CEF runtime files..."
        Copy-Item "$CefRoot/Release/libcef.dll" $OutDir -Force
        Copy-Item "$CefRoot/Release/chrome_elf.dll" $OutDir -Force
        Copy-Item "$CefRoot/Release/d3dcompiler_47.dll" $OutDir -Force
        Copy-Item "$CefRoot/Release/libEGL.dll" $OutDir -Force
        Copy-Item "$CefRoot/Release/libGLESv2.dll" $OutDir -Force
        Copy-Item "$CefRoot/Release/vk_swiftshader.dll" $OutDir -Force
        Copy-Item "$CefRoot/Release/vulkan-1.dll" $OutDir -Force
        
        # Copy resources
        Copy-Item "$CefRoot/Resources/*" $OutDir -Recurse -Force
    }
    "linux" {
        $env:CGO_LDFLAGS = "-L${CefRoot}/Release -lcef -Wl,-rpath,'\`$ORIGIN'"
        
        # Build main executable
        Write-Host "Building main executable..."
        go build -o "$OutDir/demo" ./cmd/demo
        
        # Copy CEF runtime files
        Write-Host "Copying CEF runtime files..."
        Copy-Item "$CefRoot/Release/libcef.so" $OutDir -Force
        if (Test-Path "$CefRoot/Release/chrome-sandbox") {
            Copy-Item "$CefRoot/Release/chrome-sandbox" $OutDir -Force
        }
        Copy-Item "$CefRoot/Resources/*" $OutDir -Recurse -Force
    }
    "darwin" {
        $env:CGO_LDFLAGS = "-F${CefRoot}/Release -framework 'Chromium Embedded Framework' -rpath @executable_path"
        
        # Build main executable
        Write-Host "Building main executable..."
        go build -o "$OutDir/demo" ./cmd/demo
        
        # Copy CEF runtime files
        Write-Host "Copying CEF runtime files..."
        Copy-Item "$CefRoot/Release/Chromium Embedded Framework.framework" $OutDir -Recurse -Force
    }
}

Write-Host "Build complete: $OutDir"
