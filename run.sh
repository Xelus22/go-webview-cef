#!/bin/bash

# Go + CEF WebView Demo Runner Script
# This script builds and runs the CEF demo applications

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BUILD_DIR="build"
CEF_DIR="third_party/cef/linux_64"
DEFAULT_EXAMPLE="tauri_style"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if CEF is downloaded
check_cef() {
    if [ ! -d "$CEF_DIR" ]; then
        print_error "CEF not found at $CEF_DIR"
        print_status "Downloading CEF..."
        go run scripts/setup_cef.go
    else
        print_status "CEF found at $CEF_DIR"
    fi
}

# Build the demo
build_demo() {
    local example_name="$1"
    print_status "Building example: $example_name"
    go build -o "$BUILD_DIR/$example_name" "./example/$example_name"
    print_status "Build successful!"
}

# Copy CEF runtime files
copy_runtime_files() {
    print_status "Copying CEF runtime files..."
    
    # Create build directory if it doesn't exist
    mkdir -p "$BUILD_DIR"
    
    # Copy main library
    if [ -f "$CEF_DIR/Release/libcef.so" ]; then
        cp "$CEF_DIR/Release/libcef.so" "$BUILD_DIR/"
        print_status "Copied libcef.so"
    else
        print_error "libcef.so not found!"
        exit 1
    fi
    
    # Copy V8 snapshot files
    for bin in "$CEF_DIR/Release"/*.bin; do
        if [ -f "$bin" ]; then
            cp "$bin" "$BUILD_DIR/"
            print_status "Copied $(basename "$bin")"
        fi
    done
    
    # Copy resource pak files
    for pak in "$CEF_DIR/Resources"/*.pak; do
        if [ -f "$pak" ]; then
            cp "$pak" "$BUILD_DIR/"
            print_status "Copied $(basename "$pak")"
        fi
    done
    
    # Copy ICU data file
    if [ -f "$CEF_DIR/Resources/icudtl.dat" ]; then
        cp "$CEF_DIR/Resources/icudtl.dat" "$BUILD_DIR/"
        print_status "Copied icudtl.dat"
    fi
    
    # Copy locales directory
    if [ -d "$CEF_DIR/Resources/locales" ]; then
        cp -r "$CEF_DIR/Resources/locales" "$BUILD_DIR/"
        print_status "Copied locales directory"
    fi
    
    # Copy chrome-sandbox if it exists
    if [ -f "$CEF_DIR/Release/chrome-sandbox" ]; then
        cp "$CEF_DIR/Release/chrome-sandbox" "$BUILD_DIR/"
        chmod 4755 "$BUILD_DIR/chrome-sandbox" 2>/dev/null || true
        print_status "Copied chrome-sandbox"
    fi
    
    print_status "All runtime files copied successfully!"
}

# Run the demo
run_demo() {
    local example_name="$1"
    shift
    print_status "Running example: $example_name"
    print_status "Build directory: $(pwd)/$BUILD_DIR"

    # Set library path and run
    export LD_LIBRARY_PATH="$(pwd)/$BUILD_DIR:$LD_LIBRARY_PATH"

    # Set additional environment variables for CEF
    export DISPLAY="${DISPLAY:-:0}"

    print_status "Environment:"
    echo "  DISPLAY=$DISPLAY"
    echo "  LD_LIBRARY_PATH=$LD_LIBRARY_PATH"
    echo "  CWD=$(pwd)"

    print_status "Starting demo..."
    print_status "Note: CEF requires a display server (X11/Wayland)"
    print_status "If you see GPU errors, the app needs a desktop environment"
    echo "========================================"
    cd "$BUILD_DIR" && "./$example_name" "$@"
}

# Clean build artifacts
clean() {
    print_status "Cleaning build artifacts..."
    rm -rf "$BUILD_DIR"
    print_status "Clean complete!"
}

# Show help
show_help() {
    cat << EOF
Go + CEF WebView Demo Runner

Usage: $0 [command] [example] [options]

Commands:
    build       Build the specified example (default)
    run         Build and run the specified example
    clean       Remove build artifacts
    help        Show this help message

Examples available:
    tauri_style  Tauri-style API demo with chromeless/frameless options (default)
    basic        Basic webview usage example

Options:
    --rebuild   Force rebuild even if binary exists
    --debug     Run with debug output

Environment Requirements:
    - X11 or Wayland display server
    - Desktop environment (for GPU support)

Usage Examples:
    $0                           # Build default example (tauri_style)
    $0 build                     # Build default example (tauri_style)
    $0 build basic               # Build basic example
    $0 run                       # Build and run default example
    $0 run tauri_style           # Build and run tauri_style example
    $0 run basic --rebuild       # Force rebuild and run basic example
    $0 clean                     # Clean build files

Note: If running in a headless/container environment, you may need:
    - Xvfb for virtual display
    - GPU passthrough or software rendering
EOF
}

# Main execution
main() {
    local command="${1:-build}"
    local example_name="$DEFAULT_EXAMPLE"
    local rebuild=false
    local debug=false

    # Check if second argument is an example name or option
    if [ -n "$2" ] && [[ "$2" != --* ]]; then
        example_name="$2"
        shift 2
    else
        shift 1
    fi

    # Parse remaining arguments for options
    for arg in "$@"; do
        case "$arg" in
            --rebuild)
                rebuild=true
                ;;
            --debug)
                debug=true
                ;;
        esac
    done

    # Validate example exists
    if [ ! -d "example/$example_name" ]; then
        print_error "Example not found: $example_name"
        print_status "Available examples:"
        for ex in example/*/; do
            echo "  - $(basename "$ex")"
        done
        exit 1
    fi

    case "$command" in
        build)
            check_cef
            if [ "$rebuild" = true ] || [ ! -f "$BUILD_DIR/$example_name" ]; then
                build_demo "$example_name"
                copy_runtime_files
            else
                print_status "Demo already built. Use --rebuild to force rebuild."
            fi
            print_status "Build complete! Run with: $0 run $example_name"
            ;;
        run)
            check_cef
            if [ "$rebuild" = true ] || [ ! -f "$BUILD_DIR/$example_name" ]; then
                build_demo "$example_name"
                copy_runtime_files
            fi
            run_demo "$example_name" "$@"
            ;;
        clean)
            clean
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
