package websocket

import (
	"testing"
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
