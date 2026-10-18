package review

import (
	"fmt"
	"strings"

	"chess-platform/server/internal/chess"
)

// TacticalMotif describes a tactical pattern discovered during move evaluation
type TacticalMotif string

const (
	MotifCheckmate        TacticalMotif = "checkmate"
	MotifMissedMate       TacticalMotif = "missed_mate"
	MotifMissedWin        TacticalMotif = "missed_win"
	MotifBrilliantSac     TacticalMotif = "brilliant_sacrifice"
	MotifKnightFork       TacticalMotif = "knight_fork"
	MotifPawnFork         TacticalMotif = "pawn_fork"
	MotifDiscoveredCheck  TacticalMotif = "discovered_check"
	MotifDoubleCheck      TacticalMotif = "double_check"
	MotifHangingBlunder   TacticalMotif = "hanging_blunder"
	MotifCapturedHanging  TacticalMotif = "captured_hanging"
	MotifQueenTradeWin    TacticalMotif = "queen_trade_win"
	MotifPromotion        TacticalMotif = "promotion"
	MotifBook             TacticalMotif = "book"
	MotifForced           TacticalMotif = "forced"
	MotifGreatFind        TacticalMotif = "great_find"
	MotifBest             TacticalMotif = "best"
	MotifExcellent        TacticalMotif = "excellent"
	MotifGood             TacticalMotif = "good"
	MotifInaccuracy       TacticalMotif = "inaccuracy"
	MotifMistake          TacticalMotif = "mistake"
	MotifBlunder          TacticalMotif = "blunder"
)

// ClassifyAndExplain analyzes positions and evaluations to produce Chess.com-grade classification and coaching commentary
func ClassifyAndExplain(
	posBefore, posAfter chess.Position,
	m chess.Move,
	san string,
	isWhite bool,
	evalBefore, evalAfter int,
	winBefore, winAfter float64,
	bestSAN, bestUCI string,
	isForced bool,
	isBook bool,
	openingName string,
) (MoveClassification, string) {
	playerWinBefore := winBefore
	playerWinAfter := winAfter
	if !isWhite {
		playerWinBefore = 100.0 - winBefore
		playerWinAfter = 100.0 - winAfter
	}

	winDelta := playerWinBefore - playerWinAfter
	if winDelta < 0 {
		winDelta = 0
	}

	playedUCI := m.From.String() + m.To.String()
	isBest := playedUCI == bestUCI
	playerColor := posBefore.SideToMove
	oppColor := playerColor.Other()

	// 1. Forced Move
	if isForced {
		return ClassForced, fmt.Sprintf("Forced move. %s is the only legal response available.", san)
	}

	// 2. Checkmate Delivered
	if len(chess.LegalMoves(posAfter)) == 0 && chess.IsCheck(posAfter) {
		return ClassBest, fmt.Sprintf("Checkmate! %s delivers the final winning blow.", san)
	}

	// 3. Official Opening Theory (Book)
	if isBook {
		if openingName != "" {
			return ClassBook, fmt.Sprintf("Book move. Standard opening theory in the %s.", openingName)
		}
		return ClassBook, "Book move. Standard opening theory."
	}

	// 4. Brilliant Move (!!) - Sound piece sacrifice leading to winning advantage
	isSacrifice := isPieceSacrifice(posBefore, m)
	if isBest && isSacrifice && playerWinAfter >= 52.0 {
		piece := posBefore.Board.Get(m.From)
		return ClassBrilliant, fmt.Sprintf("Brilliant move!! You sacrificed your %s on %s to seize a decisive advantage.", piece.Type.String(), m.To.String())
	}

	// 5. Great Move (!) - Solitary winning path in a sharp, contested position
	if isBest && playerWinBefore < 55.0 && playerWinAfter >= 60.0 && winDelta <= 0.2 {
		return ClassGreat, fmt.Sprintf("Great move! %s was the solitary winning continuation in a complex position.", san)
	}

	// 6. Missed Win / Missed Mate (⨉)
	if (evalBefore >= 9000 || playerWinBefore >= 70.0) && playerWinAfter < 50.0 {
		if evalBefore >= 9000 {
			return ClassMiss, fmt.Sprintf("Missed win! You had a forced checkmate with %s.", bestSAN)
		}
		return ClassMiss, fmt.Sprintf("Missed win! You had a decisive winning opportunity with %s.", bestSAN)
	}

	// Check tactical motifs on the board
	isCheck := chess.IsCheck(posAfter)
	isDiscCheck, isDblCheck := detectDiscoveredCheck(posAfter, m.To, oppColor)
	isFork, forkDesc := detectFork(posAfter.Board, m.To, playerColor)
	isHanging, hangPiece := detectHangingBlunder(posBefore, posAfter, m, playerColor)

	// 7. Blunder (??) - Major loss of evaluation
	if winDelta > 21.0 {
		if isHanging {
			return ClassBlunder, fmt.Sprintf("Blunder?? %s leaves your %s on %s completely undefended. %s was best.", san, hangPiece, m.To.String(), bestSAN)
		}
		return ClassBlunder, fmt.Sprintf("Blunder?? %s significantly alters the game. %s was the best continuation.", san, bestSAN)
	}

	// 8. Mistake (?) - Notable tactical or positional concession
	if winDelta > 11.5 {
		if isHanging {
			return ClassMistake, fmt.Sprintf("Mistake? %s leaves your %s under heavy attack. %s was safer.", san, hangPiece, bestSAN)
		}
		return ClassMistake, fmt.Sprintf("Mistake? %s gives away the edge. %s was a much stronger continuation.", san, bestSAN)
	}

	// 9. Inaccuracy (?!) - Suboptimal move
	if winDelta > 5.0 {
		return ClassInaccuracy, fmt.Sprintf("Inaccuracy?! %s would have maintained a tighter grip on the position.", bestSAN)
	}

	// 10. Best Move (★) / Excellent (✓) / Good (✓)
	if isBest {
		if isDblCheck {
			return ClassBest, fmt.Sprintf("Best move! %s delivers a devastating double check.", san)
		}
		if isDiscCheck {
			return ClassBest, fmt.Sprintf("Best move! %s unleashes a powerful discovered check.", san)
		}
		if isFork {
			return ClassBest, fmt.Sprintf("Best move! %s executes a tactical fork attacking %s.", san, forkDesc)
		}
		if isCheck {
			return ClassBest, fmt.Sprintf("Best move! %s puts the enemy King in check with tempo.", san)
		}
		if m.Promotion != nil {
			return ClassBest, fmt.Sprintf("Best move! Promotes the pawn to a %s.", m.Promotion.String())
		}
		if isQueenTrade(posBefore, posAfter) && playerWinAfter >= 60.0 {
			return ClassBest, fmt.Sprintf("Best move! Trading Queens simplifies directly into a winning endgame.")
		}
		return ClassBest, fmt.Sprintf("Best move! %s is the top engine recommendation.", san)
	}

	if winDelta <= 1.5 {
		if isFork {
			return ClassExcellent, fmt.Sprintf("Excellent move! %s attacks multiple enemy targets (%s).", san, forkDesc)
		}
		if isCheck {
			return ClassExcellent, fmt.Sprintf("Excellent move! %s keeps the offensive pressure on the King.", san)
		}
		return ClassExcellent, "Excellent move! A very strong choice that keeps the pressure."
	}

	// Default solid play
	return ClassGood, "Good move. Solid play that preserves the position."
}

