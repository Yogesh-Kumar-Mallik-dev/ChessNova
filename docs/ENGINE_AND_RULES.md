# Pure Go Chess Engine & FIDE Rules Specification

The `internal/chess` package is a ground-up, pure Go implementation of the complete Laws of Chess as defined by the International Chess Federation (FIDE). It contains **zero external dependencies** and possesses no knowledge of networking, databases, or HTTP.

---

## 1. Board Representation & Primitives

### 1.1 Square & Coordinates
Squares are represented as a flat integer index from `0` to `63`:
$$\text{Index} = \text{Rank} \times 8 + \text{File}$$
- `a1 = 0`, `h1 = 7`, `a8 = 56`, `h8 = 63`
- Ranks are indexed `0` (rank 1) through `7` (rank 8).
- Files are indexed `0` (file a) through `7` (file h).

```text
8 | 56 57 58 59 60 61 62 63    (a8 ... h8)
7 | 48 49 50 51 52 53 54 55
6 | 40 41 42 43 44 45 46 47
5 | 32 33 34 35 36 37 38 39
4 | 24 25 26 27 28 29 30 31
3 | 16 17 18 19 20 21 22 23
2 |  8  9 10 11 12 13 14 15
1 |  0  1  2  3  4  5  6  7    (a1 ... h1)
  +------------------------
     a  b  c  d  e  f  g  h
```

### 1.2 Piece & Color Primitives
```go
type Color uint8
const (
    White Color = iota
    Black
    NoColor
)

type PieceType uint8
const (
    NoPiece PieceType = iota
    Pawn
    Knight
    Bishop
    Rook
    Queen
    King
)

type Piece struct {
    Type  PieceType
    Color Color
}
```

### 1.3 Position State
The `Position` struct encapsulates complete instantaneous game state:
```go
type Position struct {
    Board           [64]Piece
    ActiveColor     Color
    Castling        CastlingRights // WhiteKingside, WhiteQueenside, BlackKingside, BlackQueenside
    EnPassantTarget *Square        // Valid target square or nil
    HalfmoveClock   int            // Fifty-move counter (resets on pawn move or capture)
    FullmoveNumber  int            // Incremented after Black moves
}
```

---

## 2. Move Generation & Validation Pipeline

The engine separates move generation into two rigorous phases:

```
[ Position ]
     │
     ▼
[ Generate Pseudo-Legal Moves ]
  - Sliding rays: Rook (+8, -8, +1, -1), Bishop (+9, -9, +7, -7), Queen (both)
  - Steppers: Knight (8 L-offsets), King (8 adjacent squares)
  - Pawns: 1-step, 2-step from home rank, diagonal captures, en passant
  - Castling moves if paths empty
     │
     ▼
[ Apply Move to Trial Board ]
     │
     ▼
[ King Safety Verification ] ──> Attacked by opponent? ──> [ REJECT MOVE ]
     │ (No)
     ▼
[ ACCEPT LEGAL MOVE ]
```

### 2.1 Pins, Discovered Checks, and Absolute Checks
- Absolute pins are enforced naturally: moving a pinned piece out of a line of attack leaves the king under check on the trial board, immediately disqualifying the move.
- In-check evasion: when in single or double check, only moves that eliminate the threat (capturing the checker, blocking the ray, or moving the king to a safe square) pass verification. Double check forces a king move.

---

## 3. FIDE Special Rules Compliance

### 3.1 Castling Rights & Execution
Castling is subject to all FIDE requirements:
1. **Unmoved Pieces**: Neither the King nor the chosen Rook may have previously moved.
2. **Clear Path**: All squares between the King and Rook must be unoccupied (`f1, g1` for White kingside; `b1, c1, d1` for White queenside).
3. **No Check Constraints**:
   - The King cannot be currently in check.
   - The King cannot pass through any square that is attacked by an enemy piece.
   - The King cannot land on a square that is attacked by an enemy piece.
4. **Castling Rights Lifecycle**:
   - If the King moves, all castling rights for that color are permanently revoked.
   - If a Rook moves, rights for that specific corner are revoked.
   - If an enemy piece captures a corner Rook on its home square (`a1`, `h1`, `a8`, `h8`), the castling right for that corner is revoked.

