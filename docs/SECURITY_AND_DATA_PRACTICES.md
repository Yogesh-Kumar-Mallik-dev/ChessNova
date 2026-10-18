# Backend Security, Authorization & Data Protection Practices

This document specifies the comprehensive security architecture, authentication and authorization enforcement, rate limiting, S3-compatible object storage lifecycle, and data protection standards implemented in ChessNova.

---

## 1. Threat Model & Security Posture

ChessNova operates on a **zero-trust, server-authoritative** model:
- **Clients never declare game positions, results, or evaluation scores.**
- **The client only submits intent** (e.g., `{ "from": "e2", "to": "e4" }`).
- **All rule validations, clock decrements, ratings, and Game Reviews are authoritatively executed on the Go backend.**
- **Anonymous/spectator clients have strictly read-only visibility into multiplayer games.**

```
Client (Browser / Tauri) ──[ Intent Payload / WS Ticket ]──> Go Server ──[ Validates Legality, Clock, Identity ]──> Authoritative State
```

---

## 2. Authentication & Credential Lifecycle

### 2.1 JWT Token Architecture & Secret Isolation
Authentication is implemented via stateless JSON Web Tokens (JWT) signed exclusively with HMAC-SHA256 (`HS256`):
- **Access Tokens (`accessToken`)**:
  - Expiry: **15 minutes**.
  - Payload claims: `userId`, `username`, `jti` (unique UUID), `tokenType: "access"`, `exp`, `iat`, `nbf`.
  - Transmitted via standard `Authorization: Bearer <token>` header or `access_token` cookie.
- **Refresh Tokens (`refreshToken`)**:
  - Expiry: **7 days**.
  - Payload claims: `userId`, `username`, `jti`, `tokenType: "refresh"`, `exp`, `iat`, `nbf`.
  - Signed with an independent refresh secret (`JWT_REFRESH_SECRET`) separate from the access secret (`JWT_ACCESS_SECRET`).
- **Production Secret Validation (`validateSecretsConfig`)**:
  - In production (`ENV=production`), application startup fails fast if `JWT_ACCESS_SECRET` or `JWT_REFRESH_SECRET` is unset, shorter than 32 characters, or identical to each other.

### 2.2 Dual-Mode Auth: HttpOnly Cookies & Bearer Headers
To shield browser users from token theft via potential Cross-Site Scripting (XSS) while maintaining native desktop app and external API compatibility:
- **HttpOnly Cookies**: On login, registration, and token refresh, the server sets:
  - `access_token` cookie: `HttpOnly`, `SameSite=Lax`, `Path=/`, `MaxAge=15m`, `Secure` (in production).
  - `refresh_token` cookie: `HttpOnly`, `SameSite=Lax`, `Path=/api/v1/auth`, `MaxAge=7d`, `Secure` (in production).
- **Desktop & API Fallback**: The `withAuth` middleware checks `Authorization: Bearer <token>` first; if absent, it checks the `access_token` cookie.
- **Web Client Credentials**: Web client API requests configure `credentials: 'include'` to automatically transmit session cookies.

### 2.3 Refresh Token Rotation & Theft Detection
To prevent replay attacks and token compromise:
1. Every token pair carries a cryptographically random `jti` (token ID).
2. When `POST /api/v1/auth/refresh` is invoked:
   - The presenting refresh token's `jti` is marked as revoked.
   - A completely new token pair with new `jti` values and updated `HttpOnly` cookies is issued.
3. **Theft Detection**: If an already rotated or revoked refresh token is presented again, the server assumes token theft, rejects the request (`401 Unauthorized`), and immediately triggers `RevokeUserSessions`, invalidating all active sessions for that user account.

### 2.4 Explicit Logout & Session Revocation
- When users log out (`POST /api/v1/auth/logout`), the active access token and refresh token `jti` values are written to Redis (or in-memory cache) with matching TTLs, and auth cookies are immediately expired (`MaxAge: -1`).
- `ValidateAccessToken` validates tokens against revocation state on every authenticated request.
- Frontend client (`apps/web/src/lib/api/client.ts` and `apps/web/src/lib/stores/auth.ts`) sends a revocation notification to `/auth/logout` prior to clearing local storage.

### 2.5 OTP Verification & Password Policies
- **Consistent Password Constraints**: 6 to 128 characters across both direct and OTP registration paths.
- **Password Hashing**: Passwords use **bcrypt** (`golang.org/x/crypto/bcrypt`). Plaintext passwords are never logged or serialized (`json:"-"`).
- **OTP Hardening**: OTP codes are 6-digit cryptographically random strings expiring in 10 minutes.
- **Zero Logging Policy**: Verification OTP codes are never logged to stdout, logs, or error responses. In production, unconfigured email delivery returns an error rather than outputting codes.
- **Email Verification**: Required for competitive multiplayer matchmaking (`EMAIL_NOT_VERIFIED` check).

