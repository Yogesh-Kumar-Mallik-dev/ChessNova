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

// Pure Go fallback evaluation based on piece-square material balance
func fallbackEval(fen string) *Evaluation {
	parts := strings.Split(fen, " ")
	if len(parts) == 0 {
		return &Evaluation{ScoreCp: 0, BestMove: "e2e4", PV: []string{"e2e4"}}
	}

	board := parts[0]
	score := 0
	pieceVals := map[rune]int{
		'p': 100, 'n': 320, 'b': 330, 'r': 500, 'q': 900, 'k': 20000,
		'P': 100, 'N': 320, 'B': 330, 'R': 500, 'Q': 900, 'K': 20000,
	}

	for _, ch := range board {
		if val, exists := pieceVals[ch]; exists {
			if ch >= 'A' && ch <= 'Z' {
				score += val
			} else {
				score -= val
			}
		}
	}

	return &Evaluation{
		ScoreCp:  score,
		IsMate:   false,
		BestMove: "",
		PV:       []string{},
		Depth:    1,
	}
}
