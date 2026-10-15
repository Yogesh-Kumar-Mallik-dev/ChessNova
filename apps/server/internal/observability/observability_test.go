package observability

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gorillaWs "github.com/gorilla/websocket"
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

type mockHijackWriter struct {
	http.ResponseWriter
	hijacked bool
	flushed  bool
}

func (m *mockHijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	m.hijacked = true
	return nil, nil, nil
}

func (m *mockHijackWriter) Flush() {
	m.flushed = true
}

func TestStatusWriterInterfaces(t *testing.T) {
	mock := &mockHijackWriter{ResponseWriter: httptest.NewRecorder()}
	sw := &statusWriter{ResponseWriter: mock}

	// Verify http.Hijacker and http.Flusher interfaces are satisfied
	var _ http.Hijacker = sw
	var _ http.Flusher = sw

	_, _, err := sw.Hijack()
	if err != nil {
		t.Fatalf("unexpected hijack error: %v", err)
	}
	if !mock.hijacked {
		t.Errorf("expected underlying ResponseWriter to be hijacked")
	}

	sw.Flush()
	if !mock.flushed {
		t.Errorf("expected underlying ResponseWriter to be flushed")
	}

	// Test non-hijacker fallback error
	plainSw := &statusWriter{ResponseWriter: httptest.NewRecorder()}
	_, _, errNon := plainSw.Hijack()
	if errNon == nil {
		t.Errorf("expected error when ResponseWriter does not support Hijacker")
	}
}

func TestWebSocketUpgradePassesThroughMiddleware(t *testing.T) {
	logger := NewLogger("test", false)
	metrics := NewMetricRegistry("test")

	upgrader := gorillaWs.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	wsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("failed to upgrade websocket: %v", err)
			return
		}
		defer conn.Close()
		_ = conn.WriteMessage(gorillaWs.TextMessage, []byte(`{"type":"connected"}`))
	})

	wrapped := HTTPLoggingMiddleware(logger, metrics, wsHandler)
	ts := httptest.NewServer(wrapped)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	dialer := gorillaWs.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("expected successful websocket upgrade through middleware, got error: %v (response status %v)", err, resp)
	}
	defer conn.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("expected status 101 Switching Protocols, got %d", resp.StatusCode)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message from websocket: %v", err)
	}
	if !strings.Contains(string(msg), "connected") {
		t.Errorf("unexpected websocket message: %s", string(msg))
	}
}

func TestMatchmakingMetrics(t *testing.T) {
	metrics := NewMetricRegistry("test-mm-inst")
	metrics.IncMatchmakingJoin()
	metrics.IncMatchmakingJoin()
	metrics.IncMatchmakingLeave()
	metrics.IncMatchmakingMatch()

	// Check Prometheus output
	rrProm := httptest.NewRecorder()
	reqProm := httptest.NewRequest("GET", "/metrics", nil)
	metrics.PrometheusHandler()(rrProm, reqProm)
	body := rrProm.Body.String()

	if !strings.Contains(body, "chessnova_matchmaking_joins_total{instance=\"test-mm-inst\"} 2") {
		t.Errorf("Prometheus missing joins total: %s", body)
	}
	if !strings.Contains(body, "chessnova_matchmaking_leaves_total{instance=\"test-mm-inst\"} 1") {
		t.Errorf("Prometheus missing leaves total: %s", body)
	}
	if !strings.Contains(body, "chessnova_matchmaking_matches_total{instance=\"test-mm-inst\"} 1") {
		t.Errorf("Prometheus missing matches total: %s", body)
	}

	// Check JSON output
	rrJSON := httptest.NewRecorder()
	reqJSON := httptest.NewRequest("GET", "/api/v1/metrics", nil)
	metrics.JSONHandler()(rrJSON, reqJSON)

	var data map[string]interface{}
	if err := json.Unmarshal(rrJSON.Body.Bytes(), &data); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}
	chess := data["chess"].(map[string]interface{})
	if chess["matchmakingJoins"].(float64) != 2 {
		t.Errorf("expected 2 joins, got %v", chess["matchmakingJoins"])
	}
	if chess["matchmakingLeaves"].(float64) != 1 {
		t.Errorf("expected 1 leave, got %v", chess["matchmakingLeaves"])
	}
	if chess["matchmakingMatches"].(float64) != 1 {
		t.Errorf("expected 1 match, got %v", chess["matchmakingMatches"])
	}
}

func TestLoggerRingBufferAndHTTPHandler(t *testing.T) {
	logger := NewLogger("logger-inst", false)
	logger.Info("info message 1", map[string]interface{}{"key": "val1"})
	logger.Warn("warn message 2")
	logger.Error("error message 3", map[string]interface{}{"errCode": 500})

	// Test RecentLogs
	allLogs := logger.RecentLogs(10, "")
	if len(allLogs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(allLogs))
	}
	// RecentLogs returns newest first
	if allLogs[0].Message != "error message 3" {
		t.Errorf("expected newest log to be error message 3, got %s", allLogs[0].Message)
	}

	// Filter by level
	errLogs := logger.RecentLogs(10, LevelError)
	if len(errLogs) != 1 || errLogs[0].Message != "error message 3" {
		t.Fatalf("expected 1 error log, got %d", len(errLogs))
	}

	// Test HTTPHandler
	req := httptest.NewRequest("GET", "/api/v1/logs?level=ERROR&limit=5", nil)
	rr := httptest.NewRecorder()
	logger.HTTPHandler()(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		InstanceID string     `json:"instanceId"`
		Count      int        `json:"count"`
		Logs       []LogEntry `json:"logs"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal log response: %v", err)
	}
	if resp.Count != 1 || len(resp.Logs) != 1 {
		t.Errorf("expected 1 log in HTTP response, got %d", resp.Count)
	}
	if resp.Logs[0].Level != "ERROR" {
		t.Errorf("expected ERROR level, got %s", resp.Logs[0].Level)
	}
}


