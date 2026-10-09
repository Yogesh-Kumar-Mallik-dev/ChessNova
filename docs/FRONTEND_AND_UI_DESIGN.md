# Frontend Architecture & UI/UX Design System

The ChessNova frontend (`apps/web`) is built with **SvelteKit 2**, **TypeScript 5**, and **Tailwind CSS 3**, prioritizing ultra-fast reactive state, modern desktop ergonomics, zero layout shifts, and visual aesthetics.

---

## 1. The "Midnight Slate" Design System

Rather than generic green/beige boards, ChessNova introduces a custom dark-mode aesthetic:

### 1.1 Color Palette
- **App Canvas**: Deep Obsidian (`slate-950` / `#020617`) with subtle ambient gradient lighting.
- **Card & Panel Surfaces**: Midnight Slate (`slate-900` / `#0f172a` and `slate-800/80` glassmorphism).
- **Light Squares**: Frosted Zinc (`#e2e8f0` / `#cbd5e1`).
- **Dark Squares**: Slate Harbor (`#475569` / `#334155`).
- **Last Move Highlight**: Indigo Glow (`rgba(99, 102, 241, 0.4)`).
- **Check State Warning**: Crimson Alert (`rgba(239, 68, 68, 0.5)` with pulsating red ring).
- **Target Indicators**: Translucent dots for empty legal squares, luminous outer rings for captures.

### 1.2 Zero-Emoji Design Standard
All emojis have been eliminated from the platform UI. All icons are rendered via lightweight, sharp vector SVGs (`apps/web/src/lib/components/icons/Icon.svelte`) ensuring crisp presentation on Retina and 4K displays.

---

## 2. Bespoke Vector Staunton Pieces

Instead of external sprite sheets or low-res raster images, ChessNova renders custom Staunton chess pieces via inline SVG vectors (`apps/web/src/lib/components/chess/ChessPiece.svelte`):

```text
       ┌──────────┐
      /  ♔ White  \  Linear gradient: #ffffff -> #e2e8f0
     │   Staunton  │ Stroke: #1e293b, Drop-shadow: rgba(0,0,0,0.25)
      \          /
       └──────────┘
       ┌──────────┐
      /  ♚ Black  \  Linear gradient: #334155 -> #0f172a
     │   Staunton  │ Stroke: #020617, Highlight inner shadow
      \          /
       └──────────┘
```

- **Vector Scalability**: Pieces scale losslessly to any resolution without artifacting.
- **Instant Rendering**: Embedded directly in the client bundle; zero network latency or flickering on page load.
- **Drop Shadows & Gradients**: Subtle dual-tone depth gives pieces a physical, tactile presence on the board.

---

## 3. Viewport Height Clamping & Board Scaling

A common flaw in online chess interfaces is board overflow on 13"–15" laptop screens, forcing players to scroll to see the whole board. ChessNova prevents this with strict responsive viewport clamping:

```css
/* Responsive Board Container */
.board-container {
    width: 100%;
    max-width: min(100%, calc(100dvh - 180px), calc(100vw - 32px));
    aspect-ratio: 1 / 1;
}
```

- **100% Full-Board Visibility**: The 8x8 grid is fully visible at all times across all screen ratios without triggering page scrollbars.
- **Side Panel Stacking**: On desktop ($>1024\text{px}$), clocks and move history sit in a dedicated column beside the board. On mobile/tablets, clocks stack cleanly above and below the board.

---

## 4. Board Interaction Model

The chessboard component (`ChessBoard.svelte`) supports both desktop and mobile interaction patterns:

1. **Click-to-Move**: Click a piece to highlight all authoritative legal destinations; click destination square to execute move.
2. **Drag-and-Drop**: Smooth pointer tracking across squares with release detection.
3. **Pawn Promotion Modal**: When a pawn reaches the terminal rank, an accessible modal (`PromotionDialog.svelte`) renders piece options (Queen, Knight, Rook, Bishop) for selection.
4. **Authoritative State Reconciliation**: Client-side board state syncs instantaneously upon receiving server WebSocket broadcasts. If a move is rejected by the server, the client immediately snaps back to canonical state.

---

## 5. Game Review & Evaluation Laboratory UI

The Game Review interface (`apps/web/src/routes/analysis` and `GameReview.svelte`) provides an analysis experience:

```
┌─────────────────────────────────────────────────────────────┐
│ White Accuracy: 92.4%               Black Accuracy: 68.1%   │
│ Opening: C20 King's Pawn Game: Wayward Queen Attack         │
├─────────────────────────────────────────────────────────────┤
│                     [ Evaluation Graph ]                    │
│   +3.5 ───────────────────▲─────────▲──────────────         │
│    0.0 ────────────────────\───────/─\─────────────         │
│   -2.0 ─────────────────────▼─────/───▼────────────         │
├─────────────────────────────────────────────────────────────┤
│ Move 14: 14... Nf6?? (Blunder)                              │
│ Best move was 14... Be6. Knight move blunders the queen!    │
├─────────────────────────────────────────────────────────────┤
│ [◄◄ First]  [◄ Previous]    [Play/Pause]    [Next ►]  [Last ►►]│
└─────────────────────────────────────────────────────────────┘
```

- **Dynamic Evaluation Bar (`EvalBar.svelte`)**: Animated vertical or horizontal bar showing real-time win probability and engine advantage.
- **Classification Badges (`ReviewBadge.svelte`)**: Tiers for Brilliant, Great, Best, Excellent, Good, Book, Forced, Inaccuracy, Mistake, Miss, and Blunder.
- **Move Scrubbing**: Jump to any ply by clicking move cells or using keyboard arrow navigation.

---

## 6. Local Guest Game Archive (`localGames.ts`)

Unauthenticated and guest players enjoy full match history and post-game review:
- Whenever a pass-and-play, bot, or live game completes, it is saved into browser `localStorage` (`chessnova_local_games_v1`).
- Stores up to 50 games with complete PGN, timestamps, results, and outcome metadata.
- Enables 1-click **"Review Game"** from the local match archive, transmitting PGN to the Stockfish laboratory.

---

## 7. Web Audio Sound Engine (`sounds.ts`)

ChessNova includes an audio synthesizer and sound effect player:
- **Events**: Move, Capture, Check, Castle, Game Start, Victory, Defeat.
- **Resilient Fallback**: Uses pre-rendered high-fidelity audio samples with an automated fallback to the browser's Web Audio API (`AudioContext` oscillator synthesis) if audio files are blocked or offline.
