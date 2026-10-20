package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"chess-platform/server/internal/auth"
	"chess-platform/server/internal/engine"
	"chess-platform/server/internal/game"
	"chess-platform/server/internal/review"
	"chess-platform/server/internal/user"
	"chess-platform/server/internal/websocket"

	"go.mongodb.org/mongo-driver/bson/primitive"
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

	// 4. Test logs route with query secret -> MUST be rejected with 401
	req = httptest.NewRequest("GET", "/api/v1/logs?secret=test-secret-12345", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 rejecting query parameter secret, got %d", w.Code)
	}

	// 5. Test logs route with header secret -> authorized (not 401)
	req = httptest.NewRequest("GET", "/api/v1/logs", nil)
	req.Header.Set("X-Admin-Secret", "test-secret-12345")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Errorf("expected header secret authorized, got 401")
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := newIPRateLimiter("test", 3, 500*time.Millisecond)

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

type testUserRepo struct {
	users map[string]*user.User
}

func newTestUserRepo() *testUserRepo {
	return &testUserRepo{users: make(map[string]*user.User)}
}

func (m *testUserRepo) Create(ctx context.Context, u *user.User) error {
	u.ID = primitive.NewObjectID()
	m.users[u.Username] = u
	return nil
}
func (m *testUserRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*user.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, user.ErrNotFound
}
func (m *testUserRepo) FindByUsername(ctx context.Context, username string) (*user.User, error) {
	if u, ok := m.users[username]; ok {
		return u, nil
	}
	return nil, user.ErrNotFound
}
func (m *testUserRepo) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, user.ErrNotFound
}
func (m *testUserRepo) UpdateProfile(ctx context.Context, id primitive.ObjectID, bio, country string) error {
	return nil
}
func (m *testUserRepo) UpdateAvatar(ctx context.Context, id primitive.ObjectID, avatarURL string) error {
	return nil
}
func (m *testUserRepo) UpdateLinkedAccounts(ctx context.Context, id primitive.ObjectID, chessCom, lichess string) error {
	return nil
}
func (m *testUserRepo) SearchUsers(ctx context.Context, query string, limit int) ([]*user.User, error) {
	var res []*user.User
	for _, u := range m.users {
		if query == "" || strings.Contains(strings.ToLower(u.Username), strings.ToLower(query)) {
			res = append(res, u)
			if limit > 0 && len(res) >= limit {
				break
			}
		}
	}
	return res, nil
}
func (m *testUserRepo) DeleteAllUsers(ctx context.Context) error {
	m.users = make(map[string]*user.User)
	return nil
}
func (m *testUserRepo) EnsureIndexes(ctx context.Context) error {
	return nil
}

func TestPublicProfileDoesNotLeakPrivateData(t *testing.T) {
	repo := newTestUserRepo()
	testUser := &user.User{
		ID:            primitive.NewObjectID(),
		Username:      "grandmaster",
		Email:         "private_grandmaster@chesspasant.com",
		PasswordHash:  "$2a$10$insecurehashvalueshouldneverbeexposed",
		EmailVerified: true,
		Bio:           "Candidate Master",
		Country:       "NO",
	}
	_ = repo.Create(context.Background(), testUser)

	router := NewRouter(nil, repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "*")

	req := httptest.NewRequest("GET", "/api/v1/users/grandmaster", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	respBody := w.Body.String()

	// Verify public fields are present
	if !strings.Contains(respBody, `"username":"grandmaster"`) {
		t.Errorf("expected username in public profile response")
	}

	// Verify private fields are strictly absent
	if strings.Contains(respBody, "private_grandmaster@chesspasant.com") {
		t.Errorf("SECURITY VULNERABILITY: User email address leaked in public profile: %s", respBody)
	}
	if strings.Contains(respBody, "insecurehashvalue") {
		t.Errorf("SECURITY VULNERABILITY: Password hash leaked in public profile: %s", respBody)
	}
	if strings.Contains(respBody, "emailVerified") {
		t.Errorf("SECURITY VULNERABILITY: Internal emailVerified flag leaked in public profile")
	}
}

func TestAdminEndpointsInaccessibleWhenNotConfigured(t *testing.T) {
	router := setupTestRouter()
	router.SetAdminSecret("") // explicitly unconfigured

	req := httptest.NewRequest("POST", "/api/v1/games/close-all", nil)
	req.Header.Set("X-Admin-Secret", "some-arbitrary-secret")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden when admin secret is unconfigured, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ADMIN_DISABLED") {
		t.Errorf("expected ADMIN_DISABLED error code, got %s", w.Body.String())
	}
}

