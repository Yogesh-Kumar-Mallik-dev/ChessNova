# ==============================================================================
# scripts/uenvi.ps1 - 1-Click Environment Teardown
# ==============================================================================
$ErrorActionPreference = "Stop"
$RootDir = Split-Path -Parent $PSScriptRoot

if (Get-Command docker -ErrorAction SilentlyContinue) {
    Write-Host "==> [uenvi] Stopping and removing containers..." -ForegroundColor Cyan
    Set-Location $RootDir
    docker compose down
    Write-Host "✅ [uenvi] Containers stopped." -ForegroundColor Green
}
