package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"chess-platform/server/internal/engine"
	"chess-platform/server/internal/review"
)

func setupTestRouter() *Router {
	eng := engine.NewStockfishEngine()
	reviewSrv := review.NewReviewService(eng)
	return NewRouter(nil, nil, nil, nil, nil, nil, nil, eng, reviewSrv, nil, "*")
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
