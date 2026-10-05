package review

import (
	"context"
	"math"

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
	WhiteAccuracy float64                       `json:"whiteAccuracy"`
	BlackAccuracy float64                       `json:"blackAccuracy"`
	Opening       *openings.Opening             `json:"opening,omitempty"`
	Moves         []ReviewedMove                `json:"moves"`
	Stats         map[string]map[MoveClassification]int `json:"stats"`
	EvalGraph     []EvalPoint                   `json:"evalGraph"`
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
		return 0
	}
	if acc > 100 {
		return 100
	}
	return math.Round(acc*10) / 10
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

		if m.FEN == "" {
			if len(m.From) >= 2 && len(m.To) >= 2 {
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

		classification := ClassGood
		explanation := "Good move. A solid choice that maintains balance."

		if isBook {
			classification = ClassBook
			explanation = "Book move. Standard opening theory."
			if openingInfo != nil {
				explanation = "Book move. Follows " + openingInfo.Name
			}
		} else if isBest || winDelta <= 0.5 {
			classification = ClassBest
			explanation = "Best move! You found the optimal continuation."
		} else if winDelta <= 2.2 {
			classification = ClassExcellent
			explanation = "Excellent move! A very strong choice that keeps the advantage."
		} else if winDelta <= 5.5 {
			classification = ClassGood
			explanation = "Good move. Solid play that keeps your position safe."
		} else if winDelta <= 11.5 {
			classification = ClassInaccuracy
			explanation = "Inaccuracy. There were more active alternatives."
		} else if winDelta <= 21.0 {
			classification = ClassMistake
			explanation = "Mistake. This concedes territory or tactical initiative."
		} else if playerWinBefore >= 68 && playerWinAfter < 50 {
			classification = ClassMiss
			explanation = "Missed win! You had a decisive tactical advantage."
		} else {
			classification = ClassBlunder
			explanation = "Blunder! This loses significant material or swings the game."
		}

		acc := calculateMoveAccuracy(winBefore, winAfter, isWhite)
		if classification == ClassBook {
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

		bestSAN := bestUCI // fallback
		if pos, err := chess.FromFEN(prevFEN); err == nil && len(bestUCI) >= 4 {
			fromSq, _ := chess.ParseSquare(bestUCI[:2])
			toSq, _ := chess.ParseSquare(bestUCI[2:4])
			mObj := chess.Move{From: fromSq, To: toSq}
			if s, err := chess.MoveToSAN(pos, mObj); err == nil {
				bestSAN = s
			}
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

	return &ReviewResult{
		WhiteAccuracy: whiteAccuracy,
		BlackAccuracy: blackAccuracy,
		Opening:       openingInfo,
		Moves:         reviewedMoves,
		Stats:         stats,
		EvalGraph:     evalGraph,
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
