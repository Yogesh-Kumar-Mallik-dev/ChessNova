package matchmaking

import (
	"context"
	"testing"

	"chess-platform/server/internal/game"
)

func TestToleranceForWait(t *testing.T) {
	tests := []struct {
		seconds  float64
		expected int
	}{
		{0.0, 50},
		{9.9, 50},
		{10.0, 100},
		{19.9, 100},
		{20.0, 200},
		{29.9, 200},
		{30.0, 300},
		{100.0, 300},
	}

	for _, tt := range tests {
		got := toleranceForWait(tt.seconds)
		if got != tt.expected {
			t.Errorf("toleranceForWait(%.1f) = %d; want %d", tt.seconds, got, tt.expected)
		}
	}
}

func TestMatchmakingQueueLifecycle(t *testing.T) {
	svc := NewService(nil, nil)
	ctx := context.Background()

	t1 := &Ticket{
		ID:       "t1",
		UserID:   "user1",
		Username: "player1",
		Rating:   500,
		TimeControl: game.TimeControl{
			InitialSeconds: 300,
			Increment:      0,
		},
	}

	// 1. Join queue
	if err := svc.JoinQueue(ctx, t1); err != nil {
		t.Fatalf("failed to join queue: %v", err)
	}

	// 2. Duplicate join is idempotent
	if err := svc.JoinQueue(ctx, t1); err != nil {
		t.Fatalf("failed to join queue second time: %v", err)
	}

	// 3. Status
	st, err := svc.GetStatus(ctx, "user1")
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}
	if st == nil || st.UserID != "user1" {
		t.Errorf("expected ticket for user1, got %v", st)
	}

	// 4. Leave queue
	if err := svc.LeaveQueue(ctx, "user1"); err != nil {
		t.Fatalf("failed to leave queue: %v", err)
	}

	stAfter, err := svc.GetStatus(ctx, "user1")
	if err != nil {
		t.Fatalf("error getting status after leave: %v", err)
	}
	if stAfter != nil {
		t.Errorf("expected nil ticket after leaving queue, got %v", stAfter)
	}
}
