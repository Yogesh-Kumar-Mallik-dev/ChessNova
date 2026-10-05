package chess

import (
	"fmt"
)

type Move struct {
	From      Square     `json:"from"`
	To        Square     `json:"to"`
	Promotion *PieceType `json:"promotion,omitempty"`
}

func NewMove(from, to Square, promo *PieceType) Move {
	return Move{
		From:      from,
		To:        to,
		Promotion: promo,
	}
}

func (m Move) Equals(other Move) bool {
	if m.From != other.From || m.To != other.To {
		return false
	}
	if m.Promotion == nil && other.Promotion == nil {
		return true
	}
	if m.Promotion != nil && other.Promotion != nil {
		return *m.Promotion == *other.Promotion
	}
	return false
}

func (m Move) UCI() string {
	res := m.From.String() + m.To.String()
	if m.Promotion != nil {
		switch *m.Promotion {
		case Queen:
			res += "q"
		case Rook:
			res += "r"
		case Bishop:
			res += "b"
		case Knight:
			res += "n"
		}
	}
	return res
}

func (m Move) String() string {
	return m.UCI()
}

func ParseUCIMove(uci string) (Move, error) {
	if len(uci) < 4 || len(uci) > 5 {
		return Move{}, fmt.Errorf("invalid UCI move string: %s", uci)
	}
	from, err := ParseSquare(uci[0:2])
	if err != nil {
		return Move{}, err
	}
	to, err := ParseSquare(uci[2:4])
	if err != nil {
		return Move{}, err
	}
	var promo *PieceType
	if len(uci) == 5 {
		var pt PieceType
		switch uci[4] {
		case 'q', 'Q':
			pt = Queen
		case 'r', 'R':
			pt = Rook
		case 'b', 'B':
			pt = Bishop
		case 'n', 'N':
			pt = Knight
		default:
			return Move{}, fmt.Errorf("invalid promotion piece: %c", uci[4])
		}
		promo = &pt
	}
	return Move{From: from, To: to, Promotion: promo}, nil
}
