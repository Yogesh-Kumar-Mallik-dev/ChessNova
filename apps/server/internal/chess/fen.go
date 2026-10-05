package chess

import (
	"fmt"
	"strconv"
	"strings"
)

func fenPlacement(b Board) string {
	var sb strings.Builder
	for rank := 7; rank >= 0; rank-- {
		emptyCount := 0
		for file := 0; file < 8; file++ {
			sq := NewSquare(file, rank)
			piece := b.Get(sq)
			if piece.IsEmpty() {
				emptyCount++
			} else {
				if emptyCount > 0 {
					sb.WriteString(strconv.Itoa(emptyCount))
					emptyCount = 0
				}
				sb.WriteRune(piece.Rune())
			}
		}
		if emptyCount > 0 {
			sb.WriteString(strconv.Itoa(emptyCount))
		}
		if rank > 0 {
			sb.WriteByte('/')
		}
	}
	return sb.String()
}

func ToFEN(p Position) string {
	placement := fenPlacement(p.Board)

	stm := "w"
	if p.SideToMove == Black {
		stm = "b"
	}

	castling := p.Castling.String()

	ep := "-"
	if p.EnPassant != nil && p.EnPassant.IsValid() {
		ep = p.EnPassant.String()
	}

	return fmt.Sprintf("%s %s %s %s %d %d", placement, stm, castling, ep, p.Halfmove, p.Fullmove)
}

func FromFEN(fen string) (Position, error) {
	parts := strings.Fields(strings.TrimSpace(fen))
	if len(parts) < 4 {
		return Position{}, fmt.Errorf("invalid FEN string, expected at least 4 fields, got %d", len(parts))
	}

	var pos Position

	// 1. Board placement
	ranks := strings.Split(parts[0], "/")
	if len(ranks) != 8 {
		return Position{}, fmt.Errorf("invalid FEN board rank count: %d", len(ranks))
	}

	for rankIdx, rankStr := range ranks {
		rank := 7 - rankIdx
		file := 0
		for _, r := range rankStr {
			if r >= '1' && r <= '8' {
				file += int(r - '0')
			} else {
				piece, err := PieceFromRune(r)
				if err != nil {
					return Position{}, fmt.Errorf("invalid piece rune in FEN: %w", err)
				}
				if file >= 8 {
					return Position{}, fmt.Errorf("file overflow on rank %d", rank+1)
				}
				pos.Board.Set(NewSquare(file, rank), piece)
				file++
			}
		}
		if file != 8 {
			return Position{}, fmt.Errorf("invalid file count %d on rank %d", file, rank+1)
		}
	}

	// 2. Active color
	switch parts[1] {
	case "w":
		pos.SideToMove = White
	case "b":
		pos.SideToMove = Black
	default:
		return Position{}, fmt.Errorf("invalid active color in FEN: %s", parts[1])
	}

	// 3. Castling availability
	pos.Castling = ParseCastlingRights(parts[2])

	// 4. En passant target square
	if parts[3] != "-" {
		sq, err := ParseSquare(parts[3])
		if err != nil {
			return Position{}, fmt.Errorf("invalid en passant square: %w", err)
		}
		pos.EnPassant = &sq
	} else {
		pos.EnPassant = nil
	}

	// 5. Halfmove clock (optional, default 0)
	if len(parts) >= 5 {
		hm, err := strconv.Atoi(parts[4])
		if err != nil {
			return Position{}, fmt.Errorf("invalid halfmove clock in FEN: %s", parts[4])
		}
		pos.Halfmove = hm
	} else {
		pos.Halfmove = 0
	}

	// 6. Fullmove number (optional, default 1)
	if len(parts) >= 6 {
		fm, err := strconv.Atoi(parts[5])
		if err != nil {
			return Position{}, fmt.Errorf("invalid fullmove number in FEN: %s", parts[5])
		}
		pos.Fullmove = fm
	} else {
		pos.Fullmove = 1
	}

	return pos, nil
}
