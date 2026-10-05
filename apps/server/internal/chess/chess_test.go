package chess

import (
	"testing"
)

func TestStartingPosition(t *testing.T) {
	pos := InitialPosition()
	fen := ToFEN(pos)
	if fen != StartingFEN {
		t.Fatalf("expected initial FEN %s, got %s", StartingFEN, fen)
	}

	legal := LegalMoves(pos)
	// White has 16 pawn moves (8 single + 8 double) + 4 knight moves = 20 legal moves
	if len(legal) != 20 {
		t.Fatalf("expected 20 opening legal moves, got %d", len(legal))
	}

	if IsCheck(pos) {
		t.Fatalf("starting position should not be in check")
	}
	if IsCheckmate(pos) {
		t.Fatalf("starting position should not be checkmate")
	}
	if IsStalemate(pos) {
		t.Fatalf("starting position should not be stalemate")
	}
}

func TestFoolsMate(t *testing.T) {
	g := NewGame()
	moves := []string{"f3", "e5", "g4", "Qh4#"}

	for _, m := range moves {
		_, err := g.MakeMoveSAN(m)
		if err != nil {
			t.Fatalf("failed move %s: %v", m, err)
		}
	}

	if g.Outcome != OutcomeCheckmate {
		t.Fatalf("expected checkmate, got %s", g.Outcome)
	}
	if g.Winner == nil || *g.Winner != Black {
		t.Fatalf("expected Black to win")
	}
	if g.Result() != "0-1" {
		t.Fatalf("expected 0-1, got %s", g.Result())
	}
}

func TestScholarsMate(t *testing.T) {
	g := NewGame()
	moves := []string{"e4", "e5", "Bc4", "Nc6", "Qh5", "Nf6", "Qxf7#"}

	for _, m := range moves {
		_, err := g.MakeMoveSAN(m)
		if err != nil {
			t.Fatalf("failed move %s: %v", m, err)
		}
	}

	if g.Outcome != OutcomeCheckmate {
		t.Fatalf("expected checkmate, got %s", g.Outcome)
	}
	if g.Winner == nil || *g.Winner != White {
		t.Fatalf("expected White to win")
	}
	if g.Result() != "1-0" {
		t.Fatalf("expected 1-0, got %s", g.Result())
	}
}

func TestCastlingRules(t *testing.T) {
	// Position where White can castle Kingside and Queenside
	fen := "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1"
	pos, err := FromFEN(fen)
	if err != nil {
		t.Fatalf("failed to parse FEN: %v", err)
	}

	moves := LegalMoves(pos)
	hasKingside := false
	hasQueenside := false
	for _, m := range moves {
		if m.From == E1 && m.To == G1 {
			hasKingside = true
		}
		if m.From == E1 && m.To == C1 {
			hasQueenside = true
		}
	}
	if !hasKingside || !hasQueenside {
		t.Fatalf("expected both castling moves to be legal")
	}

	// Apply Kingside Castling
	nextPos, err := ApplyMove(pos, Move{From: E1, To: G1})
	if err != nil {
		t.Fatalf("failed to apply castling: %v", err)
	}
	// King should be on G1, Rook should be on F1
	if nextPos.Board.Get(G1).Type != King || nextPos.Board.Get(G1).Color != White {
		t.Fatalf("expected White King on G1")
	}
	if nextPos.Board.Get(F1).Type != Rook || nextPos.Board.Get(F1).Color != White {
		t.Fatalf("expected White Rook on F1")
	}
	if nextPos.Castling.Has(CastlingWhiteKingside) || nextPos.Castling.Has(CastlingWhiteQueenside) {
		t.Fatalf("expected White to have lost all castling rights after castling")
	}

	// Castle through check test: enemy rook attacks F1
	fenThroughCheck := "r3k2r/8/8/8/5r2/8/8/R3K2R w KQkq - 0 1"
	posThroughCheck, _ := FromFEN(fenThroughCheck)
	moves = LegalMoves(posThroughCheck)
	for _, m := range moves {
		if m.From == E1 && m.To == G1 {
			t.Fatalf("castling through check on f1 should not be legal")
		}
	}

	// Castle out of check test: enemy rook attacks E1
	fenInCheck := "r3k2r/8/8/8/4r3/8/8/R3K2R w KQkq - 0 1"
	posInCheck, _ := FromFEN(fenInCheck)
	moves = LegalMoves(posInCheck)
	for _, m := range moves {
		if m.From == E1 && (m.To == G1 || m.To == C1) {
			t.Fatalf("castling out of check should not be legal")
		}
	}
}

