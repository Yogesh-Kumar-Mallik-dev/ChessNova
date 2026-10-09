# Elo Rating System & Dynamic Matchmaking

ChessNova adopts an accessible, progressive rating curve modeled directly after **Chess.com** rather than Lichess (which defaults to 1500). This provides beginner-friendly onboarding, meaningful progression milestones, and anti-deflationary floor clamping.

---

## 1. Rating Constants & Standards

| Parameter | Value | Rationale |
| :--- | :---: | :--- |
| **Default Starting Rating** | `400` | Welcoming entry baseline for new players without intimidating 1500 starting expectations. |
| **Minimum Rating Floor** | `100` | Strict floor ensuring ratings never drop below 100, preventing negative or demoralizing micro-scores. |
| **Default K-Factor** | `32` | Standard dynamic responsiveness for competitive games. |

---

## 2. Elo Mathematics & Floor Clamping

In `internal/rating/rating.go`, rating calculations follow the standard logistic distribution curve with strict floor clamping:

### 2.1 Expected Outcome
For two players with ratings $R_A$ and $R_B$:
$$E_A = \frac{1}{1 + 10^{\frac{R_B - R_A}{400}}}$$
$$E_B = 1.0 - E_A$$

### 2.2 Rating Updates
Given actual score $S_A$ ($1.0$ for win, $0.5$ for draw, $0.0$ for loss):
$$\Delta_A = \text{round}\Big(K \times (S_A - E_A)\Big)$$
$$\Delta_B = \text{round}\Big(K \times (S_B - E_B)\Big)$$

$$R'_A = \max\Big(\text{MinRating}, R_A + \Delta_A\Big)$$
$$R'_B = \max\Big(\text{MinRating}, R_B + \Delta_B\Big)$$

### 2.3 Sub-Floor Guarantee
Even if an input rating is corrupted or passed below 100, `CalculateElo` automatically clamps inputs to 100 prior to calculating deltas, and guarantees the resulting ratings remain $\ge 100$.

---

## 3. Time Control Categories

Ratings are partitioned by time control to reflect differing skills across clock formats:

| Category | Time Interval | Typical Configurations |
| :--- | :--- | :--- |
| **Bullet** | $< 3$ minutes | `1+0`, `1+1`, `2+1` |
| **Blitz** | $3$ to $< 10$ minutes | `3+0`, `3+2`, `5+0`, `5+3` |
| **Rapid** | $10$ to $< 30$ minutes | `10+0`, `15+10`, `20+0` |
| **Classical** | $\ge 30$ minutes | `30+0`, `60+30` |
| **Puzzles** | Untimed / Per Puzzle | Tactical puzzle rating |

Category determination is executed via `DetermineCategory(initialSeconds int)`:
```go
func DetermineCategory(initialSeconds int) Category {
    mins := initialSeconds / 60
    switch {
    case mins < 3:
        return Bullet
    case mins < 10:
        return Blitz
    case mins < 30:
        return Rapid
    default:
        return Classical
    }
}
```

---

## 4. Matchmaking Engine Architecture

The matchmaking service (`internal/matchmaking/matchmaking.go`) operates a hybrid in-memory and Redis-backed ticket pool:

### 4.1 Ticket Structure
When a user requests matchmaking, a ticket is generated:
```json
{
  "id": "ticket_6512...",
  "userId": "651234567890abcdef123456",
  "username": "alex",
  "rating": 450,
  "timeControl": {
    "initialSeconds": 300,
    "increment": 0
  },
  "createdAt": "2026-10-09T01:10:00Z"
}
```
- Stored locally in a thread-safe ticket slice.
- Mirrored in Redis under `mm:ticket:{userId}` with a 5-minute TTL.

### 4.2 Expanding Rating Tolerance Window
To balance match fairness against queue wait times, the allowable rating difference expands dynamically as a player waits in queue:

$$\text{Tolerance}(\text{waitDuration}) = \begin{cases}
\pm 50 & \text{if } t < 10\text{s} \\
\pm 100 & \text{if } 10\text{s} \le t < 20\text{s} \\
\pm 200 & \text{if } 20\text{s} \le t < 30\text{s} \\
\pm 300 & \text{if } t \ge 30\text{s}
\end{cases}$$

### 4.3 Pair Matching Logic
Every 1 second, the background `matchmakingLoop` evaluates all queued tickets:
1. Filters candidates that share identical `TimeControl` (`initialSeconds` and `increment`).
2. Ensures users do not match against themselves (`ticketA.UserID != ticketB.UserID`).
3. Computes the permissible rating window:
   $$\text{allowedTolerance} = \max(\text{tol}_A, \text{tol}_B)$$
4. If $|R_A - R_B| \le \text{allowedTolerance}$, the players are paired:
   - Randomly assigns White and Black colors.
   - Instantiates a persistent game record in MongoDB.
   - Registers an authoritative active game in `game.Service`.
   - Fires `onMatched` notification callback.
   - Prunes both tickets from the matchmaking queue and Redis.
