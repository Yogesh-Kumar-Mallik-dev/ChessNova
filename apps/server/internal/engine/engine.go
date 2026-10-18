package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
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
	binaryPath   string
	remoteURL    string
	remoteSecret string
	httpClient   *http.Client
	pool         chan *stockfishWorker
	maxWorkers   int
	cache        sync.Map
	cacheKeys    []string
	cacheMu      sync.Mutex
	stopChan     chan struct{}
}

type stockfishWorker struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	writer  *bufio.Writer
	scanner *bufio.Scanner
	alive   bool
	mu      sync.Mutex
	skill   int
}

var (
	depthRegex = regexp.MustCompile(`\bdepth\s+(\d+)`)
	cpRegex    = regexp.MustCompile(`\bscore\s+cp\s+(-?\d+)`)
	mateRegex  = regexp.MustCompile(`\bscore\s+mate\s+(-?\d+)`)
	pvRegex    = regexp.MustCompile(`\bpv\s+(.*)`)
)

func NewStockfishEngine() *StockfishEngine {
	bin := findStockfishBinary()
	remoteURL := strings.TrimRight(os.Getenv("STOCKFISH_REMOTE_URL"), "/")
	remoteSecret := os.Getenv("STOCKFISH_REMOTE_SECRET")

	// If remote is configured, we only need 1 local fallback worker to minimize memory footprint!
	poolSize := 1
	if remoteURL == "" {
		poolSize = 1 // 1 worker per instance prevents CPU starvation on 1 vCPU
	}

	eng := &StockfishEngine{
		binaryPath:   bin,
		remoteURL:    remoteURL,
		remoteSecret: remoteSecret,
		httpClient:   &http.Client{Timeout: 90 * time.Second},
		pool:         make(chan *stockfishWorker, poolSize),
		maxWorkers:   poolSize,
		cacheKeys:    make([]string, 0, 5000),
		stopChan:     make(chan struct{}),
	}

	if bin != "" {
		for i := 0; i < poolSize; i++ {
			w, err := newStockfishWorker(bin)
			if err == nil {
				eng.pool <- w
			}
		}
	}

	return eng
}

func (e *StockfishEngine) RemoteURL() string {
	return e.remoteURL
}

func (e *StockfishEngine) RemoteSecret() string {
	return e.remoteSecret
}

func (e *StockfishEngine) HTTPClient() *http.Client {
	return e.httpClient
}

func newStockfishWorker(binaryPath string) (*stockfishWorker, error) {
	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(), "PATH=/usr/games:/usr/local/bin:/usr/bin:/bin")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, err
	}

	w := &stockfishWorker{
		cmd:     cmd,
		stdin:   stdin,
		writer:  bufio.NewWriter(stdin),
		scanner: bufio.NewScanner(stdout),
		alive:   true,
		skill:   -1,
	}

	// Initialize UCI
	if _, err := w.writer.WriteString("uci\nisready\n"); err != nil {
		w.close()
		return nil, err
	}
	if err := w.writer.Flush(); err != nil {
		w.close()
		return nil, err
	}

	ready := make(chan bool, 1)
	go func() {
		for w.scanner.Scan() {
			line := w.scanner.Text()
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
			w.close()
			return nil, fmt.Errorf("stockfish worker readyok failed")
		}
	case <-time.After(5 * time.Second):
		w.close()
		return nil, fmt.Errorf("timeout waiting for stockfish readyok")
	}

	return w, nil
}

func (w *stockfishWorker) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.alive = false
	if w.writer != nil {
		_, _ = w.writer.WriteString("quit\n")
		_ = w.writer.Flush()
	}
	if w.stdin != nil {
		_ = w.stdin.Close()
	}
	if w.cmd != nil && w.cmd.Process != nil {
		_ = w.cmd.Process.Kill()
		_ = w.cmd.Wait()
	}
}

