package chess

import (
	"fmt"
	"unicode"
)

type PieceType uint8

const (
	NoPiece PieceType = 0
	Pawn    PieceType = 1
	Knight  PieceType = 2
	Bishop  PieceType = 3
	Rook    PieceType = 4
	Queen   PieceType = 5
	King    PieceType = 6
)

func (pt PieceType) String() string {
	switch pt {
	case Pawn:
		return "pawn"
	case Knight:
		return "knight"
	case Bishop:
		return "bishop"
	case Rook:
		return "rook"
	case Queen:
		return "queen"
	case King:
		return "king"
	default:
		return "none"
	}
}

func (pt PieceType) Rune(c Color) rune {
	var r rune
	switch pt {
	case Pawn:
		r = 'p'
	case Knight:
		r = 'n'
	case Bishop:
		r = 'b'
	case Rook:
		r = 'r'
	case Queen:
		r = 'q'
	case King:
		r = 'k'
	default:
		return ' '
	}
	if c == White {
		return unicode.ToUpper(r)
	}
	return r
}

type Piece struct {
	Type  PieceType `json:"type"`
	Color Color     `json:"color"`
}

func NewPiece(pt PieceType, c Color) Piece {
	return Piece{Type: pt, Color: c}
}

func (p Piece) IsEmpty() bool {
	return p.Type == NoPiece
}

func (p Piece) Rune() rune {
	return p.Type.Rune(p.Color)
}

func PieceFromRune(r rune) (Piece, error) {
	c := Black
	if unicode.IsUpper(r) {
		c = White
	}
	lr := unicode.ToLower(r)
	var pt PieceType
	switch lr {
	case 'p':
		pt = Pawn
	case 'n':
		pt = Knight
	case 'b':
		pt = Bishop
	case 'r':
		pt = Rook
	case 'q':
		pt = Queen
	case 'k':
		pt = King
	default:
		return Piece{}, fmt.Errorf("invalid piece rune: %c", r)
	}
	return Piece{Type: pt, Color: c}, nil
}
