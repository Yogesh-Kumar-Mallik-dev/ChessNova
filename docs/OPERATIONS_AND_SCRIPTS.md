# Operations, Lifecycle Scripts & Deployment Runbook

ChessPasant follows the unified script orchestration standard. All platform tasks are driven through a single root entrypoint:
- **Linux / macOS / POSIX**: `./script.sh <command>`
- **Windows PowerShell**: `.\script.ps1 <command>`

---

## 1. Unified Lifecycle Command Reference

| Command | Action | Sub-Script Invoked |
| :--- | :--- | :--- |
| **`dev`** | Verifies port availability, spins up Docker dependencies (MongoDB/Redis/MinIO), and launches Go backend (`:8080`) & SvelteKit frontend (`:5173`) in parallel with coordinated shutdown. | `scripts/dev.sh` / `scripts/dev.ps1` |
| **`build`** | Compiles Go server binary (`apps/server/bin/server`), engine microservice (`apps/server/bin/engine-service`), and builds SvelteKit production bundle. | `scripts/build.sh` / `scripts/build.ps1` |
| **`check`** | Executes static analysis: Go `vet` and SvelteKit `svelte-check` type checking. | `scripts/check.sh` / `scripts/check.ps1` |
| **`test`** | Executes pure Go chess rules engine tests, HTTP handler tests, rating tests, review laboratory tests, end-to-end integration tests, and Vitest frontend unit tests. | `scripts/test.sh` / `scripts/test.ps1` |
| **`deps`** | Installs backend Go modules, downloads Stockfish binary if absent, and installs npm dependencies for `apps/web` and `apps/desktop`. | `scripts/deps.sh` / `scripts/deps.ps1` |
| **`envi`** | Scaffolds `.env` from `.env.example` if absent and starts Docker containers (`chess-mongo`, `chess-redis`, `chess-minio`). | `scripts/envi.sh` / `scripts/envi.ps1` |
| **`uenvi`** | Tears down and stops all Docker infrastructure containers (`chess-mongo`, `chess-redis`, `chess-minio`). | `scripts/uenvi.sh` / `scripts/uenvi.ps1` |
| **`flush`** | Flushes Redis memory cache and drops development (`chess`) and test (`chess_test`) databases in MongoDB. | `scripts/flush_db.sh` / `scripts/flush_db.ps1` |
| **`desktop`** | Launches the ChessPasant Tauri v2 desktop application in development mode with hot-reloading. | `scripts/desktop.sh` / `scripts/desktop.ps1` |
| **`desktop-build`** | Compiles the production native desktop application release bundle (.AppImage, .deb, .msi, etc.). | `scripts/desktop.sh` / `scripts/desktop.ps1` |
| **`desktop-arch`** | Packages ChessPasant for Arch Linux (.pkg.tar.zst & .tar.gz). | `scripts/desktop.sh` |
| **`engine`** | Launches the standalone Stockfish 19 engine microservice on port `:8081`. | `apps/server/cmd/engine-service/main.go` |

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
# Environment Mode (development | production)
ENV=development

# MongoDB Configuration
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=chess

# Redis Configuration
REDIS_URL=localhost:6379

# JWT Secrets (Must be distinct and >= 32 characters in production)
JWT_ACCESS_SECRET=super-secret-access-token-key-change-in-production-12345
JWT_REFRESH_SECRET=super-secret-refresh-token-key-change-in-production-67890

# Administrative Credentials (Strictly independent from JWT secrets; >= 32 chars in production)
ADMIN_SECRET=super-secret-admin-key-change-in-production-abcdef12345

# Trusted Reverse Proxies (Comma-separated CIDRs/IPs evaluated for X-Forwarded-For rate limiting)
TRUSTED_PROXIES=127.0.0.1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16

# Server Port & CORS Configuration
SERVER_PORT=8080
CORS_ORIGIN=http://localhost:5173

# S3-Compatible Object Storage (AWS S3, MinIO, Cloudflare R2, Wasabi)
# Leave empty for automatic local storage fallback to ./uploads/avatars
S3_ENDPOINT=localhost:9000
S3_BUCKET=chesspasant-avatars
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=miniopassword123
S3_REGION=us-east-1
S3_USE_SSL=false
S3_PUBLIC_URL=/api/v1/storage/avatars
LOCAL_UPLOAD_DIR=uploads/avatars

# Engine Binary Configuration
STOCKFISH_PATH=./apps/server/bin/stockfish
```

---

## 4. Docker Compose Infrastructure

`docker-compose.yml` orchestrates local persistent databases and object storage:

```yaml
name: chesspasant

services:
  mongodb:
    image: mongo:7
    container_name: chess-mongo
    restart: always
    ports:
      - '127.0.0.1:27017:27017'
    volumes:
      - mongo_data:/data/db
    environment:
      - MONGO_INITDB_DATABASE=chess

  redis:
    image: redis:7-alpine
    container_name: chess-redis
    restart: always
    ports:
      - '127.0.0.1:6379:6379'
    volumes:
      - redis_data:/data

  minio:
    image: minio/minio:latest
    container_name: chess-minio
    restart: always
    command: server /data --console-address ":9001"
    ports:
      - '127.0.0.1:9000:9000'
      - '127.0.0.1:9001:9001'
    volumes:
      - minio_data:/data
    environment:
      - MINIO_ROOT_USER=minioadmin
      - MINIO_ROOT_PASSWORD=miniopassword123

volumes:
  mongo_data:
  redis_data:
  minio_data:
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