func (w *stockfishWorker) evaluate(ctx context.Context, fen string, depth int, moveTimeMs int) (*Evaluation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.alive {
		return nil, fmt.Errorf("worker not alive")
	}

	cmdStr := fmt.Sprintf("position fen %s\ngo depth %d movetime %d\n", fen, depth, moveTimeMs)
	if _, err := w.writer.WriteString(cmdStr); err != nil {
		w.alive = false
		return nil, err
	}
	if err := w.writer.Flush(); err != nil {
		w.alive = false
		return nil, err
	}

	eval := &Evaluation{Depth: depth}
	isBlackToMove := strings.Contains(fen, " b ")

	type readResult struct {
		eval *Evaluation
		err  error
	}

	resCh := make(chan readResult, 1)

	go func() {
		for w.scanner.Scan() {
			line := w.scanner.Text()

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
				if len(parts) > 1 && parts[1] != "(none)" {
					eval.BestMove = parts[1]
				}
				resCh <- readResult{eval: eval, err: nil}
				return
			}
		}

		if err := w.scanner.Err(); err != nil {
			resCh <- readResult{eval: nil, err: err}
			return
		}
		resCh <- readResult{eval: nil, err: fmt.Errorf("unexpected EOF from stockfish")}
	}()

	select {
	case res := <-resCh:
		if res.err != nil {
			w.alive = false
			return nil, res.err
		}
		if isBlackToMove {
			res.eval.ScoreCp = -res.eval.ScoreCp
			if res.eval.IsMate {
				res.eval.MateIn = -res.eval.MateIn
			}
		}
		if len(res.eval.PV) == 0 && res.eval.BestMove != "" {
			res.eval.PV = []string{res.eval.BestMove}
		}
		return res.eval, nil

	case <-ctx.Done():
		_, _ = w.writer.WriteString("stop\n")
		_ = w.writer.Flush()
		select {
		case <-resCh:
		case <-time.After(300 * time.Millisecond):
			w.alive = false
			w.close()
		}
		return nil, ctx.Err()
	}
}

func (w *stockfishWorker) botMove(ctx context.Context, fen string, depth int, skillLevel int, moveTimeMs int) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.alive {
		return "", fmt.Errorf("worker not alive")
	}

	if w.skill != skillLevel {
		_, _ = w.writer.WriteString(fmt.Sprintf("setoption name Skill Level value %d\nisready\n", skillLevel))
		_ = w.writer.Flush()
		w.skill = skillLevel

		ready := make(chan bool, 1)
		go func() {
			for w.scanner.Scan() {
				if w.scanner.Text() == "readyok" {
					ready <- true
					return
				}
			}
			ready <- false
		}()

		select {
		case ok := <-ready:
			if !ok {
				w.alive = false
				return "", fmt.Errorf("stockfish worker failed during skill update")
			}
		case <-time.After(2 * time.Second):
			w.alive = false
			w.close()
			return "", fmt.Errorf("stockfish readyok timeout during skill update")
		}
	}

	cmdStr := fmt.Sprintf("position fen %s\ngo depth %d movetime %d\n", fen, depth, moveTimeMs)
	if _, err := w.writer.WriteString(cmdStr); err != nil {
		w.alive = false
		return "", err
	}
	if err := w.writer.Flush(); err != nil {
		w.alive = false
		return "", err
	}

	type readResult struct {
		move string
		err  error
	}

	resCh := make(chan readResult, 1)
	go func() {
		for w.scanner.Scan() {
			line := w.scanner.Text()
			if strings.HasPrefix(line, "bestmove") {
				parts := strings.Fields(line)
				if len(parts) > 1 && parts[1] != "(none)" {
					resCh <- readResult{move: parts[1], err: nil}
				} else {
					resCh <- readResult{move: "", err: fmt.Errorf("no valid bestmove")}
				}
				return
			}
		}
		if err := w.scanner.Err(); err != nil {
			resCh <- readResult{move: "", err: err}
			return
		}
		resCh <- readResult{move: "", err: fmt.Errorf("unexpected EOF")}
	}()

	select {
	case res := <-resCh:
		if res.err != nil {
			w.alive = false
			return "", res.err
		}
		return res.move, nil

	case <-ctx.Done():
		_, _ = w.writer.WriteString("stop\n")
		_ = w.writer.Flush()
		select {
		case <-resCh:
		case <-time.After(300 * time.Millisecond):
			w.alive = false
			w.close()
		}
		return "", ctx.Err()
	}
}

func (e *StockfishEngine) replenishWorker() {
	if e.binaryPath == "" {
		return
	}
	w, err := newStockfishWorker(e.binaryPath)
	if err == nil {
		e.pool <- w
	}
}

func (e *StockfishEngine) Close() {
	close(e.stopChan)
	for {
		select {
		case w := <-e.pool:
			w.close()
		default:
			return
		}
	}
}

func canonicalFenKey(fen string, depth int) string {
	parts := strings.Fields(strings.TrimSpace(fen))
	if len(parts) >= 4 {
		return fmt.Sprintf("%s %s %s %s:%d", parts[0], parts[1], parts[2], parts[3], depth)
	}
	return fmt.Sprintf("%s:%d", strings.TrimSpace(fen), depth)
}

