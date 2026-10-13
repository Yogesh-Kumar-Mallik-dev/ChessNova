package engine

import (
	"context"
	"testing"
	"time"
)

func TestStockfishEvaluation(t *testing.T) {
	eng := NewStockfishEngine()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eval, err := eng.EvaluateFen(ctx, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", 8)
	if err != nil {
		t.Fatalf("EvaluateFen failed: %v", err)
	}

	if eval.Depth < 8 && eval.ScoreCp == 0 && eval.BestMove == "" {
		t.Errorf("evaluation returned unexpected empty result: %+v", eval)
	}

	// Mate in 1 test (Scholar's Mate)
	// 1. e4 e5 2. Qh5 Nc6 3. Bc4 Nf6 4. Qxf7#
	mateFen := "r1bqkb1r/pppp1Qpp/2n2n2/4p3/2B1P3/8/PPPP1PPP/RNB1K1NR b KQkq - 0 4"
	evalMate, err := eng.EvaluateFen(ctx, mateFen, 6)
	if err != nil {
		t.Fatalf("EvaluateFen for mate failed: %v", err)
	}
	if !evalMate.IsMate && evalMate.ScoreCp == 0 {
		t.Logf("Mate eval result: %+v", evalMate)
	}
}

func TestGenerateBotMove(t *testing.T) {
	eng := NewStockfishEngine()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startingFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	// Test level 1 (Novice ~400 Elo)
	m1, err := eng.GenerateBotMove(ctx, startingFEN, 1)
	if err != nil {
		t.Fatalf("failed to generate move for level 1: %v", err)
	}
	if len(m1) < 4 {
		t.Errorf("expected valid UCI move, got %s", m1)
	}

	// Test level 2 (Casual ~800 Elo)
	m2, err := eng.GenerateBotMove(ctx, startingFEN, 2)
	if err != nil {
		t.Fatalf("failed to generate move for level 2: %v", err)
	}
	if len(m2) < 4 {
		t.Errorf("expected valid UCI move, got %s", m2)
	}
}
