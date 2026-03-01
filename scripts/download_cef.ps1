# CEF Downloader PowerShell Script
# Usage: .\download_cef.ps1

$ErrorActionPreference = "Stop"

Write-Host "CEF Downloader Script" -ForegroundColor Cyan
Write-Host "=====================" -ForegroundColor Cyan

# Ensure we're in the repo root
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
Set-Location (Join-Path $scriptDir "..")

# Check if Go is installed
$goPath = Get-Command "go" -ErrorAction SilentlyContinue
if (-not $goPath) {
    Write-Error "Error: Go is not installed or not in PATH"
    exit 1
}

# Run the Go setup script
Write-Host "Running CEF setup..." -ForegroundColor Green
go run scripts/setup_cef.go

if ($LASTEXITCODE -ne 0) {
    Write-Error "CEF setup failed with exit code $LASTEXITCODE"
    exit $LASTEXITCODE
}

Write-Host "CEF setup completed successfully!" -ForegroundColor Green
