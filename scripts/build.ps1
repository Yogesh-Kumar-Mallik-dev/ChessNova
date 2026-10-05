# ==============================================================================
# scripts/build.ps1 - Compile production assets and binaries
# ==============================================================================
$ErrorActionPreference = "Stop"
$RootDir = Split-Path -Parent $PSScriptRoot

Write-Host "==> [build] Compiling Go server binary..." -ForegroundColor Cyan
Set-Location (Join-Path $RootDir "apps\server")
if (-not (Test-Path "bin")) { New-Item -ItemType Directory -Path "bin" | Out-Null }
go build -o bin\server.exe .\cmd\server\main.go
Write-Host "✅ [build] Go server binary built: apps/server/bin/server.exe" -ForegroundColor Green

Write-Host "==> [build] Building SvelteKit production bundle..." -ForegroundColor Cyan
Set-Location (Join-Path $RootDir "apps\web")
npm run build
Write-Host "✅ [build] SvelteKit frontend bundle built." -ForegroundColor Green

Write-Host "🎉 [build] All production artifacts built successfully." -ForegroundColor Green
