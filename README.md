# ChessNova — Next-Generation Online Chess Platform

A production-grade, server-authoritative online chess platform featuring a custom pure-Go chess engine, Stockfish 19 UCI Game Review Laboratory, real-time WebSockets, modern SvelteKit + TypeScript frontend, MongoDB persistence, and Redis matchmaking/caching.

---

## Documentation

Comprehensive architectural specifications, mathematical formulations, and operational guides are documented in the [`docs/`](file:///home/yogesh/ChessNova/docs/README.md) directory:

- 📖 **[Documentation Portal (`docs/README.md`)](file:///home/yogesh/ChessNova/docs/README.md)** — Master guide, technology stack, and high-level topology.
- 🏗️ **[System Architecture (`docs/ARCHITECTURE.md`)](file:///home/yogesh/ChessNova/docs/ARCHITECTURE.md)** — Modular Go monolith, concurrency model, data flow, MongoDB and Redis schemas.
- ♟️ **[Chess Engine & FIDE Rules (`docs/ENGINE_AND_RULES.md`)](file:///home/yogesh/ChessNova/docs/ENGINE_AND_RULES.md)** — Pure Go rules engine: move generation, castling lifecycle, en passant, promotions, check/checkmate/stalemate, terminal detection, SAN disambiguation, and PGN.
- 🔬 **[Game Review & Evaluation Laboratory (`docs/GAME_REVIEW_AND_EVALUATION.md`)](file:///home/yogesh/ChessNova/docs/GAME_REVIEW_AND_EVALUATION.md)** — Stockfish 19 UCI integration, move classifications (Brilliant, Great, Best, etc.), CAPS win-chance curves, and ECO opening book.
- 📈 **[Rating & Matchmaking (`docs/RATING_AND_MATCHMAKING.md`)](file:///home/yogesh/ChessNova/docs/RATING_AND_MATCHMAKING.md)** — Chess.com-adapted Elo formulas (400 default, 100 floor, K-32), time categories, Redis matchmaking queues, and expanding tolerance windows.
- 🔌 **[REST API & WebSocket Protocol (`docs/API_AND_WEBSOCKET_PROTOCOL.md`)](file:///home/yogesh/ChessNova/docs/API_AND_WEBSOCKET_PROTOCOL.md)** — Complete specification of HTTP REST endpoints and real-time bidirectional WebSocket events.
- 🎨 **[Frontend & UI/UX Design System (`docs/FRONTEND_AND_UI_DESIGN.md`)](file:///home/yogesh/ChessNova/docs/FRONTEND_AND_UI_DESIGN.md)** — SvelteKit client architecture, Midnight Slate palette, bespoke vector Staunton SVG pieces, viewport clamping, and local guest archives.
- 🛠️ **[Operations & Orchestration (`docs/OPERATIONS_AND_SCRIPTS.md`)](file:///home/yogesh/ChessNova/docs/OPERATIONS_AND_SCRIPTS.md)** — Developer lifecycle runbook via `./script.sh` (`dev`, `build`, `check`, `test`, `deps`, `envi`, `uenvi`, `flush`), Docker Compose, and environment variables.

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
       ├──────── Pure Go Chess Rules Engine (100% FIDE)
       ├──────── Stockfish 19 UCI Game Review Laboratory
       ├──────── Server-Side Authoritative Clocks
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
- **Stockfish 19 Game Review Laboratory (`internal/engine`, `internal/review`, `internal/openings`)**:
  - Stockfish 19 UCI engine subprocess with pure-Go minimax & PST fallback.
  - Official CAPS win-probability curve:
    $$\text{WinChance}(cp) = 50 + 50 \times \left( \frac{2}{1 + e^{-0.00368208 \times cp}} - 1 \right)$$
  - Full move classification tiers: Brilliant (`!!`), Great (`!`), Best (`★`), Excellent (`✓`), Good (`✓`), Book (`📖`), Forced (`□`), Inaccuracy (`?!`), Mistake (`?`), Miss (`⨉`), and Blunder (`??`).
  - ECO opening theory catalog (A00–E99) recognizing hundreds of openings and variations.
- **Chess.com-Adapted Elo System (`internal/rating`)**:
  - New players start at **400 Elo** rather than Lichess's 1500.
  - Strict minimum rating floor of **100 Elo** preventing negative or demoralizing micro-scores.
  - Dedicated rating tracking across Bullet, Blitz, Rapid, Classical, and Tactical Puzzles.
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
  - Elo rating tracking per user and category with MongoDB compound indexes.
- **Modern ChessNova Design System (`apps/web`)**:
  - Midnight slate aesthetic (`slate-950` / `slate-900` with luminous sky & indigo accents) rather than drab greens.
  - Bespoke vector Staunton SVG chess pieces with dual-layer gradients and subtle drop shadows.
  - Responsive viewport clamping (`calc(100dvh - 180px)`) ensuring the full board is always 100% visible without scrolling.
  - Local guest game archive (`localGames.ts`) preserving up to 50 games for instant offline or post-game review.
  - Clean, zero-emoji UI with custom vector SVG icons (`Icon.svelte`).

---

## Repository Structure

```text
chess-platform/
├── apps/
│   ├── web/                     # SvelteKit + TypeScript + Tailwind CSS
│   │   ├── src/
│   │   │   ├── routes/          # Pages (+page, /play, /play/online, /game/[id], /leaderboard, /puzzles, /analysis, /profile)
│   │   │   ├── lib/
│   │   │   │   ├── components/  # ChessBoard, ChessSquare, GameReview, ReviewBadge, EvalBar, GameClock, MoveList, Icon
│   │   │   │   ├── api/         # Typed REST client
│   │   │   │   ├── websocket/   # WebSocket client with auto-reconnection
│   │   │   │   ├── stores/      # Reactive Auth, Game, Preferences, and localGames stores
│   │   │   │   ├── chess/       # Client board helpers & vector Staunton pieces
│   │   │   │   └── audio/       # Web Audio synthesizer & sound player
│   │   │   └── app.html
│   │   └── package.json
│   │
│   └── server/                  # Modular Go Monolith
│       ├── cmd/server/main.go   # Server entrypoint with graceful shutdown
│       ├── internal/
│       │   ├── chess/           # Pure Go Chess Rules Engine & test suite
│       │   ├── engine/          # Stockfish 19 UCI integration & minimax fallback
│       │   ├── review/          # Game Review laboratory, CAPS accuracy & classifications
│       │   ├── openings/        # ECO opening book database & identification
│       │   ├── game/            # Game lifecycle, clocks, MongoDB persistence
│       │   ├── matchmaking/     # Redis queues & expanding tolerance matcher
│       │   ├── rating/          # Elo rating engine (400 baseline, 100 floor)
│       │   ├── user/            # User model & MongoDB repository
│       │   ├── auth/            # JWT authentication & bcrypt password hashing
│       │   ├── websocket/       # Real-time WebSocket hub & rooms
│       │   ├── puzzle/          # Tactical puzzles & MongoDB repository
│       │   └── http/            # REST API router & middleware
│       ├── bin/
│       │   └── stockfish        # Bundled Stockfish 19 UCI binary
│       ├── go.mod
│       └── go.sum
│
├── docs/                        # Comprehensive Documentation Suite
│   ├── README.md                # Documentation portal & index
│   ├── ARCHITECTURE.md          # Monolith topology, concurrency, data flow, schemas
│   ├── ENGINE_AND_RULES.md      # Pure Go FIDE rules engine specification
│   ├── GAME_REVIEW_AND_EVALUATION.md # Stockfish laboratory, CAPS scoring, classifications
│   ├── RATING_AND_MATCHMAKING.md # Chess.com Elo system, time controls, Redis queues
│   ├── API_AND_WEBSOCKET_PROTOCOL.md # REST endpoints & WebSocket protocol reference
│   ├── FRONTEND_AND_UI_DESIGN.md # Midnight slate design system, viewport clamping, SVG pieces
│   └── OPERATIONS_AND_SCRIPTS.md # Lifecycle scripts, Docker compose, runbook
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

All lifecycle tasks are orchestrated through the root entrypoint:

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
