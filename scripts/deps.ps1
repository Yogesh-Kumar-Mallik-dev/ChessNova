# ==============================================================================
# scripts/deps.ps1 - Install repository dependencies
# ==============================================================================
$ErrorActionPreference = "Stop"
$RootDir = Split-Path -Parent $PSScriptRoot

Write-Host "==> [deps] Checking toolchains..." -ForegroundColor Cyan
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go is required but not installed."
}
if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    Write-Error "npm is required but not installed."
}

Write-Host "==> [deps] Installing Go backend dependencies..." -ForegroundColor Cyan
Set-Location (Join-Path $RootDir "apps\server")
go mod download
go mod tidy

Write-Host "==> [deps] Installing web frontend dependencies..." -ForegroundColor Cyan
Set-Location (Join-Path $RootDir "apps\web")
npm install

$DesktopDir = Join-Path $RootDir "apps\desktop"
if (Test-Path $DesktopDir) {
    Write-Host "==> [deps] Installing desktop application dependencies..." -ForegroundColor Cyan
    Set-Location $DesktopDir
    npm install
}

Write-Host "✅ [deps] All dependencies installed successfully." -ForegroundColor Green
