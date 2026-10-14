# ChessNova — Next-Generation Online Chess Platform

A production-grade, server-authoritative online chess platform featuring a custom pure-Go chess engine, Stockfish 19 UCI Game Review Laboratory, real-time WebSockets, modern SvelteKit + TypeScript frontend, MongoDB persistence, and Redis matchmaking/caching.

---

## Documentation

Comprehensive architectural specifications, mathematical formulations, and operational guides are documented in the [`docs/`](./docs/README.md) directory:

- 📖 **[Documentation Portal (`docs/README.md`)](./docs/README.md)** — Master guide, technology stack, and high-level topology.
- 🏗️ **[System Architecture (`docs/ARCHITECTURE.md`)](./docs/ARCHITECTURE.md)** — Modular Go monolith, concurrency model, data flow, MongoDB and Redis schemas.
- ♟️ **[Chess Engine & FIDE Rules (`docs/ENGINE_AND_RULES.md`)](./docs/ENGINE_AND_RULES.md)** — Pure Go rules engine: move generation, castling lifecycle, en passant, promotions, check/checkmate/stalemate, terminal detection, SAN disambiguation, and PGN.
- 🔬 **[Game Review & Evaluation Laboratory (`docs/GAME_REVIEW_AND_EVALUATION.md`)](./docs/GAME_REVIEW_AND_EVALUATION.md)** — Stockfish 19 UCI integration, move classifications (Brilliant, Great, Best, etc.), CAPS win-chance curves, ECO opening book, phase accuracies, and calibrated platform Elo rating.
- 📈 **[Rating & Matchmaking (`docs/RATING_AND_MATCHMAKING.md`)](./docs/RATING_AND_MATCHMAKING.md)** — Chess.com-adapted Elo formulas (400 default, 100 floor, K-32), time categories, Redis matchmaking queues, and expanding tolerance windows.
- 🔌 **[REST API & WebSocket Protocol (`docs/API_AND_WEBSOCKET_PROTOCOL.md`)](./docs/API_AND_WEBSOCKET_PROTOCOL.md)** — Complete specification of HTTP REST endpoints, external platforms integration, and real-time bidirectional WebSocket events.
- 🔒 **[Security & Data Practices (`docs/SECURITY_AND_DATA_PRACTICES.md`)](./docs/SECURITY_AND_DATA_PRACTICES.md)** — JWT authentication, bcrypt passwords, ownership authorization, S3-compatible avatar storage, magic-byte sniffing, and security headers.
- 🎨 **[Frontend & UI/UX Design System (`docs/FRONTEND_AND_UI_DESIGN.md`)](./docs/FRONTEND_AND_UI_DESIGN.md)** — SvelteKit client architecture, dynamic multi-theme engine (Dark Slate, Chess Classic, OLED Black, Emerald Studio), bespoke vector Staunton SVG pieces, viewport clamping, and local guest archives.
- 🤖 **[Bot Persona & Personality Blueprint (`docs/BOT_PERSONA_BLUEPRINT.md`)](./docs/BOT_PERSONA_BLUEPRINT.md)** — Architecture for human-like bots, phase blunder curves, sacrifice biases, tactical vs positional heuristics, and trash-talk engine.
- 🛠️ **[Operations & Orchestration (`docs/OPERATIONS_AND_SCRIPTS.md`)](./docs/OPERATIONS_AND_SCRIPTS.md)** — Developer lifecycle runbook via `./script.sh` (`dev`, `build`, `check`, `test`, `deps`, `envi`, `uenvi`, `flush`), Docker Compose, and environment variables.

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
       ┌────────────────────┼────────────────────┬────────────────────┐
       │                    │                    │                    │
       ▼                    ▼                    ▼                    ▼
   Game Service       Matchmaking Service    Auth Service      Storage & External
       │                    │                                   (S3 / MinIO /
       │                    ▼                                    Chess.com /
       │                  Redis (Match queues, ephemeral state)  Lichess APIs)
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
  - Official CAPS win-probability curve with calibrated platform Elo ratings ($400\text{--}2850$).
  - Separate accuracy percentages computed for **Opening**, **Middlegame**, and **Endgame** phases.
  - Automatic Stockfish depth boosting ($14\text{ plies}$) when $\le 10$ pieces remain on the board for theoretical endgame precision.
  - Full move classification tiers: Brilliant (`!!`), Great (`!`), Best (`★`), Excellent (`✓`), Good (`✓`), Book (`📖`), Forced (`□`), Inaccuracy (`?!`), Mistake (`?`), Miss (`⨉`), and Blunder (`??`).
  - ECO opening theory catalog (A00–E99) recognizing hundreds of openings and variations.
