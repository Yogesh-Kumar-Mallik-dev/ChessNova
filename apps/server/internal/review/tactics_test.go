package review

import (
	"math"
	"strings"
	"testing"

	"chess-platform/server/internal/chess"
)

func TestDetectFork(t *testing.T) {
	// 1. Knight Royal Fork (King and Queen)
	// White Knight on e7 attacks Black King on c8 and Black Queen on g8
	// FEN: 2k3q1/8/8/8/8/8/4N3/4K3 w - - 0 1
	b1 := chess.Board{}
	b1.Set(chess.C8, chess.Piece{Type: chess.King, Color: chess.Black})
	b1.Set(chess.G8, chess.Piece{Type: chess.Queen, Color: chess.Black})
	b1.Set(chess.E7, chess.Piece{Type: chess.Knight, Color: chess.White})

	isFork, desc := detectFork(b1, chess.E7, chess.White)
	if !isFork {
		t.Fatalf("expected knight royal fork to be detected")
	}
	if !strings.Contains(desc, "King") || !strings.Contains(desc, "queen") {
		t.Errorf("expected fork description to mention King and queen, got: %s", desc)
	}

	// 2. Pawn Fork attacking two minor pieces
	// White pawn on d4 attacks Black Knight on c5 and Black Bishop on e5
	b2 := chess.Board{}
	b2.Set(chess.C5, chess.Piece{Type: chess.Knight, Color: chess.Black})
	b2.Set(chess.E5, chess.Piece{Type: chess.Bishop, Color: chess.Black})
	b2.Set(chess.D4, chess.Piece{Type: chess.Pawn, Color: chess.White})

	isFork2, desc2 := detectFork(b2, chess.D4, chess.White)
	if !isFork2 {
		t.Fatalf("expected pawn fork to be detected")
	}
	if !strings.Contains(desc2, "knight") || !strings.Contains(desc2, "bishop") {
		t.Errorf("expected pawn fork to mention knight and bishop, got: %s", desc2)
	}

	// 3. Blocked ray should NOT be a fork
	// White Rook on d1, enemy Rook on d4, enemy Queen on d7.
	// Rook on d1 only directly attacks d4; d7 is blocked by d4.
	b3 := chess.Board{}
	b3.Set(chess.D1, chess.Piece{Type: chess.Rook, Color: chess.White})
	b3.Set(chess.D4, chess.Piece{Type: chess.Rook, Color: chess.Black})
	b3.Set(chess.D7, chess.Piece{Type: chess.Queen, Color: chess.Black})

	isFork3, _ := detectFork(b3, chess.D1, chess.White)
	if isFork3 {
		t.Errorf("rook ray is blocked by first piece; should not be a multi-piece fork")
	}

	// 4. Friendly piece cannot be forked
	b4 := chess.Board{}
	b4.Set(chess.D1, chess.Piece{Type: chess.Rook, Color: chess.White})
	b4.Set(chess.D4, chess.Piece{Type: chess.Rook, Color: chess.White})
	b4.Set(chess.B1, chess.Piece{Type: chess.Knight, Color: chess.White})

	isFork4, _ := detectFork(b4, chess.D1, chess.White)
	if isFork4 {
		t.Errorf("friendly pieces cannot be forked")
	}
}

func TestIsDirectlyAttackedBy(t *testing.T) {
	b := chess.Board{}
	// White Queen on d4
	b.Set(chess.D4, chess.Piece{Type: chess.Queen, Color: chess.White})

	// Test orthogonal attack unblocked
	if !isDirectlyAttackedBy(b, chess.D4, chess.D8) {
		t.Errorf("expected d4 Queen to directly attack d8")
	}

	// Block with piece on d6
	b.Set(chess.D6, chess.Piece{Type: chess.Pawn, Color: chess.Black})
	if isDirectlyAttackedBy(b, chess.D4, chess.D8) {
		t.Errorf("d4 Queen ray to d8 should be blocked by pawn on d6")
	}
	if !isDirectlyAttackedBy(b, chess.D4, chess.D6) {
		t.Errorf("d4 Queen should directly attack blocker on d6")
	}

	// Test diagonal attack
	if !isDirectlyAttackedBy(b, chess.D4, chess.G7) {
		t.Errorf("expected d4 Queen to directly attack g7")
	}

	// Non-ray square
	if isDirectlyAttackedBy(b, chess.D4, chess.E6) {
		t.Errorf("d4 Queen cannot attack e6 (knight offset)")
	}

	// Same square should return false
	if isDirectlyAttackedBy(b, chess.D4, chess.D4) {
		t.Errorf("square cannot attack itself")
	}
}

