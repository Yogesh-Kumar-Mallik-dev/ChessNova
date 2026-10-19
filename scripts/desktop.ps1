# ==============================================================================
# scripts/desktop.ps1 - Launch or build Tauri desktop application on Windows
# ==============================================================================
[CmdletBinding()]
param (
    [Parameter(Position = 0)]
    [ValidateSet("dev", "build", "check")]
    [string]$Action = "dev",

    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$RemainingArgs
)

$ErrorActionPreference = "Stop"
$RootDir = Split-Path -Parent $PSScriptRoot
$DesktopDir = Join-Path $RootDir "apps\desktop"

Set-Location $DesktopDir

# Ensure dependencies installed
if (-not (Test-Path "node_modules")) {
    Write-Host "==> [desktop] Installing desktop dependencies..." -ForegroundColor Cyan
    npm install
}

switch ($Action.ToLower()) {
    "dev" {
        Write-Host "==> [desktop] Launching ChessPasant Tauri desktop in development mode..." -ForegroundColor Cyan
        npm run dev -- @RemainingArgs
    }
    "build" {
        Write-Host "==> [desktop] Building ChessPasant Tauri desktop production binary..." -ForegroundColor Cyan
        npm run build -- @RemainingArgs
    }
    "check" {
        Write-Host "==> [desktop] Running cargo check on Tauri desktop crate..." -ForegroundColor Cyan
        cargo check --manifest-path src-tauri\Cargo.toml
    }
}
