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

Write-Host "==> [flush] Resetting MongoDB databases..." -ForegroundColor Cyan
$DbName = if ($env:MONGODB_DATABASE) { $env:MONGODB_DATABASE } else { "chess" }
if (Get-Command mongosh -ErrorAction SilentlyContinue) {
    & mongosh --eval "use $DbName; db.dropDatabase(); use chess_test; db.dropDatabase();"
} elseif (Get-Command docker -ErrorAction SilentlyContinue) {
    & docker exec chess-mongo mongosh --eval "use $DbName; db.dropDatabase(); use chess_test; db.dropDatabase();"
}

Write-Host "✅ [flush] Database and caches flushed successfully." -ForegroundColor Green