func TestEnPassant(t *testing.T) {
	// White pawn on e5, Black pawn on d7
	fen := "rnbqkbnr/pppppppp/8/4P3/8/8/PPPP1PPP/RNBQKBNR b KQkq - 0 2"
	pos, err := FromFEN(fen)
	if err != nil {
		t.Fatalf("failed to parse FEN: %v", err)
	}

	// Black plays d7 to d5
	d5Move := Move{From: D7, To: D5}
	posAfterD5, err := ApplyMove(pos, d5Move)
	if err != nil {
		t.Fatalf("failed to play d5: %v", err)
	}

	if posAfterD5.EnPassant == nil || *posAfterD5.EnPassant != D6 {
		t.Fatalf("expected en passant square d6, got %v", posAfterD5.EnPassant)
	}

	// White plays exd6 e.p.
	epMove := Move{From: E5, To: D6}
	san, err := MoveToSAN(posAfterD5, epMove)
	if err != nil {
		t.Fatalf("expected exd6 to be legal: %v", err)
	}
	if san != "exd6" {
		t.Fatalf("expected SAN 'exd6', got '%s'", san)
	}

	posAfterEP, err := ApplyMove(posAfterD5, epMove)
	if err != nil {
		t.Fatalf("failed to apply en passant move: %v", err)
	}

	// Pawn on d5 should be captured and empty
	if !posAfterEP.Board.IsEmpty(D5) {
		t.Fatalf("expected square d5 to be empty after en passant capture")
	}
	if posAfterEP.Board.Get(D6) != NewPiece(Pawn, White) {
		t.Fatalf("expected White pawn on d6")
	}
}

func TestPawnPromotion(t *testing.T) {
	// White pawn on a7
	fen := "8/P7/8/8/8/8/8/4K2k w - - 0 1"
	pos, _ := FromFEN(fen)

	moves := LegalMoves(pos)
	promoQueen := false
	promoRook := false
	promoBishop := false
	promoKnight := false

	for _, m := range moves {
		if m.From == A7 && m.To == A8 {
			if m.Promotion != nil {
				switch *m.Promotion {
				case Queen:
					promoQueen = true
				case Rook:
					promoRook = true
				case Bishop:
					promoBishop = true
				case Knight:
					promoKnight = true
				}
			}
		}
	}

	if !promoQueen || !promoRook || !promoBishop || !promoKnight {
		t.Fatalf("expected all 4 promotion options to be legal")
	}

	// Promote to Knight
	pr := Knight
	knightPromo := Move{From: A7, To: A8, Promotion: &pr}
	san, err := MoveToSAN(pos, knightPromo)
	if err != nil {
		t.Fatalf("MoveToSAN failed: %v", err)
	}
	if san != "a8=N" && san != "a8=N+" {
		t.Fatalf("expected a8=N, got %s", san)
	}

	nextPos, err := ApplyMove(pos, knightPromo)
	if err != nil {
		t.Fatalf("failed promotion move: %v", err)
	}
	if nextPos.Board.Get(A8) != NewPiece(Knight, White) {
		t.Fatalf("expected Knight on a8")
	}
}

func TestStalemate(t *testing.T) {
	// Classic stalemate: Black King on a8, White King on c7, White Queen on b6
	fen := "k7/2K5/1Q6/8/8/8/8/8 b - - 0 1"
	pos, err := FromFEN(fen)
	if err != nil {
		t.Fatalf("failed to parse FEN: %v", err)
	}

	if IsCheck(pos) {
		t.Fatalf("position should not be check")
	}
	if !IsStalemate(pos) {
		t.Fatalf("position should be stalemate")
	}
	if !IsDraw(pos) {
		t.Fatalf("stalemate should be draw")
	}
}

