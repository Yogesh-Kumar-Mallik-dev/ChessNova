# System Architecture

This document details the architectural design, component topology, concurrency model, data flow, and persistence schemas powering the ChessNova platform.

---

## 1. Architectural Philosophy: Strict Server Authority

Online chess platforms face unique vulnerabilities: client-side tampering, invalid position injection, desynchronized clocks, and clock-freezing cheats. ChessNova is designed around strict **Server Authority**:

1. **Clients Do Not Transmit FENs or Board States**: Clients only send move intents (e.g., `{ "from": "e2", "to": "e4", "promotion": "q" }`).
2. **Server Validates Against Internal State**: The server holds the canonical `chess.Game` struct in memory, verifies player authorization, turns, and moves via the pure Go engine, executes the state transition, recalculates remaining clock time, and broadcasts the authoritative outcome.
3. **Deterministic Verification**: Every move is recorded with ply, SAN, FEN before, FEN after, and timestamp, guaranteeing complete auditable history and reproducible game state.

```
+---------------+           Move Intent (e2 -> e4)         +--------------------+
|               | ---------------------------------------> |                    |
| Browser /     |                                          | ChessNova Server   |
| Web Client    | <--------------------------------------- | (internal/game)    |
|               |   Authoritative Move Broadcast + Clock   |                    |
+---------------+                                          +--------------------+
```

---

## 2. Monolith Module Structure (`apps/server`)

ChessNova uses a modular Go monolith where domain boundaries are clearly defined with minimal coupling:

```text
apps/server/
├── cmd/
│   └── server/
│       └── main.go           # Dependency injection, configuration, graceful shutdown
├── internal/
│   ├── auth/                 # Password hashing (bcrypt) and JWT claims generation/validation
│   ├── chess/                # 100% pure Go FIDE chess rules engine (zero network/DB knowledge)
│   ├── engine/               # Stockfish 19 UCI bridge + pure Go minimax fallback
│   ├── game/                 # Game lifecycle, ActiveGame memory cache, authoritative clocks
│   ├── http/                 # REST HTTP router, endpoints, authentication middleware
│   ├── matchmaking/          # In-memory + Redis queue, dynamic tolerance window matcher
│   ├── openings/             # ECO opening database (A00-E99) & book move identification
│   ├── puzzle/               # Tactical puzzle models, random picker, and validation
│   ├── rating/               # Elo calculation (400 baseline, 100 floor, K-32), leaderboards
│   ├── review/               # Game Review laboratory: CAPS accuracy, move classifications
│   ├── user/                 # User domain entity, MongoDB repository
│   └── websocket/            # Gorilla WebSocket hub, game rooms, broadcast engine
└── bin/
    └── stockfish             # Bundled Stockfish 19 UCI binary
```

---

## 3. Concurrency & Synchronization Model

The backend leverages Go's concurrency primitives (`sync.Mutex`, `sync.RWMutex`, channels, and goroutines) to handle concurrent games, clock ticks, and spectator streams safely:

### 3.1 Active Game Synchronization
Each ongoing game is managed by an `ActiveGame` struct in `internal/game/service.go`:
- Protected by a dedicated `sync.Mutex`.
- Serializes moves: two simultaneous requests from the same or different players are queued safely; race conditions on clock deductions or turns are impossible.
- Authoritative clock elapsed time is computed as:
  $$\text{elapsed} = \text{now} - \text{lastMoveAt}$$
  and subtracted from the active player's remaining time.

### 3.2 WebSocket Hub & Rooms
In `internal/websocket/websocket.go`:
- The `Hub` maintains a thread-safe map of active rooms (`map[string]*GameRoom`) guarded by `sync.RWMutex`.
- Each `GameRoom` tracks connected `*Client` instances with its own `sync.RWMutex`.
- Each `Client` possesses a non-blocking buffered write channel (`chan []byte, 256`). If a client's buffer fills (e.g. stalled connection), the connection is cleanly closed and pruned to prevent backpressure from starving other clients.

### 3.3 Authoritative Clock Timeout Daemon
- When an active game starts, a periodic background ticker (`time.NewTicker(250 * time.Millisecond)`) monitors elapsed time against the active player's clock.
- If remaining time drops to $\le 0$, the server immediately triggers a loss on time (`OutcomeTimeout`), updates the MongoDB record, recalculates player ratings, and broadcasts `game_finished` to all room participants.

### 3.4 Matchmaking Daemon
- A background goroutine (`matchmakingLoop`) triggers every 1 second.
- Compares tickets across identical time controls with dynamic rating expansion tolerance.

---

## 4. Data Flow

### 4.1 Live Matchmaking to Game Start Flow
```
Client A                Client B                  Server/Redis
   |                       |                           |
   |--- POST /join ------->|                           | Ticket A created (Rating 450, Blitz)
   |                       |--- POST /join ----------->| Ticket B created (Rating 480, Blitz)
   |                       |                           |
   |                       |                           | Matchmaker detects delta=30 <= tolerance
   |                       |                           | Initializes Game in MongoDB & ActiveGames
   |<-- MatchFound Event --|-- MatchFound Event ------>| Game ID, Color Assignment, Clock Init
   |                       |                           |
   |=== WS Connect =======>|                           | Hub joins Client A to Room
   |                       |=== WS Connect ===========>| Hub joins Client B to Room
   |<================= Authoritative Start Broadcast ==| Both receive initial FEN & 300s clocks
```

