# ChessNova Documentation

Welcome to the comprehensive documentation suite for **ChessNova**, a next-generation, server-authoritative online chess platform and analysis laboratory.

ChessNova combines a 100% FIDE-compliant pure Go chess rules engine, a high-performance Stockfish 19 UCI game evaluation and review system, real-time WebSocket gameplay, dynamic Redis matchmaking, Chess.com-adapted Elo rating curves (starting at 400 with a 100 floor), and a modern SvelteKit + Tailwind CSS "Midnight Slate" client.

---

## Documentation Index

Explore the detailed architecture, specifications, and operational manuals:

| Document | Description |
| :--- | :--- |
| **[System Architecture](./ARCHITECTURE.md)** | High-level topology, modular Go monolith, SvelteKit client, concurrency model, MongoDB & Redis persistence schemas. |
| **[Chess Engine & Rules](./ENGINE_AND_RULES.md)** | Pure Go FIDE rules engine specification: move generation, castling lifecycle, en passant, promotions, check/stalemate, terminal detection, SAN & PGN. |
| **[Game Review & Stockfish Laboratory](./GAME_REVIEW_AND_EVALUATION.md)** | Stockfish 19 UCI & WASM integration, minimax/PST fallback, tactical motif pattern engine (forks, hanging blunders, discovered checks, queen trades), move classification (Brilliant, Great, Best, etc.), and natural coaching commentary. |
| **[Rating & Matchmaking](./RATING_AND_MATCHMAKING.md)** | Chess.com-adapted Elo formulas (400 baseline, 100 floor, K-32), time-control categories, Redis matchmaking queues, expanding tolerance windows. |
| **[REST API & WebSocket Protocol](./API_AND_WEBSOCKET_PROTOCOL.md)** | Full specification for HTTP endpoints (Auth, Games, Review, Puzzles, Leaderboards, Profiles, External Sync) and bidirectional WebSocket events. |
| **[Security & Data Practices](./SECURITY_AND_DATA_PRACTICES.md)** | JWT authentication, bcrypt passwords, ownership authorization, S3-compatible avatar storage, magic-byte sniffing, and security headers. |
| **[Frontend & UI/UX Design System](./FRONTEND_AND_UI_DESIGN.md)** | SvelteKit client architecture, dynamic multi-theme engine, custom vector Staunton SVG pieces, viewport clamping, desktop review layout, and local guest archives. |
| **[Bot Persona & Personality Blueprint](./BOT_PERSONA_BLUEPRINT.md)** | Architecture for human-like bots, phase blunder curves, sacrifice biases, tactical vs positional heuristics, and trash-talk engine. |
| **[Operations & Orchestration](./OPERATIONS_AND_SCRIPTS.md)** | Developer lifecycle runbook via `./script.sh` (`dev`, `build`, `check`, `test`, `deps`, `envi`, `uenvi`, `flush`, `desktop`, `desktop-build`, `engine`), Docker compose, and environment setup. |

---

## High-Level Topology

```
                              [ Web Browser / Client ]
                             /                        \
                  HTTP REST /                          \ Real-Time WebSocket
                           v                            v
               ┌──────────────────────────────────────────────┐
               │         ChessNova Modular Go Monolith         │
               │                   (apps/server)              │
               ├──────────────────────┬───────────────────────┤
               │   Auth & Accounts    │   Matchmaking Engine   │
               │     (JWT/bcrypt)     │  (Redis Queues/Window) │
               ├──────────────────────┼───────────────────────┤
               │ Pure Go Rules Engine │  Authoritative Clocks │
               │   (100% FIDE Spec)   │  (Active Game State)  │
               ├──────────────────────┼───────────────────────┤
               │  Stockfish Analysis  │   ECO Opening Book    │
               │ (UCI / CAPS Curves)  │  (Classifications)    │
               └───────────┬──────────────────────┬───────────┘
                           │                      │
                           v                      v
                 ┌───────────────────┐  ┌───────────────────┐
                 │      MongoDB      │  │       Redis       │
                 │   Users, Games,   │  │   Match Queues,   │
                 │ Ratings, Puzzles  │  │  Ephemeral State  │
                 └───────────────────┘  └───────────────────┘
```

---

## Technology Stack

### Backend (`apps/server`)
- **Language**: Go 1.24+
- **HTTP Routing**: Go standard library `net/http` ServeMux (Go 1.22+ pattern matching)
- **WebSockets**: Gorilla WebSocket (`github.com/gorilla/websocket`)
- **Persistence**: MongoDB Go Driver (`go.mongodb.org/mongo-driver`)
- **In-Memory Cache / Matchmaking**: Go-Redis (`github.com/redis/go-redis/v9`)
- **Chess Engine**: Custom pure-Go rules engine (`internal/chess`) with zero external chess dependencies
- **Engine Analysis**: Stockfish 19 UCI binary integration (`internal/engine`) with pure-Go PST/minimax fallback

### Frontend (`apps/web`)
- **Framework**: SvelteKit 2 + Vite
- **Language**: TypeScript 5
- **Styling**: Tailwind CSS 3 (Midnight Slate palette)
- **Icons**: Custom vector Lucide-derived SVG components
- **Audio**: Web Audio API / Synthesized sound engine
- **Local Persistence**: Browser `localStorage` for guest game preservation and review

### Infrastructure & Operations
- **Containerization**: Docker Compose (`mongo:7`, `redis:7-alpine`)
- **Lifecycle Orchestration**: Cross-platform scripts (`script.sh` / `script.ps1`)
