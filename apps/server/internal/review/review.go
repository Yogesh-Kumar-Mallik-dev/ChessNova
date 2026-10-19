package review

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
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
	Phase           string             `json:"phase"` // "opening", "middlegame", "endgame"
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

type PhaseAccuracy struct {
	Opening    float64 `json:"opening"`
	Middlegame float64 `json:"middlegame"`
	Endgame    float64 `json:"endgame"`
}

type PerformanceRating struct {
	White int `json:"white"`
	Black int `json:"black"`
}

type ReviewResult struct {
	WhiteAccuracy float64                               `json:"whiteAccuracy"`
	BlackAccuracy float64                               `json:"blackAccuracy"`
	WhitePhases   PhaseAccuracy                         `json:"whitePhases"`
	BlackPhases   PhaseAccuracy                         `json:"blackPhases"`
	EstimatedElo  PerformanceRating                     `json:"estimatedElo"`
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

func countPiecesInFen(fen string) int {
	parts := strings.Split(fen, " ")
	if len(parts) == 0 {
		return 32
	}
	count := 0
	for _, ch := range parts[0] {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			count++
		}
	}
	return count
}

func determineGamePhase(ply int, fen string) string {
	parts := strings.Split(fen, " ")
	if len(parts) == 0 {
		return "opening"
	}
	pieceCount := 0
	hasWhiteQueen := false
	hasBlackQueen := false
	for _, ch := range parts[0] {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			pieceCount++
			if ch == 'Q' {
				hasWhiteQueen = true
			}
			if ch == 'q' {
				hasBlackQueen = true
			}
		}
	}

	// Endgame: <= 12 total pieces, or <= 16 pieces with queens traded off
	if pieceCount <= 12 || (!hasWhiteQueen && !hasBlackQueen && pieceCount <= 16) {
		return "endgame"
	}

	if ply <= 20 {
		return "opening"
	}

	return "middlegame"
}