func TestSpoofedIPRateLimiting(t *testing.T) {
	// 1. Untrusted peer trying to spoof X-Forwarded-For
	req := httptest.NewRequest("GET", "/api/v1/games", nil)
	req.RemoteAddr = "203.0.113.195:54321" // Public internet IP (not trusted proxy)
	req.Header.Set("X-Forwarded-For", "127.0.0.1, 10.0.0.1")
	req.Header.Set("X-Real-IP", "1.1.1.1")

	extractedIP := getClientIP(req)
	if extractedIP != "203.0.113.195" {
		t.Errorf("SECURITY VULNERABILITY: Blindly trusted X-Forwarded-For from untrusted remote peer! Got %s, expected 203.0.113.195", extractedIP)
	}

	// 2. Trusted reverse proxy forwarding client IP
	proxyReq := httptest.NewRequest("GET", "/api/v1/games", nil)
	proxyReq.RemoteAddr = "127.0.0.1:45123" // Localhost Nginx reverse proxy
	proxyReq.Header.Set("X-Forwarded-For", "198.51.100.77")

	trustedExtracted := getClientIP(proxyReq)
	if trustedExtracted != "198.51.100.77" {
		t.Errorf("expected trusted proxy forwarding to extract client IP 198.51.100.77, got %s", trustedExtracted)
	}
}

func TestWSTicketCreationAndConsumption(t *testing.T) {
	authSvc := auth.NewService(nil, "access-secret-32-chars-long-test1", "refresh-secret-32-chars-long-test2")
	hub := websocket.NewHub(nil, authSvc)

	ticket := hub.CreateTicket("user-123", "charlie")
	if ticket == "" {
		t.Fatalf("expected non-empty ticket")
	}

	// First consumption succeeds
	uid, uname, ok := hub.ConsumeTicket(ticket)
	if !ok || uid != "user-123" || uname != "charlie" {
		t.Fatalf("expected successful ticket consumption, got ok=%v uid=%s uname=%s", ok, uid, uname)
	}

	// Second consumption fails (single-use ticket)
	_, _, ok2 := hub.ConsumeTicket(ticket)
	if ok2 {
		t.Errorf("SECURITY VULNERABILITY: WebSocket connection ticket was re-used after consumption!")
	}
}

func TestRefreshTokenRotationAndSessionRevocation(t *testing.T) {
	repo := newTestUserRepo()
	authSvc := auth.NewService(repo, "access-secret-32-chars-long-test1", "refresh-secret-32-chars-long-test2")
	ctx := context.Background()

	u := &user.User{Username: "david"}
	_ = repo.Create(ctx, u)
	tokens, err := authSvc.GenerateTokenPair(u)
	if err != nil {
		t.Fatalf("failed to generate tokens: %v", err)
	}

	// 1. Initial access token is valid
	claims, err := authSvc.ValidateAccessToken(tokens.AccessToken)
	if err != nil || claims.Username != "david" {
		t.Fatalf("expected valid access token: %v", err)
	}

	// 2. Refresh token rotation: using refresh token issues new pair and revokes old
	newPair, err := authSvc.Refresh(ctx, tokens.RefreshToken)
	if err != nil {
		t.Fatalf("expected successful refresh: %v", err)
	}
	if newPair.AccessToken == "" || newPair.RefreshToken == "" {
		t.Fatalf("expected new tokens")
	}

	// 3. Reuse detection: using the old refresh token MUST fail and trigger session revocation!
	_, err = authSvc.Refresh(ctx, tokens.RefreshToken)
	if err != auth.ErrInvalidToken {
		t.Errorf("SECURITY VULNERABILITY: Rotated refresh token was accepted again without reuse error! Got %v", err)
	}

	// 4. Logout revokes token
	logoutPair, _ := authSvc.GenerateTokenPair(u)
	err = authSvc.Logout(ctx, logoutPair.AccessToken, logoutPair.RefreshToken)
	if err != nil {
		t.Fatalf("logout error: %v", err)
	}

	// Validating logged-out access token must fail
	_, err = authSvc.ValidateAccessToken(logoutPair.AccessToken)
	if err != auth.ErrInvalidToken {
		t.Errorf("SECURITY VULNERABILITY: Logged out access token was still accepted as valid!")
	}
}

func TestUserSearchEndpoint(t *testing.T) {
	repo := newTestUserRepo()
	_ = repo.Create(context.Background(), &user.User{
		Username: "yogesh",
		Email:    "yogesh1@example.com",
	})
	_ = repo.Create(context.Background(), &user.User{
		Username: "Yogesh",
		Email:    "yogesh2@example.com",
	})
	_ = repo.Create(context.Background(), &user.User{
		Username: "magnus",
		Email:    "magnus@example.com",
	})

	router := NewRouter(nil, repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "*")

	req := httptest.NewRequest("GET", "/api/v1/users/search?q=yog", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'yog', got %d: %v", len(results), results)
	}
}

func TestAdminResetUsersEndpoint(t *testing.T) {
	repo := newTestUserRepo()
	_ = repo.Create(context.Background(), &user.User{
		Username: "tempuser",
		Email:    "temp@example.com",
	})

	router := NewRouter(nil, repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "*")
	router.SetAdminSecret("secret-admin-key")

	// Unauthorized attempt
	req1 := httptest.NewRequest("POST", "/api/v1/admin/users/reset", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", w1.Code)
	}

	// Authorized attempt
	req2 := httptest.NewRequest("POST", "/api/v1/admin/users/reset", nil)
	req2.Header.Set("X-Admin-Secret", "secret-admin-key")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w2.Code, w2.Body.String())
	}

	if len(repo.users) != 0 {
		t.Fatalf("expected 0 users remaining after reset, got %d", len(repo.users))
	}
}


