# REST API & WebSocket Protocol Reference

This document provides the complete API reference for ChessNova's REST endpoints and real-time bidirectional WebSocket protocol.

---

## 1. REST API Specification

### Base URL & Standards
- Base URL: `http://localhost:8080` (production: configured domain)
- Data format: JSON (`Content-Type: application/json`)
- Authentication: Standard HTTP Authorization header with JWT Bearer token:
  ```text
  Authorization: Bearer <accessToken>
  ```

---

### 1.1 System & Health
#### `GET /health`
Returns system health, database status, and current UTC server time.
- **Response**: `200 OK`
  ```json
  {
    "status": "ok",
    "time": "2026-10-09T01:14:00Z"
  }
  ```

---

### 1.2 Authentication
#### `POST /api/v1/auth/register`
Register a new player account. Default Elo rating of 400 is initialized across all categories.
- **Request Body**:
  ```json
  {
    "username": "grandmaster_alex",
    "email": "alex@chessnova.com",
    "password": "SecurePassword123!"
  }
  ```
- **Response**: `201 Created`
  ```json
  {
    "user": {
      "id": "651234567890abcdef123456",
      "username": "grandmaster_alex",
      "email": "alex@chessnova.com",
      "createdAt": "2026-10-09T01:14:00Z"
    },
    "accessToken": "eyJhbGciOiJIUzI1NiIs...",
    "refreshToken": "eyJhbGciOiJIUzI1NiIs..."
  }
  ```

#### `POST /api/v1/auth/login`
Authenticate with credentials and obtain JWT tokens.
- **Request Body**:
  ```json
  {
    "username": "grandmaster_alex",
    "password": "SecurePassword123!"
  }
  ```
- **Response**: `200 OK` (returns user and tokens).

#### `POST /api/v1/auth/refresh`
Exchange a valid refresh token for a newly signed access token.
- **Request Body**:
  ```json
  { "refreshToken": "eyJhbGciOiJIUzI1NiIs..." }
  ```

---

### 1.3 Users & Profiles
#### `GET /api/v1/users/me` *(Auth Required)*
Fetch the authenticated user's profile, bio, country, linked accounts, and current Elo ratings across all categories.
- **Response**: `200 OK`
  ```json
  {
    "user": {
      "id": "651234567890abcdef123456",
      "username": "grandmaster_alex",
      "email": "alex@chessnova.com",
      "avatar": "/uploads/avatars/avatar_6512_abc123.png",
      "bio": "Tactical attacking player.",
      "country": "United States",
      "chessComUsername": "alex_chess",
      "lichessUsername": "alex_li"
    },
    "ratings": {
      "bullet": 400,
      "blitz": 520,
      "rapid": 400,
      "classical": 400
    }
  }
  ```

#### `GET /api/v1/users/{username}`
Public profile endpoint retrieving public stats, bio, country, ratings, and recent games.

#### `PUT /api/v1/users/profile` *(Auth Required)*
Update the authenticated user's profile details.
- **Request Body**:
  ```json
  {
    "bio": "Grandmaster calculation with aggressive tactical openings.",
    "country": "Norway"
  }
  ```
- **Constraints**: `bio` max 500 characters; `country` max 50 characters.

#### `POST /api/v1/users/avatar` *(Auth Required)*
Upload a new avatar picture to S3-compatible object storage (or local fallback storage).
- **Request Format**: `multipart/form-data` with `avatar` file field.
- **Constraints**: Maximum size 5MB; enforced MIME signatures: `image/jpeg`, `image/png`, `image/webp`, `image/gif`.
- **Response**: `200 OK`
  ```json
  {
    "avatar": "https://cdn.chessnova.com/avatars/avatar_6512_abc123.jpg"
  }
  ```

#### `POST /api/v1/users/link-accounts` *(Auth Required)*
Link Chess.com and Lichess accounts to the user profile with automatic API existence verification.
- **Request Body**:
  ```json
  {
    "chessComUsername": "hikaru",
    "lichessUsername": "DrNykterstein"
  }
  ```
- **Response**: `200 OK` (or `400 Bad Request` if external accounts cannot be verified).