func TestInsufficientMaterial(t *testing.T) {
	tests := []struct {
		fen      string
		expected bool
	}{
		{"8/8/8/4k3/8/8/4K3/8 w - - 0 1", true},                // K vs K
		{"8/8/8/4k3/8/5B2/4K3/8 w - - 0 1", true},              // K+B vs K
		{"8/8/8/4k3/8/5N2/4K3/8 w - - 0 1", true},              // K+N vs K
		{"8/8/8/4k3/8/5P2/4K3/8 w - - 0 1", false},             // K+P vs K
		{"8/8/8/4k3/8/5R2/4K3/8 w - - 0 1", false},             // K+R vs K
		{"8/1b6/8/4k3/8/5B2/4K3/8 w - - 0 1", true},            // K+B vs K+B same color (b7 is light, f3 is light)
		{"8/2b5/8/4k3/8/5B2/4K3/8 w - - 0 1", false},           // K+B vs K+B opposite color (c7 is dark, f3 is light)
	}

	for _, tc := range tests {
		pos, err := FromFEN(tc.fen)
		if err != nil {
			t.Fatalf("failed to parse fen %s: %v", tc.fen, err)
		}
		got := IsInsufficientMaterial(pos)
		if got != tc.expected {
			t.Errorf("for FEN %s, expected %v, got %v", tc.fen, tc.expected, got)
		}
	}
}

func TestThreefoldRepetition(t *testing.T) {
	g := NewGame()
	// Knights moving back and forth
	moves := []string{
		"Nf3", "Nf6", "Ng1", "Ng8", // 2nd time
		"Nf3", "Nf6", "Ng1", "Ng8", // 3rd time
	}

	for _, m := range moves {
		_, err := g.MakeMoveSAN(m)
		if err != nil {
			t.Fatalf("move %s failed: %v", m, err)
		}
	}

	if g.Outcome != OutcomeThreefoldRepetition {
		t.Fatalf("expected threefold repetition, got %s", g.Outcome)
	}
}

func TestFiftyMoveRule(t *testing.T) {
	fen := "8/8/8/4k3/8/8/4K3/8 w - - 99 50"
	pos, err := FromFEN(fen)
	if err != nil {
		t.Fatalf("failed to parse fen: %v", err)
	}

	// Move king: halfmove becomes 100
	nextPos, err := ApplyMove(pos, Move{From: E2, To: E3})
	if err != nil {
		t.Fatalf("king move failed: %v", err)
	}

	if !IsFiftyMoveRule(nextPos) {
		t.Fatalf("expected 50-move rule to be active at halfmove=100")
	}
}

func TestPinnedPieceCannotMoveIntoCheck(t *testing.T) {
	// White King on e1, White Rook on e2, Black Rook on e8 (Rook is pinned to King)
	fen := "4r3/8/8/8/8/8/4R3/4K3 w - - 0 1"
	pos, _ := FromFEN(fen)

	// Rook moving along rank (e.g. e2 to d2) should be illegal
	moves := LegalMoves(pos)
	for _, m := range moves {
		if m.From == E2 && m.To != E3 && m.To != E4 && m.To != E5 && m.To != E6 && m.To != E7 && m.To != E8 {
			t.Fatalf("pinned rook should only be allowed to move along pinned file, but found move %s", m.UCI())
		}
	}
}

func TestDisambiguationSAN(t *testing.T) {
	// Two White knights can move to d2 (Nbd2 vs Nfd2). d2 is empty (d4 pawn).
	fen := "rnbqkbnr/pppppppp/8/8/3P4/5N2/PPP1PPPP/RNBQKB1R w KQkq - 0 1"
	pos, _ := FromFEN(fen)

	// Knights are at b1 and f3, both can move to d2
	m := Move{From: B1, To: D2}
	san, err := MoveToSAN(pos, m)
	if err != nil {
		t.Fatalf("MoveToSAN failed: %v", err)
	}
	if san != "Nbd2" {
		t.Fatalf("expected 'Nbd2', got '%s'", san)
	}
}

