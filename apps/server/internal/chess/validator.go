package chess

import (
	"errors"
	"fmt"
)

var (
	ErrNotYourTurn     = errors.New("not your turn")
	ErrNoPieceAtSquare = errors.New("no piece at source square")
	ErrNotYourPiece    = errors.New("cannot move opponent's piece")
	ErrIllegalMove     = errors.New("illegal move")
	ErrKingInCheck     = errors.New("move leaves king in check")
)

// GeneratePseudoMoves generates all pseudo-legal moves (not checking for self-check yet)
func GeneratePseudoMoves(p Position) []Move {
	var moves []Move
	color := p.SideToMove
	opp := color.Other()

	for sq := Square(0); sq < 64; sq++ {
		piece := p.Board.Get(sq)
		if piece.IsEmpty() || piece.Color != color {
			continue
		}

		f := sq.File()
		r := sq.Rank()

		switch piece.Type {
		case Pawn:
			dir := 1
			startRank := 1
			promoRank := 7
			if color == Black {
				dir = -1
				startRank = 6
				promoRank = 0
			}

			// Single push
			fwdSq := NewSquare(f, r+dir)
			if fwdSq.IsValid() && p.Board.IsEmpty(fwdSq) {
				if fwdSq.Rank() == promoRank {
					for _, promo := range []PieceType{Queen, Rook, Bishop, Knight} {
						pr := promo
						moves = append(moves, Move{From: sq, To: fwdSq, Promotion: &pr})
					}
				} else {
					moves = append(moves, Move{From: sq, To: fwdSq})

					// Double push from start rank
					if r == startRank {
						doubleSq := NewSquare(f, r+2*dir)
						if doubleSq.IsValid() && p.Board.IsEmpty(doubleSq) {
							moves = append(moves, Move{From: sq, To: doubleSq})
						}
					}
				}
			}

			// Captures
			for _, df := range []int{-1, 1} {
				capSq := NewSquare(f+df, r+dir)
				if !capSq.IsValid() {
					continue
				}
				targetPiece := p.Board.Get(capSq)

				// Standard capture
				if !targetPiece.IsEmpty() && targetPiece.Color == opp {
					if capSq.Rank() == promoRank {
						for _, promo := range []PieceType{Queen, Rook, Bishop, Knight} {
							pr := promo
							moves = append(moves, Move{From: sq, To: capSq, Promotion: &pr})
						}
					} else {
						moves = append(moves, Move{From: sq, To: capSq})
					}
				} else if p.EnPassant != nil && *p.EnPassant == capSq {
					// En passant capture
					moves = append(moves, Move{From: sq, To: capSq})
				}
			}

		case Knight:
			for _, offset := range knightOffsets {
				toSq := NewSquare(f+offset[0], r+offset[1])
				if toSq.IsValid() {
					targetPiece := p.Board.Get(toSq)
					if targetPiece.IsEmpty() || targetPiece.Color == opp {
						moves = append(moves, Move{From: sq, To: toSq})
					}
				}
			}

		case Bishop:
			moves = append(moves, generateSlidingMoves(p, sq, bishopDirections, opp)...)

		case Rook:
			moves = append(moves, generateSlidingMoves(p, sq, rookDirections, opp)...)

		case Queen:
			moves = append(moves, generateSlidingMoves(p, sq, bishopDirections, opp)...)
			moves = append(moves, generateSlidingMoves(p, sq, rookDirections, opp)...)

		case King:
			for _, offset := range kingOffsets {
				toSq := NewSquare(f+offset[0], r+offset[1])
				if toSq.IsValid() {
					targetPiece := p.Board.Get(toSq)
					if targetPiece.IsEmpty() || targetPiece.Color == opp {
						moves = append(moves, Move{From: sq, To: toSq})
					}
				}
			}

			// Castling moves
			if color == White {
				if sq == E1 && !IsSquareAttacked(&p.Board, E1, Black) {
					// Kingside: E1 -> G1, through F1
					if p.Castling.Has(CastlingWhiteKingside) &&
						p.Board.IsEmpty(F1) && p.Board.IsEmpty(G1) &&
						p.Board.Get(H1) == NewPiece(Rook, White) &&
						!IsSquareAttacked(&p.Board, F1, Black) &&
						!IsSquareAttacked(&p.Board, G1, Black) {
						moves = append(moves, Move{From: E1, To: G1})
					}
					// Queenside: E1 -> C1, through D1, B1 empty
					if p.Castling.Has(CastlingWhiteQueenside) &&
						p.Board.IsEmpty(D1) && p.Board.IsEmpty(C1) && p.Board.IsEmpty(B1) &&
						p.Board.Get(A1) == NewPiece(Rook, White) &&
						!IsSquareAttacked(&p.Board, D1, Black) &&
						!IsSquareAttacked(&p.Board, C1, Black) {
						moves = append(moves, Move{From: E1, To: C1})
					}
				}
			} else {
				if sq == E8 && !IsSquareAttacked(&p.Board, E8, White) {
					// Kingside: E8 -> G8, through F8
					if p.Castling.Has(CastlingBlackKingside) &&
						p.Board.IsEmpty(F8) && p.Board.IsEmpty(G8) &&
						p.Board.Get(H8) == NewPiece(Rook, Black) &&
						!IsSquareAttacked(&p.Board, F8, White) &&
						!IsSquareAttacked(&p.Board, G8, White) {
						moves = append(moves, Move{From: E8, To: G8})
					}
					// Queenside: E8 -> C8, through D8, B8 empty
					if p.Castling.Has(CastlingBlackQueenside) &&
						p.Board.IsEmpty(D8) && p.Board.IsEmpty(C8) && p.Board.IsEmpty(B8) &&
						p.Board.Get(A8) == NewPiece(Rook, Black) &&
						!IsSquareAttacked(&p.Board, D8, White) &&
						!IsSquareAttacked(&p.Board, C8, White) {
						moves = append(moves, Move{From: E8, To: C8})
					}
				}
			}
		}
	}

	return moves
}