### 3.2 En Passant
- When a pawn advances two squares from its starting rank (`rank 2` to `4` for White, `rank 7` to `5` for Black) and lands adjacent to an enemy pawn, the intermediate square is set as the `EnPassantTarget`.
- **Single-Ply Expiry**: The `EnPassantTarget` is valid exclusively for the player's immediate subsequent turn. If not exercised, it is cleared (`nil`).
- When captured en passant, the target pawn on the adjacent file is excised from the board.

### 3.3 Pawn Promotion
- Triggered when a pawn reaches rank 8 (squares 56–63 for White) or rank 1 (squares 0–7 for Black).
- Supported promotion targets: Queen (`Q`), Rook (`R`), Bishop (`B`), and Knight (`N`).
- Moves omitting promotion when required are flagged as invalid.

---

## 4. Terminal Game State Detection

After every applied move, the engine evaluates the position against all five FIDE end conditions:

### 4.1 Checkmate
- The active player's King is in check: `IsInCheck(pos, activeColor) == true`.
- The active player has zero legal moves available: `len(LegalMoves(pos)) == 0`.
- **Result**: `OutcomeCheckmate` (`1-0` or `0-1`).

### 4.2 Stalemate
- The active player's King is **not** in check: `IsInCheck(pos, activeColor) == false`.
- The active player has zero legal moves available: `len(LegalMoves(pos)) == 0`.
- **Result**: `OutcomeStalemate` (`1/2-1/2`).

### 4.3 Insufficient Material
A draw is immediately declared if neither player has mating material:
1. **King vs King** ($K \text{ vs } K$)
2. **King and Bishop vs King** ($KB \text{ vs } K$)
3. **King and Knight vs King** ($KN \text{ vs } K$)
4. **King and Bishop vs King and Bishop with same-colored bishops** (both bishops reside on light squares or both on dark squares)

### 4.4 Threefold Repetition
- Every applied move generates a canonical `RepetitionKey`:
  $$\text{Key} = \text{PiecePlacement} + \text{ActiveColor} + \text{CastlingRights} + \text{EnPassantTarget}$$
- The key does not consider move counts or clock times.
- If any repetition key reaches a frequency $\ge 3$, the game state reports `OutcomeThreefoldRepetition` (`1/2-1/2`).

### 4.5 Fifty-Move Rule
- The `HalfmoveClock` tracks the number of halfmoves (plies) elapsed since the last pawn advance or piece capture.
- When `HalfmoveClock >= 100` (50 full moves for both sides), the game state reports `OutcomeFiftyMoves` (`1/2-1/2`).

---

## 5. Notation, SAN, and PGN Serialization

### 5.1 Standard Algebraic Notation (SAN)
The engine produces and parses strict FIDE SAN:
- **Piece Prefixes**: `K`, `Q`, `R`, `B`, `N` (pawns have no prefix).
- **Captures**: `x` (e.g. `Bxf7`, `exd5`).
- **Disambiguation**:
  1. When two pieces of the same type can move to the same destination, disambiguation by starting file is attempted (e.g. `Rad1`).
  2. If on the same file, disambiguation by rank is attempted (e.g. `R1d1`).
  3. If both file and rank could be ambiguous (e.g. three promoted Queens), full coordinate is used (e.g. `Qh4e1`).
- **Suffixes**:
  - `+` appended for check.
  - `#` appended for checkmate.
- **Castling**: `O-O` (kingside), `O-O-O` (queenside).
- **Promotion**: `=Q`, `=R`, `=B`, `=N` (e.g. `e8=Q#`).

### 5.2 FEN (Forsyth–Edwards Notation)
- `ToFEN(pos)` generates standard 6-field strings:
  `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1`
- `FromFEN(fen)` validates and deserializes arbitrary FEN strings into an authoritative `Position`.

### 5.3 Portable Game Notation (PGN)
- **`ExportPGN(game)`**: Formats Seven Tag Roster (`Event`, `Site`, `Date`, `Round`, `White`, `Black`, `Result`) alongside formatted plies and terminal outcome.
- **`ParsePGN(pgnStr)`**: Robust streaming tokenizer supporting headers, move numbering, recursive move playback, SAN interpretation, and trailing outcome tags.
