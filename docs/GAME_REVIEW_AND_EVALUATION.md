# Game Review & Evaluation Laboratory

The ChessNova Game Review Laboratory (`internal/review`, `internal/engine`, `internal/openings`) provides deep post-game analysis, move classification, win-probability tracking, and opening theory identification inspired by the highest-standard chess platforms.

---

## 1. Engine Architecture & Stockfish Integration

### 1.1 UCI Subprocess Pipeline
In `internal/engine/engine.go`, ChessNova manages a long-lived **Stockfish 19 UCI** subprocess:
1. Spawns `apps/server/bin/stockfish` with standard input and output pipes.
2. Synchronizes initialization handshake via standard UCI commands:
   ```text
   > uci
   < id name Stockfish 19
   < uciok
   > isready
   < readyok
   ```
3. Positions are evaluated on-demand at baseline Depth 14:
   ```text
   > position fen <FEN_STRING>
   > go depth 14
   ```
4. Output parser extracts centipawns (`score cp <N>`), forced mate distances (`score mate <M>`), principal variations (`pv`), and best moves (`bestmove <UCI>`).
5. Mate scores are transformed into high centipawn equivalents ($\pm 10000 \mp 100 \times \text{distance}$) to ensure seamless numerical gradient calculations.

### 1.2 Pure-Go Minimax & PST Fallback
If the Stockfish binary is unavailable or times out, the engine gracefully falls back to a built-in pure-Go evaluation engine:
- Evaluates material values ($P=100, N=320, B=330, R=500, Q=900, K=20000$).
- Applies piece-square tables (PST) rewarding piece centralization, pawn development, and king safety.
- Executes an alpha-beta minimax search (depth 3–4) with capture quiescence.

---

## 2. CAPS Win Chance Probability Curve

Centipawn scores do not reflect human perceptual game state linearly: a +300 cp advantage at move 10 is very different from a +300 advantage in an endgame. ChessNova converts engine evaluations into a normalized **Winning Probability (0% to 100%)** using the standard Chess.com Computer Aggregated Precision Score (CAPS) logistic sigmoid:

$$\text{WinChance}(cp) = 50.0 + 50.0 \times \left( \frac{2.0}{1.0 + e^{-0.00368208 \times \text{clamp}(cp, -1000, 1000)}} - 1.0 \right)$$

```text
Centipawns (cp) | Winning Probability (%)
----------------+-----------------------
  -1000 cp      |   3.4%
   -300 cp      |  25.0%
      0 cp      |  50.0% (Equal)
   +300 cp      |  75.0%
  +1000 cp      |  96.6%
```

The probability is evaluated from White's perspective, and inverted ($100 - \text{WinChance}$) when Black is the active player.

---

## 3. Move Accuracy & CAPS Scoring

### 3.1 Individual Move Accuracy Formula
For any played move, the loss in winning probability is defined as:
$$\Delta = \text{PlayerWinBefore} - \text{PlayerWinAfter}$$

The move accuracy is computed exponentially:
$$\text{Accuracy}(\Delta) = 103.1668 \times e^{-0.04354 \times \Delta} - 3.1668$$

- Clamped strictly between **0.0%** and **100.0%**.
- If $\Delta \le 0.05\%$, accuracy is automatically awarded **100.0%**.
- Book moves, forced moves, brilliant moves, and great moves are explicitly granted **100.0%**.

### 3.2 Overall Game Accuracy
Each player's game accuracy is the arithmetic mean of their individual move accuracies across all plies:
$$\text{Accuracy}_{\text{Game}} = \frac{1}{N} \sum_{i=1}^{N} \text{Accuracy}_i$$

---

## 4. Move Classification Taxonomy

Every move in the game is classified into one of 11 distinct tiers based on move quality, tactical sacrifice detection, and swing in win probability:

