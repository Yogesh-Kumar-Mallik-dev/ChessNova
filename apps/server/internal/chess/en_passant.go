package chess

func IsEnPassantMove(p Position, m Move) bool {
	piece := p.Board.Get(m.From)
	if piece.Type != Pawn {
		return false
	}
	if p.EnPassant == nil || *p.EnPassant != m.To {
		return false
	}
	// Pawn must move diagonally
	if m.From.File() == m.To.File() {
		return false
	}
	return true
}

func EnPassantCapturedSquare(to Square, color Color) Square {
	if color == White {
		return NewSquare(to.File(), to.Rank()-1)
	}
	return NewSquare(to.File(), to.Rank()+1)
}
