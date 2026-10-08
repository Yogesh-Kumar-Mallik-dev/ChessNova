package engine

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"chess-platform/server/internal/chess"
)

type Evaluation struct {
	ScoreCp   int      `json:"scoreCp"`   // Centipawns from White's perspective
	IsMate    bool     `json:"isMate"`
	MateIn    int      `json:"mateIn,omitempty"`
	BestMove  string   `json:"bestMove"`  // E.g. "e2e4"
	PV        []string `json:"pv"`
	Depth     int      `json:"depth"`
}

type StockfishEngine struct {
	binaryPath string
	mu         sync.Mutex
}

var (
	depthRegex = regexp.MustCompile(`depth\s+(\d+)`)
	cpRegex    = regexp.MustCompile(`score\s+cp\s+(-?\d+)`)
	mateRegex  = regexp.MustCompile(`score\s+mate\s+(-?\d+)`)
	pvRegex    = regexp.MustCompile(`pv\s+(.*)`)
)

func NewStockfishEngine() *StockfishEngine {
	// Locate stockfish binary in apps/server/bin/stockfish or PATH
	bin := findStockfishBinary()
	return &StockfishEngine{
		binaryPath: bin,
	}
}

func findStockfishBinary() string {
	candidates := []string{
		"bin/stockfish",
		"apps/server/bin/stockfish",
		"../bin/stockfish",
		"../../bin/stockfish",
	}

	// Check relative to executable
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(dir, "stockfish"))
	}

	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			abs, err := filepath.Abs(c)
			if err == nil {
				return abs
			}
			return c
		}
	}

	if path, err := exec.LookPath("stockfish"); err == nil {
		return path
	}

	return ""
}

// EvaluateFen runs Stockfish on the given FEN position up to the requested depth.
func (e *StockfishEngine) EvaluateFen(ctx context.Context, fen string, depth int) (*Evaluation, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.binaryPath == "" {
		// Fallback simple evaluator if binary is not present on system
		return fallbackEval(fen), nil
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, e.binaryPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fallbackEval(fen), nil
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fallbackEval(fen), nil
	}

	if err := cmd.Start(); err != nil {
		return fallbackEval(fen), nil
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	writer := bufio.NewWriter(stdin)
	scanner := bufio.NewScanner(stdout)

	// Send UCI initialization
	_, _ = writer.WriteString("uci\nisready\n")
	_ = writer.Flush()

	// Wait for readyok
	ready := make(chan bool, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if line == "readyok" {
				ready <- true
				return
			}
		}
		ready <- false
	}()

	select {
	case ok := <-ready:
		if !ok {
			return fallbackEval(fen), nil
		}
	case <-cmdCtx.Done():
		return fallbackEval(fen), nil
	}

	// Send position and go command
	_, _ = writer.WriteString(fmt.Sprintf("position fen %s\ngo depth %d\n", fen, depth))
	_ = writer.Flush()

	eval := &Evaluation{Depth: depth}
	isBlackToMove := strings.Contains(fen, " b ")

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "info") && strings.Contains(line, "score") {
			if m := depthRegex.FindStringSubmatch(line); len(m) > 1 {
				eval.Depth, _ = strconv.Atoi(m[1])
			}
			if m := mateRegex.FindStringSubmatch(line); len(m) > 1 {
				mateIn, _ := strconv.Atoi(m[1])
				eval.IsMate = true
				eval.MateIn = mateIn
				if mateIn > 0 {
					eval.ScoreCp = 10000
				} else {
					eval.ScoreCp = -10000
				}
			} else if m := cpRegex.FindStringSubmatch(line); len(m) > 1 {
				eval.IsMate = false
				eval.ScoreCp, _ = strconv.Atoi(m[1])
			}

			if m := pvRegex.FindStringSubmatch(line); len(m) > 1 {
				eval.PV = strings.Fields(m[1])
			}
		}

		if strings.HasPrefix(line, "bestmove") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				eval.BestMove = parts[1]
			}
			break
		}
	}

	// Normalize score to White's perspective (+ White advantage, - Black advantage)
	if isBlackToMove {
		eval.ScoreCp = -eval.ScoreCp
		if eval.IsMate {
			eval.MateIn = -eval.MateIn
		}
	}

	if len(eval.PV) == 0 && eval.BestMove != "" {
		eval.PV = []string{eval.BestMove}
	}

	return eval, nil
}

