# ==============================================================================
# scripts/test.ps1 - Run test suites
# ==============================================================================
$ErrorActionPreference = "Stop"
$RootDir = Split-Path -Parent $PSScriptRoot

Write-Host "==> [test] Running Pure Go Chess Engine unit tests..." -ForegroundColor Cyan
Set-Location (Join-Path $RootDir "apps\server")
go test -v .\internal\chess\...

Write-Host "==> [test] Running end-to-end integration tests..." -ForegroundColor Cyan
go test -v .\internal\...

Write-Host "✅ [test] All test suites passed." -ForegroundColor Green
