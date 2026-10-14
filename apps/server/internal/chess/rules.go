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

// IsFiftyMoveRule returns true when 50 full moves (100 halfmoves) have passed without capture or pawn move
// Under FIDE 9.3, this gives the right to claim a draw.
func IsFiftyMoveRule(p Position) bool {
	return p.Halfmove >= 100
}

// IsSeventyFiveMoveRule returns true when 75 full moves (150 halfmoves) have passed without capture or pawn move
// Under FIDE 9.6.2, this is an automatic draw.
func IsSeventyFiveMoveRule(p Position) bool {
	return p.Halfmove >= 150
}

// CanOpponentMate checks whether the opponent has sufficient material to theoretically deliver checkmate
// under any legal series of moves (per FIDE Article 6.9).
func CanOpponentMate(p Position, opp Color) bool {
	var oppPawns, oppRooks, oppQueens, oppKnights, oppBishops int
	var otherPieces int

	for sq := Square(0); sq < 64; sq++ {
		piece := p.Board.Get(sq)
		if piece.IsEmpty() || piece.Type == King {
			continue
		}
		if piece.Color == opp {
			switch piece.Type {
			case Pawn:
				oppPawns++
			case Rook:
				oppRooks++
			case Queen:
				oppQueens++
			case Knight:
				oppKnights++
			case Bishop:
				oppBishops++
			}
		} else {
			otherPieces++
		}
	}

	// If opponent has pawns, rooks, or queens, checkmate is always theoretically possible
	if oppPawns > 0 || oppRooks > 0 || oppQueens > 0 {
		return true
	}

	// Lone king cannot mate
	if oppKnights == 0 && oppBishops == 0 {
		return false
	}

	// Single minor piece (K+B or K+N) vs lone king cannot deliver mate under any legal move sequence
	if oppKnights+oppBishops == 1 && otherPieces == 0 {
		return false
	}

	return true
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
	return IsStalemate(p) || IsFiftyMoveRule(p) || IsInsufficientMaterial(p) || IsSeventyFiveMoveRule(p)
}