func generateSlidingMoves(p Position, from Square, directions [][2]int, opp Color) []Move {
	var moves []Move
	f := from.File()
	r := from.Rank()

	for _, dir := range directions {
		curF, curR := f+dir[0], r+dir[1]
		for curF >= 0 && curF < 8 && curR >= 0 && curR < 8 {
			toSq := NewSquare(curF, curR)
			piece := p.Board.Get(toSq)
			if piece.IsEmpty() {
				moves = append(moves, Move{From: from, To: toSq})
			} else {
				if piece.Color == opp {
					moves = append(moves, Move{From: from, To: toSq})
				}
				break
			}
			curF += dir[0]
			curR += dir[1]
		}
	}
	return moves
}

// LegalMoves returns all strictly legal moves in position
func LegalMoves(p Position) []Move {
	pseudo := GeneratePseudoMoves(p)
	legal := make([]Move, 0, len(pseudo))

	for _, m := range pseudo {
		nextPos, err := ApplyMoveWithoutTurnValidation(p, m)
		if err != nil {
			continue
		}
		// King of the side that just moved must not be attacked
		kingSq := nextPos.Board.KingSquare(p.SideToMove)
		if !kingSq.IsValid() || IsSquareAttacked(&nextPos.Board, kingSq, p.SideToMove.Other()) {
			continue
		}
		legal = append(legal, m)
	}
	return legal
}

