package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chess-platform/server/internal/auth"
	"chess-platform/server/internal/engine"
	"chess-platform/server/internal/review"
)

func TestHTTPSecurityHeaders(t *testing.T) {
	router := setupTestRouter()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for k, expected := range headers {
		got := w.Header().Get(k)
		if got != expected {
			t.Errorf("header %s: expected %q, got %q", k, expected, got)
		}
	}

	csp := w.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src 'self'") {
		t.Errorf("CSP missing default-src 'self': %s", csp)
	}
	if !strings.Contains(csp, "wasm-unsafe-eval") {
		t.Errorf("CSP missing wasm-unsafe-eval for Stockfish: %s", csp)
	}

	hsts := w.Header().Get("Strict-Transport-Security")
	if !strings.Contains(hsts, "max-age=31536000") {
		t.Errorf("HSTS missing or incorrect: %s", hsts)
	}
}

func TestCORSPreflightOptions(t *testing.T) {
	router := setupTestRouter()
	req := httptest.NewRequest("OPTIONS", "/api/v1/chess/move", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for OPTIONS, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Errorf("missing Access-Control-Allow-Origin")
	}
	if !strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), "POST") {
		t.Errorf("missing POST in Access-Control-Allow-Methods")
	}
	if !strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Errorf("missing Authorization in Access-Control-Allow-Headers")
	}
}

func TestAdminAuthProtection(t *testing.T) {
	t.Run("Disabled admin secret returns 403", func(t *testing.T) {
		router := setupTestRouter() // adminSecret is empty by default
		req := httptest.NewRequest("GET", "/api/v1/logs", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden when adminSecret is unset, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Valid secret via X-Admin-Secret header", func(t *testing.T) {
		router := setupTestRouter()
		router.SetAdminSecret("super-secret-admin-token-12345")

		req := httptest.NewRequest("GET", "/api/v1/logs", nil)
		req.Header.Set("X-Admin-Secret", "super-secret-admin-token-12345")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK with valid secret, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Valid secret via Authorization Bearer header", func(t *testing.T) {
		router := setupTestRouter()
		router.SetAdminSecret("super-secret-admin-token-12345")

		req := httptest.NewRequest("GET", "/api/v1/logs", nil)
		req.Header.Set("Authorization", "Bearer super-secret-admin-token-12345")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK with valid Bearer secret, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Invalid secret rejected with 401", func(t *testing.T) {
		router := setupTestRouter()
		router.SetAdminSecret("super-secret-admin-token-12345")

		req := httptest.NewRequest("GET", "/api/v1/logs", nil)
		req.Header.Set("X-Admin-Secret", "wrong-secret")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized with invalid secret, got %d", w.Code)
		}
	})

	t.Run("Secret in URL query param rejected with 401", func(t *testing.T) {
		router := setupTestRouter()
		router.SetAdminSecret("super-secret-admin-token-12345")

		// Query param is purposefully NOT supported for security reasons
		req := httptest.NewRequest("GET", "/api/v1/logs?secret=super-secret-admin-token-12345", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized when secret is passed in query, got %d", w.Code)
		}
	})
}

func TestChessEndpointsMalformedAndEdgeCases(t *testing.T) {
	router := setupTestRouter()

	t.Run("LegalMoves with non-existent square", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"fen":    "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			"square": "z9",
		})
		req := httptest.NewRequest("POST", "/api/v1/chess/legal-moves", bytes.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for square 'z9', got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("LegalMoves with corrupted JSON body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/chess/legal-moves", bytes.NewReader([]byte("{invalid-json")))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for malformed json, got %d", w.Code)
		}
	})

	t.Run("LegalMoves with invalid FEN", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"fen":    "gibberish fen string",
			"square": "e2",
		})
		req := httptest.NewRequest("POST", "/api/v1/chess/legal-moves", bytes.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for corrupt FEN, got %d", w.Code)
		}
	})

	t.Run("ExecuteMove rejecting illegal move", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"fen":  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			"from": "e2",
			"to":   "e5", // pawn can't move 3 squares
		})
		req := httptest.NewRequest("POST", "/api/v1/chess/move", bytes.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for illegal move, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("ExecuteMove rejecting pinned piece moving into check", func(t *testing.T) {
		// White King on e1, White Rook on e2, Black Rook on e8.
		// Moving Rook off e-file (e.g. e2->a2) exposes King to check.
		pinnedFen := "4r3/8/8/8/8/8/4R3/4K3 w - - 0 1"
		body, _ := json.Marshal(map[string]string{
			"fen":  pinnedFen,
			"from": "e2",
			"to":   "a2",
		})
		req := httptest.NewRequest("POST", "/api/v1/chess/move", bytes.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for moving pinned piece into check, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("BotMove invalid FEN", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"fen":   "not a valid fen",
			"level": 3,
		})
		req := httptest.NewRequest("POST", "/api/v1/chess/bot-move", bytes.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for invalid FEN, got %d", w.Code)
		}
	})
}

func TestAuthValidationEdgeCases(t *testing.T) {
	authSrv := auth.NewService(nil, "access-secret-32-chars-long-test1", "refresh-secret-32-chars-long-test2")
	eng := engine.NewStockfishEngine()
	reviewSrv := review.NewReviewService(eng)
	router := NewRouter(authSrv, nil, nil, nil, nil, nil, nil, eng, reviewSrv, nil, nil, nil, "*")

	t.Run("Missing auth token on protected endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
		var errResp struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		json.Unmarshal(w.Body.Bytes(), &errResp)
		if errResp.Error.Code != "AUTH_REQUIRED" {
			t.Errorf("expected AUTH_REQUIRED, got %v", errResp.Error.Code)
		}
	})

	t.Run("Corrupted auth token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
		req.Header.Set("Authorization", "Bearer invalid.jwt.token.string")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized for corrupted token, got %d", w.Code)
		}
		var errResp struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		json.Unmarshal(w.Body.Bytes(), &errResp)
		if errResp.Error.Code != "AUTH_INVALID" {
			t.Errorf("expected AUTH_INVALID, got %v", errResp.Error.Code)
		}
	})
}
