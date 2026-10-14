package chess

import (
	"testing"
)

// Perft computes the number of leaf nodes at given depth
func Perft(pos Position, depth int) uint64 {
	if depth <= 0 {
		return 1
	}

	moves := LegalMoves(pos)
	if depth == 1 {
		return uint64(len(moves))
	}

	var nodes uint64
	for _, m := range moves {
		nextPos, err := ApplyMoveWithoutTurnValidation(pos, m)
		if err != nil {
			continue
		}
		nodes += Perft(nextPos, depth-1)
	}
	return nodes
}

func TestPerftInitialPosition(t *testing.T) {
	pos := InitialPosition()

	expected := map[int]uint64{
		1: 20,
		2: 400,
		3: 8902,
		4: 197281,
	}

	for depth, want := range expected {
		got := Perft(pos, depth)
		if got != want {
			t.Errorf("Perft(InitialPosition, depth %d) = %d; want %d", depth, got, want)
		}
	}
}

func TestPerftKiwipete(t *testing.T) {
	// Position 2: Kiwipete - tests en passant, castling, pins, promotions
	fen := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	pos, err := FromFEN(fen)
	if err != nil {
		t.Fatalf("failed to parse Kiwipete FEN: %v", err)
	}

	expected := map[int]uint64{
		1: 48,
		2: 2039,
		3: 97862,
	}

	for depth, want := range expected {
		got := Perft(pos, depth)
		if got != want {
			t.Errorf("Perft(Kiwipete, depth %d) = %d; want %d", depth, got, want)
		}
	}
}

func TestPerftPosition3(t *testing.T) {
	fen := "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1"
	pos, err := FromFEN(fen)
	if err != nil {
		t.Fatalf("failed to parse FEN: %v", err)
	}

	expected := map[int]uint64{
		1: 14,
		2: 191,
		3: 2812,
		4: 43238,
	}

	for depth, want := range expected {
		got := Perft(pos, depth)
		if got != want {
			t.Errorf("Perft(Pos3, depth %d) = %d; want %d", depth, got, want)
		}
	}
}

func TestPerftPosition4(t *testing.T) {
	fen := "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1"
	pos, err := FromFEN(fen)
	if err != nil {
		t.Fatalf("failed to parse FEN: %v", err)
	}

	expected := map[int]uint64{
		1: 6,
		2: 264,
		3: 9467,
	}

	for depth, want := range expected {
		got := Perft(pos, depth)
		if got != want {
			t.Errorf("Perft(Pos4, depth %d) = %d; want %d", depth, got, want)
		}
	}
}

func TestPerftPosition5(t *testing.T) {
	fen := "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8"
	pos, err := FromFEN(fen)
	if err != nil {
		t.Fatalf("failed to parse FEN: %v", err)
	}

	expected := map[int]uint64{
		1: 44,
		2: 1486,
		3: 62379,
	}

	for depth, want := range expected {
		got := Perft(pos, depth)
		if got != want {
			t.Errorf("Perft(Pos5, depth %d) = %d; want %d", depth, got, want)
		}
	}
}
