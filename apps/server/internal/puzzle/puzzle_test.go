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
