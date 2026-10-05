package chess

type Board [64]Piece

func (b *Board) Get(sq Square) Piece {
	if !sq.IsValid() {
		return Piece{}
	}
	return b[sq]
}

func (b *Board) Set(sq Square, p Piece) {
	if sq.IsValid() {
		b[sq] = p
	}
}

func (b *Board) Clear(sq Square) {
	if sq.IsValid() {
		b[sq] = Piece{}
	}
}

func (b *Board) IsEmpty(sq Square) bool {
	if !sq.IsValid() {
		return false
	}
	return b[sq].IsEmpty()
}

func (b *Board) KingSquare(color Color) Square {
	for sq := Square(0); sq < 64; sq++ {
		p := b[sq]
		if p.Type == King && p.Color == color {
			return sq
		}
	}
	return NoSquare
}
