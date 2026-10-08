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
	if isBlackToMove && !eval.IsMate {
		eval.ScoreCp = -eval.ScoreCp
	}

	if len(eval.PV) == 0 && eval.BestMove != "" {
		eval.PV = []string{eval.BestMove}
	}

	return eval, nil
}

// Pure Go fallback evaluation based on legal move search & material balance
func fallbackEval(fen string) *Evaluation {
	pos, err := chess.FromFEN(fen)
	if err != nil {
		return &Evaluation{ScoreCp: 0, BestMove: "e2e4", PV: []string{"e2e4"}, Depth: 1}
	}

	// Calculate base material score from White's perspective
	score := 0
	pieceVals := map[chess.PieceType]int{
		chess.Pawn: 100, chess.Knight: 320, chess.Bishop: 330,
		chess.Rook: 500, chess.Queen: 900, chess.King: 20000,
	}

	for sq := 0; sq < 64; sq++ {
		p := pos.Board[sq]
		if !p.IsEmpty() {
			val := pieceVals[p.Type]
			if p.Color == chess.White {
				score += val
			} else {
				score -= val
			}
		}
	}

	legalMoves := chess.LegalMoves(pos)
	if len(legalMoves) == 0 {
		if chess.IsCheck(pos) {
			if pos.SideToMove == chess.White {
				return &Evaluation{ScoreCp: -10000, IsMate: true, MateIn: -1, Depth: 1}
			}
			return &Evaluation{ScoreCp: 10000, IsMate: true, MateIn: 1, Depth: 1}
		}
		// Stalemate
		return &Evaluation{ScoreCp: 0, IsMate: false, Depth: 1}
	}

	// Pick best candidate move using 1-ply material lookahead and central dominance
	bestMove := legalMoves[0]
	bestEval := -999999
	if pos.SideToMove == chess.Black {
		bestEval = 999999
	}

	for _, m := range legalMoves {
		nextPos, err := chess.ApplyMove(pos, m)
		if err != nil {
			continue
		}

		mScore := 0
		for sq := 0; sq < 64; sq++ {
			p := nextPos.Board[sq]
			if !p.IsEmpty() {
				val := pieceVals[p.Type]
				if p.Color == chess.White {
					mScore += val
				} else {
					mScore -= val
				}
			}
		}

		// Small bonus for center control (d4, e4, d5, e5)
		toStr := m.To.String()
		if toStr == "e4" || toStr == "d4" || toStr == "e5" || toStr == "d5" {
			if pos.SideToMove == chess.White {
				mScore += 15
			} else {
				mScore -= 15
			}
		}

		if pos.SideToMove == chess.White {
			if mScore > bestEval {
				bestEval = mScore
				bestMove = m
			}
		} else {
			if mScore < bestEval {
				bestEval = mScore
				bestMove = m
			}
		}
	}

	bestUCI := bestMove.UCI()
	return &Evaluation{
		ScoreCp:  score,
		IsMate:   false,
		BestMove: bestUCI,
		PV:       []string{bestUCI},
		Depth:    1,
	}
}

