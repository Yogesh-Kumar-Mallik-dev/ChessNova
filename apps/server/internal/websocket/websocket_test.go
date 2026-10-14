package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gorillaWs "github.com/gorilla/websocket"
)

func TestHubRoomCreationAndBroadcast(t *testing.T) {
	hub := NewHub(nil, nil)

	// 1. Create/Get Room
	room1 := hub.getOrCreateRoom("game-123")
	if room1 == nil {
		t.Fatal("expected valid room")
	}
	if room1.gameID != "game-123" {
		t.Errorf("expected gameID game-123, got %s", room1.gameID)
	}

	// 2. Reuse existing room
	room2 := hub.getOrCreateRoom("game-123")
	if room1 != room2 {
		t.Error("expected getOrCreateRoom to return identical instance")
	}

	// 3. Broadcast to empty room (should not panic or deadlock)
	hub.Broadcast("game-123", ServerMessage{
		Type:   "test_message",
		GameID: "game-123",
	})

	// 4. Broadcast to non-existent room (should be a clean no-op)
	hub.Broadcast("game-999", ServerMessage{
		Type: "noop",
	})
}

func TestWebSocketOriginValidation(t *testing.T) {
	cfg := HubConfig{
		AllowedOrigins: []string{
			"http://localhost:5173",
			"https://chessnova.com",
			"tauri://localhost",
			"http://tauri.localhost",
		},
		AllowMissingOrigin: true,
		AllowDevLocalhost:  true,
	}

	tests := []struct {
		name    string
		origin  string
		allowed bool
	}{
		// Trusted origins
		{"Allowed configured origin", "http://localhost:5173", true},
		{"Allowed production origin", "https://chessnova.com", true},
		{"Allowed desktop tauri scheme", "tauri://localhost", true},
		{"Allowed desktop http tauri", "http://tauri.localhost", true},
		{"Allowed dev localhost with port", "http://localhost:3000", true},
		{"Allowed dev loopback 127.0.0.1", "http://127.0.0.1:8080", true},

		// Hostile origins
		{"Hostile external origin", "https://evil-attacker.com", false},
		{"Hostile subdomain trick", "http://localhost.evil.com", false},
		{"Hostile suffix attack", "http://chessnova.com.attacker.com", false},
		{"Hostile http instead of https prod", "http://chessnova.com", false},

		// Malformed origins
		{"Malformed empty scheme", "://broken", false},
		{"Malformed javascript scheme", "javascript:alert(1)", false},
		{"Malformed random text", "not-a-valid-origin", false},
		{"Malformed empty host", "http://", false},

		// Missing origin
		{"Missing origin when allowed", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateOrigin(tc.origin, cfg)
			if got != tc.allowed {
				t.Errorf("ValidateOrigin(%q) = %v; want %v", tc.origin, got, tc.allowed)
			}
		})
	}

	// Test missing origin rejection when disabled
	strictCfg := cfg
	strictCfg.AllowMissingOrigin = false
	if ValidateOrigin("", strictCfg) {
		t.Error("expected missing origin to be rejected when AllowMissingOrigin is false")
	}

	// Test same-host matching (e.g. Origin: https://chess.yogeshmallik.in with Host: chess.yogeshmallik.in)
	if !ValidateOrigin("https://chess.yogeshmallik.in", strictCfg, "chess.yogeshmallik.in") {
		t.Error("expected same-host origin to be allowed")
	}
	if !ValidateOrigin("https://chess.yogeshmallik.in", strictCfg, "chess.yogeshmallik.in:443") {
		t.Error("expected same-host origin with port to be allowed")
	}
}

func TestWebSocketUpgradeOriginRejection(t *testing.T) {
	cfg := HubConfig{
		AllowedOrigins:     []string{"http://localhost:5173"},
		AllowMissingOrigin: false,
		AllowDevLocalhost:  false,
	}
	hub := NewHub(nil, nil, cfg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r, "test-game")
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// 1. Connection with untrusted origin should fail
	dialer := gorillaWs.Dialer{}
	header := http.Header{}
	header.Set("Origin", "https://evil.com")
	_, resp, err := dialer.Dial(wsURL, header)
	if err == nil {
		t.Fatal("expected error dialing with untrusted origin")
	}
	if resp != nil && resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", resp.StatusCode)
	}

	// 2. Connection with trusted origin should succeed
	header.Set("Origin", "http://localhost:5173")
	conn, resp, err := dialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("expected dial to succeed with trusted origin: %v", err)
	}
	conn.Close()
}
