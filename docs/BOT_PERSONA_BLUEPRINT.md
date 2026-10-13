# ChessNova Bot Persona & Personality Architecture Blueprint

This document defines the architectural blueprint for ChessNova's **Personality-Driven Bot Simulation System**. It transitions the platform from a generic 5-level difficulty slider into a rich, human-like simulation engine capable of modeling distinct player archetypes, phase-specific blunder curves, tactical biases, material sacrifice desires, and contextual in-game trash talk.

---

## 1. Architectural Philosophy: Beyond Stockfish Depth

A standard chess engine evaluates moves along a single axis: **objective mathematical truth**. Setting lower depth or a flat blunder rate merely produces a drunk grandmaster.

Real human players—from 250 Elo novices to 2200 Elo masters—have **distinct psychological profiles**:
- **Phase Variance**: A player might memorize 15 moves of opening theory, blunder pieces in the chaotic middlegame, but play textbook king-and-pawn endgames.
- **Evaluation Bias**: An aggressive attacker values an open h-file and king pressure significantly higher than a 1-pawn material deficit; a materialist hoards pawns even at the cost of king safety.
- **Piece Obsessions**: Players like Chess.com's "Nelson" bring their Queen out on move 2; a romantic attacker eagerly sacrifices rooks on `f7` or `h7`.
- **Ego & Psychology**: An overconfident player trash-talks after a dubious sacrifice and becomes panicked or passive when their bluff is refuted.

---

## 2. Core Persona Data Model (`BotProfile`)

```go
package bot

import "chess-platform/server/internal/chess"

type GamePhase string
const (
    PhaseOpening    GamePhase = "opening"
    PhaseMiddlegame GamePhase = "middlegame"
    PhaseEndgame    GamePhase = "endgame"
)

type BotProfile struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Title       string   `json:"title"`       // e.g. "The Rook Gambler", "The Trash Talker"
    Avatar      string   `json:"avatar"`      // SVG or avatar identifier
    Rating      int      `json:"rating"`      // Target display rating (e.g. 400, 1100, 1750)
    Bio         string   `json:"bio"`
    Playstyle   string   `json:"playstyle"`   // "Romantic Attacker", "Positional Grinder", etc.

    // 1. Opening Repertoire
    PreferredOpenings []string `json:"preferredOpenings"` // ECO prefixes e.g. ["C20", "C30", "B01"]
    QueenAttackEarly  bool     `json:"queenAttackEarly"`  // E.g. Nelson bot (Wayward Queen attack)

    // 2. Phase-Dependent Blunder Curves (0.0 to 1.0)
    PhaseBlunderProb map[GamePhase]float64 `json:"phaseBlunderProb"`

    // 3. Evaluation & Behavioral Weight Vector
    Weights StyleWeights `json:"weights"`

    // 4. Trash-Talk & Dialogue Configuration
    Dialogue BotDialogue `json:"dialogue"`
}

type StyleWeights struct {
    // Material Preferences
    SacrificeRookDesire    float64 // > 0 encourages Rook sacrifices on f7/h7/g7
    SacrificeKnightDesire  float64 // > 0 encourages Greek gift / knight sac
    AvoidPieceTrades       float64 // > 0 keeps pieces on board; < 0 forces simplifications
    QueenActivityWeight    float64 // Multiplier on early queen movement evaluation

    // Positional vs Tactical Bias
    Aggression             float64 // Weight on attacking king safety & checks (+2.0 = berserker)
    PositionalPawnStructure float64 // Weight on doubled pawns, isolated pawns
    PawnStormDesire        float64 // Weight on advancing f, g, h pawns against enemy king
    DefensiveCaution       float64 // Negative rewards for risky king exposure
}

type BotDialogue struct {
    Tone           string            // "arrogant_jerk", "friendly_grandmaster", "clueless_noob"
    OnGameStart    []string          `json:"onGameStart"`
    OnTakeQueen    []string          `json:"onTakeQueen"`
    OnLoseQueen    []string          `json:"onLoseQueen"`
    OnBlunder      []string          `json:"onBlunder"`       // When the bot makes a mistake
    OnPlayerBlunder []string         `json:"onPlayerBlunder"` // When player blunders
    OnGiveCheck    []string          `json:"onGiveCheck"`
    OnCheckmate    []string          `json:"onCheckmate"`
    OnDefeat       []string          `json:"onDefeat"`
}
```

---

## 3. The 5-Step Move Decision Pipeline

```
  [ Current FEN ] ───► [ 1. Game Phase Detection ] 
                              │
                              ▼
                       [ 2. Opening Book Match ] ──(Book Move Found)──► [ PLAY MOVE ]
                              │ (Deviated from Book)
                              ▼
                       [ 3. Stockfish Multi-PV Search ]
                         (Generates Top 5 Candidate Moves)
                              │
                              ▼
                       [ 4. Persona Utility Scoring ]
                         U(m) = EngineEval(m) 
                              + AggressionBonus(m) 
                              + SacrificeBonus(m)
                              + TradePenalty(m)
                              │
                              ▼
                       [ 5. Phase-Blunder Sampling ]
                         Roll random float against PhaseBlunderProb[currentPhase]
                              ├─► If Blunder Triggered: Pick 3rd/4th best or random legal move
                              └─► If Normal: Pick highest scored persona move
```