// detectFork checks if a piece on `sq` attacks 2 or more valuable enemy pieces
func detectFork(b chess.Board, sq chess.Square, attacker chess.Color) (bool, string) {
	piece := b.Get(sq)
	if piece.IsEmpty() || piece.Color != attacker {
		return false, ""
	}

	oppColor := attacker.Other()
	var targets []string

	// Check if attacking King
	oppKingSq := b.KingSquare(oppColor)
	if oppKingSq.IsValid() && isDirectlyAttackedBy(b, sq, oppKingSq) {
		targets = append(targets, "King")
	}

	// Check attacks on other enemy pieces (Queen, Rook, Bishop, Knight)
	for s := chess.Square(0); s < 64; s++ {
		if s == oppKingSq {
			continue
		}
		targetPiece := b.Get(s)
		if !targetPiece.IsEmpty() && targetPiece.Color == oppColor && targetPiece.Type != chess.Pawn {
			if isDirectlyAttackedBy(b, sq, s) {
				targets = append(targets, strings.ToLower(targetPiece.Type.String()))
			}
		}
	}

	if len(targets) >= 2 {
		return true, strings.Join(targets[:2], " and ")
	}
	return false, ""
}

// isDirectlyAttackedBy tests if piece on fromSq directly attacks toSq
func isDirectlyAttackedBy(b chess.Board, fromSq, toSq chess.Square) bool {
	piece := b.Get(fromSq)
	if piece.IsEmpty() || fromSq == toSq {
		return false
	}
	color := piece.Color

	f1, r1 := fromSq.File(), fromSq.Rank()
	f2, r2 := toSq.File(), toSq.Rank()
	df := f2 - f1
	dr := r2 - r1
	absDf := df
	if absDf < 0 {
		absDf = -absDf
	}
	absDr := dr
	if absDr < 0 {
		absDr = -absDr
	}

	switch piece.Type {
	case chess.Knight:
		return (absDf == 1 && absDr == 2) || (absDf == 2 && absDr == 1)
	case chess.King:
		return absDf <= 1 && absDr <= 1
	case chess.Pawn:
		pawnDir := 1
		if color == chess.Black {
			pawnDir = -1
		}
		return dr == pawnDir && absDf == 1
	case chess.Bishop, chess.Rook, chess.Queen:
		isDiag := absDf == absDr && absDf > 0
		isOrth := (absDf == 0 && absDr > 0) || (absDr == 0 && absDf > 0)

		if piece.Type == chess.Bishop && !isDiag {
			return false
		}
		if piece.Type == chess.Rook && !isOrth {
			return false
		}
		if piece.Type == chess.Queen && !isDiag && !isOrth {
			return false
		}

		stepF := 0
		if df > 0 {
			stepF = 1
		} else if df < 0 {
			stepF = -1
		}
		stepR := 0
		if dr > 0 {
			stepR = 1
		} else if dr < 0 {
			stepR = -1
		}

		curF, curR := f1+stepF, r1+stepR
		for curF != f2 || curR != r2 {
			if !b.Get(chess.NewSquare(curF, curR)).IsEmpty() {
				return false // Ray is blocked
			}
			curF += stepF
			curR += stepR
		}
		return true
	}
	return false
}