---

## 3. Administrative Authentication & Secrets Isolation

### 3.1 Independent Administrative Credentials
- Administrative operations (such as `POST /api/v1/games/close-all`) are governed by `ADMIN_SECRET`.
- `ADMIN_SECRET` is strictly isolated from JWT signing keys and must be $\ge 32$ characters in production environments.
- Query parameter authentication (`?secret=`) is completely disabled. Credentials are only accepted via:
  - Header `X-Admin-Secret: <secret>`
  - Header `Authorization: Bearer <secret>`
- Comparison is performed using constant-time evaluation (`crypto/subtle.ConstantTimeCompare`) to prevent timing side-channel attacks.
- If `ADMIN_SECRET` is unset or empty, administrative endpoints are disabled and return `403 Forbidden` (`ADMIN_DISABLED`).

---

## 4. Public Profile Data Segregation

To prevent sensitive user information leakage:
- The public profile endpoint (`GET /api/v1/users/{username}`) serializes a dedicated `PublicUser` DTO (`internal/user/user.go`) containing only:
  - `id`, `username`, `avatar`, `bio`, `country`, `chessComUsername`, `lichessUsername`, `createdAt`.
- Private fields (`email`, `passwordHash`, `emailVerified`, `updatedAt`) are strictly omitted from public views.
- Private account data is accessible exclusively by the account owner via `GET /api/v1/users/me` with valid access token claims.

---

## 5. Network Hardening, Rate Limiting & Proxy Trust

