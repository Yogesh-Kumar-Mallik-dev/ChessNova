package review

import (
	"context"
	"fmt"
	"math"
	"time"

	"chess-platform/server/internal/chess"
	"chess-platform/server/internal/engine"
	"chess-platform/server/internal/openings"
)

type MoveClassification string

const (
	ClassBrilliant  MoveClassification = "brilliant"  // !!
	ClassGreat      MoveClassification = "great"      // !
	ClassBest       MoveClassification = "best"       // ★
	ClassExcellent  MoveClassification = "excellent"  // ✓
	ClassGood       MoveClassification = "good"       // ✓
	ClassBook       MoveClassification = "book"       // 📖
	ClassInaccuracy MoveClassification = "inaccuracy" // ?!
	ClassMistake    MoveClassification = "mistake"    // ?
	ClassMiss       MoveClassification = "miss"       // ⨉
	ClassBlunder    MoveClassification = "blunder"    // ??
	ClassForced     MoveClassification = "forced"     // □
)

type ReviewedMove struct {
	Ply             int                `json:"ply"`
	MoveNumber      int                `json:"moveNumber"`
	Color           string             `json:"color"`
	From            string             `json:"from"`
	To              string             `json:"to"`
	SAN             string             `json:"san"`
	FENBefore       string             `json:"fenBefore"`
	FENAfter        string             `json:"fenAfter"`
	EvalBefore      int                `json:"evalBefore"`
	EvalAfter       int                `json:"evalAfter"`
	WinChanceBefore float64            `json:"winChanceBefore"`
	WinChanceAfter  float64            `json:"winChanceAfter"`
	Accuracy        float64            `json:"accuracy"`
	Classification  MoveClassification `json:"classification"`
	BestMoveSAN     string             `json:"bestMoveSan"`
	BestMoveUCI     string             `json:"bestMoveUci"`
	Explanation     string             `json:"explanation"`
}

type ReviewResult struct {
	WhiteAccuracy float64                               `json:"whiteAccuracy"`
	BlackAccuracy float64                               `json:"blackAccuracy"`
	Opening       *openings.Opening                     `json:"opening,omitempty"`
	Moves         []ReviewedMove                        `json:"moves"`
	Stats         map[string]map[MoveClassification]int `json:"stats"`
	EvalGraph     []EvalPoint                           `json:"evalGraph"`
	PGN           string                                `json:"pgn,omitempty"`
	Headers       map[string]string                     `json:"headers,omitempty"`
	White         string                                `json:"white,omitempty"`
	Black         string                                `json:"black,omitempty"`
	Result        string                                `json:"result,omitempty"`
}

type EvalPoint struct {
	Ply     int    `json:"ply"`
	ScoreCp int    `json:"scoreCp"`
	SAN     string `json:"san"`
}

type InputMove struct {
	From string `json:"from"`
	To   string `json:"to"`
	SAN  string `json:"san"`
	FEN  string `json:"fen"`
}

type ReviewService struct {
	engine *engine.StockfishEngine
}

func NewReviewService(eng *engine.StockfishEngine) *ReviewService {
	return &ReviewService{engine: eng}
}

// CentipawnsToWinChance implements the official Chess.com CAPS winning probability curve
func CentipawnsToWinChance(cp int) float64 {
	clamped := float64(cp)
	if clamped > 1000 {
		clamped = 1000
	} else if clamped < -1000 {
		clamped = -1000
	}

	winProb := 50.0 + 50.0*(2.0/(1.0+math.Exp(-0.00368208*clamped))-1.0)
	if winProb < 0 {
		return 0
	}
	if winProb > 100 {
		return 100
	}
	return winProb
}

func calculateMoveAccuracy(winBefore, winAfter float64, isWhite bool) float64 {
	playerWinBefore := winBefore
	playerWinAfter := winAfter
	if !isWhite {
		playerWinBefore = 100.0 - winBefore
		playerWinAfter = 100.0 - winAfter
	}

	delta := playerWinBefore - playerWinAfter
	if delta <= 0.05 {
		return 100.0
	}

	acc := 103.1668*math.Exp(-0.04354*delta) - 3.1668
	if acc < 0 {
		acc = 0
	}
	if acc > 100 {
		acc = 100
	}
	return math.Round(acc*10) / 10
}

// isPieceSacrifice returns true if a minor or major piece moved into an attacked square
func isPieceSacrifice(pos chess.Position, m chess.Move) bool {
	movedPiece := pos.Board.Get(m.From)
	if movedPiece.Type != chess.Knight && movedPiece.Type != chess.Bishop &&
		movedPiece.Type != chess.Rook && movedPiece.Type != chess.Queen {
		return false
	}

	nextPos, err := chess.ApplyMove(pos, m)
	if err != nil {
		return false
	}

	oppMoves := chess.LegalMoves(nextPos)
	for _, oppM := range oppMoves {
		if oppM.To == m.To {
			attacker := nextPos.Board.Get(oppM.From)
			if attacker.Type == chess.Pawn || attacker.Type < movedPiece.Type {
				return true
			}
		}
	}
	return false
}