func (e *StockfishEngine) getCache(key string) (*Evaluation, bool) {
	if val, ok := e.cache.Load(key); ok {
		if eval, ok := val.(*Evaluation); ok {
			cp := *eval
			return &cp, true
		}
	}
	return nil, false
}

func (e *StockfishEngine) putCache(key string, eval *Evaluation) {
	if eval == nil {
		return
	}
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()

	if len(e.cacheKeys) >= 5000 {
		oldest := e.cacheKeys[0]
		e.cacheKeys = e.cacheKeys[1:]
		e.cache.Delete(oldest)
	}

	e.cacheKeys = append(e.cacheKeys, key)
	cp := *eval
	e.cache.Store(key, &cp)
}

func findStockfishBinary() string {
	candidates := []string{
		os.Getenv("STOCKFISH_PATH"),
		"/usr/games/stockfish",
		"/usr/local/bin/stockfish",
		"/usr/bin/stockfish",
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
		if c == "" {
			continue
		}
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
	if depth <= 0 {
		depth = 10
	}
	if depth > 16 {
		depth = 16
	}

	// 1. Fast cache lookup
	cacheKey := canonicalFenKey(fen, depth)
	if cached, ok := e.getCache(cacheKey); ok {
		return cached, nil
	}

	// Check if already evaluated at a higher depth
	for d := depth + 1; d <= depth+4 && d <= 16; d++ {
		higherKey := canonicalFenKey(fen, d)
		if cached, ok := e.getCache(higherKey); ok {
			return cached, nil
		}
	}

	// 2. Query remote showcase Stockfish engine if configured
	if e.remoteURL != "" {
		eval, err := e.evaluateRemote(ctx, fen, depth)
		if err == nil && eval != nil {
			e.putCache(cacheKey, eval)
			return eval, nil
		}
	}

	if e.binaryPath == "" {
		eval := fallbackEval(fen)
		e.putCache(cacheKey, eval)
		return eval, nil
	}

	moveTimeMs := 350
	if depth <= 8 {
		moveTimeMs = 150
	} else if depth <= 10 {
		moveTimeMs = 250
	} else if depth <= 12 {
		moveTimeMs = 500
	} else {
		moveTimeMs = 1000
	}

	// Lease an active worker from the pool
	var w *stockfishWorker
	select {
	case w = <-e.pool:
	case <-ctx.Done():
		return fallbackEval(fen), ctx.Err()
	case <-time.After(2 * time.Second):
		// Workers busy under heavy concurrency: fallback immediately to avoid stalling client
		return fallbackEval(fen), nil
	}

	defer func() {
		if w.alive {
			e.pool <- w
		} else {
			w.close()
			go e.replenishWorker()
		}
	}()

	eval, err := w.evaluate(ctx, fen, depth, moveTimeMs)
	if err != nil {
		fallback := fallbackEval(fen)
		return fallback, nil
	}

	e.putCache(cacheKey, eval)
	return eval, nil
}

// BotDifficulty maps levels 1..5 to Stockfish parameters
type BotDifficulty struct {
	Level      int
	SkillLevel int // Stockfish UCI Skill Level 0..20
	Depth      int // Search depth limit
	MoveTimeMs int // Movetime limit
}

var BotLevels = map[int]BotDifficulty{
	1: {Level: 1, SkillLevel: 0, Depth: 1, MoveTimeMs: 80},    // ~400 Elo (Novice)
	2: {Level: 2, SkillLevel: 3, Depth: 3, MoveTimeMs: 150},   // ~800 Elo (Casual)
	3: {Level: 3, SkillLevel: 8, Depth: 6, MoveTimeMs: 300},   // ~1200 Elo (Intermediate)
	4: {Level: 4, SkillLevel: 14, Depth: 10, MoveTimeMs: 500}, // ~1600 Elo (Advanced)
	5: {Level: 5, SkillLevel: 20, Depth: 14, MoveTimeMs: 1000},// ~2000+ Elo (Master)
}

func (e *StockfishEngine) GenerateBotMove(ctx context.Context, fen string, level int) (string, error) {
	var thinkDuration time.Duration
	var blunderChance float64

	switch level {
	case 1:
		thinkMs := 400 + rand.Intn(1000)
		thinkDuration = time.Duration(thinkMs) * time.Millisecond
		normalized := float64(thinkMs-400) / 1000.0
		blunderChance = 0.80 - (normalized * 0.35)
	case 2:
		thinkMs := 500 + rand.Intn(1200)
		thinkDuration = time.Duration(thinkMs) * time.Millisecond
		normalized := float64(thinkMs-500) / 1200.0
		blunderChance = 0.45 - (normalized * 0.25)
	case 3:
		thinkMs := 600 + rand.Intn(1200)
		thinkDuration = time.Duration(thinkMs) * time.Millisecond
		normalized := float64(thinkMs-600) / 1200.0
		blunderChance = 0.18 - (normalized * 0.12)
	case 4:
		thinkMs := 700 + rand.Intn(1100)
		thinkDuration = time.Duration(thinkMs) * time.Millisecond
		normalized := float64(thinkMs-700) / 1100.0
		blunderChance = 0.06 - (normalized * 0.04)
	default:
		thinkMs := 800 + rand.Intn(1200)
		thinkDuration = time.Duration(thinkMs) * time.Millisecond
		blunderChance = 0.0
	}

	select {
	case <-time.After(thinkDuration):
	case <-ctx.Done():
		return "", ctx.Err()
	}

	if pos, err := chess.FromFEN(fen); err == nil {
		legalMoves := chess.LegalMoves(pos)
		if len(legalMoves) > 0 {
			r := rand.Float64()
			if r < blunderChance {
				return legalMoves[rand.Intn(len(legalMoves))].UCI(), nil
			}
		}
	}

	diff, ok := BotLevels[level]
	if !ok {
		diff = BotLevels[2]
	}

	// Query remote showcase Stockfish engine if configured
	if e.remoteURL != "" {
		move, err := e.botMoveRemote(ctx, fen, level)
		if err == nil && move != "" {
			return move, nil
		}
	}

	if e.binaryPath == "" {
		return fallbackBotMove(fen, level), nil
	}

	var w *stockfishWorker
	select {
	case w = <-e.pool:
	case <-ctx.Done():
		return fallbackBotMove(fen, level), ctx.Err()
	case <-time.After(2 * time.Second):
		return fallbackBotMove(fen, level), nil
	}

	defer func() {
		if w.alive {
			e.pool <- w
		} else {
			w.close()
			go e.replenishWorker()
		}
	}()

	move, err := w.botMove(ctx, fen, diff.Depth, diff.SkillLevel, diff.MoveTimeMs)
	if err != nil || move == "" {
		return fallbackBotMove(fen, level), nil
	}

	return move, nil
}

func (e *StockfishEngine) evaluateRemote(ctx context.Context, fen string, depth int) (*Evaluation, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"fen":   fen,
		"depth": depth,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", e.remoteURL+"/evaluate", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.remoteSecret != "" {
		req.Header.Set("X-Engine-Secret", e.remoteSecret)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote engine returned status %d", resp.StatusCode)
	}

	var eval Evaluation
	if err := json.NewDecoder(resp.Body).Decode(&eval); err != nil {
		return nil, err
	}
	return &eval, nil
}

