# ==============================================================================
# scripts/dev.ps1 - Start local development servers
# ==============================================================================
$ErrorActionPreference = "Stop"
$RootDir = Split-Path -Parent $PSScriptRoot

$EnviScript = Join-Path $RootDir "scripts\envi.ps1"
if (Test-Path $EnviScript) {
    & $EnviScript
}

# 2. Deterministic Port Cleanup
@(8080, 5173) | ForEach-Object {
    $port = $_
    $conns = Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue
    if ($conns) {
        $pids = $conns | Select-Object -ExpandProperty OwningProcess -Unique
        foreach ($p in $pids) {
            Write-Host "==> [dev] Port $port in use by PID $p. Terminating..." -ForegroundColor Yellow
            Stop-Process -Id $p -Force -ErrorAction SilentlyContinue
        }
    }
}

Write-Host "==> [dev] Starting Go backend on http://localhost:8080..." -ForegroundColor Cyan
$BackendJob = Start-Process -FilePath "go" -ArgumentList "run", "cmd/server/main.go" -WorkingDirectory (Join-Path $RootDir "apps\server") -PassThru

Write-Host "==> [dev] Starting SvelteKit frontend on http://localhost:5173..." -ForegroundColor Cyan
$FrontendJob = Start-Process -FilePath "npm" -ArgumentList "run", "dev" -WorkingDirectory (Join-Path $RootDir "apps\web") -PassThru

Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host "  ♟ ChessNova Platform is running in dev mode!" -ForegroundColor Cyan
Write-Host "  Frontend: http://localhost:5173" -ForegroundColor Cyan
Write-Host "  Backend:  http://localhost:8080" -ForegroundColor Cyan
Write-Host "  Press Ctrl+C to stop all services." -ForegroundColor Cyan
Write-Host "==================================================================" -ForegroundColor Cyan

try {
    Wait-Process -Id $BackendJob.Id, $FrontendJob.Id
} finally {
    Write-Host "`n==> [dev] Shutting down services..." -ForegroundColor Cyan
    Stop-Process -Id $BackendJob.Id -Force -ErrorAction SilentlyContinue
    Stop-Process -Id $FrontendJob.Id -Force -ErrorAction SilentlyContinue
}
