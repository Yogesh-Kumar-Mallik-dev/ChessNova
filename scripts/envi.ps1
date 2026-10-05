# ==============================================================================
# scripts/envi.ps1 - 1-Click Environment Setup & Docker Infrastructure
# ==============================================================================
$ErrorActionPreference = "Stop"
$RootDir = Split-Path -Parent $PSScriptRoot

Write-Host "==> [envi] Initializing environment configuration..." -ForegroundColor Cyan
$EnvFile = Join-Path $RootDir ".env"
$EnvExample = Join-Path $RootDir ".env.example"

if (-not (Test-Path $EnvFile)) {
    if (Test-Path $EnvExample) {
        Copy-Item $EnvExample $EnvFile
        Write-Host "Created .env from .env.example"
    }
}

if (Get-Command docker -ErrorAction SilentlyContinue) {
    Write-Host "==> [envi] Starting MongoDB and Redis via Docker Compose..." -ForegroundColor Cyan
    Set-Location $RootDir
    docker compose up -d
    Write-Host "✅ [envi] Infrastructure containers started." -ForegroundColor Green
} else {
    Write-Warning "Docker not detected. Please ensure MongoDB (27017) and Redis (6379) are running."
}
