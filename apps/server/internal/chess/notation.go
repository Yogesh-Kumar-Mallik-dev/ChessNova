package chess

import (
	"fmt"
	"strings"
)

func MoveToSAN(p Position, m Move) (string, error) {
	if err := ValidateMove(p, m); err != nil {
		return "", err
	}

	piece := p.Board.Get(m.From)
	target := p.Board.Get(m.To)
	isCapture := !target.IsEmpty() || (piece.Type == Pawn && p.EnPassant != nil && m.To == *p.EnPassant)

	// Castling
	if piece.Type == King {
		if (m.From == E1 && m.To == G1) || (m.From == E8 && m.To == G8) {
			nextPos, _ := ApplyMove(p, m)
			suffix := getCheckSuffix(nextPos)
			return "O-O" + suffix, nil
		}
		if (m.From == E1 && m.To == C1) || (m.From == E8 && m.To == C8) {
			nextPos, _ := ApplyMove(p, m)
			suffix := getCheckSuffix(nextPos)
			return "O-O-O" + suffix, nil
		}
	}

	var sb strings.Builder

	if piece.Type == Pawn {
		if isCapture {
			sb.WriteByte(byte('a' + m.From.File()))
			sb.WriteByte('x')
		}
		sb.WriteString(m.To.String())
		if m.Promotion != nil {
			sb.WriteByte('=')
			switch *m.Promotion {
			case Queen:
				sb.WriteByte('Q')
			case Rook:
				sb.WriteByte('R')
			case Bishop:
				sb.WriteByte('B')
			case Knight:
				sb.WriteByte('N')
			}
		}
	} else {
		// Non-pawn piece
		var pieceChar byte
		switch piece.Type {
		case Knight:
			pieceChar = 'N'
		case Bishop:
			pieceChar = 'B'
		case Rook:
			pieceChar = 'R'
		case Queen:
			pieceChar = 'Q'
		case King:
			pieceChar = 'K'
		}
		sb.WriteByte(pieceChar)

		// Disambiguation
		if piece.Type != King {
			disambiguation := getDisambiguation(p, m, piece.Type)
			sb.WriteString(disambiguation)
		}

		if isCapture {
			sb.WriteByte('x')
		}
		sb.WriteString(m.To.String())
	}

	nextPos, _ := ApplyMove(p, m)
	sb.WriteString(getCheckSuffix(nextPos))

	return sb.String(), nil
}

func getDisambiguation(p Position, m Move, pt PieceType) string {
	legal := LegalMoves(p)
	var candidates []Move
	for _, lm := range legal {
		if lm.To == m.To && lm.From != m.From {
			candPiece := p.Board.Get(lm.From)
			if candPiece.Type == pt && candPiece.Color == p.SideToMove {
				candidates = append(candidates, lm)
			}
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	sameFile := false
	sameRank := false

	for _, cand := range candidates {
		if cand.From.File() == m.From.File() {
			sameFile = true
		}
		if cand.From.Rank() == m.From.Rank() {
			sameRank = true
		}
	}

	fileChar := string(byte('a' + m.From.File()))
	rankChar := string(byte('1' + m.From.Rank()))

	if !sameFile {
		return fileChar
	}
	if !sameRank {
		return rankChar
	}
	return fileChar + rankChar
}

func getCheckSuffix(p Position) string {
	if IsCheckmate(p) {
		return "#"
	}
	if IsCheck(p) {
		return "+"
	}
	return ""
}

func ParseSAN(p Position, san string) (Move, error) {
	cleanSAN := strings.TrimRight(san, "+#?! ")
	legalMoves := LegalMoves(p)

	for _, m := range legalMoves {
		generatedSAN, err := MoveToSAN(p, m)
		if err != nil {
			continue
		}
		cleanGen := strings.TrimRight(generatedSAN, "+#?! ")
		if cleanGen == cleanSAN {
			return m, nil
		}
	}

	return Move{}, fmt.Errorf("unable to parse SAN move '%s' for position: %s", san, ToFEN(p))
}
