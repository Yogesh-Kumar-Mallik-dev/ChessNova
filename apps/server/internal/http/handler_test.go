package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chess-platform/server/internal/engine"
	"chess-platform/server/internal/game"
	"chess-platform/server/internal/review"
)

func setupTestRouter() *Router {
	eng := engine.NewStockfishEngine()
	reviewSrv := review.NewReviewService(eng)
	return NewRouter(nil, nil, nil, nil, nil, nil, nil, eng, reviewSrv, nil, nil, nil, "*")
}

func TestHandleHealth(t *testing.T) {
	router := setupTestRouter()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestHandleLegalMoves(t *testing.T) {
	router := setupTestRouter()

	body, _ := json.Marshal(map[string]string{
		"fen":    "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"square": "e2",
	})
	req := httptest.NewRequest("POST", "/api/v1/chess/legal-moves", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Targets []struct {
			To        string `json:"to"`
			SAN       string `json:"san"`
			IsCapture bool   `json:"isCapture"`
			Category  string `json:"category"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Targets) != 2 {
		t.Fatalf("expected 2 legal moves for e2 (e3, e4), got %d", len(resp.Targets))
	}
}

func TestHandleExecuteMove(t *testing.T) {
	router := setupTestRouter()

	body, _ := json.Marshal(map[string]string{
		"fen":  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"from": "e2",
		"to":   "e4",
	})
	req := httptest.NewRequest("POST", "/api/v1/chess/move", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Valid  bool   `json:"valid"`
		NewFen string `json:"newFen"`
		SAN    string `json:"san"`
		Turn   string `json:"turn"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Valid || resp.SAN != "e4" || resp.Turn != "black" {
		t.Errorf("unexpected move result: %+v", resp)
	}
}

func TestHandleEvaluatePosition(t *testing.T) {
	router := setupTestRouter()

	body, _ := json.Marshal(map[string]interface{}{
		"fen":   "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"depth": 8,
	})
	req := httptest.NewRequest("POST", "/api/v1/analysis/evaluate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		FEN       string  `json:"fen"`
		ScoreCp   int     `json:"scoreCp"`
		WinChance float64 `json:"winChance"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.WinChance <= 0 {
		t.Errorf("expected positive win chance, got %f", resp.WinChance)
	}
}

func TestHandleReviewGame(t *testing.T) {
	router := setupTestRouter()

	body, _ := json.Marshal(map[string]interface{}{
		"moves": []map[string]string{
			{"from": "e2", "to": "e4", "san": "e4"},
			{"from": "e7", "to": "e5", "san": "e5"},
		},
	})
	req := httptest.NewRequest("POST", "/api/v1/analysis/review", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		WhiteAccuracy float64 `json:"whiteAccuracy"`
		BlackAccuracy float64 `json:"blackAccuracy"`
		Moves         []any   `json:"moves"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Moves) != 2 {
		t.Errorf("expected 2 reviewed moves, got %d", len(resp.Moves))
	}
}

func TestHandleReviewGamePGN(t *testing.T) {
	router := setupTestRouter()

	body, _ := json.Marshal(map[string]interface{}{
		"pgn": `[Event "World Championship"]
[White "Ding Liren"]
[Black "Nepomniachtchi"]
[Result "1-0"]

1. d4 Nf6 2. c4 e6 1-0`,
	})
	req := httptest.NewRequest("POST", "/api/v1/analysis/review", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		WhiteAccuracy float64 `json:"whiteAccuracy"`
		BlackAccuracy float64 `json:"blackAccuracy"`
		Moves         []any   `json:"moves"`
		White         string  `json:"white"`
		Black         string  `json:"black"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Moves) != 4 {
		t.Errorf("expected 4 reviewed moves, got %d", len(resp.Moves))
	}
	if resp.White != "Ding Liren" || resp.Black != "Nepomniachtchi" {
		t.Errorf("unexpected players: %s vs %s", resp.White, resp.Black)
	}
}

func TestSecurityHeaders(t *testing.T) {
	router := setupTestRouter()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("expected X-Content-Type-Options: nosniff, got %s", w.Header().Get("X-Content-Type-Options"))
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("expected X-Frame-Options: DENY, got %s", w.Header().Get("X-Frame-Options"))
	}
	if w.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Errorf("expected Referrer-Policy: strict-origin-when-cross-origin, got %s", w.Header().Get("Referrer-Policy"))
	}
}

func TestRegisterValidation(t *testing.T) {
	router := setupTestRouter()

	// 1. Invalid username
	body, _ := json.Marshal(map[string]string{
		"username": "a!", // invalid chars
		"email":    "test@example.com",
		"password": "validPassword123",
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid username, got %d", w.Code)
	}

	// 2. Invalid email
	body, _ = json.Marshal(map[string]string{
		"username": "validUser",
		"email":    "notanemail",
		"password": "validPassword123",
	})
	req = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid email, got %d", w.Code)
	}

	// 3. Short password
	body, _ = json.Marshal(map[string]string{
		"username": "validUser",
		"email":    "test@example.com",
		"password": "123",
	})
	req = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short password, got %d", w.Code)
	}
}

func TestUnauthorizedProfileUpdate(t *testing.T) {
	router := setupTestRouter()
	body, _ := json.Marshal(map[string]string{
		"bio":     "New Bio",
		"country": "US",
	})
	req := httptest.NewRequest("PUT", "/api/v1/users/profile", bytes.NewReader(body))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for profile update without token, got %d", w.Code)
	}
}

type testGameRepo struct {
	games map[string]*game.GameDocument
}

func (m *testGameRepo) Save(ctx context.Context, doc *game.GameDocument) error {
	m.games[doc.ID] = doc
	return nil
}

func (m *testGameRepo) SaveReview(ctx context.Context, id string, rev *review.ReviewResult) error {
	if g, ok := m.games[id]; ok {
		g.IsEvaluated = true
		g.Review = rev
		return nil
	}
	return game.ErrGameNotFound
}

func (m *testGameRepo) FindByID(ctx context.Context, id string) (*game.GameDocument, error) {
	if g, ok := m.games[id]; ok {
		return g, nil
	}
	return nil, game.ErrGameNotFound
}

func (m *testGameRepo) FindUserGames(ctx context.Context, userID string, limit, skip int) ([]game.GameDocument, error) {
	return nil, nil
}

func (m *testGameRepo) FindRecentGames(ctx context.Context, limit int) ([]game.GameDocument, error) {
	return nil, nil
}

func (m *testGameRepo) FindUnreviewedGames(ctx context.Context, limit int) ([]game.GameDocument, error) {
	var list []game.GameDocument
	for _, g := range m.games {
		if !g.IsEvaluated && len(g.Moves) > 0 {
			list = append(list, *g)
		}
	}
	return list, nil
}

func (m *testGameRepo) EnsureIndexes(ctx context.Context) error {
	return nil
}

func TestHandleReviewGameByID_CachedAndEvaluated(t *testing.T) {
	eng := engine.NewStockfishEngine()
	reviewSrv := review.NewReviewService(eng)
	repo := &testGameRepo{
		games: make(map[string]*game.GameDocument),
	}

	gameID := "test-game-123"
	repo.games[gameID] = &game.GameDocument{
		ID:  gameID,
		PGN: "1. e4 e5 2. Nf3 Nc6",
		Moves: []game.PersistedMove{
			{Ply: 1, From: "e2", To: "e4", SAN: "e4"},
			{Ply: 2, From: "e7", To: "e5", SAN: "e5"},
		},
		IsEvaluated: false,
	}

	router := NewRouter(nil, nil, nil, repo, nil, nil, nil, eng, reviewSrv, nil, nil, nil, "*")

	// 1. First review request: evaluates, persists to repo, sets IsEvaluated = true
	req := httptest.NewRequest("POST", "/api/v1/games/"+gameID+"/review", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	saved := repo.games[gameID]
	if !saved.IsEvaluated || saved.Review == nil {
		t.Fatalf("expected game to be marked as evaluated with review saved")
	}

	// 2. Second review request: should return cached review directly
	saved.Review.WhiteAccuracy = 99.9 // Custom value to verify cached return
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	var res review.ReviewResult
	_ = json.Unmarshal(w2.Body.Bytes(), &res)
	if res.WhiteAccuracy != 99.9 {
		t.Fatalf("expected cached review with WhiteAccuracy 99.9, got %f", res.WhiteAccuracy)
	}
}

func TestAdminAuth(t *testing.T) {
	router := setupTestRouter()
	router.SetAdminSecret("test-secret-12345")

	// 1. Without secret -> 401
	req := httptest.NewRequest("POST", "/api/v1/games/close-all", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without secret, got %d", w.Code)
	}

	// 2. With wrong secret -> 401
	req = httptest.NewRequest("POST", "/api/v1/games/close-all", nil)
	req.Header.Set("X-Admin-Secret", "wrong-secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong secret, got %d", w.Code)
	}

	// 3. With correct secret -> 200
	req = httptest.NewRequest("POST", "/api/v1/games/close-all", nil)
	req.Header.Set("X-Admin-Secret", "test-secret-12345")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with correct secret, got %d", w.Code)
	}

	// 4. Test logs route with query secret
	req = httptest.NewRequest("GET", "/api/v1/logs?secret=test-secret-12345", nil)
	w = httptest.NewRecorder()
	// Just start request to verify it does not reject with 401
	router.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Errorf("expected authorized access with query secret, got 401")
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := newIPRateLimiter(3, 500*time.Millisecond)

	ip := "192.168.1.100"
	if !limiter.allow(ip) {
		t.Fatalf("first request should be allowed")
	}
	if !limiter.allow(ip) {
		t.Fatalf("second request should be allowed")
	}
	if !limiter.allow(ip) {
		t.Fatalf("third request should be allowed")
	}
	if limiter.allow(ip) {
		t.Fatalf("fourth request should exceed rate limit")
	}

	// Wait for window to reset
	time.Sleep(550 * time.Millisecond)
	if !limiter.allow(ip) {
		t.Fatalf("request after window expiry should be allowed")
	}
}