#### `GET /api/v1/users/{username}/external-stats`
Retrieves live synchronized rating comparisons across ChessNova, Chess.com, and Lichess.
- **Response**: `200 OK`
  ```json
  {
    "username": "grandmaster_alex",
    "chessnova": { "bullet": 400, "blitz": 520, "rapid": 400 },
    "chesscom": { "bullet": 1420, "blitz": 1580, "rapid": 1620, "puzzle": 1850 },
    "lichess": { "bullet": 1500, "blitz": 1610, "rapid": 1690, "puzzle": 1920 },
    "chessComUser": "alex_chess",
    "lichessUser": "alex_li"
  }
  ```

#### `GET /api/v1/users/{username}/external-games?platform=all&limit=10`
Fetches recent games played by the user on linked Chess.com and Lichess accounts with full PGN for 1-click Game Review.
- **Query Params**: `platform` (`all`, `chesscom`, `lichess`), `limit` ($1\text{--}20$, default 10).
- **Response**: `200 OK` (array of `ExternalGame` records).

---

### 1.4 Matchmaking
#### `POST /api/v1/matchmaking/join` *(Auth Required)*
Enqueues player into matchmaking pool for a specified time control.
- **Request Body**:
  ```json
  {
    "timeControl": {
      "initialSeconds": 300,
      "increment": 0
    }
  }
  ```
- **Response**: `200 OK`
  ```json
  {
    "status": "queued",
    "ticketId": "ticket_6512..."
  }
  ```

#### `POST /api/v1/matchmaking/leave` *(Auth Required)*
Cancels and removes the active queue ticket.

#### `GET /api/v1/matchmaking/status` *(Auth Required)*
Polls current matchmaking ticket status or returns matched game metadata.

---

### 1.5 Game Management
#### `GET /api/v1/games`
List historical games with query filters (`status`, `username`, `limit`).

#### `GET /api/v1/games/{id}`
Retrieve full game record, including move list, SANs, FENs, and result.

#### `GET /api/v1/games/{id}/pgn`
Download canonical Portable Game Notation (PGN) export.
- **Response Content-Type**: `application/x-chess-pgn`

---

### 1.6 Authoritative Pure Engine
#### `POST /api/v1/chess/legal-moves`
Queries legal destination squares from a given FEN for a chosen square.
- **Request Body**:
  ```json
  {
    "fen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
    "square": "e2"
  }
  ```
- **Response**: `200 OK`
  ```json
  {
    "square": "e2",
    "legalMoves": ["e3", "e4"]
  }
  ```

#### `POST /api/v1/chess/move`
Validates and executes a move against an arbitrary FEN, returning the resulting FEN, SAN, and check state.
- **Request Body**:
  ```json
  {
    "fen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
    "from": "e2",
    "to": "e4",
    "promotion": "q"
  }
  ```

---

### 1.7 Stockfish Analysis & Game Review
#### `POST /api/v1/analysis/evaluate`
Direct position evaluation via Stockfish 19 UCI.
- **Request Body**:
  ```json
  {
    "fen": "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
    "depth": 10
  }
  ```
- **Response**: `200 OK`
  ```json
  {
    "fen": "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
    "depth": 10,
    "scoreCp": 32,
    "isMate": false,
    "bestMove": "e7e5"
  }
  ```

#### `POST /api/v1/analysis/review`
Full Game Review laboratory endpoint supporting either move lists or raw PGN strings.
- **Request Body (PGN format)**:
  ```json
  {
    "pgn": "[Event \"Live\"]\n1. e4 e5 2. Qh5 Nc6 3. Bc4 Nf6 4. Qxf7# 1-0"
  }
  ```
