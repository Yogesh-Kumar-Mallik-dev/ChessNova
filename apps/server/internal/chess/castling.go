package chess

type CastlingRights uint8

const (
	CastlingWhiteKingside  CastlingRights = 1 << 0 // 1 (K)
	CastlingWhiteQueenside CastlingRights = 1 << 1 // 2 (Q)
	CastlingBlackKingside  CastlingRights = 1 << 2 // 4 (k)
	CastlingBlackQueenside CastlingRights = 1 << 3 // 8 (q)
	CastlingAll            CastlingRights = CastlingWhiteKingside | CastlingWhiteQueenside | CastlingBlackKingside | CastlingBlackQueenside
	CastlingNone           CastlingRights = 0
)

func (cr CastlingRights) Has(right CastlingRights) bool {
	return (cr & right) == right
}

func (cr CastlingRights) Add(right CastlingRights) CastlingRights {
	return cr | right
}

func (cr CastlingRights) Remove(right CastlingRights) CastlingRights {
	return cr &^ right
}

func (cr CastlingRights) String() string {
	if cr == CastlingNone {
		return "-"
	}
	res := ""
	if cr.Has(CastlingWhiteKingside) {
		res += "K"
	}
	if cr.Has(CastlingWhiteQueenside) {
		res += "Q"
	}
	if cr.Has(CastlingBlackKingside) {
		res += "k"
	}
	if cr.Has(CastlingBlackQueenside) {
		res += "q"
	}
	if res == "" {
		return "-"
	}
	return res
}

func ParseCastlingRights(s string) CastlingRights {
	if s == "-" {
		return CastlingNone
	}
	var cr CastlingRights
	for _, r := range s {
		switch r {
		case 'K':
			cr |= CastlingWhiteKingside
		case 'Q':
			cr |= CastlingWhiteQueenside
		case 'k':
			cr |= CastlingBlackKingside
		case 'q':
			cr |= CastlingBlackQueenside
		}
	}
	return cr
}
