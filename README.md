# ChessNova — Next-Generation Online Chess Platform

A production-grade, server-authoritative online chess platform featuring a custom pure-Go chess engine, real-time WebSockets, modern SvelteKit + TypeScript frontend, MongoDB persistence, and Redis matchmaking/caching.

---

## Architecture Overview

```text
                         Internet
                            │
                  ┌─────────┴─────────┐
                  │                   │
              SvelteKit            WebSocket
             (TypeScript)              │
                  │                   │
                  └─────────┬─────────┘
                            │
                        Go Backend
                            │
       ┌────────────────────┼────────────────────┐
       │                    │                    │
       ▼                    ▼                    ▼
   Game Service       Matchmaking Service    Auth Service
       │                    │
       │                    ▼
       │                  Redis (Match queues, ephemeral state)
       │
       ▼
   Game Manager
       │
       ├──────── Pure Go Chess Rules Engine
       ├──────── Server-Side Authoritative Clock
       └──────── Event Broadcaster
       │
       ▼
    MongoDB (Users, Games, Ratings, Puzzles)
```

---

## Key Highlights

- **Pure Go Chess Rules Engine (`internal/chess`)**:
  - Implements 100% of FIDE chess rules: pawn movements, captures, double pawn move, en passant, promotion (Q, R, B, N), castling (kingside/queenside, castling rights lifecycle, preventing castling out of, through, or into check), pinned pieces, discovered checks, double checks, checkmate, stalemate, insufficient material (K vs K, K+B vs K, K+N vs K, same-color bishops), threefold repetition, and 50-move rule.
  - Zero external chess dependencies or networking knowledge in engine core.
  - Comprehensive unit test suite covering full game sequences and complex edge cases.
- **Server Authority**:
  - Browser never submits positions or FENs.
  - Clients send intent `{ type: "move", from: "e2", to: "e4" }`.
  - Go server verifies player identity, turn order, legality via the pure engine, clock elapsed time, and broadcasts authoritative state.
- **Authoritative Real-Time Clocks**:
  - Server calculates remaining time from `lastMoveAt` and base time.
  - Periodic timeout detector triggers loss on time automatically.
- **Dynamic Matchmaking (`internal/matchmaking`)**:
  - Redis-backed matchmaking queue by time control (Bullet, Blitz, Rapid, Classical).
  - Expanding rating tolerance window: 0-10s (±50), 10-20s (±100), 20-30s (±200), 30+s (±300).
- **Persistent Storage (`internal/repository/mongodb`)**:
  - Complete game records with full move list, ply count, SAN, FEN after each move, time spent, and PGN export.
  - Elo rating tracking per user and category with MongoDB indexes.
- **Modern ChessNova Design System (`apps/web`)**:
  - Midnight slate aesthetic (`slate-950` / `slate-900` with luminous sky & indigo accents) rather than chess.com drab greens.
  - Bespoke vector Staunton SVG chess pieces with dual-layer gradients and subtle drop shadows.
  - Clean, zero-emoji UI with custom vector SVG icons (`Icon.svelte`).
  - Click-to-move, drag-and-drop, legal move indicator rings/dots, last-move highlights, check glow, and promotion picker.
  - Authoritative live clocks, move history list, captured pieces counter with material score.

---

## Repository Structure

```text
chess-platform/
├── apps/
│   ├── web/                     # SvelteKit + TypeScript + Tailwind CSS
│   │   ├── src/
│   │   │   ├── routes/          # Pages (+page, /play, /play/online, /game/[id], /leaderboard, /puzzles, /analysis, /profile)
│   │   │   ├── lib/
│   │   │   │   ├── components/  # ChessBoard, ChessSquare, MoveIndicator, GameClock, MoveList, GameResult, icons/Icon
│   │   │   │   ├── api/         # Typed REST client
│   │   │   │   ├── websocket/   # WebSocket client with auto-reconnection
│   │   │   │   ├── stores/      # Reactive Auth and Game stores
│   │   │   │   └── chess/       # Client board helpers & vector Staunton pieces
│   │   │   └── app.html
│   │   └── package.json
│   │
│   └── server/                  # Modular Go Monolith
│       ├── cmd/server/main.go   # Server entrypoint with graceful shutdown
│       ├── internal/
│       │   ├── chess/           # Pure Go Chess Rules Engine & test suite
│       │   ├── game/            # Game lifecycle, clocks, MongoDB persistence
│       │   ├── matchmaking/     # Redis queues & expanding tolerance matcher
│       │   ├── rating/          # Elo rating engine & category rankings
│       │   ├── user/            # User model & MongoDB repository
│       │   ├── auth/            # JWT authentication & bcrypt password hashing
│       │   ├── websocket/       # Real-time WebSocket hub & rooms
│       │   ├── puzzle/          # Tactical puzzles & MongoDB repository
│       │   └── http/            # REST API router & middleware
│       ├── go.mod
│       └── go.sum
│
├── packages/
│   └── shared/                  # Common TypeScript interfaces
│
├── scripts/                     # Cross-platform lifecycle sub-scripts
│   ├── build.sh / build.ps1     # Compile binaries & web bundle
│   ├── check.sh / check.ps1     # Static analysis & typechecking
│   ├── deps.sh / deps.ps1       # Dependency installer
│   ├── dev.sh / dev.ps1         # Local dev orchestrator with port cleanup
│   ├── envi.sh / envi.ps1       # Docker infrastructure spin-up
│   ├── flush_db.sh / flush_db.ps1 # Database & cache flush
│   ├── test.sh / test.ps1       # Test runners
│   └── uenvi.sh / uenvi.ps1     # Docker infrastructure teardown
│
├── script.sh / script.ps1       # Single unified entrypoint per engineering standards
├── docker-compose.yml           # Local MongoDB & Redis infrastructure
├── README.md
└── .env.example
```

---

## Unified Lifecycle Commands

Conforming to the repository orchestration standard (`engineering-standards/docs/SCRIPT_ORCHESTRATION_GUIDE.md`), all lifecycle tasks are orchestrated through the root entrypoint:

- **Linux / macOS / POSIX:** `./script.sh <command>`
- **Windows PowerShell:** `.\script.ps1 <command>`

| Command | Action |
| :--- | :--- |
| `dev` | Verify ports, launch MongoDB/Redis, start Go backend (`:8080`) & SvelteKit frontend (`:5173`) |
| `build` | Compile Go binary to `apps/server/bin/server` and SvelteKit frontend bundle |
| `check` | Run Go `vet` and SvelteKit `svelte-check` typechecker |
| `test` | Execute pure Go chess rules engine test suite and end-to-end integration tests |
| `deps` | Install backend Go modules and frontend npm packages |
| `envi` | Scaffold `.env` and start Docker infrastructure containers (`chess-mongo`, `chess-redis`) |
| `uenvi` | Stop and tear down Docker containers |
| `flush` | Flush Redis cache and reset MongoDB database |