### Step 1: Game Phase Detection
Phase is calculated authority-side using non-pawn material count:
- **Opening**: Move count $< 10$ and both sides have $\ge 30$ points of non-pawn material.
- **Middlegame**: Queens on the board with minor/major pieces active.
- **Endgame**: Both players have $\le 13$ points of material (e.g. Queen traded, or only R+B vs R+N).

### Step 2: Persona Evaluation Adjustment Formula
For each candidate legal move $m$, the bot recalculates its subjective utility $U(m)$:

$$U(m) = \text{Score}_{\text{cp}}(m) + W_{\text{aggression}} \cdot \Delta_{\text{KingSafety}} + W_{\text{sac}} \cdot \mathbb{I}_{\text{Sacrifice}}(m) - W_{\text{trade}} \cdot \mathbb{I}_{\text{EqualTrade}}(m)$$

- If the persona **"Loves Sacrificing the Rook"** ($W_{\text{sac}} = +400\text{cp}$), a rook sacrifice that evaluates at $-150\text{cp}$ in Stockfish will be boosted to $+250\text{cp}$ in the bot's mind, making it enthusiastically sacrifice the piece!
- If the persona is **"Overconfident Jerk"**, it avoids queen trades ($W_{\text{trade}} = -300\text{cp}$) because it believes it can checkmate you with the queen.

---

## 4. Persona Archetype Specifications

### Archetype 1: `"martin_novice"` (Rating: 400 — The Clueless Beginner)
- **Phase Blunders**: Opening $40\%$, Middlegame $70\%$, Endgame $60\%$.
- **Behavior**: Shuffles pieces aimlessly, moves pawns 1 square, frequently walks into Scholar's mate and tactical forks.
- **Dialogue Tone**: *"Wait, how does the horsey move again?"*, *"Oops, did I need that queen?"*

### Archetype 2: `"chad_overconfident"` (Rating: 1100 — The Arrogant Brawler)
- **Phase Blunders**: Opening $15\%$, Middlegame $45\%$, Endgame $30\%$.
- **Behavior**: Premature queen attacks (Wayward Queen / Scholar attempts). Refuses all piece trades because *"trading is for cowards"*. Pushes h and g pawns recklessly.
- **Dialogue Tone**: *"Is that your best move? My grandma plays faster."*, *"Enjoy your last 5 moves."*, *"Whatever, you got lucky."*

### Archetype 3: `"rocky_sacrificer"` (Rating: 1350 — The Romantic Gambiteer)
- **Phase Blunders**: Opening $10\%$, Middlegame $25\%$, Endgame $15\%$.
- **Openings**: King's Gambit (`C30`), Danish Gambit (`C21`), Evans Gambit (`C51`).
- **Behavior**: $W_{\text{sac}} = +500\text{cp}$. Whenever a minor piece or rook can be sacrificed near the enemy king, it plays it immediately. Relentless piece-for-initiative play.
- **Dialogue Tone**: *"Material is an illusion, attack is everything!"*, *"Take my rook, I dare you!"*

### Archetype 4: `"tortoise_grinder"` (Rating: 1600 — The Defensive Iron Wall)
- **Phase Blunders**: Opening $5\%$, Middlegame $10\%$, Endgame $5\%$.
- **Openings**: London System (`D02`), Caro-Kann (`B12`), Petroff (`C42`).
- **Behavior**: Zero sacrifices ($W_{\text{sac}} = -500\text{cp}$), extreme caution, actively forces piece trades to drag the player into a grinding endgame where it plays with near-perfect technique.
- **Dialogue Tone**: *"A solid position cannot be breached."*, *"Welcome to the endgame."*

---

## 5. In-Game Trash Talk & Chat Engine

### 5.1 WebSocket Event Protocol
When the bot plays a move or responds to a game trigger, the server broadcasts a `bot_chat` event to the room:

```json
{
  "type": "bot_chat",
  "gameId": "game_12345",
  "botId": "chad_overconfident",
  "text": "Did you really think I wouldn't see that fork? Nice try, amateur.",
  "mood": "taunting",
  "avatar": "/avatars/chad.svg"
}
```

### 5.2 Mood States
- `smug`: When ahead by $\ge +300\text{cp}$ or when player blunders.
- `aggressive`: When delivering check or launching a piece sacrifice.
- `panicked`: When the bot's evaluation collapses below $-400\text{cp}$.
- `confused`: Novice bots wondering why a piece disappeared.
- `gracious`: Polite bots complimenting a brilliant move.

---

## 6. Implementation & Roadmap Architecture

```text
apps/server/internal/bot/
├── persona.go         # BotProfile, StyleWeights, Dialogue definitions
├── catalog.go         # Built-in bot roster (Martin, Chad, Rocky, Tortoise, etc.)
├── evaluator.go       # Utility function U(m) biasing Stockfish Multi-PV outputs
├── engine.go          # BotMoveService orchestrating phase detection & sampling
└── chat.go            # Context-sensitive dialogue trigger matcher
```

### Frontend Integration (`apps/web`):
1. **Persona Selection Grid**:
   - Replaces the generic "Level 1–5" buttons with visual persona cards showing avatar, rating, playstyle badge, and quote.
2. **Live Bot Companion Card**:
   - Displays bot portrait above the board with animated speech bubbles and reactive facial expressions (smug, shocked, calm) synchronized with the evaluation bar.