// detectHangingBlunder checks if the moved piece landed on an undefended attacked square
func detectHangingBlunder(posBefore, posAfter chess.Position, m chess.Move, playerColor chess.Color) (bool, string) {
	movedPiece := posAfter.Board.Get(m.To)
	if movedPiece.IsEmpty() || movedPiece.Type == chess.King || movedPiece.Type == chess.Pawn {
		return false, ""
	}

	oppColor := playerColor.Other()
	isUnderAttack := chess.IsSquareAttacked(&posAfter.Board, m.To, oppColor)
	if !isUnderAttack {
		return false, ""
	}

	isDefended := chess.IsSquareAttacked(&posAfter.Board, m.To, playerColor)
	if !isDefended {
		return true, strings.ToLower(movedPiece.Type.String())
	}

	// Also check if high-value piece (Queen/Rook) is attacked by low-value piece (Pawn)
	if movedPiece.Type == chess.Queen || movedPiece.Type == chess.Rook {
		pawnAttacked := false
		f := m.To.File()
		r := m.To.Rank()
		pawnRank := r + 1
		if oppColor == chess.White {
			pawnRank = r - 1
		}
		for _, df := range []int{-1, 1} {
			pSq := chess.NewSquare(f+df, pawnRank)
			if pSq.IsValid() {
				p := posAfter.Board.Get(pSq)
				if p.Type == chess.Pawn && p.Color == oppColor {
					pawnAttacked = true
					break
				}
			}
		}
		if pawnAttacked {
			return true, strings.ToLower(movedPiece.Type.String())
		}
	}

	return false, ""
}

// detectDiscoveredCheck determines if check was delivered by a piece other than the moved piece
func detectDiscoveredCheck(posAfter chess.Position, movedTo chess.Square, oppColor chess.Color) (bool, bool) {
	if !chess.IsCheck(posAfter) {
		return false, false
	}

	attacker := oppColor.Other()
	kingSq := posAfter.Board.KingSquare(oppColor)
	if !kingSq.IsValid() {
		return false, false
	}

	movedPieceAttacks := isDirectlyAttackedBy(posAfter.Board, movedTo, kingSq)
	// If moved piece does not attack king directly, but king is in check -> 100% discovered check!
	if !movedPieceAttacks {
		return true, false
	}

	// Check if another friendly piece also attacks king -> double check!
	var checkingCount int
	for s := chess.Square(0); s < 64; s++ {
		p := posAfter.Board.Get(s)
		if !p.IsEmpty() && p.Color == attacker {
			if isDirectlyAttackedBy(posAfter.Board, s, kingSq) {
				checkingCount++
			}
		}
	}

	if checkingCount >= 2 {
		return true, true // Double check
	}

	return false, false
}

// isQueenTrade returns true if Queens were present before and absent after
func isQueenTrade(posBefore, posAfter chess.Position) bool {
	var whiteQBefore, blackQBefore bool
	var whiteQAfter, blackQAfter bool

	for sq := chess.Square(0); sq < 64; sq++ {
		p1 := posBefore.Board.Get(sq)
		if p1.Type == chess.Queen {
			if p1.Color == chess.White {
				whiteQBefore = true
			} else {
				blackQBefore = true
			}
		}
		p2 := posAfter.Board.Get(sq)
		if p2.Type == chess.Queen {
			if p2.Color == chess.White {
				whiteQAfter = true
			} else {
				blackQAfter = true
			}
		}
	}

	return whiteQBefore && blackQBefore && !whiteQAfter && !blackQAfter
}
