package observability

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsAndExposition(t *testing.T) {
	metrics := NewMetricRegistry("test-instance-1")
	metrics.RecordHTTPRequest("GET", "/api/v1/games/1234567890abcdef12345678", 200, 15)
	metrics.RecordHTTPRequest("POST", "/api/v1/auth/login", 401, 45)
	metrics.IncActiveWebSocket()
	metrics.SetActiveGames(3)
	metrics.SetMatchmakingQueueSize(4)
	metrics.IncStockfishEvaluation()

	// Test Prometheus Output
	reqProm := httptest.NewRequest("GET", "/metrics", nil)
	rrProm := httptest.NewRecorder()
	metrics.PrometheusHandler()(rrProm, reqProm)

	if rrProm.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /metrics, got %d", rrProm.Code)
	}

	bodyProm := rrProm.Body.String()
	if !strings.Contains(bodyProm, "chessnova_active_websocket_connections{instance=\"test-instance-1\"} 1") {
		t.Errorf("Prometheus output missing active websockets: %s", bodyProm)
	}
	if !strings.Contains(bodyProm, "chessnova_active_games{instance=\"test-instance-1\"} 3") {
		t.Errorf("Prometheus output missing active games: %s", bodyProm)
	}
	if !strings.Contains(bodyProm, "chessnova_matchmaking_queue_size{instance=\"test-instance-1\"} 4") {
		t.Errorf("Prometheus output missing queue size: %s", bodyProm)
	}
	if !strings.Contains(bodyProm, "http_requests_total{instance=\"test-instance-1\",method=\"GET\",path=\"/api/v1/games/{id}\",status=\"200\"} 1") {
		t.Errorf("Prometheus output missing normalized http request counter: %s", bodyProm)
	}

	// Test JSON Output
	reqJSON := httptest.NewRequest("GET", "/api/v1/metrics", nil)
	rrJSON := httptest.NewRecorder()
	metrics.JSONHandler()(rrJSON, reqJSON)

	if rrJSON.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/v1/metrics, got %d", rrJSON.Code)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(rrJSON.Body.Bytes(), &data); err != nil {
		t.Fatalf("failed to decode JSON metrics: %v", err)
	}
	if data["instanceId"] != "test-instance-1" {
		t.Errorf("expected instanceId test-instance-1, got %v", data["instanceId"])
	}

	chessStats := data["chess"].(map[string]interface{})
	if chessStats["activeGames"].(float64) != 3 {
		t.Errorf("expected 3 active games in JSON, got %v", chessStats["activeGames"])
	}
}

func TestHTTPLoggingMiddleware(t *testing.T) {
	metrics := NewMetricRegistry("test-inst")
	logger := NewLogger("test-inst", false)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"created":true}`))
	})

	wrapped := HTTPLoggingMiddleware(logger, metrics, handler)

	req := httptest.NewRequest("POST", "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}
