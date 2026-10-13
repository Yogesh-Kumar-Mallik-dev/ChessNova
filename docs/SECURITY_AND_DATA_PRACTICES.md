# Backend Security, Authorization & Data Protection Practices

This document specifies the security architecture, authorization enforcement, S3-compatible object storage lifecycle, and data protection standards implemented in ChessNova.

---

## 1. Threat Model & Security Posture

ChessNova operates on a **zero-trust, server-authoritative** model:
- **Clients never declare game positions, results, or evaluation scores.**
- **The client only submits intent** (e.g., `{ "from": "e2", "to": "e4" }`).
- **All rule validations, clock decrements, ratings, and Game Reviews are authoritatively executed on the Go backend.**

```
Client (Browser) ──[ Intent Payload ]──> Go Server ──[ Validates Legality, Clock, Identity ]──> Authoritative State
```

---

## 2. Authentication & Authorization

### 2.1 JWT Token Architecture
Authentication is implemented via stateless JSON Web Tokens (JWT) signed with HMAC-SHA256:
- **Access Tokens (`accessToken`)**:
  - Expiry: **15 minutes**.
  - Payload: `userId`, `username`, `exp`, `iat`, `nbf`.
  - Transmitted via standard `Authorization: Bearer <token>` header.
- **Refresh Tokens (`refreshToken`)**:
  - Expiry: **7 days**.
  - Signed with an independent refresh secret (`JWT_REFRESH_SECRET`) separate from the access secret (`JWT_ACCESS_SECRET`).
  - Stored securely in client storage; exchanged via `POST /api/v1/auth/refresh`.

### 2.2 Password Security
- Passwords are encrypted using **bcrypt** with `DefaultCost` (`golang.org/x/crypto/bcrypt`).
- Plaintext passwords are never logged, stored, or returned in API responses (`json:"-"` BSON/JSON tag).
- Minimum password length: **8 characters**; maximum length: **128 characters** (preventing bcrypt silent truncation beyond 72 bytes).

### 2.3 Strict Ownership Authorization
- Profile edits (`PUT /api/v1/users/profile`), avatar uploads (`POST /api/v1/users/avatar`), and account links (`POST /api/v1/users/link-accounts`) strictly bind the operation to the authenticated `claims.UserID`.
- Users cannot modify or overwrite another user's bio, avatar, or linked accounts under any condition.
- Game moves and resignations (`POST /api/v1/games/{id}/moves`, `resign`) verify that `claims.UserID` matches either the White or Black player registered for that active match.

---

## 3. Defense-in-Depth & Network Hardening

### 3.1 Security Headers
All HTTP responses from the Go server (`internal/http/handler.go`) enforce modern defensive headers:

| Header | Value | Purpose |
| :--- | :--- | :--- |
| `X-Content-Type-Options` | `nosniff` | Disables MIME sniffing, preventing script execution from uploaded media. |
| `X-Frame-Options` | `DENY` | Prevents embedding within external `<iframe>` elements (Clickjacking mitigation). |
| `X-XSS-Protection` | `1; mode=block` | Activates browser XSS filtering and terminates page rendering if reflected attacks are detected. |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Protects privacy by preserving origin details only across encrypted requests. |

### 3.2 Granular CORS Policy
- Configurable via `CORS_ORIGIN` (defaults to `http://localhost:5173` in development).
- Explicit allowed methods: `GET, POST, PUT, DELETE, OPTIONS`.
- Explicit allowed headers: `Content-Type, Authorization, Accept, X-Requested-With`.
- Credentials enabled: `Access-Control-Allow-Credentials: true`.

### 3.3 Request Body Limits & DoS Mitigation
To prevent memory exhaustion attacks and Denial-of-Service via unbounded request streams:
- Standard JSON endpoints enforce `http.MaxBytesReader(w, r.Body, 1<<20)` (strictly $1\text{ MB}$).
- Avatar upload multipart endpoints enforce `http.MaxBytesReader(w, r.Body, 6*1024*1024)` ($6\text{ MB}$, covering the $5\text{ MB}$ file + boundary headers).

---

## 4. S3-Compatible Object Storage for Profile Avatars

### 4.1 Storage Architecture
The storage subsystem (`internal/storage/storage.go`) leverages `minio-go/v7` to support any standard S3-compliant object store:
- **Supported Providers**: AWS S3, Cloudflare R2, MinIO, Wasabi, Backblaze B2, Ceph.
- **Environment Variables**:
  - `S3_ENDPOINT`: S3 host (e.g., `s3.amazonaws.com` or Cloudflare R2 endpoint).
  - `S3_BUCKET`: Target bucket name (e.g., `chessnova-avatars`).
  - `S3_ACCESS_KEY` & `S3_SECRET_KEY`: IAM / API credentials.
  - `S3_REGION`: Storage region (e.g., `us-east-1` or `auto`).
  - `S3_USE_SSL`: `true` / `false`.
  - `S3_PUBLIC_URL`: Optional custom CDN or public domain prefix (e.g., `https://cdn.chessnova.com`).
  - `LOCAL_UPLOAD_DIR`: Local disk directory for zero-config fallback (default: `uploads/avatars`).

### 4.2 MIME Validation & Magic-Byte Sniffing
To prevent malicious file uploads (e.g., SVG-based stored XSS or disguised executables):
1. The server reads the raw byte stream into memory.
2. The byte stream is inspected via `http.DetectContentType(fileData)` for true file signature magic numbers.
3. Only the following MIME types are accepted:
   - `image/jpeg` (`.jpg`)
   - `image/png` (`.png`)
   - `image/webp` (`.webp`)
   - `image/gif` (`.gif`)
4. Files disguised with false headers or SVG markup are rejected immediately with `400 Bad Request`.

### 4.3 Automatic Local Storage Fallback
When running locally or during self-hosted deployment without S3 credentials, the storage engine:
- Automatically detects missing S3 configuration.
- Safely writes validated avatar images to `./uploads/avatars/avatar_<userID>_<randomHex>.<ext>`.
- Serves avatars directly through a protected static HTTP file handler at `/uploads/avatars/*`.

---

## 5. External Platform Verification (Chess.com & Lichess)

To preserve data integrity when users link external accounts:
- Usernames are validated against alphanumeric and hyphen/underscore rules.
- When `POST /api/v1/users/link-accounts` is invoked, the Go server performs a synchronous verification request to Chess.com (`/pub/player/{username}/stats`) and Lichess (`/api/user/{username}`).
- If the account does not exist on the external platform, the linking attempt is rejected with a clear error message.
- External game archives imported for Game Review are sanitized and parsed through the authoritative pure-Go PGN parser before evaluation.