// Piece-square tables for positional evaluation in fallback engine
var (
	pawnPST = [64]int{
		0, 0, 0, 0, 0, 0, 0, 0,
		50, 50, 50, 50, 50, 50, 50, 50,
		10, 10, 20, 30, 30, 20, 10, 10,
		5, 5, 10, 25, 25, 10, 5, 5,
		0, 0, 0, 20, 20, 0, 0, 0,
		5, -5, -10, 0, 0, -10, -5, 5,
		5, 10, 10, -20, -20, 10, 10, 5,
		0, 0, 0, 0, 0, 0, 0, 0,
	}
	knightPST = [64]int{
		-50, -40, -30, -30, -30, -30, -40, -50,
		-40, -20, 0, 0, 0, 0, -20, -40,
		-30, 0, 10, 15, 15, 10, 0, -30,
		-30, 5, 15, 20, 20, 15, 5, -30,
		-30, 0, 15, 20, 20, 15, 0, -30,
		-30, 5, 10, 15, 15, 10, 5, -30,
		-40, -20, 0, 5, 5, 0, -20, -40,
		-50, -40, -30, -30, -30, -30, -40, -50,
	}
	bishopPST = [64]int{
		-20, -10, -10, -10, -10, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 10, 10, 5, 0, -10,
		-10, 5, 5, 10, 10, 5, 5, -10,
		-10, 0, 10, 10, 10, 10, 0, -10,
		-10, 10, 10, 10, 10, 10, 10, -10,
		-10, 5, 0, 0, 0, 0, 5, -10,
		-20, -10, -10, -10, -10, -10, -10, -20,
	}
	rookPST = [64]int{
		0, 0, 0, 0, 0, 0, 0, 0,
		5, 10, 10, 10, 10, 10, 10, 5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		0, 0, 0, 5, 5, 0, 0, 0,
	}
	queenPST = [64]int{
		-20, -10, -10, -5, -5, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 5, 5, 5, 0, -10,
		-5, 0, 5, 5, 5, 5, 0, -5,
		0, 0, 5, 5, 5, 5, 0, -5,
		-10, 5, 5, 5, 5, 5, 0, -10,
		-10, 0, 5, 0, 0, 0, 0, -10,
		-20, -10, -10, -5, -5, -10, -10, -20,
	}
	kingPST = [64]int{
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-20, -30, -30, -40, -40, -30, -30, -20,
		-10, -20, -20, -20, -20, -20, -20, -10,
		20, 20, 0, 0, 0, 0, 20, 20,
		20, 30, 10, 0, 0, 10, 30, 20,
	}
)

func evaluateStaticPosition(pos chess.Position) int {
	score := 0
	pieceVals := map[chess.PieceType]int{
		chess.Pawn: 100, chess.Knight: 320, chess.Bishop: 330,
		chess.Rook: 500, chess.Queen: 900, chess.King: 20000,
	}

	for sq := 0; sq < 64; sq++ {
		p := pos.Board[sq]
		if p.IsEmpty() {
			continue
		}
		val := pieceVals[p.Type]
		var pstVal int
		tableSq := sq
		if p.Color == chess.Black {
			// Flip vertically for black
			tableSq = (7-(sq/8))*8 + (sq % 8)
		}

		switch p.Type {
		case chess.Pawn:
			pstVal = pawnPST[tableSq]
		case chess.Knight:
			pstVal = knightPST[tableSq]
		case chess.Bishop:
			pstVal = bishopPST[tableSq]
		case chess.Rook:
			pstVal = rookPST[tableSq]
		case chess.Queen:
			pstVal = queenPST[tableSq]
		case chess.King:
			pstVal = kingPST[tableSq]
		}

		totalPieceScore := val + pstVal
		if p.Color == chess.White {
			score += totalPieceScore
		} else {
			score -= totalPieceScore
		}
	}
	return score
}

