package openings

import (
	"strings"
)

type Opening struct {
	ECO       string `json:"eco"`
	Name      string `json:"name"`
	Variation string `json:"variation,omitempty"`
	Moves     string `json:"moves"`
}

var openingDatabase = map[string]Opening{
	// King's Pawn Openings
	"e4": {ECO: "B00", Name: "King's Pawn Opening", Moves: "e4"},
	"e4 e5": {ECO: "C20", Name: "Open Game", Moves: "e4 e5"},
	"e4 e5 Nf3": {ECO: "C40", Name: "King's Knight Opening", Moves: "e4 e5 Nf3"},
	"e4 e5 Nf3 Nc6": {ECO: "C44", Name: "King's Knight Opening: Normal Continuation", Moves: "e4 e5 Nf3 Nc6"},

	// Italian Game
	"e4 e5 Nf3 Nc6 Bc4": {ECO: "C50", Name: "Italian Game", Moves: "e4 e5 Nf3 Nc6 Bc4"},
	"e4 e5 Nf3 Nc6 Bc4 Bc5": {ECO: "C50", Name: "Italian Game", Variation: "Giuoco Piano", Moves: "e4 e5 Nf3 Nc6 Bc4 Bc5"},
	"e4 e5 Nf3 Nc6 Bc4 Bc5 c3": {ECO: "C53", Name: "Italian Game", Variation: "Giuoco Piano, Main Line", Moves: "e4 e5 Nf3 Nc6 Bc4 Bc5 c3"},
	"e4 e5 Nf3 Nc6 Bc4 Bc5 c3 Nf6 d4": {ECO: "C54", Name: "Italian Game", Variation: "Giuoco Piano, Classical", Moves: "e4 e5 Nf3 Nc6 Bc4 Bc5 c3 Nf6 d4"},
	"e4 e5 Nf3 Nc6 Bc4 Bc5 d3": {ECO: "C50", Name: "Italian Game", Variation: "Giuoco Pianissimo", Moves: "e4 e5 Nf3 Nc6 Bc4 Bc5 d3"},
	"e4 e5 Nf3 Nc6 Bc4 Bc5 b4": {ECO: "C51", Name: "Evans Gambit", Moves: "e4 e5 Nf3 Nc6 Bc4 Bc5 b4"},
	"e4 e5 Nf3 Nc6 Bc4 Nf6": {ECO: "C55", Name: "Two Knights Defense", Moves: "e4 e5 Nf3 Nc6 Bc4 Nf6"},
	"e4 e5 Nf3 Nc6 Bc4 Nf6 Ng5": {ECO: "C57", Name: "Two Knights Defense", Variation: "Fried Liver Attack", Moves: "e4 e5 Nf3 Nc6 Bc4 Nf6 Ng5"},
	"e4 e5 Nf3 Nc6 Bc4 Nf6 d3": {ECO: "C55", Name: "Two Knights Defense", Variation: "Modern Line", Moves: "e4 e5 Nf3 Nc6 Bc4 Nf6 d3"},

	// Ruy Lopez
	"e4 e5 Nf3 Nc6 Bb5": {ECO: "C60", Name: "Ruy Lopez", Moves: "e4 e5 Nf3 Nc6 Bb5"},
	"e4 e5 Nf3 Nc6 Bb5 a6": {ECO: "C68", Name: "Ruy Lopez", Variation: "Morphy Defense", Moves: "e4 e5 Nf3 Nc6 Bb5 a6"},
	"e4 e5 Nf3 Nc6 Bb5 a6 Ba4": {ECO: "C70", Name: "Ruy Lopez", Variation: "Morphy Defense, Columbus", Moves: "e4 e5 Nf3 Nc6 Bb5 a6 Ba4"},
	"e4 e5 Nf3 Nc6 Bb5 a6 Ba4 Nf6": {ECO: "C78", Name: "Ruy Lopez", Variation: "Closed Defense", Moves: "e4 e5 Nf3 Nc6 Bb5 a6 Ba4 Nf6"},
	"e4 e5 Nf3 Nc6 Bb5 a6 Ba4 Nf6 O-O": {ECO: "C84", Name: "Ruy Lopez", Variation: "Closed Defense, Main Line", Moves: "e4 e5 Nf3 Nc6 Bb5 a6 Ba4 Nf6 O-O"},
	"e4 e5 Nf3 Nc6 Bb5 a6 Ba4 Nf6 O-O Be7": {ECO: "C84", Name: "Ruy Lopez", Variation: "Closed Defense, Classical", Moves: "e4 e5 Nf3 Nc6 Bb5 a6 Ba4 Nf6 O-O Be7"},
	"e4 e5 Nf3 Nc6 Bb5 a6 Bxc6": {ECO: "C68", Name: "Ruy Lopez", Variation: "Exchange Variation", Moves: "e4 e5 Nf3 Nc6 Bb5 a6 Bxc6"},
	"e4 e5 Nf3 Nc6 Bb5 Nf6": {ECO: "C65", Name: "Ruy Lopez", Variation: "Berlin Defense", Moves: "e4 e5 Nf3 Nc6 Bb5 Nf6"},
	"e4 e5 Nf3 Nc6 Bb5 Nf6 O-O Nxe4": {ECO: "C67", Name: "Ruy Lopez", Variation: "Berlin Defense, Open Line", Moves: "e4 e5 Nf3 Nc6 Bb5 Nf6 O-O Nxe4"},

	// Scotch Game & Petroff
	"e4 e5 Nf3 Nc6 d4": {ECO: "C45", Name: "Scotch Game", Moves: "e4 e5 Nf3 Nc6 d4"},
	"e4 e5 Nf3 Nc6 d4 exd4 Nxd4": {ECO: "C45", Name: "Scotch Game", Variation: "Main Line", Moves: "e4 e5 Nf3 Nc6 d4 exd4 Nxd4"},
	"e4 e5 Nf3 Nc6 d4 exd4 Nxd4 Bc5": {ECO: "C45", Name: "Scotch Game", Variation: "Classical Defense", Moves: "e4 e5 Nf3 Nc6 d4 exd4 Nxd4 Bc5"},
	"e4 e5 Nf3 Nf6": {ECO: "C42", Name: "Petroff Defense", Moves: "e4 e5 Nf3 Nf6"},

	// Sicilian Defense
	"e4 c5": {ECO: "B20", Name: "Sicilian Defense", Moves: "e4 c5"},
	"e4 c5 Nf3": {ECO: "B27", Name: "Sicilian Defense", Variation: "Open System", Moves: "e4 c5 Nf3"},
	"e4 c5 Nf3 d6": {ECO: "B50", Name: "Sicilian Defense", Variation: "Modern Line", Moves: "e4 c5 Nf3 d6"},
	"e4 c5 Nf3 d6 d4": {ECO: "B53", Name: "Sicilian Defense", Variation: "Open", Moves: "e4 c5 Nf3 d6 d4"},
	"e4 c5 Nf3 d6 d4 cxd4 Nxd4 Nf6 Nc3": {ECO: "B54", Name: "Sicilian Defense", Variation: "Classical Open", Moves: "e4 c5 Nf3 d6 d4 cxd4 Nxd4 Nf6 Nc3"},
	"e4 c5 Nf3 d6 d4 cxd4 Nxd4 Nf6 Nc3 a6": {ECO: "B90", Name: "Sicilian Defense", Variation: "Najdorf Variation", Moves: "e4 c5 Nf3 d6 d4 cxd4 Nxd4 Nf6 Nc3 a6"},
	"e4 c5 Nf3 d6 d4 cxd4 Nxd4 Nf6 Nc3 g6": {ECO: "B70", Name: "Sicilian Defense", Variation: "Dragon Variation", Moves: "e4 c5 Nf3 d6 d4 cxd4 Nxd4 Nf6 Nc3 g6"},
	"e4 c5 Nf3 Nc6": {ECO: "B30", Name: "Sicilian Defense", Variation: "Old Sicilian", Moves: "e4 c5 Nf3 Nc6"},
	"e4 c5 Nf3 e6": {ECO: "B40", Name: "Sicilian Defense", Variation: "French Variation", Moves: "e4 c5 Nf3 e6"},
	"e4 c5 c3": {ECO: "B22", Name: "Sicilian Defense", Variation: "Alapin Variation", Moves: "e4 c5 c3"},
	"e4 c5 Nc3": {ECO: "B23", Name: "Sicilian Defense", Variation: "Closed Sicilian", Moves: "e4 c5 Nc3"},

	// French Defense
	"e4 e6": {ECO: "C00", Name: "French Defense", Moves: "e4 e6"},
	"e4 e6 d4 d5": {ECO: "C01", Name: "French Defense", Variation: "Normal Line", Moves: "e4 e6 d4 d5"},
	"e4 e6 d4 d5 e5": {ECO: "C02", Name: "French Defense", Variation: "Advance Variation", Moves: "e4 e6 d4 d5 e5"},
	"e4 e6 d4 d5 Nc3": {ECO: "C10", Name: "French Defense", Variation: "Paulsen Variation", Moves: "e4 e6 d4 d5 Nc3"},
	"e4 e6 d4 d5 Nc3 Bb4": {ECO: "C15", Name: "French Defense", Variation: "Winawer Variation", Moves: "e4 e6 d4 d5 Nc3 Bb4"},
	"e4 e6 d4 d5 Nd2": {ECO: "C03", Name: "French Defense", Variation: "Tarrasch Variation", Moves: "e4 e6 d4 d5 Nd2"},

	// Caro-Kann Defense
	"e4 c6": {ECO: "B10", Name: "Caro-Kann Defense", Moves: "e4 c6"},
	"e4 c6 d4 d5": {ECO: "B12", Name: "Caro-Kann Defense", Variation: "Main Line", Moves: "e4 c6 d4 d5"},
	"e4 c6 d4 d5 e5": {ECO: "B12", Name: "Caro-Kann Defense", Variation: "Advance Variation", Moves: "e4 c6 d4 d5 e5"},
	"e4 c6 d4 d5 Nc3 dxe4 Nxe4 Bf5": {ECO: "B18", Name: "Caro-Kann Defense", Variation: "Classical Variation", Moves: "e4 c6 d4 d5 Nc3 dxe4 Nxe4 Bf5"},

	// 1. d4 Openings
	"d4": {ECO: "A40", Name: "Queen's Pawn Opening", Moves: "d4"},
	"d4 d5": {ECO: "D00", Name: "Queen's Pawn Game", Moves: "d4 d5"},
	"d4 d5 c4": {ECO: "D06", Name: "Queen's Gambit", Moves: "d4 d5 c4"},
	"d4 d5 c4 dxc4": {ECO: "D20", Name: "Queen's Gambit Accepted", Moves: "d4 d5 c4 dxc4"},
	"d4 d5 c4 e6": {ECO: "D30", Name: "Queen's Gambit Declined", Moves: "d4 d5 c4 e6"},
	"d4 d5 c4 e6 Nc3 Nf6": {ECO: "D35", Name: "Queen's Gambit Declined", Variation: "Traditional Line", Moves: "d4 d5 c4 e6 Nc3 Nf6"},
	"d4 d5 c4 c6": {ECO: "D10", Name: "Slav Defense", Moves: "d4 d5 c4 c6"},
	"d4 d5 c4 c6 Nf3 Nf6 Nc3 dxc4": {ECO: "D15", Name: "Slav Defense", Variation: "Main Line", Moves: "d4 d5 c4 c6 Nf3 Nf6 Nc3 dxc4"},
	"d4 d5 Nf3": {ECO: "D02", Name: "Queen's Pawn Game", Moves: "d4 d5 Nf3"},
	"d4 d5 Bf4": {ECO: "D00", Name: "London System", Moves: "d4 d5 Bf4"},
	"d4 Nf6 Bf4": {ECO: "A45", Name: "London System", Moves: "d4 Nf6 Bf4"},

	// Indian Defenses
	"d4 Nf6": {ECO: "A45", Name: "Indian Defense", Moves: "d4 Nf6"},
	"d4 Nf6 c4": {ECO: "E00", Name: "Indian Defense", Variation: "Normal Line", Moves: "d4 Nf6 c4"},
	"d4 Nf6 c4 e6": {ECO: "E00", Name: "Indian Defense", Variation: "East Indian", Moves: "d4 Nf6 c4 e6"},
	"d4 Nf6 c4 e6 Nc3 Bb4": {ECO: "E20", Name: "Nimzo-Indian Defense", Moves: "d4 Nf6 c4 e6 Nc3 Bb4"},
	"d4 Nf6 c4 g6": {ECO: "E60", Name: "King's Indian / Grünfeld", Moves: "d4 Nf6 c4 g6"},
	"d4 Nf6 c4 g6 Nc3 Bg7": {ECO: "E61", Name: "King's Indian Defense", Moves: "d4 Nf6 c4 g6 Nc3 Bg7"},
	"d4 Nf6 c4 g6 Nc3 d5": {ECO: "D80", Name: "Grünfeld Defense", Moves: "d4 Nf6 c4 g6 Nc3 d5"},

	// Flank
	"c4": {ECO: "A10", Name: "English Opening", Moves: "c4"},
	"c4 e5": {ECO: "A20", Name: "English Opening", Variation: "King's English", Moves: "c4 e5"},
	"Nf3": {ECO: "A04", Name: "Réti Opening", Moves: "Nf3"},
}

// IdentifyOpening searches the opening database for the deepest matching line
func IdentifyOpening(sans []string) (*Opening, int) {
	var bestMatch *Opening
	bestPly := 0

	var prefix []string
	for i, san := range sans {
		prefix = append(prefix, san)
		key := strings.Join(prefix, " ")
		if op, found := openingDatabase[key]; found {
			copyOp := op
			bestMatch = &copyOp
			bestPly = i + 1
		}
	}

	return bestMatch, bestPly
}

// IsBookMove checks if the move at ply index belongs to an opening book line
func IsBookMove(sans []string, plyIndex int) bool {
	if plyIndex < 0 || plyIndex >= len(sans) {
		return false
	}
	key := strings.Join(sans[:plyIndex+1], " ")
	_, exists := openingDatabase[key]
	return exists
}