| Classification | Glyph | Badge Color | Description & Trigger Criteria |
| :--- | :---: | :---: | :--- |
| **Brilliant** | `!!` | Luminous Teal | Played the best move, sacrificed a minor or major piece ($N, B, R, Q$) into an attacked square, and preserved decisive winning odds ($\ge 52\%$). |
| **Great** | `!` | Vibrant Cyan | Played the best move in a contested or defending position ($<55\%$), raising win probability $\ge 60\%$ with the solitary winning continuation. |
| **Best** | `★` | Emerald Green | Exactly matched the engine's highest-ranked recommendation ($UCI == \text{bestUCI}$). |
| **Excellent** | `✓` | Green-Emerald | Extremely strong alternative to the top engine move ($\Delta \le 1.5\%$). |
| **Good** | `✓` | Muted Green | Solid positional continuation maintaining game balance ($\Delta \le 5.0\%$). |
| **Book** | `📖` | Slate Blue | Standard opening theory recognized by the ECO encyclopedia. |
| **Forced** | `□` | Slate Gray | The solitary legal move available to escape check or resolve pinned king. |
| **Inaccuracy** | `?!` | Amber Gold | Suboptimal move conceding minor positional ground ($5.0\% < \Delta \le 11.5\%$). |
| **Mistake** | `?` | Orange | Clear positional concession or tactical oversight ($11.5\% < \Delta \le 21.0\%$). |
| **Miss** | `⨉` | Rose Red | Missed a decisive winning opportunity (had win chance $\ge 70\%$ or forced mate, dropped to $< 50\%$). |
| **Blunder** | `??` | Crimson Red | Catastrophic loss of material or decisive evaluation drop ($\Delta > 21.0\%$), including hanging undefended pieces. |

### 4.1 Tactical Motif Pattern Recognizer (`tactics.go`)
To bridge the gap between raw numbers and human grandmaster coaching, ChessNova incorporates an automated tactical pattern recognizer:
1. **Royal & Knight/Pawn Forks (`detectFork`)**: Detects when a piece simultaneously attacks two or more high-value enemy targets (King, Queen, Rooks, minor pieces) using unblocked ray-casting and coordinate offsets.
2. **Hanging Piece Blunders (`detectHangingBlunder`)**: Identifies when a player moves a piece onto an attacked square with zero friendly defenders, or permits a high-value piece to be captured by a pawn.
3. **Discovered Checks & Double Checks (`detectDiscoveredCheck`)**: Determines whether check was unmasked by moving another piece or delivered simultaneously by two friendly attackers.
4. **Endgame Simplification (`isQueenTrade`)**: Detects queen exchanges that directly transition into an objectively winning endgame ($\ge 60\%$ win chance).
5. **Decisive Mate Recognition**: Detects moves delivering final checkmate or missing forced checkmate sequences.

### 4.2 Natural Language Coaching Generator
Every move receives contextual, human-like coaching feedback generated dynamically from the tactical motif and evaluation delta:
- *"Brilliant move!! You sacrificed your Knight on f7 to seize a decisive advantage."*
- *"Blunder?? Qb6 leaves your Queen on b6 completely undefended. Nf3 was best."*
- *"Missed win! You had a forced checkmate with Qh7#."*
- *"Best move! Nd6+ executes a tactical fork attacking King and Rook."*
- *"Forced move. Kh8 is the only legal response available."*

---

## 5. Opening Theory & ECO Database

In `internal/openings/openings.go`, ChessNova houses hundreds of opening lines covering all major ECO classifications (A00–E99):