// AnalyzePGN performs production-grade game review directly from a standard PGN string
func (s *ReviewService) AnalyzePGN(ctx context.Context, pgnStr string) (*ReviewResult, error) {
	parsedGame, err := chess.ParsePGN(pgnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PGN: %w", err)
	}

	if len(parsedGame.Moves) == 0 {
		return nil, fmt.Errorf("PGN contains no moves to review")
	}

	inputMoves := make([]InputMove, len(parsedGame.Moves))
	for i, m := range parsedGame.Moves {
		inputMoves[i] = InputMove{
			From: m.From,
			To:   m.To,
			SAN:  m.SAN,
			FEN:  m.FEN,
		}
	}

	result, err := s.AnalyzeMoves(ctx, inputMoves)
	if err != nil {
		return nil, err
	}

	result.PGN = pgnStr
	result.Headers = parsedGame.Headers
	if w, ok := parsedGame.Headers["White"]; ok {
		result.White = w
	}
	if b, ok := parsedGame.Headers["Black"]; ok {
		result.Black = b
	}
	if r, ok := parsedGame.Headers["Result"]; ok {
		result.Result = r
	} else {
		result.Result = parsedGame.Result
	}

	return result, nil
}

func (s *ReviewService) AnalyzeMoves(ctx context.Context, moves []InputMove) (*ReviewResult, error) {
	sans := make([]string, len(moves))
	for i, m := range moves {
		sans[i] = m.SAN
	}

	openingInfo, _ := openings.IdentifyOpening(sans)

	stats := map[string]map[MoveClassification]int{
		"white": initStatsMap(),
		"black": initStatsMap(),
	}

	reviewedMoves := make([]ReviewedMove, 0, len(moves))
	evalGraph := make([]EvalPoint, 0, len(moves)+1)

	currentPos := chess.InitialPosition()
	prevFEN := chess.StartingFEN
	prevEval, _ := s.engine.EvaluateFen(ctx, prevFEN, 10)

	evalGraph = append(evalGraph, EvalPoint{
		Ply:     0,
		ScoreCp: prevEval.ScoreCp,
		SAN:     "",
	})

	var totalWhiteAcc, totalBlackAcc float64
	var whiteCount, blackCount int

	for i, m := range moves {
		ply := i + 1
		moveNum := (i / 2) + 1
		isWhite := i%2 == 0
		color := "white"
		if !isWhite {
			color = "black"
		}

		// Ensure From/To and FEN are populated correctly from SAN or positions
		if (m.From == "" || m.To == "") && m.SAN != "" {
			if parsedM, err := chess.ParseSAN(currentPos, m.SAN); err == nil {
				m.From = parsedM.From.String()
				m.To = parsedM.To.String()
			}
		}

		if m.FEN == "" {
			if m.SAN != "" {
				if parsedM, err := chess.ParseSAN(currentPos, m.SAN); err == nil {
					if nextPos, err := chess.ApplyMove(currentPos, parsedM); err == nil {
						currentPos = nextPos
						m.FEN = chess.ToFEN(currentPos)
					}
				}
			} else if len(m.From) >= 2 && len(m.To) >= 2 {
				fromSq, err1 := chess.ParseSquare(m.From)
				toSq, err2 := chess.ParseSquare(m.To)
				if err1 == nil && err2 == nil {
					mObj := chess.Move{From: fromSq, To: toSq}
					if nextPos, err := chess.ApplyMove(currentPos, mObj); err == nil {
						currentPos = nextPos
						m.FEN = chess.ToFEN(currentPos)
					}
				}
			}
		} else {
			if pos, err := chess.FromFEN(m.FEN); err == nil {
				currentPos = pos
			}
		}

		afterEval, _ := s.engine.EvaluateFen(ctx, m.FEN, 10)

		winBefore := CentipawnsToWinChance(prevEval.ScoreCp)
		winAfter := CentipawnsToWinChance(afterEval.ScoreCp)

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

		bestUCI := prevEval.BestMove
		playedUCI := m.From + m.To
		isBest := playedUCI == bestUCI
		isBook := openings.IsBookMove(sans, i)

		// Determine best move in SAN for clear human coaching
		bestSAN := bestUCI
		var prevPos chess.Position
		hasPrevPos := false
		if p, err := chess.FromFEN(prevFEN); err == nil {
			prevPos = p
			hasPrevPos = true
			if len(bestUCI) >= 4 {
				if mObj, err := chess.ParseUCIMove(bestUCI); err == nil {
					if s, err := chess.MoveToSAN(prevPos, mObj); err == nil {
						bestSAN = s
					}
				}
			}
		}

		// Check if position had only 1 legal move (Forced)
		isForced := false
		if hasPrevPos && len(chess.LegalMoves(prevPos)) == 1 {
			isForced = true
		}

		mObj, _ := chess.ParseUCIMove(playedUCI)
		isSacrifice := hasPrevPos && isPieceSacrifice(prevPos, mObj)

		classification := ClassGood
		explanation := "Good move. Solid play that preserves the position."

		if isForced {
			classification = ClassForced
			explanation = fmt.Sprintf("Forced move. %s is the only legal response available.", m.SAN)
		} else if isBook {
			classification = ClassBook
			if openingInfo != nil {
				explanation = fmt.Sprintf("Book move. Standard opening theory (%s).", openingInfo.Name)
			} else {
				explanation = "Book move. Standard opening theory."
			}
		} else if isBest && isSacrifice && playerWinAfter >= 50.0 {
			classification = ClassBrilliant
			explanation = fmt.Sprintf("Brilliant move!! You sacrificed material on %s to seize a decisive advantage.", m.To)
		} else if isBest && playerWinBefore < 55.0 && playerWinAfter >= 60.0 && winDelta <= 0.2 {
			classification = ClassGreat
			explanation = fmt.Sprintf("Great move! %s was the solitary winning continuation in a complex position.", m.SAN)
		} else if isBest {
			classification = ClassBest
			explanation = fmt.Sprintf("Best move! You found the optimal continuation (%s).", m.SAN)
		} else if winDelta <= 1.5 {
			classification = ClassExcellent
			explanation = "Excellent move! A very strong choice that keeps the pressure."
		} else if winDelta <= 5.0 {
			classification = ClassGood
			explanation = "Good move. Solid choice that preserves the balance."
		} else if winDelta <= 11.5 {
			classification = ClassInaccuracy
			explanation = fmt.Sprintf("Inaccuracy. %s would have maintained a stronger grip on the position.", bestSAN)
		} else if winDelta <= 21.0 {
			classification = ClassMistake
			explanation = fmt.Sprintf("Mistake. %s was a better choice to preserve your advantage.", bestSAN)
		} else if playerWinBefore >= 68 && playerWinAfter < 50 {
			classification = ClassMiss
			explanation = fmt.Sprintf("Missed win! You had a decisive winning opportunity with %s.", bestSAN)
		} else {
			classification = ClassBlunder
			explanation = fmt.Sprintf("Blunder! %s gives away the advantage. %s was best.", m.SAN, bestSAN)
		}

		acc := calculateMoveAccuracy(winBefore, winAfter, isWhite)
		if classification == ClassBook || classification == ClassForced || classification == ClassBrilliant || classification == ClassGreat {
			acc = 100.0
		}

		stats[color][classification]++

		if isWhite {
			totalWhiteAcc += acc
			whiteCount++
		} else {
			totalBlackAcc += acc
			blackCount++
		}

		reviewedMoves = append(reviewedMoves, ReviewedMove{
			Ply:             ply,
			MoveNumber:      moveNum,
			Color:           color,
			From:            m.From,
			To:              m.To,
			SAN:             m.SAN,
			FENBefore:       prevFEN,
			FENAfter:        m.FEN,
			EvalBefore:      prevEval.ScoreCp,
			EvalAfter:       afterEval.ScoreCp,
			WinChanceBefore: math.Round(winBefore*10) / 10,
			WinChanceAfter:  math.Round(winAfter*10) / 10,
			Accuracy:        acc,
			Classification:  classification,
			BestMoveSAN:     bestSAN,
			BestMoveUCI:     bestUCI,
			Explanation:     explanation,
		})

		evalGraph = append(evalGraph, EvalPoint{
			Ply:     ply,
			ScoreCp: afterEval.ScoreCp,
			SAN:     m.SAN,
		})

		prevFEN = m.FEN
		prevEval = afterEval
	}

	whiteAccuracy := 100.0
	if whiteCount > 0 {
		whiteAccuracy = math.Round((totalWhiteAcc/float64(whiteCount))*10) / 10
	}

	blackAccuracy := 100.0
	if blackCount > 0 {
		blackAccuracy = math.Round((totalBlackAcc/float64(blackCount))*10) / 10
	}

	// Always generate clean authoritative PGN for the reviewed game
	pgnBuilder := chess.NewPGNGame()
	if openingInfo != nil {
		pgnBuilder.SetHeader("Event", openingInfo.Name)
		pgnBuilder.SetHeader("ECO", openingInfo.ECO)
	} else {
		pgnBuilder.SetHeader("Event", "Game Review")
	}
	pgnBuilder.SetHeader("Site", "ChessNova")
	pgnBuilder.SetHeader("Date", time.Now().UTC().Format("2006.01.02"))
	for _, rm := range reviewedMoves {
		pgnBuilder.AddMoveWithSquares(rm.Ply, rm.From, rm.To, rm.SAN, rm.FENAfter)
	}

	return &ReviewResult{
		WhiteAccuracy: whiteAccuracy,
		BlackAccuracy: blackAccuracy,
		Opening:       openingInfo,
		Moves:         reviewedMoves,
		Stats:         stats,
		EvalGraph:     evalGraph,
		PGN:           pgnBuilder.Export(),
	}, nil
}

func initStatsMap() map[MoveClassification]int {
	return map[MoveClassification]int{
		ClassBrilliant:  0,
		ClassGreat:      0,
		ClassBest:       0,
		ClassExcellent:  0,
		ClassGood:       0,
		ClassBook:       0,
		ClassInaccuracy: 0,
		ClassMistake:    0,
		ClassMiss:       0,
		ClassBlunder:    0,
		ClassForced:     0,
	}
}
