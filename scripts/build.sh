#!/bin/bash
set -e

# Detect platform
OS=$(go env GOOS)
ARCH=$(go env GOARCH)
OUT_DIR="build/${OS}_${ARCH}"
CEF_ROOT="third_party/cef/${OS}_${ARCH}"

echo "Building for ${OS}_${ARCH}..."

# Ensure CEF is downloaded
if [ ! -d "$CEF_ROOT" ]; then
    echo "CEF not found. Run: go run scripts/setup_cef.go"
    exit 1
fi

# Setup environment
export CEF_ROOT=$(realpath $CEF_ROOT)

# Create output directory
mkdir -p "$OUT_DIR"

# Platform-specific build flags
case "$OS" in
    "darwin")
        export CGO_CFLAGS="-I${CEF_ROOT}/include"
        export CGO_LDFLAGS="-F${CEF_ROOT}/Release -framework 'Chromium Embedded Framework' -rpath @executable_path"
        ;;
    "linux")
        export CGO_CFLAGS="-I${CEF_ROOT}/include"
        export CGO_LDFLAGS="-L${CEF_ROOT}/Release -lcef -Wl,-rpath,'\$ORIGIN'"
        ;;
    "windows")
        export CGO_CFLAGS="-I${CEF_ROOT}/include"
        export CGO_LDFLAGS="-L${CEF_ROOT}/Release -lcef"
        ;;
esac

# Build main executable
echo "Building main executable..."
go build -o "$OUT_DIR/demo" ./cmd/demo

# Copy CEF runtime files
echo "Copying CEF runtime files..."
case "$OS" in
    "darwin")
        cp -r "$CEF_ROOT/Release/Chromium Embedded Framework.framework" "$OUT_DIR/"
        ;;
    "linux")
        cp "$CEF_ROOT/Release/libcef.so" "$OUT_DIR/"
        cp "$CEF_ROOT/Release/chrome-sandbox" "$OUT_DIR/" 2>/dev/null || true
        cp -r "$CEF_ROOT/Resources/"* "$OUT_DIR/"
        ;;
    "windows")
        cp "$CEF_ROOT/Release/libcef.dll" "$OUT_DIR/"
        cp "$CEF_ROOT/Release/chrome_elf.dll" "$OUT_DIR/"
        cp -r "$CEF_ROOT/Resources/"* "$OUT_DIR/"
        ;;
esac

echo "Build complete: $OUT_DIR"