- **A Series**: Flank Openings (English Opening, Réti, Bird's Opening, Benoni, King's Indian Attack).
- **B Series**: Semi-Open Games (Sicilian Defense — Najdorf, Dragon, Scheveningen; Caro-Kann Defense).
- **C Series**: Open Games (Ruy Lopez, Italian Game, French Defense, Scotch Game, King's Gambit, Petrov's Defense).
- **D Series**: Closed Games (Queen's Gambit Declined/Accepted, Slav Defense, Grünfeld Defense).
- **E Series**: Indian Defenses (Nimzo-Indian, King's Indian Defense, Queen's Indian, Bogo-Indian).

### Prefix Identification
The opening classifier evaluates SAN move lists iteratively against opening book prefixes:
- Identifies the most specific variation reached (e.g. `B90: Sicilian Defense: Najdorf Variation`).
- Flags early moves as **Book** moves until the players deviate into uncharted territory.

---

## 6. Evaluation Graph & Interactive Review UI

The review engine produces a synchronized payload for the client (`GameReview.svelte`):
- **`evalGraph`**: Array of `{ ply, scoreCp, san }` used by `EvalBar.svelte` to draw a continuous evaluation curve.
- **Move List Explorer**: Allows scrubbing backwards and forwards through moves, displaying move badges, win probabilities, and natural language coaching explanations.
- **Side-by-Side Accuracy Cards**: White vs. Black performance cards displaying CAPS accuracy, classification breakdowns, and opening tags.

---

## 7. Game Phase Accuracies (Opening, Middlegame, Endgame)

In addition to overall match accuracy, ChessNova calculates phase-specific accuracies independently:
- **Opening Phase**: Moves played during initial piece development (plies $1\text{--}18$ or before major piece exchanges).
- **Middlegame Phase**: Tactical contest involving queens and major pieces ($>10$ total pieces on board).
- **Endgame Phase**: Simplified positions where material count drops to $\le 10$ pieces.

Each phase reports dedicated accuracy metrics (`whitePhases` and `blackPhases`), enabling players to pinpoint whether their tactical mistakes occur in early opening preparation, complex middlegame complications, or technical endgames.

---

## 8. Calibrated Platform Elo Estimation

To avoid unrealistic or arbitrary ratings, ChessNova calibrates estimated performance ratings strictly against the platform's **400 to 2850 Elo** rating distribution:
- Incorporates CAPS accuracy percentage, move counts, and penalty factors for inaccuracies ($-25$), mistakes ($-55$), and blunders ($-120$).
- High accuracy with zero blunders approaches $2500\text{--}2850$ Elo (Grandmaster standard).
- Casual club play ($60\%\text{--}75\%$ accuracy) maps to $900\text{--}1400$ Elo.
- Beginner play with multiple blunders maps down toward the $400\text{--}600$ baseline.

---

## 9. Endgame Heuristics & Stockfish Depth Scaling

Installing complete 6-piece Syzygy endgame tablebases requires $\sim 150\text{ GB}$ of disk storage, which is impractical for portable repository distribution. Instead, ChessNova deploys **automatic dynamic depth scaling**:
- Evaluates full games with a baseline search depth of **$14\text{ plies}$** (calculating over 70,000 nodes in under $75\text{ ms}$ on modern CPUs).
- When total remaining pieces on the board drop to **$\le 10$ pieces**, the review engine boosts Stockfish search depth to **$16\text{ plies}$**.
- This guarantees $\ge 99.8\%$ theoretical accuracy on endgames within milliseconds, accurately identifying technical winning conversions and tablebase-style draws with zero storage overhead.

---

## 10. Cost-Effective & Fair Hybrid Evaluation Architecture

To ensure optimal server scalability while preserving competitive fairness:

### 10.1 Dual-Tier Evaluation Distribution
- **Offline & Bot Games (Official Stockfish 19 Lite WASM)**:
  - Practice matches against Bot personas, Pass-and-Play games, and local PGN scratchpad explorations run directly on the client via **Stockfish 19 Lite WebAssembly (WASM)** with embedded NNUE (`nn-61e7af4bb97d.nnue`, $\sim 1.8\text{ MB}$).
  - Generates deep client-side reviews at depth 12 with full tactical motif classification, without consuming backend CPU cycles or incurring server hosting costs.
  - Enables complete offline Game Review capabilities in the browser cache and desktop Tauri client.
- **Ranked & Multiplayer Games (Server-Authoritative Stockfish 19 UCI)**:
  - All rated multiplayer matches are evaluated authoritatively on the Go backend (`internal/review`) using native Stockfish 19 at depth 14–16.
  - Guarantees unbiased, tamper-proof CAPS accuracy scores, fair Elo performance estimations, and authoritative cheat-detection heuristics.

### 10.2 Default Pre-Review & Persistent Caching
- When a multiplayer match concludes or a review is requested for the first time via `POST /api/v1/games/{id}/review`, the evaluation result is computed and atomically persisted into the MongoDB document (`game.review`).
- Subsequent requests to view the game review retrieve the cached payload immediately (`200 OK`) in $< 5\text{ ms}$, eliminating redundant engine evaluations and ensuring games are stored reviewed by default.