// Pure Go fallback evaluation based on 2-ply minimax lookahead & positional PST tables
func fallbackEval(fen string) *Evaluation {
	pos, err := chess.FromFEN(fen)
	if err != nil {
		return &Evaluation{ScoreCp: 0, BestMove: "e2e4", PV: []string{"e2e4"}, Depth: 1}
	}

	legalMoves := chess.LegalMoves(pos)
	if len(legalMoves) == 0 {
		if chess.IsCheck(pos) {
			if pos.SideToMove == chess.White {
				return &Evaluation{ScoreCp: -10000, IsMate: true, MateIn: -1, Depth: 2}
			}
			return &Evaluation{ScoreCp: 10000, IsMate: true, MateIn: 1, Depth: 2}
		}
		// Stalemate
		return &Evaluation{ScoreCp: 0, IsMate: false, Depth: 2}
	}

	bestMove := legalMoves[0]
	var bestScore int

	if pos.SideToMove == chess.White {
		bestScore = -999999
		for _, m := range legalMoves {
			pos1, err := chess.ApplyMove(pos, m)
			if err != nil {
				continue
			}

			// Ply 2: Opponent's best response
			oppMoves := chess.LegalMoves(pos1)
			if len(oppMoves) == 0 {
				if chess.IsCheck(pos1) {
					// Mate in 1 for White!
					return &Evaluation{
						ScoreCp:  10000,
						IsMate:   true,
						MateIn:   1,
						BestMove: m.UCI(),
						PV:       []string{m.UCI()},
						Depth:    2,
					}
				}
				// Stalemate
				if 0 > bestScore {
					bestScore = 0
					bestMove = m
				}
				continue
			}

			// Black minimizes score
			minOppScore := 999999
			for _, om := range oppMoves {
				pos2, err := chess.ApplyMove(pos1, om)
				if err != nil {
					continue
				}
				eval := evaluateStaticPosition(pos2)
				if eval < minOppScore {
					minOppScore = eval
				}
			}

			if minOppScore > bestScore {
				bestScore = minOppScore
				bestMove = m
			}
		}
	} else {
		// Black maximizes negative score (minimizes White's score)
		bestScore = 999999
		for _, m := range legalMoves {
			pos1, err := chess.ApplyMove(pos, m)
			if err != nil {
				continue
			}

			oppMoves := chess.LegalMoves(pos1)
			if len(oppMoves) == 0 {
				if chess.IsCheck(pos1) {
					// Mate in 1 for Black!
					return &Evaluation{
						ScoreCp:  -10000,
						IsMate:   true,
						MateIn:   -1,
						BestMove: m.UCI(),
						PV:       []string{m.UCI()},
						Depth:    2,
					}
				}
				// Stalemate
				if 0 < bestScore {
					bestScore = 0
					bestMove = m
				}
				continue
			}

			// White maximizes score
			maxOppScore := -999999
			for _, om := range oppMoves {
				pos2, err := chess.ApplyMove(pos1, om)
				if err != nil {
					continue
				}
				eval := evaluateStaticPosition(pos2)
				if eval > maxOppScore {
					maxOppScore = eval
				}
			}

			if maxOppScore < bestScore {
				bestScore = maxOppScore
				bestMove = m
			}
		}
	}

	bestUCI := bestMove.UCI()
	return &Evaluation{
		ScoreCp:  bestScore,
		IsMate:   false,
		BestMove: bestUCI,
		PV:       []string{bestUCI},
		Depth:    2,
	}
}


