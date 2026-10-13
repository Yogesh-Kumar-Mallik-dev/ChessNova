package puzzle

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestPuzzleToPublic(t *testing.T) {
	oid := primitive.NewObjectID()
	p := Puzzle{
		ID:       oid,
		FEN:      "6k1/5ppp/8/8/8/8/5PPP/4R1K1 w - - 0 1",
		Solution: []string{"e1e8"},
		Rating:   1000,
		Themes:   []string{"mateIn1", "backRankMate"},
	}

	pub := p.ToPublic()
	if pub.ID != oid {
		t.Errorf("expected ID %v, got %v", oid, pub.ID)
	}
	if pub.FEN != p.FEN {
		t.Errorf("expected FEN %s, got %s", p.FEN, pub.FEN)
	}
	if pub.Rating != 1000 {
		t.Errorf("expected rating 1000, got %d", pub.Rating)
	}
	if pub.TotalMoves != 1 {
		t.Errorf("expected total moves 1, got %d", pub.TotalMoves)
	}
}

func TestCuratedLichessPuzzlesIntegrity(t *testing.T) {
	if len(CuratedLichessPuzzles) < 100 {
		t.Fatalf("expected at least 100 curated lichess puzzles, got %d", len(CuratedLichessPuzzles))
	}

	for i, p := range CuratedLichessPuzzles {
		if p.LichessID == "" {
			t.Errorf("puzzle #%d has empty LichessID", i)
		}
		if p.FEN == "" {
			t.Errorf("puzzle %s has empty FEN", p.LichessID)
		}
		if len(p.Solution) == 0 {
			t.Errorf("puzzle %s has no solution moves", p.LichessID)
		}
		if p.Rating < 400 {
			t.Errorf("puzzle %s has unusually low rating %d", p.LichessID, p.Rating)
		}
		if len(p.Themes) == 0 {
			t.Errorf("puzzle %s has no tactical themes", p.LichessID)
		}
	}
}

func TestDetermineCategory(t *testing.T) {
	tests := []struct {
		themes   []string
		expected string
	}{
		{[]string{"mate", "mateIn1", "kingsideAttack"}, "mate"},
		{[]string{"endgame", "fork", "short"}, "fork"},
		{[]string{"pin", "middlegame"}, "pin"},
		{[]string{"skewer", "endgame"}, "skewer"},
		{[]string{"hangingPiece", "advantage"}, "capture"},
		{[]string{"crushing", "quietMove"}, "position"},
		{[]string{"zugzwang", "endgame"}, "position"},
	}

	for _, tt := range tests {
		got := DetermineCategory(tt.themes)
		if got != tt.expected {
			t.Errorf("DetermineCategory(%v) = %s; want %s", tt.themes, got, tt.expected)
		}
	}
}

func TestDetermineTier(t *testing.T) {
	if DetermineTier(600) != 1 {
		t.Errorf("expected 600 to be Tier 1")
	}
	if DetermineTier(1150) != 2 {
		t.Errorf("expected 1150 to be Tier 2")
	}
	if DetermineTier(1450) != 3 {
		t.Errorf("expected 1450 to be Tier 3")
	}
	if DetermineTier(1850) != 4 {
		t.Errorf("expected 1850 to be Tier 4")
	}
	if DetermineTier(2400) != 5 {
		t.Errorf("expected 2400 to be Tier 5")
	}
}