// LegalMovesFrom returns all legal moves originating from square sq
func LegalMovesFrom(p Position, sq Square) []Move {
	all := LegalMoves(p)
	filtered := make([]Move, 0, len(all))
	for _, m := range all {
		if m.From == sq {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// ValidateMove validates that a move is completely legal in the given position
func ValidateMove(p Position, m Move) error {
	legal := LegalMoves(p)
	for _, lm := range legal {
		if lm.Equals(m) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrIllegalMove, m.UCI())
}

// ApplyMove applies move to position, returning new position or error if illegal
func ApplyMove(p Position, m Move) (Position, error) {
	if err := ValidateMove(p, m); err != nil {
		return Position{}, err
	}
	return ApplyMoveWithoutTurnValidation(p, m)
}

// ApplyMoveWithoutTurnValidation executes the move changes without checking legality
func ApplyMoveWithoutTurnValidation(p Position, m Move) (Position, error) {
	movingPiece := p.Board.Get(m.From)
	if movingPiece.IsEmpty() {
		return Position{}, ErrNoPieceAtSquare
	}
	if movingPiece.Color != p.SideToMove {
		return Position{}, ErrNotYourPiece
	}

	next := p.Clone()
	capturedPiece := next.Board.Get(m.To)

	// Halfmove clock reset condition
	isCapture := !capturedPiece.IsEmpty()
	isPawnMove := movingPiece.Type == Pawn

	if isCapture || isPawnMove {
		next.Halfmove = 0
	} else {
		next.Halfmove++
	}

	// Handle Castling moves (King moving 2 squares)
	if movingPiece.Type == King {
		if m.From == E1 && m.To == G1 {
			// Move White Kingside Rook: H1 -> F1
			next.Board.Clear(H1)
			next.Board.Set(F1, NewPiece(Rook, White))
		} else if m.From == E1 && m.To == C1 {
			// Move White Queenside Rook: A1 -> D1
			next.Board.Clear(A1)
			next.Board.Set(D1, NewPiece(Rook, White))
		} else if m.From == E8 && m.To == G8 {
			// Move Black Kingside Rook: H8 -> F8
			next.Board.Clear(H8)
			next.Board.Set(F8, NewPiece(Rook, Black))
		} else if m.From == E8 && m.To == C8 {
			// Move Black Queenside Rook: A8 -> D8
			next.Board.Clear(A8)
			next.Board.Set(D8, NewPiece(Rook, Black))
		}
	}

	// Handle En Passant capture
	if movingPiece.Type == Pawn && p.EnPassant != nil && m.To == *p.EnPassant {
		capSq := EnPassantCapturedSquare(m.To, movingPiece.Color)
		next.Board.Clear(capSq)
		isCapture = true
		next.Halfmove = 0
	}

	// Update piece at destination
	placedPiece := movingPiece
	if m.Promotion != nil && movingPiece.Type == Pawn {
		placedPiece = NewPiece(*m.Promotion, movingPiece.Color)
	}

	next.Board.Clear(m.From)
	next.Board.Set(m.To, placedPiece)

	// Update Castling Rights
	if movingPiece.Type == King {
		if movingPiece.Color == White {
			next.Castling = next.Castling.Remove(CastlingWhiteKingside | CastlingWhiteQueenside)
		} else {
			next.Castling = next.Castling.Remove(CastlingBlackKingside | CastlingBlackQueenside)
		}
	} else if movingPiece.Type == Rook {
		switch m.From {
		case A1:
			next.Castling = next.Castling.Remove(CastlingWhiteQueenside)
		case H1:
			next.Castling = next.Castling.Remove(CastlingWhiteKingside)
		case A8:
			next.Castling = next.Castling.Remove(CastlingBlackQueenside)
		case H8:
			next.Castling = next.Castling.Remove(CastlingBlackKingside)
		}
	}

	// If a rook was captured on its home square, remove castling right
	switch m.To {
	case A1:
		next.Castling = next.Castling.Remove(CastlingWhiteQueenside)
	case H1:
		next.Castling = next.Castling.Remove(CastlingWhiteKingside)
	case A8:
		next.Castling = next.Castling.Remove(CastlingBlackQueenside)
	case H8:
		next.Castling = next.Castling.Remove(CastlingBlackKingside)
	}

	// Update En Passant square for next move
	if movingPiece.Type == Pawn && (m.To.Rank()-m.From.Rank() == 2 || m.From.Rank()-m.To.Rank() == 2) {
		epSq := NewSquare(m.From.File(), (m.From.Rank()+m.To.Rank())/2)
		next.EnPassant = &epSq
	} else {
		next.EnPassant = nil
	}

	// Update Fullmove count
	if p.SideToMove == Black {
		next.Fullmove++
	}

	// Toggle active color
	next.SideToMove = p.SideToMove.Other()

	return next, nil
}
