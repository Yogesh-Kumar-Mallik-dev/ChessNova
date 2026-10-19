# ==============================================================================
# Universal Project Lifecycle Orchestrator (Windows PowerShell Entrypoint)
# Conforming to engineering-standards / SCRIPT_ORCHESTRATION_GUIDE.md
# Usage: .\script.ps1 [dev|build|check|test|deps|envi|uenvi|flush]
# ==============================================================================
[CmdletBinding()]
param (
    [Parameter(Position = 0)]
    [string]$Command = "dev",

    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$RemainingArgs
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

function Show-Help {
    Write-Host "==================================================================" -ForegroundColor Cyan
    Write-Host "  ♟ ChessPasant Platform Lifecycle Orchestrator" -ForegroundColor Cyan
    Write-Host "==================================================================" -ForegroundColor Cyan
    Write-Host "Usage: .\script.ps1 <command> [args...]" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Available commands:"
    Write-Host "  dev      Start local infrastructure and launch servers"
    Write-Host "  build    Build Go backend binary and SvelteKit frontend bundle"
    Write-Host "  check    Run typecheckers, linters, and static analysis"
    Write-Host "  test     Run chess engine unit tests and integration tests"
    Write-Host "  deps     Install Go and npm dependencies"
    Write-Host "  envi     Initialize local environment and start Docker services"
    Write-Host "  uenvi    Tear down local Docker containers"
    Write-Host "  desktop  Launch ChessPasant Tauri desktop application in dev mode"
    Write-Host "  desktop-build Build ChessPasant Tauri desktop native release binary"
    Write-Host "  engine   Launch standalone Stockfish 19 engine microservice (port 8081)"
    Write-Host "  flush    Flush database (MongoDB) and Redis caches"
    Write-Host "  help     Display this help message"
    Write-Host "==================================================================" -ForegroundColor Cyan
}

switch ($Command.ToLower()) {
    "dev" {
        $target = Join-Path $ScriptDir "scripts\dev.ps1"
        if (Test-Path $target) { & $target @RemainingArgs } else { Write-Host "scripts\dev.ps1 not found" }
    }
    "build" {
        $target = Join-Path $ScriptDir "scripts\build.ps1"
        if (Test-Path $target) { & $target @RemainingArgs } else { Write-Host "scripts\build.ps1 not found" }
    }
    "check" {
        $target = Join-Path $ScriptDir "scripts\check.ps1"
        if (Test-Path $target) { & $target @RemainingArgs } else { Write-Host "scripts\check.ps1 not found" }
    }
    "test" {
        $target = Join-Path $ScriptDir "scripts\test.ps1"
        if (Test-Path $target) { & $target @RemainingArgs } else { Write-Host "scripts\test.ps1 not found" }
    }
    "deps" {
        $target = Join-Path $ScriptDir "scripts\deps.ps1"
        if (Test-Path $target) { & $target @RemainingArgs } else { Write-Host "scripts\deps.ps1 not found" }
    }
    "envi" {
        $target = Join-Path $ScriptDir "scripts\envi.ps1"
        if (Test-Path $target) { & $target @RemainingArgs } else { Write-Host "scripts\envi.ps1 not found" }
    }
    "uenvi" {
        $target = Join-Path $ScriptDir "scripts\uenvi.ps1"
        if (Test-Path $target) { & $target @RemainingArgs } else { Write-Host "scripts\uenvi.ps1 not found" }
    }
    "desktop" {
        $target = Join-Path $ScriptDir "scripts\desktop.ps1"
        if (Test-Path $target) { & $target "dev" @RemainingArgs } else { Write-Host "scripts\desktop.ps1 not found" }
    }
    "desktop-build" {
        $target = Join-Path $ScriptDir "scripts\desktop.ps1"
        if (Test-Path $target) { & $target "build" @RemainingArgs } else { Write-Host "scripts\desktop.ps1 not found" }
    }
    "engine" {
        Write-Host "==> [engine] Launching standalone Stockfish 19 engine microservice on port 8081..." -ForegroundColor Cyan
        Set-Location (Join-Path $ScriptDir "apps\server")
        go run cmd\engine-service\main.go @RemainingArgs
    }
    { $_ -in "flush", "flush-db", "flush_db" } {
        $target = Join-Path $ScriptDir "scripts\flush_db.ps1"
        if (Test-Path $target) { & $target @RemainingArgs } else { Write-Host "scripts\flush_db.ps1 not found" }
    }
    { $_ -in "help", "--help", "-h" } {
        Show-Help
    }
    Default {
        Write-Warning "Unknown command: $Command"
        Show-Help
        exit 1
    }
}