func TestPGNExportAndParse(t *testing.T) {
	g := NewGame()
	_, _ = g.MakeMoveSAN("e4")
	_, _ = g.MakeMoveSAN("e5")
	_, _ = g.MakeMoveSAN("Nf3")
	_, _ = g.MakeMoveSAN("Nc6")

	pgnStr := g.ToPGN(map[string]string{
		"Event": "Antigravity Open",
		"White": "Carlsen",
		"Black": "Nakamura",
	})

	parsed, err := ParsePGN(pgnStr)
	if err != nil {
		t.Fatalf("ParsePGN failed: %v", err)
	}

	if parsed.Headers["White"] != "Carlsen" || parsed.Headers["Black"] != "Nakamura" {
		t.Fatalf("headers mismatch: %v", parsed.Headers)
	}
	if len(parsed.Moves) != 4 {
		t.Fatalf("expected 4 moves parsed, got %d", len(parsed.Moves))
	}
	if parsed.Moves[0].SAN != "e4" || parsed.Moves[3].SAN != "Nc6" {
		t.Fatalf("unexpected parsed moves: %v", parsed.Moves)
	}
}

func TestDiscoveredCheckAndDoubleCheck(t *testing.T) {
	// White King on e1, White Bishop on e4, White Rook on e1 is not there, let's set up:
	// White Queen on e2, White Knight on e4, Black King on e8.
	// When Knight moves to d6: Knight gives check AND opens Queen on e-file -> DOUBLE CHECK!
	fen := "4k3/8/8/8/4N3/8/4Q3/4K3 w - - 0 1"
	pos, err := FromFEN(fen)
	if err != nil {
		t.Fatalf("failed to parse FEN: %v", err)
	}

	d6Move := Move{From: E4, To: D6}
	san, err := MoveToSAN(pos, d6Move)
	if err != nil {
		t.Fatalf("MoveToSAN failed for double check: %v", err)
	}
	if san != "Nd6#" && san != "Nd6+" {
		t.Fatalf("expected check suffix, got %s", san)
	}

	nextPos, err := ApplyMove(pos, d6Move)
	if err != nil {
		t.Fatalf("failed to apply move: %v", err)
	}
	if !IsCheck(nextPos) {
		t.Fatalf("expected double check to leave black in check")
	}

	// In double check, the King MUST move (cannot block or capture one attacker because there are two)
	legalMoves := LegalMoves(nextPos)
	for _, lm := range legalMoves {
		piece := nextPos.Board.Get(lm.From)
		if piece.Type != King {
			t.Fatalf("in double check only King can move, but found move by %s: %s", piece.Type, lm.UCI())
		}
	}
}

func TestBlackCastling(t *testing.T) {
	fen := "r3k2r/8/8/8/8/8/8/4K3 b kq - 0 1"
	pos, _ := FromFEN(fen)

	moves := LegalMoves(pos)
	hasKingside := false
	hasQueenside := false
	for _, m := range moves {
		if m.From == E8 && m.To == G8 {
			hasKingside = true
		}
		if m.From == E8 && m.To == C8 {
			hasQueenside = true
		}
	}
	if !hasKingside || !hasQueenside {
		t.Fatalf("expected Black to be able to castle kingside and queenside")
	}

	// Black castles queenside
	nextPos, err := ApplyMove(pos, Move{From: E8, To: C8})
	if err != nil {
		t.Fatalf("black castling failed: %v", err)
	}
	if nextPos.Board.Get(C8) != NewPiece(King, Black) {
		t.Fatalf("expected Black King on c8")
	}
	if nextPos.Board.Get(D8) != NewPiece(Rook, Black) {
		t.Fatalf("expected Black Rook on d8")
	}
}