### 5.1 Trusted Proxy IP Detection
To prevent rate limiting bypass via spoofed `X-Forwarded-For` or `X-Real-IP` headers:
- `getClientIP` in `internal/http/ratelimit.go` inspects the immediate socket peer (`r.RemoteAddr`).
- Forwarding headers are **only** evaluated if the remote address is a trusted proxy:
  - Loopback addresses (`127.0.0.1`, `::1`).
  - Private networks (RFC 1918: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`).
  - Explicitly configured IPs in `TRUSTED_PROXIES`.
- Forwarding headers from untrusted connections are discarded, enforcing rate limits against the real client socket.

### 5.2 Rate Limiting Matrix
Sliding-window rate limiting is implemented via Redis atomic pipelines with in-memory mutex fallback:

| Endpoint Path | Rate Limit | Scope | Mitigation Target |
| :--- | :--- | :--- | :--- |
| `/api/v1/auth/login`, `/register` | 10 req / min | Client IP | Credential stuffing & brute-force |
| `/api/v1/auth/register/verify-otp` | 5 req / min | Client IP | OTP brute-force enumeration |
| `/api/v1/auth/register/resend-otp` | 3 req / min | Client IP | Mailer abuse & spam exhaustion |
| `/api/v1/auth/refresh` | 30 req / min | Client IP | Token abuse |
| `/api/v1/matchmaking/join` | 15 req / min | User / IP | Queue flood & resource exhaustion |
| `/api/v1/puzzles/*` | 60 req / min | Client IP | Scraping & automated solving |
| `/api/v1/engine/evaluate` | 30 req / 10s | Client IP | CPU exhaustion |
| Global API routes | 300 req / min | Client IP | DoS protection |

### 5.3 Defensive HTTP Headers
All HTTP responses enforce modern defense-in-depth headers:

| Header | Value | Purpose |
| :--- | :--- | :--- |
| `X-Content-Type-Options` | `nosniff` | Blocks MIME sniffing |
| `X-Frame-Options` | `DENY` | Clickjacking mitigation |
| `X-XSS-Protection` | `1; mode=block` | Browser XSS filtering |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Origin privacy |
| `Content-Security-Policy` | Configured strict policy | Script & injection containment |

---

## 6. WebSocket Security & Connection Tickets

### 6.1 Single-Use Connection Tickets
To avoid exposing long-lived access tokens in WebSocket connection URLs and proxy access logs:
1. Client calls `POST /api/v1/ws/ticket` with standard Bearer authorization.
2. Server generates a cryptographically random, single-use ticket expiring in 60 seconds (`internal/websocket/websocket.go`).
3. Client establishes WebSocket connection using `ws://.../ws/game/{id}?ticket={ticket}`.
4. Ticket is atomically consumed upon connection; reuse attempts are rejected.

### 6.2 Origin Validation Policy
In `internal/websocket/websocket.go`:
- Validates origin scheme, host, and port against configured `CORS_ORIGIN` and production domains (`https://chessnova.com`).
- Explicitly permits native desktop schemes: `tauri://localhost` and `http://tauri.localhost`.
- In production (`ENV=production`), missing origins and arbitrary localhost ports are strictly rejected.

### 6.3 Spectator Policy, Connection Flooding & Per-Connection Rate Limiting
- **Per-IP Connection Caps (Slowloris / Flood Prevention)**: In `internal/websocket/websocket.go`, the hub enforces a strict maximum of **30 concurrent WebSocket connections per IP address**. Connection requests exceeding this threshold are immediately rejected with HTTP `429 Too Many Requests` prior to protocol upgrade.
- **Spectator Authorization**: Spectators connecting to game rooms have strictly read-only capabilities. Any incoming move, draw offer, resignation, or timeout claim from a non-player client is rejected immediately.
- **Message Rate Limiting**: Each WebSocket client is rate-limited to 25 messages per second to prevent spamming and CPU exhaustion.
- **Deadlines & Limits**: Configured with 60s read deadlines, periodic ping/pong heartbeats, and 64KB max payload sizes.

---

## 7. Puzzle Rating Integrity

- **Server-Side Verification**: `VerifyMove` in `internal/puzzle/puzzle.go` validates user moves strictly against the authoritative puzzle solution; client-provided success booleans are ignored.
- **Assistance Detection**: Calling `/puzzles/{id}/hint` or `/puzzles/{id}/solution` sets a server-side `puzzleAssisted` flag on the session, disabling rating gains.
- **Idempotent Scoring**: Duplicate or retry attempts on already-solved puzzles award 0 Elo points (`isRetry = true`). Anonymous users cannot alter ratings.

---

## 8. Avatar Upload Security, Dedicated S3 (MinIO) & SSRF Protections

### 8.1 Image Decoding & Polyglot Defense
Avatar upload in `internal/storage/storage.go` enforces multi-layer inspection:
1. Max request body bounded to 6 MB.
2. Sniffs MIME magic numbers (`image/png`, `image/jpeg`, `image/webp`).
3. Uses Go's `image.DecodeConfig` to parse actual image dimensions and block SVG XSS, polyglot shells, and decompression bombs ($> 4096 \times 4096\text{ px}$).

### 8.2 Dedicated On-Server MinIO S3 Storage & Secure Proxy
- **Self-Hosted MinIO**: Provided out-of-the-box via `docker-compose.yml`, MinIO provides high-performance on-server S3-compatible object storage isolated to `127.0.0.1:9000`.
- **Public URL Serving Proxy**: The Go backend exposes `GET /api/v1/storage/avatars/{filename}`, streaming avatars directly from S3 or local disk fallback:
  - Sanitizes input using `filepath.Base` to guarantee path traversal cannot escape the storage boundary.
  - Returns `Cache-Control: public, max-age=86400, stale-while-revalidate=604800` and `X-Content-Type-Options: nosniff`.
  - Avoids exposing internal S3 credentials or raw bucket endpoints to client browsers.

### 8.3 SSRF Protections for External Profiles
In `internal/external/external.go`:
- Usernames for Chess.com and Lichess are validated against strict regex: `^[a-zA-Z0-9_\-\.]{1,50}$`.
- The outgoing HTTP client implements a custom `CheckRedirect` interceptor blocking any redirect targeting private or loopback IP ranges (RFC 1918, `127.0.0.0/8`, `169.254.0.0/16`).

---

## 9. Desktop (Tauri) Content Security Policy

In `apps/desktop/src-tauri/tauri.conf.json`, the native webview enforces a strict Content Security Policy:
```text
default-src 'self' tauri: ipc:; script-src 'self' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https: blob:; connect-src 'self' https: http: ws: wss: ipc: tauri:; font-src 'self' data:;
```
- Allows WebAssembly execution for local Stockfish analysis (`'wasm-unsafe-eval'`).
- Blocks unauthorized external scripts and inline script execution.

---

## 10. Anti-Cheat & Fair Play Integrity

### 10.1 Server-Side Move Timing Heuristics
Scripted engine tampering and bot automation often submit moves with unnaturally mechanical timing:
- **Move Duration Tracking**: In `internal/game/service.go`, elapsed time per move is measured and persisted in `PersistedMove.TimeSpent` and `ActiveGame` duration series.
- **Variance Analysis**: For games with $\ge 12$ moves per player, `DetectSuspiciousTiming` computes the sample standard deviation ($\sigma$) of move times.
- **Detection Criteria**: A standard deviation $\sigma < 40\text{ms}$ across 12+ moves strongly indicates robotic automation or scripted injection. The game document is flagged with `suspiciousTiming: true` for administrative review.

---

## 11. Infrastructure & Data Store Isolation

In `docker-compose.yml`:
- **Strict Private Binding**: MongoDB (`27017:27017`), Redis (`6379:6379`), and MinIO (`9000:9000`, `9001:9001`) are bound exclusively to the loopback interface (`127.0.0.1`).
- **Container Network**: Database and object storage containers communicate over the internal Docker bridge network (`chess-net`), completely unreachable from the public internet.
