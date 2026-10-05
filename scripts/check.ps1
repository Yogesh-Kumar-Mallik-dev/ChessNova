# ==============================================================================
# scripts/check.ps1 - Run static analyzers and typecheckers
# ==============================================================================
$ErrorActionPreference = "Stop"
$RootDir = Split-Path -Parent $PSScriptRoot

Write-Host "==> [check] Running Go static analysis (go vet)..." -ForegroundColor Cyan
Set-Location (Join-Path $RootDir "apps\server")
go vet .\...

Write-Host "==> [check] Running SvelteKit typecheck (svelte-check)..." -ForegroundColor Cyan
Set-Location (Join-Path $RootDir "apps\web")
npm run check

Write-Host "✅ [check] All static and type checks passed." -ForegroundColor Green