### 4.2 Move Execution Flow
```
Client White                     Server                           Client Black
     |                              |                                   |
     |-- WS { type: "move" } ------>|                                   |
     |                              | 1. Lock ActiveGame mutex          |
     |                              | 2. Verify White's turn            |
     |                              | 3. Validate move in pure engine   |
     |                              | 4. Deduct White clock + Increment |
     |                              | 5. Checkmate / Stalemate checks   |
     |                              | 6. Persist ply to MongoDB         |
     |                              | 7. Unlock ActiveGame mutex        |
     |<- WS { move, FEN, clocks } --|-- WS { move, FEN, clocks } ------>|
```

---

## 5. Persistence & Database Schemas

ChessNova uses **MongoDB** as its primary persistent store and **Redis** for ephemeral matchmaking tickets and caching.

### 5.1 MongoDB Collections

#### Collection: `users`
Stores account identities, hashed credentials, and timestamps.
```json
{
  "_id": ObjectId("651234567890abcdef123456"),
  "username": "grandmaster_alex",
  "email": "alex@chessnova.com",
  "passwordHash": "$2a$10$eWkZ0yV...",
  "createdAt": "2026-10-08T03:32:00Z",
  "updatedAt": "2026-10-09T01:14:00Z"
}
```
**Indexes**:
- Unique index on `username`
- Unique index on `email`

#### Collection: `user_ratings`
Maintains individual Elo ratings, win/loss records, and categories per player.
```json
{
  "_id": ObjectId("651234567890abcdef123457"),
  "userId": ObjectId("651234567890abcdef123456"),
  "username": "grandmaster_alex",
  "category": "blitz",
  "rating": 520,
  "games": 18,
  "wins": 12,
  "losses": 4,
  "draws": 2,
  "updatedAt": "2026-10-09T01:12:00Z"
}
```
**Indexes**:
- Compound unique index on `{ userId: 1, category: 1 }`
- Compound index on `{ category: 1, rating: -1 }` (for sub-millisecond leaderboard queries)

#### Collection: `games`
The authoritative immutable ledger of played matches.
```json
{
  "_id": "6705db859df6e...",
  "white": {
    "userId": "65123...",
    "username": "alex",
    "rating": 520
  },
  "black": {
    "userId": "65124...",
    "username": "beatrice",
    "rating": 480
  },
  "timeControl": {
    "initialSeconds": 300,
    "increment": 0
  },
  "status": "finished",
  "outcome": "checkmate",
  "result": "1-0",
  "moves": [
    {
      "ply": 1,
      "move": { "from": "e2", "to": "e4" },
      "san": "e4",
      "fenBefore": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
      "fenAfter": "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
    }
  ],
  "finalFen": "rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3",
  "pgn": "[Event \"ChessNova Live\"]\n[Site \"ChessNova\"]\n...",
  "startedAt": "2026-10-09T01:10:00Z",
  "finishedAt": "2026-10-09T01:14:00Z"
}
```
**Indexes**:
- Index on `white.userId`
- Index on `black.userId`
- Index on `status`
- Index on `startedAt` (descending)

#### Collection: `puzzles`
Tactical puzzles parsed from standard databases.
```json
{
  "_id": ObjectId("651234567890abcdef123499"),
  "fen": "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2N2N2/PPPP1PPP/R1BQK2R w KQkq - 0 6",
  "moves": ["c3e4", "d7d5", "c4d3"],
  "rating": 1200,
  "themes": ["fork", "tactics"],
  "opening": "Italian Game"
}
```

---

## 6. Frontend Client Topology (`apps/web`)

The client is built with SvelteKit 2 and Svelte 5 runes:
- **`src/routes`**: Route definitions matching URL paths:
  - `/` — Landing portal, quick start, featured leaderboards.
  - `/play` — Pass-and-play and bot play.
  - `/play/online` — Matchmaking lobby, queue search, live opponent pairings.
  - `/game/[gameId]` — Real-time live match room with clocks and move lists.
  - `/analysis` — Stockfish analysis board, PGN importer, and Game Review suite.
  - `/leaderboard` — Global category rankings across Bullet, Blitz, Rapid, Classical, and Puzzles.
  - `/puzzles` — Interactive tactical puzzles with move validation.
  - `/profile/[username]` — Player stats, rating history, and recent match archive.
- **`src/lib/stores`**:
  - `auth.ts` — JWT access tokens, current user session, login/logout reactive state.
  - `game.ts` — Authoritative game board state, legal move targets, clock ticks, and check alerts.
  - `localGames.ts` — Browser `localStorage` store preserving up to 50 games for guests and offline review.
  - `preferences.ts` — Board orientation, sounds, theme toggles, move hints.
- **`src/lib/chess/pieces.ts` & `ChessPiece.svelte`**:
  - Fully vector SVG Staunton pieces embedded directly into the DOM with dual-layer gradients and drop shadows. Zero external asset download latency or blurry PNG scaling.
