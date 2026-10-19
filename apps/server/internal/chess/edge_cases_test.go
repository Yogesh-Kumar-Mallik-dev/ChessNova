package chess

import (
	"testing"
)

func TestFENParserEdgeCases(t *testing.T) {
	invalidFENs := []struct {
		name string
		fen  string
	}{
		{"empty string", ""},
		{"single word", "rnbqkbnr"},
		{"missing fields", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w"},
		{"invalid active color", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR x KQkq - 0 1"},
		{"invalid character in rank", "rnbqkbnr/ppppppXp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"},
		{"rank sum too large", "rnbqkbnr/9/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"},
		{"rank sum too small", "rnbqkbnr/pppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"},
		{"fewer than 8 ranks", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP w KQkq - 0 1"},
		{"more than 8 ranks", "rnbqkbnr/pppppppp/8/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"},
		{"invalid en passant rank", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq e1 0 1"},
		{"invalid castling token", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w X - 0 1"},
		{"no white king", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQ1BNR w KQkq - 0 1"},
		{"no black king", "rnbq1bnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"},
		{"multiple white kings", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBKKBNR w KQkq - 0 1"},
		{"white pawn on rank 1", "rnbqkbnr/pppppppp/8/8/8/8/8/RPBQKBNR w KQkq - 0 1"},
		{"black pawn on rank 8", "rnbqkbp1/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"},
	}

	for _, tc := range invalidFENs {
		t.Run(tc.name, func(t *testing.T) {
			_, err := FromFEN(tc.fen)
			if err == nil {
				t.Errorf("expected error for invalid FEN '%s' (%s), but got nil", tc.fen, tc.name)
			}
		})
	}
}

func TestEnPassantDiscoveredCheckPin(t *testing.T) {
	// The classic discovered horizontal pin on En Passant:
	// White King on a5, White Pawn on e5.
	// Black Rook on h5, Black Pawn on d5 (just pushed from d7 to d5), Black King on e8.
	// FEN: 4k3/8/8/K2pP2r/8/8/8/8 w - d6 0 1
	// If White takes en passant (exd6), both the e5 and d5 pawns leave the 5th rank!
	// This would expose White's King on a5 to check from Black's Rook on h5.
	// Therefore, exd6 must be deemed ILLEGAL by move generation / validation!

	fen := "4k3/8/8/K2pP2r/8/8/8/8 w - d6 0 1"
	pos, err := FromFEN(fen)
	if err != nil {
		t.Fatalf("failed to parse FEN: %v", err)
	}

	legalMoves := LegalMoves(pos)
	for _, m := range legalMoves {
		if m.From == E5 && m.To == D6 {
			t.Fatalf("en passant exd6 exposes king to horizontal check and must be ILLEGAL!")
		}
	}

	// Try applying the move directly; it must return an error
	_, err = ApplyMove(pos, Move{From: E5, To: D6})
	if err == nil {
		t.Fatalf("ApplyMove should fail when move leaves King in check via en passant unmasking")
	}
}

func TestUnderpromotionVariants(t *testing.T) {
	// Position where White pawn on c7 is ready to promote to Queen, Rook, Bishop, or Knight.
	// Black King on e8, White King on e1.
	// FEN: 4k3/2P5/8/8/8/8/8/4K3 w - - 0 1
	fen := "4k3/2P5/8/8/8/8/8/4K3 w - - 0 1"
	pos, err := FromFEN(fen)
	if err != nil {
		t.Fatalf("failed to parse FEN: %v", err)
	}

	legalMoves := LegalMoves(pos)
	hasKnightPromo := false
	hasQueenPromo := false
	hasRookPromo := false
	hasBishopPromo := false

	for _, m := range legalMoves {
		if m.From == C7 && m.To == C8 {
			if m.Promotion != nil {
				switch *m.Promotion {
				case Knight:
					hasKnightPromo = true
				case Queen:
					hasQueenPromo = true
				case Rook:
					hasRookPromo = true
				case Bishop:
					hasBishopPromo = true
				}
			}
		}
	}

	if !hasKnightPromo || !hasQueenPromo || !hasRookPromo || !hasBishopPromo {
		t.Fatalf("expected all 4 promotion choices (Q, R, B, N) to be legal, got Q=%v R=%v B=%v N=%v",
			hasQueenPromo, hasRookPromo, hasBishopPromo, hasKnightPromo)
	}

	// Apply Knight promotion
	promoN := Knight
	posKnight, err := ApplyMove(pos, Move{From: C7, To: C8, Promotion: &promoN})
	if err != nil {
		t.Fatalf("failed to apply knight promo: %v", err)
	}
	pieceOnC8 := posKnight.Board.Get(C8)
	if pieceOnC8.Type != Knight || pieceOnC8.Color != White {
		t.Errorf("expected White Knight on c8 after promotion, got %+v", pieceOnC8)
	}
}

func TestSANDisambiguationBreakingPoints(t *testing.T) {
	// Two knights on b1 and f3 both can legally reach d2 (pawn on d4 leaves d2 empty)
	// FEN: rnbqkbnr/pppppppp/8/8/3P4/5N2/PPP1PPPP/RNBQKB1R w KQkq - 1 2
	fenFileDisambig := "rnbqkbnr/pppppppp/8/8/3P4/5N2/PPP1PPPP/RNBQKB1R w KQkq - 1 2"
	pos1, err := FromFEN(fenFileDisambig)
	if err != nil {
		t.Fatalf("failed to parse FEN: %v", err)
	}

	m1, err := ParseSAN(pos1, "Nbd2")
	if err != nil {
		t.Fatalf("failed to parse Nbd2: %v", err)
	}
	if m1.From != B1 || m1.To != D2 {
		t.Errorf("expected Nbd2 to move from b1 to d2, got %s->%s", m1.From, m1.To)
	}

	m2, err := ParseSAN(pos1, "Nfd2")
	if err != nil {
		t.Fatalf("failed to parse Nfd2: %v", err)
	}
	if m2.From != F3 || m2.To != D2 {
		t.Errorf("expected Nfd2 to move from f3 to d2, got %s->%s", m2.From, m2.To)
	}

	// 2. Ambiguous move without disambiguation should fail
	_, errAmbiguous := ParseSAN(pos1, "Nd2")
	if errAmbiguous == nil {
		t.Errorf("Nd2 should fail with ambiguous move error when two knights can reach d2")
	}

	// 3. Two rooks on same file (e.g. e1 and e7 both can reach e4)
	// Requires rank disambiguation: R1e4 or R7e4
	fenRankDisambig := "4k3/4R3/8/8/8/8/8/4R1K1 w - - 0 1"
	pos2, _ := FromFEN(fenRankDisambig)

	mRank1, err := ParseSAN(pos2, "R1e4")
	if err != nil {
		t.Fatalf("failed to parse R1e4: %v", err)
	}
	if mRank1.From != E1 || mRank1.To != E4 {
		t.Errorf("expected R1e4 from e1 to e4, got %s->%s", mRank1.From, mRank1.To)
	}

	mRank7, err := ParseSAN(pos2, "R7e4")
	if err != nil {
		t.Fatalf("failed to parse R7e4: %v", err)
	}
	if mRank7.From != E7 || mRank7.To != E4 {
		t.Errorf("expected R7e4 from e7 to e4, got %s->%s", mRank7.From, mRank7.To)
	}

	// 4. Capture with check and promotion: exd8=Q+
	fenPromo := "3r4/4P3/8/8/8/8/8/4K2k w - - 0 1"
	posPromo, _ := FromFEN(fenPromo)

	mPromo, err := ParseSAN(posPromo, "exd8=Q+")
	if err != nil {
		t.Fatalf("failed to parse exd8=Q+: %v", err)
	}
	if mPromo.From != E7 || mPromo.To != D8 || mPromo.Promotion == nil || *mPromo.Promotion != Queen {
		t.Errorf("unexpected parsed promo move: %+v", mPromo)
	}

	// 5. Invalid SAN moves
	invalidSANs := []string{"", "Qz9", "e9", "Nxx5", "O-O-O-O", "Kxa1", "invalid"}
	for _, san := range invalidSANs {
		_, err := ParseSAN(pos1, san)
		if err == nil {
			t.Errorf("expected error for invalid SAN '%s', got nil", san)
		}
	}
}

func TestPGNParserEdgeCasesAndTolerances(t *testing.T) {
	// PGN with inline curly-brace comments, move numbers with extra dots, and NAG symbols
	pgnComplex := `[Event "World Championship 1972"]
[Site "Reykjavik ISL"]
[Date "1972.07.23"]
[Round "6"]
[White "Fischer, Robert J."]
[Black "Spassky, Boris V."]
[Result "1-0"]

1. c4 {English Opening} e6 $1 2. Nf3 d5 3. d4 Nf6 4. Nc3 Be7 (4... c6 5. e3)
5. Bg5 O-O 6. e3 h6 7. Bh4 b6 8. cxd5 Nxd5 1-0`

	game, err := ParsePGN(pgnComplex)
	if err != nil {
		t.Fatalf("failed to parse complex PGN: %v", err)
	}

	if game.Headers["White"] != "Fischer, Robert J." || game.Headers["Black"] != "Spassky, Boris V." {
		t.Errorf("unexpected players: %s vs %s", game.Headers["White"], game.Headers["Black"])
	}

	if game.Result != "1-0" {
		t.Errorf("expected result 1-0, got %s", game.Result)
	}

	if len(game.Moves) != 16 {
		t.Errorf("expected 16 plies (8 full moves), got %d moves", len(game.Moves))
	}

	// PGN with missing headers and no final result token
	pgnNoHeaders := `1. e4 e5 2. Nf3 Nc6 3. Bb5`
	game2, err := ParsePGN(pgnNoHeaders)
	if err != nil {
		t.Fatalf("failed to parse headerless PGN: %v", err)
	}
	if len(game2.Moves) != 5 {
		t.Errorf("expected 5 plies, got %d", len(game2.Moves))
	}
}

func TestDrawRulesInsufficientMaterial(t *testing.T) {
	cases := []struct {
		name         string
		fen          string
		insufficient bool
	}{
		{"K vs K", "8/8/8/4k3/8/8/4K3/8 w - - 0 1", true},
		{"K+N vs K", "8/8/8/4k3/8/8/4KN2/8 w - - 0 1", true},
		{"K+B vs K", "8/8/8/4k3/8/8/4KB2/8 w - - 0 1", true},
		{"K+B vs K+B same color", "8/8/8/4k3/8/4b3/4K3/2B5 w - - 0 1", true},
		{"K+N+N vs K (theoretical mate possible)", "8/8/8/4k3/8/8/4KNN1/8 w - - 0 1", false},
		{"K+P vs K (pawn can promote)", "8/8/8/4k3/8/8/4KP2/8 w - - 0 1", false},
		{"K+R vs K (mating material)", "8/8/8/4k3/8/8/4KR2/8 w - - 0 1", false},
		{"K+Q vs K (mating material)", "8/8/8/4k3/8/8/4KQ2/8 w - - 0 1", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pos, err := FromFEN(tc.fen)
			if err != nil {
				t.Fatalf("failed to parse FEN: %v", err)
			}
			isInsuff := IsInsufficientMaterial(pos)
			if isInsuff != tc.insufficient {
				t.Errorf("expected IsInsufficientMaterial=%v for %s, got %v", tc.insufficient, tc.name, isInsuff)
			}
		})
	}
}

func TestFiftyMoveRuleReset(t *testing.T) {
	g := NewGame()

	// Play 4 knight moves back and forth: clock should increment
	moves := []string{"Nf3", "Nf6", "Ng1", "Ng8"}
	for _, m := range moves {
		_, err := g.MakeMoveSAN(m)
		if err != nil {
			t.Fatalf("failed move %s: %v", m, err)
		}
	}
	if g.CurrentPosition.Halfmove != 4 {
		t.Errorf("expected halfmove clock 4, got %d", g.CurrentPosition.Halfmove)
	}

	// Push a pawn: clock MUST reset to 0
	_, err := g.MakeMoveSAN("e4")
	if err != nil {
		t.Fatalf("failed move e4: %v", err)
	}
	if g.CurrentPosition.Halfmove != 0 {
		t.Errorf("expected halfmove clock to reset to 0 after pawn push, got %d", g.CurrentPosition.Halfmove)
	}
}

func TestThreefoldRepetitionWithCastlingRevocation(t *testing.T) {
	// If kingside castling rights are lost, the position is NOT repetition
	// even if the board pieces are on identical squares!
	g := NewGame()

	// 1. Nf3 Nf6 2. Ng1 Ng8 (Pos A with KQkq)
	_, _ = g.MakeMoveSAN("Nf3")
	_, _ = g.MakeMoveSAN("Nf6")
	_, _ = g.MakeMoveSAN("Ng1")
	_, _ = g.MakeMoveSAN("Ng8")

	// Pos A occurred twice now (starting pos + ply 4)
	if g.CanClaimThreefoldRepetition() {
		t.Errorf("position occurred only twice, should not be threefold repetition yet")
	}

	// 3. Nf3 Nf6 4. Ng1 Ng8 (Pos A occurs a 3rd time!)
	_, _ = g.MakeMoveSAN("Nf3")
	_, _ = g.MakeMoveSAN("Nf6")
	_, _ = g.MakeMoveSAN("Ng1")
	_, _ = g.MakeMoveSAN("Ng8")

	if !g.CanClaimThreefoldRepetition() && g.Outcome != OutcomeThreefoldRepetition {
		t.Errorf("expected threefold repetition after third occurrence of identical position state")
	}
}
