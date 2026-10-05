package chess

func IsPromotion(p Position, m Move) bool {
	piece := p.Board.Get(m.From)
	if piece.Type != Pawn {
		return false
	}
	if piece.Color == White && m.To.Rank() == 7 {
		return true
	}
	if piece.Color == Black && m.To.Rank() == 0 {
		return true
	}
	return false
}

func ValidPromotionPiece(pt PieceType) bool {
	return pt == Queen || pt == Rook || pt == Bishop || pt == Knight
}
