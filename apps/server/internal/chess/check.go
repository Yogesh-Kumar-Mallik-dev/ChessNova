package chess

var knightOffsets = [][2]int{
	{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2},
	{1, -2}, {1, 2}, {2, -1}, {2, 1},
}

var kingOffsets = [][2]int{
	{-1, -1}, {-1, 0}, {-1, 1},
	{0, -1}, {0, 1},
	{1, -1}, {1, 0}, {1, 1},
}

var rookDirections = [][2]int{
	{-1, 0}, {1, 0}, {0, -1}, {0, 1},
}

var bishopDirections = [][2]int{
	{-1, -1}, {-1, 1}, {1, -1}, {1, 1},
}

func IsSquareAttacked(b *Board, sq Square, attacker Color) bool {
	if !sq.IsValid() {
		return false
	}
	f := sq.File()
	r := sq.Rank()

	// 1. Pawn attacks
	if attacker == White {
		// White pawns attack sq from (f-1, r-1) and (f+1, r-1)
		for _, df := range []int{-1, 1} {
			pSq := NewSquare(f+df, r-1)
			if pSq.IsValid() {
				piece := b.Get(pSq)
				if piece.Type == Pawn && piece.Color == White {
					return true
				}
			}
		}
	} else {
		// Black pawns attack sq from (f-1, r+1) and (f+1, r+1)
		for _, df := range []int{-1, 1} {
			pSq := NewSquare(f+df, r+1)
			if pSq.IsValid() {
				piece := b.Get(pSq)
				if piece.Type == Pawn && piece.Color == Black {
					return true
				}
			}
		}
	}

	// 2. Knight attacks
	for _, offset := range knightOffsets {
		nSq := NewSquare(f+offset[0], r+offset[1])
		if nSq.IsValid() {
			piece := b.Get(nSq)
			if piece.Type == Knight && piece.Color == attacker {
				return true
			}
		}
	}

	// 3. Bishop and Queen (diagonals)
	for _, dir := range bishopDirections {
		curF, curR := f+dir[0], r+dir[1]
		for curF >= 0 && curF < 8 && curR >= 0 && curR < 8 {
			curSq := NewSquare(curF, curR)
			piece := b.Get(curSq)
			if !piece.IsEmpty() {
				if piece.Color == attacker && (piece.Type == Bishop || piece.Type == Queen) {
					return true
				}
				break // Ray blocked
			}
			curF += dir[0]
			curR += dir[1]
		}
	}

	// 4. Rook and Queen (straight)
	for _, dir := range rookDirections {
		curF, curR := f+dir[0], r+dir[1]
		for curF >= 0 && curF < 8 && curR >= 0 && curR < 8 {
			curSq := NewSquare(curF, curR)
			piece := b.Get(curSq)
			if !piece.IsEmpty() {
				if piece.Color == attacker && (piece.Type == Rook || piece.Type == Queen) {
					return true
				}
				break // Ray blocked
			}
			curF += dir[0]
			curR += dir[1]
		}
	}

	// 5. King attacks (adjacent)
	for _, offset := range kingOffsets {
		kSq := NewSquare(f+offset[0], r+offset[1])
		if kSq.IsValid() {
			piece := b.Get(kSq)
			if piece.Type == King && piece.Color == attacker {
				return true
			}
		}
	}

	return false
}

func IsCheck(p Position) bool {
	kingSq := p.Board.KingSquare(p.SideToMove)
	if !kingSq.IsValid() {
		return false
	}
	return IsSquareAttacked(&p.Board, kingSq, p.SideToMove.Other())
}