// EstimatePlayerPerformanceElo calculates calibrated performance rating (400-2850)
func EstimatePlayerPerformanceElo(accuracy float64, blunders, mistakes, inaccuracies, movesCount int) int {
	if movesCount == 0 {
		return 400
	}
	// Base rating mathematically mapped to accuracy %
	base := 400.0 + (math.Pow(math.Max(0, accuracy-30)/70.0, 1.35) * 2200.0)

	// Blunder/mistake penalty normalized by game length
	blunderPenalty := float64(blunders) * 120.0
	mistakePenalty := float64(mistakes) * 55.0
	inaccuracyPenalty := float64(inaccuracies) * 20.0

	penalty := (blunderPenalty + mistakePenalty + inaccuracyPenalty) * (15.0 / float64(math.Max(10, float64(movesCount))))
	perf := base - penalty

	if perf < 400 {
		perf = 400
	}
	if perf > 2850 {
		perf = 2850
	}
	return int(math.Round(perf/10) * 10)
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
	if s.engine != nil && s.engine.RemoteURL() != "" {
		if res, err := s.analyzeRemote(ctx, pgnStr, nil); err == nil && res != nil {
			return res, nil
		}
	}

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
	if s.engine != nil && s.engine.RemoteURL() != "" {
		if res, err := s.analyzeRemote(ctx, "", moves); err == nil && res != nil {
			return res, nil
		}
	}

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
	prevEval, _ := s.engine.EvaluateFen(ctx, prevFEN, 14)

	evalGraph = append(evalGraph, EvalPoint{
		Ply:     0,
		ScoreCp: prevEval.ScoreCp,
		SAN:     "",
	})

	var totalWhiteAcc, totalBlackAcc float64
	var whiteCount, blackCount int

	// Phase-by-phase accuracy accumulators
	var wOpenAcc, wMidAcc, wEndAcc float64
	var wOpenCnt, wMidCnt, wEndCnt int
	var bOpenAcc, bMidAcc, bEndAcc float64
	var bOpenCnt, bMidCnt, bEndCnt int

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

		// Depth scaling: Boost evaluation depth to 14 by default, 16 in endgames (<= 10 pieces)
		evalDepth := 14
		if countPiecesInFen(m.FEN) <= 10 {
			evalDepth = 16
		}

		afterEval, _ := s.engine.EvaluateFen(ctx, m.FEN, evalDepth)

		winBefore := CentipawnsToWinChance(prevEval.ScoreCp)
		winAfter := CentipawnsToWinChance(afterEval.ScoreCp)

		bestUCI := prevEval.BestMove
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

		playedUCI := m.From + m.To
		var mObj chess.Move
		if hasPrevPos && m.SAN != "" {
			if pm, err := chess.ParseSAN(prevPos, m.SAN); err == nil {
				mObj = pm
				playedUCI = pm.UCI()
			} else if parsed, err := chess.ParseUCIMove(playedUCI); err == nil {
				mObj = parsed
			}
		} else if parsed, err := chess.ParseUCIMove(playedUCI); err == nil {
			mObj = parsed
		}

		openingName := ""
		if openingInfo != nil {
			openingName = openingInfo.Name
		}

		classification, explanation := ClassifyAndExplain(
			prevPos, currentPos, mObj, m.SAN, isWhite,
			prevEval.ScoreCp, afterEval.ScoreCp,
			winBefore, winAfter,
			bestSAN, bestUCI,
			isForced, isBook,
			openingName,
		)

		acc := calculateMoveAccuracy(winBefore, winAfter, isWhite)
		if classification == ClassBook || classification == ClassForced || classification == ClassBrilliant || classification == ClassGreat {
			acc = 100.0
		}

		stats[color][classification]++

		// Game phase classification
		phase := determineGamePhase(ply, prevFEN)
		if isWhite {
			totalWhiteAcc += acc
			whiteCount++
			switch phase {
			case "opening":
				wOpenAcc += acc
				wOpenCnt++
			case "middlegame":
				wMidAcc += acc
				wMidCnt++
			case "endgame":
				wEndAcc += acc
				wEndCnt++
			}
		} else {
			totalBlackAcc += acc
			blackCount++
			switch phase {
			case "opening":
				bOpenAcc += acc
				bOpenCnt++
			case "middlegame":
				bMidAcc += acc
				bMidCnt++
			case "endgame":
				bEndAcc += acc
				bEndCnt++
			}
		}

		reviewedMoves = append(reviewedMoves, ReviewedMove{
			Ply:             ply,
			MoveNumber:      moveNum,
			Color:           color,
			Phase:           phase,
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

	calcPhase := func(sum float64, count int, defaultVal float64) float64 {
		if count > 0 {
			return math.Round((sum/float64(count))*10) / 10
		}
		return defaultVal
	}

	whitePhases := PhaseAccuracy{
		Opening:    calcPhase(wOpenAcc, wOpenCnt, whiteAccuracy),
		Middlegame: calcPhase(wMidAcc, wMidCnt, whiteAccuracy),
		Endgame:    calcPhase(wEndAcc, wEndCnt, whiteAccuracy),
	}
	blackPhases := PhaseAccuracy{
		Opening:    calcPhase(bOpenAcc, bOpenCnt, blackAccuracy),
		Middlegame: calcPhase(bMidAcc, bMidCnt, blackAccuracy),
		Endgame:    calcPhase(bEndAcc, bEndCnt, blackAccuracy),
	}

	whiteElo := EstimatePlayerPerformanceElo(
		whiteAccuracy,
		stats["white"][ClassBlunder],
		stats["white"][ClassMistake],
		stats["white"][ClassInaccuracy],
		whiteCount,
	)
	blackElo := EstimatePlayerPerformanceElo(
		blackAccuracy,
		stats["black"][ClassBlunder],
		stats["black"][ClassMistake],
		stats["black"][ClassInaccuracy],
		blackCount,
	)

	// Always generate clean authoritative PGN for the reviewed game
	pgnBuilder := chess.NewPGNGame()
	if openingInfo != nil {
		pgnBuilder.SetHeader("Event", openingInfo.Name)
		pgnBuilder.SetHeader("ECO", openingInfo.ECO)
	} else {
		pgnBuilder.SetHeader("Event", "Game Review")
	}
	pgnBuilder.SetHeader("Site", "ChessPasant")
	pgnBuilder.SetHeader("Date", time.Now().UTC().Format("2006.01.02"))
	for _, rm := range reviewedMoves {
		pgnBuilder.AddMoveWithSquares(rm.Ply, rm.From, rm.To, rm.SAN, rm.FENAfter)
	}

	return &ReviewResult{
		WhiteAccuracy: whiteAccuracy,
		BlackAccuracy: blackAccuracy,
		WhitePhases:   whitePhases,
		BlackPhases:   blackPhases,
		EstimatedElo: PerformanceRating{
			White: whiteElo,
			Black: blackElo,
		},
		Opening:   openingInfo,
		Moves:     reviewedMoves,
		Stats:     stats,
		EvalGraph: evalGraph,
		PGN:       pgnBuilder.Export(),
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

func (s *ReviewService) analyzeRemote(ctx context.Context, pgnStr string, moves []InputMove) (*ReviewResult, error) {
	if s.engine == nil || s.engine.RemoteURL() == "" {
		return nil, fmt.Errorf("remote engine not configured")
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"pgn":   pgnStr,
		"moves": moves,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", s.engine.RemoteURL()+"/review", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if sec := s.engine.RemoteSecret(); sec != "" {
		req.Header.Set("X-Engine-Secret", sec)
	}

	client := s.engine.HTTPClient()
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote review returned status %d", resp.StatusCode)
	}

	var res ReviewResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}
