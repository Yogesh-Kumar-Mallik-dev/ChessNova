package openings

import (
	"testing"
)

func TestIdentifyOpening(t *testing.T) {
	tests := []struct {
		moves    []string
		wantECO  string
		wantName string
		wantPly  int
	}{
		{
			moves:    []string{"e4", "e5", "Nf3", "Nc6", "Bc4"},
			wantECO:  "C50",
			wantName: "Italian Game",
			wantPly:  5,
		},
		{
			moves:    []string{"e4", "c5", "Nf3", "d6", "d4", "cxd4", "Nxd4", "Nf6", "Nc3", "a6"},
			wantECO:  "B90",
			wantName: "Sicilian Defense",
			wantPly:  10,
		},
		{
			moves:    []string{"d4", "d5", "c4"},
			wantECO:  "D06",
			wantName: "Queen's Gambit",
			wantPly:  3,
		},
		{
			moves:    []string{"a4"},
			wantECO:  "",
			wantName: "",
			wantPly:  0,
		},
	}

	for _, tc := range tests {
		op, ply := IdentifyOpening(tc.moves)
		if tc.wantECO == "" {
			if op != nil {
				t.Errorf("expected nil opening for %v, got %v", tc.moves, op)
			}
		} else {
			if op == nil {
				t.Fatalf("expected opening for %v, got nil", tc.moves)
			}
			if op.ECO != tc.wantECO {
				t.Errorf("expected ECO %s, got %s", tc.wantECO, op.ECO)
			}
			if op.Name != tc.wantName {
				t.Errorf("expected Name %s, got %s", tc.wantName, op.Name)
			}
			if ply != tc.wantPly {
				t.Errorf("expected ply %d, got %d", tc.wantPly, ply)
			}
		}
	}
}

func TestIsBookMove(t *testing.T) {
	moves := []string{"e4", "e5", "Nf3"}
	if !IsBookMove(moves, 0) {
		t.Errorf("expected e4 to be book move")
	}
	if !IsBookMove(moves, 1) {
		t.Errorf("expected e5 to be book move")
	}
	if !IsBookMove(moves, 2) {
		t.Errorf("expected Nf3 to be book move")
	}

	unusual := []string{"h4", "h5"}
	if IsBookMove(unusual, 0) {
		t.Errorf("expected h4 not to be book move")
	}
}
