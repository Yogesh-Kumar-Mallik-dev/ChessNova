package chess

func IsCheckmate(p Position) bool {
	if !IsCheck(p) {
		return false
	}
	return len(LegalMoves(p)) == 0
}

func IsStalemate(p Position) bool {
	if IsCheck(p) {
		return false
	}
	return len(LegalMoves(p)) == 0
}

func IsFiftyMoveRule(p Position) bool {
	return p.Halfmove >= 100
}

func IsInsufficientMaterial(p Position) bool {
	var (
		whiteKnights, blackKnights int
		whiteBishops, blackBishops int
		whiteBishopSquareColor     int = -1 // 0 for dark, 1 for light
		blackBishopSquareColor     int = -1
	)

	for sq := Square(0); sq < 64; sq++ {
		piece := p.Board.Get(sq)
		if piece.IsEmpty() {
			continue
		}
		// If pawns, rooks, or queens exist, sufficient material
		if piece.Type == Pawn || piece.Type == Rook || piece.Type == Queen {
			return false
		}
		if piece.Type == Knight {
			if piece.Color == White {
				whiteKnights++
			} else {
				blackKnights++
			}
		} else if piece.Type == Bishop {
			sqColor := (sq.File() + sq.Rank()) % 2
			if piece.Color == White {
				whiteBishops++
				whiteBishopSquareColor = sqColor
			} else {
				blackBishops++
				blackBishopSquareColor = sqColor
			}
		}
	}

	totalMinorPieces := whiteKnights + blackKnights + whiteBishops + blackBishops

	// King vs King
	if totalMinorPieces == 0 {
		return true
	}

	// King and Minor Piece vs King (K+B vs K, K+N vs K)
	if totalMinorPieces == 1 {
		return true
	}

	// King and Bishop vs King and Bishop on the same color square
	if whiteKnights == 0 && blackKnights == 0 && whiteBishops == 1 && blackBishops == 1 {
		if whiteBishopSquareColor == blackBishopSquareColor {
			return true
		}
	}

	return false
}

func IsDraw(p Position) bool {
	return IsStalemate(p) || IsFiftyMoveRule(p) || IsInsufficientMaterial(p)
}