- **Response**: `200 OK`
  ```json
  {
    "whiteAccuracy": 98.4,
    "blackAccuracy": 72.1,
    "opening": {
      "eco": "C20",
      "name": "King's Pawn Game: Wayward Queen Attack"
    },
    "moves": [
      {
        "ply": 1,
        "moveNumber": 1,
        "color": "white",
        "from": "e2",
        "to": "e4",
        "san": "e4",
        "accuracy": 100.0,
        "classification": "book",
        "explanation": "Book move. Standard opening theory."
      }
    ],
    "whitePhases": { "opening": 98.0, "middlegame": 94.5, "endgame": 96.0 },
    "blackPhases": { "opening": 85.0, "middlegame": 68.0, "endgame": 70.5 },
    "estimatedElo": { "white": 2150, "black": 1420 },
    "stats": {
      "white": { "brilliant": 0, "great": 0, "best": 2, "book": 2, "blunder": 0 },
      "black": { "brilliant": 0, "great": 0, "best": 1, "book": 1, "blunder": 1 }
    },
    "evalGraph": [
      { "ply": 0, "scoreCp": 20, "san": "" },
      { "ply": 1, "scoreCp": 35, "san": "e4" }
    ]
  }
  ```

#### `POST /api/v1/games/{id}/review`
Automated server-side review of any finished game by ID.

---

### 1.8 Leaderboards & Puzzles
- `GET /api/v1/leaderboard?category=blitz&limit=50`: Paginated top rankings per category.
- `GET /api/v1/puzzles/random?category=all&tier=1`: Retrieves a random tactical puzzle filtered by category (`mate`, `fork`, `pin`, `skewer`, `hanging_piece`, `advantage`) and progressive tier ($1\text{--}5$).
- `GET /api/v1/puzzles/daily`: Retrieves today's curated daily puzzle, deterministically cached per calendar day.
- `GET /api/v1/puzzles/weekly`: Retrieves the 7 curated weekly puzzles for the current calendar week.
- `GET /api/v1/puzzles/{id}`: Retrieves puzzle position and metadata by ID.
- `GET /api/v1/puzzles/{id}/hint`: Provides a contextual hint (e.g. piece type or tactical theme) without spoiling the solution.
- `GET /api/v1/puzzles/{id}/solution`: Retrieves the full step-by-step solution sequence.
- `POST /api/v1/puzzles/{id}/solve`: Validates user move sequence against engine solution and updates user puzzle rating.

---

## 2. Real-Time WebSocket Protocol

### Connection Handshake
Clients connect to:
```text
ws://localhost:8080/ws/game/{gameId}?token={accessToken}
```
*(Token is optional for spectators; required for players).*

Upon establishing connection, the server immediately emits the current board state (`game_start` or initial state).

---

### 2.1 Client-to-Server Messages

#### Move Intent (`move`)
Dispatched by the player whose turn it is:
```json
{
  "type": "move",
  "from": "e2",
  "to": "e4",
  "promotion": "q"
}
```

#### Resign (`resign`)
Forfeits the game immediately:
```json
{
  "type": "resign"
}
```

#### Draw Negotiation
- **Offer Draw**: `{ "type": "draw_offer" }`
- **Accept Draw**: `{ "type": "draw_accept" }`
- **Decline Draw**: `{ "type": "draw_decline" }`

---

### 2.2 Server-to-Client Broadcast Events

#### Move Executed (`move`)
Broadcasted to all players and spectators upon successful engine validation:
```json
{
  "type": "move",
  "gameId": "6705db85...",
  "move": { "from": 12, "to": 28 },
  "san": "e4",
  "fen": "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
  "turn": "black",
  "whiteTime": 298400,
  "blackTime": 300000,
  "status": "in_progress",
  "isCheck": false
}
```

#### Draw Events
- `draw_offer`: Emitted when an opponent offers a draw. Includes `offeredBy` (`"white"` or `"black"`).
- `draw_accepted`: Emitted when draw is agreed; terminal status updated to `1/2-1/2`.
- `draw_declined`: Emitted when draw offer is declined.

#### Game Finished (`game_finished`)
Emitted on checkmate, resignation, timeout, stalemate, or draw:
```json
{
  "type": "game_finished",
  "gameId": "6705db85...",
  "status": "finished",
  "outcome": "checkmate",
  "result": "1-0",
  "winner": "white",
  "fen": "rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3"
}
```

#### Error (`error`)
Emitted strictly to the initiating client if an illegal move, out-of-turn play, or invalid operation was attempted:
```json
{
  "type": "error",
  "error": "illegal move: King cannot move into check"
}
```
