.PHONY: all setup build run clean test help

# Default target
all: build

# Detect platform
OS := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)
OUT_DIR := build/$(OS)_$(ARCH)

help: ## Show this help
	@echo "Go WebView CEF - Build System"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

setup: ## Download and setup CEF
	@echo "Setting up CEF..."
	@go run scripts/setup_cef.go

build: ## Build the demo application
	@echo "Building demo..."
	@mkdir -p $(OUT_DIR)
ifeq ($(OS),windows)
	@powershell -ExecutionPolicy Bypass -File scripts/build.ps1
else
	@bash scripts/build.sh
endif

run: build ## Build and run the demo
	@echo "Running demo..."
ifeq ($(OS),windows)
	@$(OUT_DIR)/demo.exe
else
	@$(OUT_DIR)/demo
endif

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf build/

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

lint: fmt vet ## Run all linters

# Development helpers
dev: ## Run in development mode (with hot reload)
	@echo "Starting development mode..."
	@air

deps: ## Download Go dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Platform-specific builds
build-linux: ## Build for Linux
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 bash scripts/build.sh

build-darwin: ## Build for macOS
	@echo "Building for macOS..."
	@GOOS=darwin GOARCH=amd64 bash scripts/build.sh

build-windows: ## Build for Windows
	@echo "Building for Windows..."
	@GOOS=windows GOARCH=amd64 powershell -ExecutionPolicy Bypass -File scripts/build.ps1

# Release builds
release: clean ## Create release builds for all platforms
	@echo "Creating release builds..."
	@mkdir -p dist
	@bash scripts/build.sh 2>/dev/null || true
	@GOOS=linux GOARCH=amd64 bash scripts/build.sh 2>/dev/null || true
	@GOOS=darwin GOARCH=amd64 bash scripts/build.sh 2>/dev/null || true
	@echo "Release builds complete in dist/"