func TestDetectDiscoveredAndDoubleCheck(t *testing.T) {
	// Setup position for discovered check:
	// White King on g1, White Rook on e1, White Knight on e4, Black King on e8
	// When Knight moves to d6 or g5, Rook on e1 delivers check to e8!
	pos, err := chess.FromFEN("4k3/8/8/8/4N3/8/8/4R1K1 w - - 0 1")
	if err != nil {
		t.Fatalf("failed to parse FEN: %v", err)
	}

	// Move Knight to g5 (not delivering check itself, unmasking e1 rook)
	posAfter, err := chess.ApplyMove(pos, chess.Move{From: chess.E4, To: chess.G5})
	if err != nil {
		t.Fatalf("failed to apply move: %v", err)
	}

	isDisc, isDbl := detectDiscoveredCheck(posAfter, chess.G5, chess.Black)
	if !isDisc {
		t.Errorf("expected discovered check to be true")
	}
	if isDbl {
		t.Errorf("did not expect double check")
	}

	// Setup position for double check:
	// White Bishop on d5, White Rook on e1, Black King on e8.
	// Bishop moves to c6+ (delivering direct bishop check while unmasking e1 rook!)
	pos2, err := chess.FromFEN("4k3/8/8/3B4/8/8/8/4R1K1 w - - 0 1")
	if err != nil {
		t.Fatalf("failed to parse FEN: %v", err)
	}

	posAfter2, err := chess.ApplyMove(pos2, chess.Move{From: chess.D5, To: chess.C6})
	if err != nil {
		t.Fatalf("failed to apply move: %v", err)
	}

	isDisc2, isDbl2 := detectDiscoveredCheck(posAfter2, chess.C6, chess.Black)
	if !isDbl2 {
		t.Errorf("expected double check to be true, got disc=%v dbl=%v", isDisc2, isDbl2)
	}

	// Position with no check
	posNoCheck, _ := chess.FromFEN(chess.StartingFEN)
	isDisc3, isDbl3 := detectDiscoveredCheck(posNoCheck, chess.E4, chess.Black)
	if isDisc3 || isDbl3 {
		t.Errorf("expected no discovered check in starting position")
	}
}

func TestDetectHangingBlunder(t *testing.T) {
	// White moves Queen on h5 to e5 where Black pawn on d6 attacks it and White has NO defenders!
	posBefore, err := chess.FromFEN("rnb1kbnr/ppp1pppp/3p4/7Q/8/8/PPPPPPPP/RNB1KBNR w KQkq - 0 1")
	if err != nil {
		t.Fatalf("failed to parse FEN: %v", err)
	}

	// Move Queen from h5 to e5
	m := chess.Move{From: chess.H5, To: chess.E5}
	posAfter, err := chess.ApplyMove(posBefore, m)
	if err != nil {
		t.Fatalf("failed to apply move: %v", err)
	}

	isHanging, pieceName := detectHangingBlunder(posBefore, posAfter, m, chess.White)
	if !isHanging {
		t.Errorf("expected Queen on e5 attacked by d6 pawn to be detected as hanging blunder")
	}
	if pieceName != "queen" {
		t.Errorf("expected hanging piece to be queen, got %s", pieceName)
	}

	// Defended piece moved: White Knight to f3 in starting position (defended by d2 pawn/Q/K)
	startPos := chess.InitialPosition()
	nf3Move := chess.Move{From: chess.G1, To: chess.F3}
	posAfterNf3, _ := chess.ApplyMove(startPos, nf3Move)
	isHanging2, _ := detectHangingBlunder(startPos, posAfterNf3, nf3Move, chess.White)
	if isHanging2 {
		t.Errorf("Nf3 in opening should not be flagged as hanging")
	}
}

func TestIsQueenTrade(t *testing.T) {
	// Position with both queens
	posWithQueens, _ := chess.FromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	// Position after queens traded
	posNoQueens, _ := chess.FromFEN("rnb1kbnr/pppppppp/8/8/8/8/PPPPPPPP/RNB1KBNR w KQkq - 0 1")

	if !isQueenTrade(posWithQueens, posNoQueens) {
		t.Errorf("expected isQueenTrade to be true when queens disappeared")
	}

	// Position with only black queen missing (not a trade, just white took black queen)
	posOneQueen, _ := chess.FromFEN("rnb1kbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if isQueenTrade(posWithQueens, posOneQueen) {
		t.Errorf("one-sided queen loss is not a queen trade")
	}
}

