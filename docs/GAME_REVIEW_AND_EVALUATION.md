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
3. Positions are evaluated on-demand:
   ```text
   > position fen <FEN_STRING>
   > go depth 10
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
| **Brilliant** | `!!` | Luminous Teal | Played the best move, sacrificed a minor or major piece ($N, B, R, Q$) into an attacked square, and preserved winning odds ($\ge 50\%$). |
| **Great** | `!` | Vibrant Cyan | Played the best move in a contested or defending position ($<55\%$), raising win probability $\ge 60\%$ with solitary winning continuation. |
| **Best** | `★` | Emerald Green | Exactly matched the engine's highest-ranked recommendation ($UCI == \text{bestUCI}$). |
| **Excellent** | `✓` | Green-Emerald | Extremely strong alternative to the top engine move ($\Delta \le 1.5\%$). |
| **Good** | `✓` | Muted Green | Solid positional continuation maintaining game balance ($\Delta \le 5.0\%$). |
| **Book** | `📖` | Slate Blue | Standard opening theory recognized by the ECO encyclopedia. |
| **Forced** | `□` | Slate Gray | The solitary legal move available to escape check or resolve pinned king. |
| **Inaccuracy** | `?!` | Amber Gold | Suboptimal move conceding minor positional ground ($5.0\% < \Delta \le 11.5\%$). |
| **Mistake** | `?` | Orange | Clear positional concession or tactical oversight ($11.5\% < \Delta \le 21.0\%$). |
| **Miss** | `⨉` | Rose Red | Missed a decisive winning opportunity (had win chance $\ge 68\%$, dropped to $< 50\%$). |
| **Blunder** | `??` | Crimson Red | Catastrophic loss of material or decisive evaluation drop ($\Delta > 21.0\%$). |

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
