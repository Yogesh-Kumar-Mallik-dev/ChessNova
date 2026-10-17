package rating

import (
	"testing"
)

func TestRatingConstants(t *testing.T) {
	if DefaultRating != 400 {
		t.Fatalf("expected DefaultRating to be 400, got %d", DefaultRating)
	}
	if MinRating != 100 {
		t.Fatalf("expected MinRating to be 100, got %d", MinRating)
	}
}

func TestCalculateElo_EqualRatings(t *testing.T) {
	// Two default 400 rated players, White wins
	newA, newB, changeA, changeB := CalculateElo(400, 400, 1.0, DefaultKFactor)

	if newA != 416 {
		t.Fatalf("expected winner new rating 416, got %d", newA)
	}
	if changeA != 16 {
		t.Fatalf("expected winner change +16, got %d", changeA)
	}
	if newB != 384 {
		t.Fatalf("expected loser new rating 384, got %d", newB)
	}
	if changeB != -16 {
		t.Fatalf("expected loser change -16, got %d", changeB)
	}
}

func TestCalculateElo_FloorClamping(t *testing.T) {
	// Player B is at 105, loses against 400
	newA, newB, changeA, changeB := CalculateElo(400, 105, 1.0, DefaultKFactor)

	if newB < MinRating {
		t.Fatalf("expected loser rating to not drop below %d, got %d", MinRating, newB)
	}
	if newB != 100 {
		t.Fatalf("expected loser rating clamped to %d, got %d", MinRating, newB)
	}
	if changeB != -5 {
		t.Fatalf("expected loser change clamped to -5, got %d", changeB)
	}
	if newA <= 400 {
		t.Fatalf("expected winner rating to increase, got %d", newA)
	}
	if changeA <= 0 {
		t.Fatalf("expected winner change to be positive, got %d", changeA)
	}
}

func TestCalculateElo_AlreadyAtFloor(t *testing.T) {
	// Player B is already at MinRating (100) and loses
	_, newB, _, changeB := CalculateElo(400, 100, 1.0, DefaultKFactor)

	if newB != MinRating {
		t.Fatalf("expected loser rating to remain at %d, got %d", MinRating, newB)
	}
	if changeB != 0 {
		t.Fatalf("expected loser change to be 0 at floor, got %d", changeB)
	}
}

func TestCalculateElo_SubFloorInput(t *testing.T) {
	// Input rating below MinRating is automatically clamped up to MinRating
	_, newB, _, _ := CalculateElo(400, 50, 1.0, DefaultKFactor)

	if newB < MinRating {
		t.Fatalf("expected loser rating >= %d, got %d", MinRating, newB)
	}
}

func TestDetermineCategory(t *testing.T) {
	tests := []struct {
		initialSeconds int
		expected       Category
	}{
		{60, Bullet},
		{120, Bullet},
		{180, Blitz},
		{300, Blitz},
		{600, Rapid},
		{900, Rapid},
		{1800, Classical},
	}

	for _, tc := range tests {
		got := DetermineCategory(tc.initialSeconds)
		if got != tc.expected {
			t.Errorf("DetermineCategory(%d) = %s, expected %s", tc.initialSeconds, got, tc.expected)
		}
	}
}

func TestCalculateElo_AllCategoriesAndScenarios(t *testing.T) {
	categories := []Category{Bullet, Blitz, Rapid, Classical, Puzzles}
	for _, cat := range categories {
		if string(cat) == "" {
			t.Errorf("empty category found")
		}
	}

	// 1. Higher rated vs lower rated: Underdog wins (+delta should be large)
	newA, newB, diffA, diffB := CalculateElo(1200, 1600, 1.0, DefaultKFactor)
	if diffA <= 16 {
		t.Errorf("underdog win should award significant Elo (>16), got %d", diffA)
	}
	if diffB >= -16 {
		t.Errorf("favorite loss should deduct significant Elo (<-16), got %d", diffB)
	}
	if newA != 1200+diffA || newB != 1600+diffB {
		t.Errorf("mismatch in rating updates: newA=%d, newB=%d", newA, newB)
	}

	// 2. Draw between equal ratings
	newA, newB, diffA, diffB = CalculateElo(1500, 1500, 0.5, DefaultKFactor)
	if diffA != 0 || diffB != 0 {
		t.Errorf("draw between equal ratings should produce 0 diff, got diffA=%d, diffB=%d", diffA, diffB)
	}
	if newA != 1500 || newB != 1500 {
		t.Errorf("draw between equal ratings should preserve ratings, got newA=%d, newB=%d", newA, newB)
	}

	// 3. Draw between favorite and underdog: underdog gains slightly, favorite loses slightly
	newA, newB, diffA, diffB = CalculateElo(1200, 1600, 0.5, DefaultKFactor)
	if diffA <= 0 {
		t.Errorf("underdog drawing against much higher rated opponent should gain rating, got %d", diffA)
	}
	if diffB >= 0 {
		t.Errorf("favorite drawing against much lower rated opponent should lose rating, got %d", diffB)
	}

	// 4. Puzzle solve calculation (single player score against puzzle rating)
	playerRating := 1000
	puzzleRating := 1200
	// Solving correctly
	newPlayer, _, gain, _ := CalculateElo(playerRating, puzzleRating, 1.0, DefaultKFactor)
	if gain <= 16 {
		t.Errorf("solving higher rated puzzle should award >16 rating, got %d", gain)
	}
	if newPlayer != playerRating+gain {
		t.Errorf("new rating %d != %d + %d", newPlayer, playerRating, gain)
	}

	// Failing puzzle
	newPlayerFail, _, loss, _ := CalculateElo(playerRating, puzzleRating, 0.0, DefaultKFactor)
	if loss >= 0 {
		t.Errorf("failing puzzle should decrease rating, got %d", loss)
	}
	if newPlayerFail != playerRating+loss {
		t.Errorf("new rating %d != %d + %d", newPlayerFail, playerRating, loss)
	}
}

