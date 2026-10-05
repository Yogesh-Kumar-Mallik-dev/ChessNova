package review

import (
	"context"
	"testing"
	"time"

	"chess-platform/server/internal/engine"
)

func TestGameReviewService(t *testing.T) {
	eng := engine.NewStockfishEngine()
	srv := NewReviewService(eng)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	moves := []InputMove{
		{From: "e2", To: "e4", SAN: "e4"},
		{From: "e7", To: "e5", SAN: "e5"},
		{From: "g1", To: "f3", SAN: "Nf3"},
		{From: "b8", To: "c6", SAN: "Nc6"},
	}

	result, err := srv.AnalyzeMoves(ctx, moves)
	if err != nil {
		t.Fatalf("AnalyzeMoves failed: %v", err)
	}

	if len(result.Moves) != 4 {
		t.Errorf("expected 4 reviewed moves, got %d", len(result.Moves))
	}

	if result.WhiteAccuracy <= 0 || result.BlackAccuracy <= 0 {
		t.Errorf("expected positive accuracies, got white: %f, black: %f", result.WhiteAccuracy, result.BlackAccuracy)
	}

	if result.Opening == nil {
		t.Errorf("expected opening to be identified, got nil")
	} else if result.Opening.ECO != "C44" && result.Opening.ECO != "C40" {
		t.Logf("Identified opening: %+v", result.Opening)
	}

	if len(result.EvalGraph) != 5 { // starting pos + 4 moves
		t.Errorf("expected 5 eval graph points, got %d", len(result.EvalGraph))
	}
}
