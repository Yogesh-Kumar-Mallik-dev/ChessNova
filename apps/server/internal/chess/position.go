package chess

import (
	"crypto/sha256"
	"fmt"
)

type Position struct {
	Board      Board          `json:"board"`
	SideToMove Color          `json:"sideToMove"`
	Castling   CastlingRights `json:"castling"`
	EnPassant  *Square        `json:"enPassant,omitempty"`
	Halfmove   int            `json:"halfmove"`
	Fullmove   int            `json:"fullmove"`
}

const StartingFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

func InitialPosition() Position {
	pos, _ := FromFEN(StartingFEN)
	return pos
}

func (p Position) Clone() Position {
	cp := p
	if p.EnPassant != nil {
		ep := *p.EnPassant
		cp.EnPassant = &ep
	}
	return cp
}

// RepetitionKey returns a representation of the position suitable for threefold repetition checking.
// Per FIDE rules: two positions are identical if the same type of pieces of the same color occupy the same squares,
// the same player's turn to move, and the same castling rights and en-passant possibilities.
func (p Position) RepetitionKey() string {
	fenBoard := fenPlacement(p.Board)
	epStr := "-"
	if p.EnPassant != nil {
		epStr = p.EnPassant.String()
	}
	return fmt.Sprintf("%s %s %s %s", fenBoard, p.SideToMove.String(), p.Castling.String(), epStr)
}

// Hash returns a 32-byte sha256 checksum of the repetition key
func (p Position) Hash() [32]byte {
	return sha256.Sum256([]byte(p.RepetitionKey()))
}
