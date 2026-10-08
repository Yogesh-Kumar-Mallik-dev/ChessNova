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