func TestClassifyAndExplain(t *testing.T) {
	posBefore, _ := chess.FromFEN(chess.StartingFEN)
	posAfter, _ := chess.ApplyMove(posBefore, chess.Move{From: chess.E2, To: chess.E4})

	// 1. Forced move test
	class, exp := ClassifyAndExplain(
		posBefore, posAfter,
		chess.Move{From: chess.E2, To: chess.E4},
		"e4", true,
		30, 35, 52.8, 53.5,
		"e4", "e2e4",
		true, false, "",
	)
	if class != ClassForced || !strings.Contains(exp, "Forced move") {
		t.Errorf("expected ClassForced, got %s (%s)", class, exp)
	}

	// 2. Book move test
	classBook, expBook := ClassifyAndExplain(
		posBefore, posAfter,
		chess.Move{From: chess.E2, To: chess.E4},
		"e4", true,
		30, 35, 52.8, 53.5,
		"e4", "e2e4",
		false, true, "Ruy Lopez",
	)
	if classBook != ClassBook || !strings.Contains(expBook, "Ruy Lopez") {
		t.Errorf("expected ClassBook with opening name, got %s (%s)", classBook, expBook)
	}

	// 3. Missed win test
	classMiss, expMiss := ClassifyAndExplain(
		posBefore, posAfter,
		chess.Move{From: chess.E2, To: chess.E4},
		"e4", true,
		9500, 0, 75.0, 45.0, // Had decisive win (+9500 or 75%), dropped to 45%
		"Qh7#", "h5h7",
		false, false, "",
	)
	if classMiss != ClassMiss || !strings.Contains(expMiss, "Missed win") {
		t.Errorf("expected ClassMiss, got %s (%s)", classMiss, expMiss)
	}

	// 4. Blunder test
	classBlunder, expBlunder := ClassifyAndExplain(
		posBefore, posAfter,
		chess.Move{From: chess.E2, To: chess.E4},
		"e4", true,
		0, -500, 50.0, 25.0, // Dropped 25% win rate
		"Nf3", "g1f3",
		false, false, "",
	)
	if classBlunder != ClassBlunder || !strings.Contains(expBlunder, "Blunder") {
		t.Errorf("expected ClassBlunder, got %s (%s)", classBlunder, expBlunder)
	}

	// 5. Best move test
	classBest, expBest := ClassifyAndExplain(
		posBefore, posAfter,
		chess.Move{From: chess.E2, To: chess.E4},
		"e4", true,
		30, 35, 52.8, 53.5,
		"e4", "e2e4",
		false, false, "",
	)
	if classBest != ClassBest || !strings.Contains(expBest, "Best move") {
		t.Errorf("expected ClassBest, got %s (%s)", classBest, expBest)
	}
}

func TestCentipawnsToWinChanceBounds(t *testing.T) {
	// Extreme positive centipawns should clamp near 100%
	highWin := CentipawnsToWinChance(10000)
	if highWin < 95.0 || highWin > 100.0 {
		t.Errorf("expected high win chance close to 100%%, got %f", highWin)
	}

	// Extreme negative centipawns should clamp near 0%
	lowWin := CentipawnsToWinChance(-10000)
	if lowWin > 5.0 || lowWin < 0.0 {
		t.Errorf("expected low win chance close to 0%%, got %f", lowWin)
	}

	// 0 centipawns should be exactly 50%
	equalWin := CentipawnsToWinChance(0)
	if math.Abs(equalWin-50.0) > 0.01 {
		t.Errorf("expected 50%% for 0 centipawns, got %f", equalWin)
	}
}

func TestEstimatePlayerPerformanceElo(t *testing.T) {
	// 100% accuracy with 0 mistakes/blunders should yield Grandmaster rating (> 2500)
	gmElo := EstimatePlayerPerformanceElo(100.0, 0, 0, 0, 40)
	if gmElo < 2500 || gmElo > 2850 {
		t.Errorf("expected GM Elo between 2500 and 2850, got %d", gmElo)
	}

	// Horrible play with 5 blunders and low accuracy should floor at 400
	floorElo := EstimatePlayerPerformanceElo(20.0, 5, 5, 5, 20)
	if floorElo < 400 {
		t.Errorf("expected Elo floor of 400, got %d", floorElo)
	}
}
