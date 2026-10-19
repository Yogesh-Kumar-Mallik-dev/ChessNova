package rating

import (
	"math"
	"testing"
)

func TestRatingExtremeDisparities(t *testing.T) {
	// Grandmaster (2800) vs Beginner (400)
	gmRating := 2800
	beginnerRating := 400
	k := DefaultKFactor // 32

	t.Run("Expected GM Win", func(t *testing.T) {
		newGM, newBeg, deltaGM, deltaBeg := CalculateElo(gmRating, beginnerRating, 1.0, k)
		// Expected outcome is practically 1.0, so GM gains at most 1 point, Beginner loses at most 1 point
		if deltaGM < 0 || deltaGM > 1 {
			t.Errorf("GM delta should be 0 or 1, got %d", deltaGM)
		}
		if deltaBeg > 0 || deltaBeg < -1 {
			t.Errorf("Beginner delta should be 0 or -1, got %d", deltaBeg)
		}
		if newGM < gmRating || newGM > gmRating+1 {
			t.Errorf("unexpected GM new rating %d", newGM)
		}
		if newBeg > beginnerRating || newBeg < beginnerRating-1 {
			t.Errorf("unexpected Beginner new rating %d", newBeg)
		}
	})

	t.Run("Shocking Beginner Upset Win", func(t *testing.T) {
		// Beginner beats GM! (score = 0.0 for GM, 1.0 for Beginner)
		_, _, deltaGM, deltaBeg := CalculateElo(gmRating, beginnerRating, 0.0, k)
		// GM should lose almost full K points (-32), Beginner should gain almost full K points (+32)
		if deltaGM > -30 {
			t.Errorf("expected GM to lose ~32 points on huge upset, got %d", deltaGM)
		}
		if deltaBeg < 30 {
			t.Errorf("expected Beginner to gain ~32 points on huge upset, got %d", deltaBeg)
		}
		if math.Abs(float64(deltaGM+deltaBeg)) > 1 {
			t.Errorf("zero-sum delta violated beyond rounding: deltaGM=%d, deltaBeg=%d", deltaGM, deltaBeg)
		}
	})

	t.Run("Shocking Draw", func(t *testing.T) {
		// Draw (score = 0.5)
		_, _, deltaGM, deltaBeg := CalculateElo(gmRating, beginnerRating, 0.5, k)
		// Expected score for GM was ~1.0, so 0.5 - 1.0 = -0.5 -> deltaGM ~ -16
		// Expected score for Beginner was ~0.0, so 0.5 - 0.0 = +0.5 -> deltaBeg ~ +16
		if deltaGM > -14 || deltaGM < -18 {
			t.Errorf("expected GM to lose ~16 points on draw, got %d", deltaGM)
		}
		if deltaBeg < 14 || deltaBeg > 18 {
			t.Errorf("expected Beginner to gain ~16 points on draw, got %d", deltaBeg)
		}
	})
}

func TestRatingSymmetryAndZeroSum(t *testing.T) {
	ratings := [][2]int{
		{1200, 1200},
		{1500, 1600},
		{2100, 1950},
		{800, 1400},
		{2400, 2400},
	}

	for _, pair := range ratings {
		rA, rB := pair[0], pair[1]
		k := 32

		// Outcome 1: A wins vs B
		_, _, deltaA1, deltaB1 := CalculateElo(rA, rB, 1.0, k)
		// Outcome 2: B wins vs A
		_, _, deltaA2, deltaB2 := CalculateElo(rA, rB, 0.0, k)

		// Invariant 1: In any match (above floor), deltaA + deltaB == 0
		if math.Abs(float64(deltaA1+deltaB1)) > 1 {
			t.Errorf("zero-sum violated in match 1: deltaA=%d, deltaB=%d", deltaA1, deltaB1)
		}
		if math.Abs(float64(deltaA2+deltaB2)) > 1 {
			t.Errorf("zero-sum violated in match 2: deltaA=%d, deltaB=%d", deltaA2, deltaB2)
		}

		// Invariant 2: Complementary win gain equals K
		// deltaA1 (gain if A wins) + deltaB2 (gain if B wins) must equal K (+/- rounding)
		if math.Abs(float64((deltaA1+deltaB2)-k)) > 1 {
			t.Errorf("complementary win gain sum should equal K (%d), got %d (%d + %d)",
				k, deltaA1+deltaB2, deltaA1, deltaB2)
		}
	}
}

func TestRatingVariableKFactors(t *testing.T) {
	rA, rB := 1500, 1500

	// K <= 0 falls back to DefaultKFactor (32)
	_, _, dA0, dB0 := CalculateElo(rA, rB, 1.0, 0)
	if dA0 != 16 || dB0 != -16 {
		t.Errorf("K<=0 should fall back to DefaultKFactor (32), giving +/-16, got dA=%d, dB=%d", dA0, dB0)
	}

	// K = 10 (FIDE Masters / Super GMs)
	_, _, dA10, dB10 := CalculateElo(rA, rB, 1.0, 10)
	if dA10 != 5 || dB10 != -5 {
		t.Errorf("K=10 with equal ratings should give +/-5, got %d, %d", dA10, dB10)
	}

	// K = 40 (Provisional junior player)
	_, _, dA40, dB40 := CalculateElo(rA, rB, 1.0, 40)
	if dA40 != 20 || dB40 != -20 {
		t.Errorf("K=40 with equal ratings should give +/-20, got %d, %d", dA40, dB40)
	}
}

func TestRatingSequentialStreakDrift(t *testing.T) {
	// Simulate a 50-game win streak against equal opponents
	currentRating := 1000
	k := 32

	for i := 0; i < 50; i++ {
		opponent := currentRating
		newR, _, delta, _ := CalculateElo(currentRating, opponent, 1.0, k)
		if delta != 16 {
			t.Fatalf("at game %d, expected delta 16 against equal opponent, got %d", i, delta)
		}
		if newR != currentRating+16 {
			t.Fatalf("at game %d, rating calculation mismatch", i)
		}
		currentRating = newR
	}

	expectedFinal := 1000 + (50 * 16)
	if currentRating != expectedFinal {
		t.Fatalf("expected final rating %d, got %d", expectedFinal, currentRating)
	}
}