- **S3-Compatible Object Storage & Defense-in-Depth (`internal/storage`, `internal/http`)**:
  - Object storage client (`minio-go/v7`) compatible with AWS S3, Cloudflare R2, MinIO, Wasabi, etc.
  - Automatic zero-config local storage fallback (`uploads/avatars/`) when S3 is unconfigured.
  - Magic-byte sniffing via `http.DetectContentType` enforcing genuine JPEG/PNG/WebP/GIF files.
  - Hardened security headers (`X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `X-XSS-Protection`, `Referrer-Policy`) and strict payload size clamping (`MaxBytesReader`).
- **External Platform Linking & 1-Click Game Review (`internal/external`)**:
  - Link Chess.com and Lichess accounts with public existence verification.
  - Live **Tri-Platform Elo Matrix** comparing ratings side-by-side across Rapid, Blitz, Bullet, and Tactics.
  - Import recent games with PGN from Chess.com and Lichess archives and launch deep Stockfish Game Reviews directly in ChessNova with a single click.
- **Tactical Puzzles System (`internal/puzzle`)**:
  - Progressive difficulty tiers (Tier 1 to 5) and tactical categories (Mate, Fork, Pin, Skewer, Hanging Piece, etc.).
  - Daily and Weekly curated puzzles with deterministic in-memory calendar fallbacks.
  - Independent puzzle rating tracking, hints, and step-by-step solutions.
- **Dynamic Multi-Theme Design System (`apps/web`)**:
  - Dynamic HSL theme palettes: **Dark Slate**, **Chess Classic** (`#161512` / `#21201d`), **OLED Black** (`#000000`), and **Emerald Studio** (`#05140e`).
  - Bespoke vector Staunton SVG chess pieces with dual-layer gradients and subtle drop shadows.
  - Responsive viewport clamping (`calc(100dvh - 180px)`) ensuring the full board is always 100% visible without scrolling.
  - Local guest game archive (`localGames.ts`) preserving up to 50 games for instant offline or post-game review without requiring an account.

---

## Repository Structure

```text
chess-platform/
├── apps/
│   ├── web/                     # SvelteKit + TypeScript + Tailwind CSS (SPA mode)
│   │   ├── src/
│   │   │   ├── routes/          # Pages (+page, /play, /play/online, /game/[id], /leaderboard, /puzzles, /analysis, /profile)
│   │   │   ├── lib/
│   │   │   │   ├── components/  # ChessBoard, DesktopTitlebar, GameReview, ReviewBadge, EvalBar, GameClock, MoveList, Icon
│   │   │   │   ├── api/         # Typed REST client
│   │   │   │   ├── websocket/   # WebSocket client with desktop notifications & auto-reconnect
│   │   │   │   ├── stores/      # Reactive Auth, Game, Preferences, and localGames stores
│   │   │   │   ├── chess/       # Client board helpers & vector Staunton pieces
│   │   │   │   ├── utils/       # Desktop bridge & platform detection
│   │   │   │   └── audio/       # Web Audio synthesizer & sound player
│   │   │   └── app.html
│   │   └── package.json
│   │
│   ├── desktop/                 # Dedicated Native Desktop Application (Tauri v2)
│   │   ├── src-tauri/           # Rust shell, system tray, frameless window, notifications
│   │   │   ├── Cargo.toml
│   │   │   ├── tauri.conf.json
│   │   │   └── src/ (main.rs, lib.rs)
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
| `desktop` | Launch ChessNova Tauri v2 desktop application in development mode |
| `desktop-build` | Compile native desktop application release bundle (.AppImage, .deb, etc.) |

---

## Native Desktop Application (Tauri v2)

ChessNova includes a dedicated desktop shell in [`apps/desktop/`](file:///home/yogesh/ChessNova/apps/desktop) (accessible also via the repo root shortcut `desktop/`):

- **Custom Frameless Titlebar:** Seamless window header matching the theme palette with drag region (`data-tauri-drag-region`), minimize, maximize/restore, and close buttons.
- **Native System Tray:** Persistent system tray icon with "Show ChessNova" and "Quit ChessNova" context menus.
- **Desktop Notifications:** Real-time native system notifications for match start, opponent moves, draw offers, and game completion.
- **Cross-Platform CI Compilation:** Automated matrix builds via GitHub Actions ([`.github/workflows/desktop-release.yml`](file:///home/yogesh/ChessNova/.github/workflows/desktop-release.yml)) packaging:
  - **Linux:** `.deb` and `.AppImage`
  - **Windows:** `.msi` and `.exe` (NSIS)
  - **macOS:** `.dmg` and `.app` (Universal Darwin)
