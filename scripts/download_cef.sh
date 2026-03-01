#!/bin/bash
set -e

echo "CEF Downloader Script"
echo "====================="

# Ensure we're in the repo root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    exit 1
fi

# Run the Go setup script
echo "Running CEF setup..."
go run scripts/setup_cef.go