func (e *StockfishEngine) botMoveRemote(ctx context.Context, fen string, level int) (string, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"fen":   fen,
		"level": level,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", e.remoteURL+"/bot-move", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.remoteSecret != "" {
		req.Header.Set("X-Engine-Secret", e.remoteSecret)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("remote engine returned status %d", resp.StatusCode)
	}

	var res struct {
		Move string `json:"move"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.Move, nil
}

func fallbackBotMove(fen string, level int) string {
	pos, err := chess.FromFEN(fen)
	if err != nil {
		return ""
	}
	moves := chess.LegalMoves(pos)
	if len(moves) == 0 {
		return ""
	}

	if level <= 1 {
		return moves[rand.Intn(len(moves))].UCI()
	}

	if level == 2 && rand.Float64() < 0.40 {
		return moves[rand.Intn(len(moves))].UCI()
	}

	bestUCI := moves[0].UCI()
	bestScore := -999999
	if pos.SideToMove == chess.Black {
		bestScore = 999999
	}

	for _, m := range moves {
		nextPos, err := chess.ApplyMove(pos, m)
		if err != nil {
			continue
		}
		score := evaluateStaticPosition(nextPos)
		if pos.SideToMove == chess.White {
			if score > bestScore {
				bestScore = score
				bestUCI = m.UCI()
			}
		} else {
			if score < bestScore {
				bestScore = score
				bestUCI = m.UCI()
			}
		}
	}

	return bestUCI
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


