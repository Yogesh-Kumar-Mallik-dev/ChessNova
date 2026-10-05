# ==============================================================================
# scripts/flush_db.ps1 - Flush databases and caches for clean-slate testing
# ==============================================================================
$ErrorActionPreference = "Stop"

Write-Host "==> [flush] Flushing Redis cache..." -ForegroundColor Cyan
if (Get-Command redis-cli -ErrorAction SilentlyContinue) {
    & redis-cli flushall
} elseif (Get-Command docker -ErrorAction SilentlyContinue) {
    & docker exec chess-redis redis-cli flushall
}

Write-Host "==> [flush] Resetting MongoDB test data..." -ForegroundColor Cyan
if (Get-Command mongosh -ErrorAction SilentlyContinue) {
    & mongosh --eval "use chess_test; db.dropDatabase();"
} elseif (Get-Command docker -ErrorAction SilentlyContinue) {
    & docker exec chess-mongo mongosh --eval "use chess_test; db.dropDatabase();"
}

Write-Host "✅ [flush] Database and caches flushed successfully." -ForegroundColor Green
