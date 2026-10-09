# Operations, Lifecycle Scripts & Deployment Runbook

ChessNova follows the unified script orchestration standard. All platform tasks are driven through a single root entrypoint:
- **Linux / macOS / POSIX**: `./script.sh <command>`
- **Windows PowerShell**: `.\script.ps1 <command>`

---

## 1. Unified Lifecycle Command Reference

| Command | Action | Sub-Script Invoked |
| :--- | :--- | :--- |
| **`dev`** | Verifies port availability, spins up Docker dependencies (MongoDB/Redis), and launches Go backend (`:8080`) & SvelteKit frontend (`:5173`) in parallel with coordinated shutdown. | `scripts/dev.sh` |
| **`build`** | Compiles the Go server binary to `apps/server/bin/server` and builds the SvelteKit production bundle. | `scripts/build.sh` |
| **`check`** | Executes static analysis: Go `vet` and SvelteKit `svelte-check` type checking. | `scripts/check.sh` |
| **`test`** | Executes pure Go chess rules engine tests, HTTP handler tests, rating tests, review laboratory tests, and end-to-end integration tests. | `scripts/test.sh` |
| **`deps`** | Installs backend Go modules (`go mod download`) and frontend npm packages across `apps/web` and `packages/shared`. | `scripts/deps.sh` |
| **`envi`** | Scaffolds `.env` from `.env.example` if absent and starts Docker containers (`chess-mongo`, `chess-redis`). | `scripts/envi.sh` |
| **`uenvi`** | Tears down and stops all Docker infrastructure containers. | `scripts/uenvi.sh` |
| **`flush`** | Flushes the Redis memory cache and drops all collections in the MongoDB database for a clean slate. | `scripts/flush_db.sh` |

---

## 2. Deep-Dive: Script Mechanics

### 2.1 `scripts/dev.sh`
- Checks whether ports `8080` or `5173` are occupied; terminates stale processes safely before starting.
- Verifies MongoDB and Redis availability; boots containers via Docker Compose if offline.
- Starts backend and frontend concurrently with trap handlers ensuring background processes are cleanly terminated when `Ctrl+C` is pressed.

### 2.2 `scripts/check.sh`
```bash
# Backend analysis
go vet ./...

# Frontend typechecking
svelte-kit sync && svelte-check --tsconfig ./tsconfig.json
```
Enforces zero compile errors, zero type mismatches, and clean Go structs.

### 2.3 `scripts/test.sh`
Executes complete automated validation across the entire stack:
1. `chess-platform/server/internal/chess`: FIDE rules, fool's mate, scholar's mate, castling rights, en passant, promotion, stalemate, insufficient material, repetition, 50-move rule, pinned pieces, discovered checks, SAN disambiguation, PGN parser/exporter.
2. `chess-platform/server/internal/rating`: Elo formulas, floor clamping (min 100), starting rating (400), time category mappings.
3. `chess-platform/server/internal/review`: Stockfish evaluation, CAPS win probabilities, move accuracy formulas, classifications.
4. `chess-platform/server/internal/openings`: ECO book identification and book move tracking.
5. `chess-platform/server/internal/http`: REST API handlers and endpoint contracts.
6. `chess-platform/server/internal`: Full end-to-end multi-turn game integration flow.

---

## 3. Environment Configuration (`.env`)

Environment variables are loaded from the root `.env` file (copied from `.env.example`):

```bash
# Server Configuration
PORT=8080
CORS_ORIGIN=http://localhost:5173

# Persistence & Cache
MONGO_URI=mongodb://localhost:27017
MONGO_DB_NAME=chessnova
REDIS_URL=localhost:6379

# Authentication Secrets
JWT_ACCESS_SECRET=chessnova-access-secret-production-random-key-12345
JWT_REFRESH_SECRET=chessnova-refresh-secret-production-random-key-67890

# Engine Binary Configuration
STOCKFISH_PATH=/home/yogesh/ChessNova/apps/server/bin/stockfish
```

---

## 4. Docker Compose Infrastructure

`docker-compose.yml` orchestrates local persistent databases:

```yaml
version: '3.8'

services:
  chess-mongo:
    image: mongo:7
    container_name: chess-mongo
    ports:
      - "27017:27017"
    volumes:
      - mongo-data:/data/db
    restart: unless-stopped

  chess-redis:
    image: redis:7-alpine
    container_name: chess-redis
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    restart: unless-stopped

volumes:
  mongo-data:
  redis-data:
```

---

## 5. Production Deployment Considerations

### 5.1 Go Binary Build
Compile an optimized production binary with debug symbols stripped:
```bash
cd apps/server
CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/server cmd/server/main.go
```

### 5.2 Stockfish 19 UCI Binary
Ensure the Stockfish executable has execution permissions:
```bash
chmod +x apps/server/bin/stockfish
```

### 5.3 Web Client Build
Build the client into optimized static/node output:
```bash
cd apps/web
npm run build
```

---

## 6. Troubleshooting & Diagnostics

- **Port Conflict (`listen tcp :8080: bind: address already in use`)**:
  Run `./scripts/dev.sh` which automatically clears orphan processes, or manually inspect:
  ```bash
  fuser -k 8080/tcp
  fuser -k 5173/tcp
  ```
- **MongoDB Connection Error**:
  Verify container status:
  ```bash
  docker ps -f name=chess-mongo
  ```
  Restart via `./script.sh envi`.
- **Database Reset Needed**:
  Run `./script.sh flush` to empty Redis tickets and drop MongoDB collections.
