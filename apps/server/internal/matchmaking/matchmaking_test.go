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

func TestJoinQueueAutoResignsExistingGame(t *testing.T) {
	ctx := context.Background()
	gameSvc := game.NewService(nil, nil)
	mmSvc := NewService(nil, gameSvc)

	u1 := game.PlayerInfo{UserID: "alice", Username: "Alice", Rating: 1500}
	u2 := game.PlayerInfo{UserID: "bob", Username: "Bob", Rating: 1500}
	tc := game.TimeControl{InitialSeconds: 300, Increment: 0}

	// Alice starts a match with Bob
	ag, err := gameSvc.CreateGame(u1, u2, tc)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}
	if ag.Status != game.StatusPlaying {
		t.Fatalf("expected game status to be playing, got %s", ag.Status)
	}

	// Alice tries to join matchmaking while match is still active
	ticket := &Ticket{
		ID:          "ticket-alice",
		UserID:      "alice",
		Username:    "Alice",
		Rating:      1500,
		TimeControl: tc,
	}

	// Joining queue MUST auto-resign the previous match without error
	if err := mmSvc.JoinQueue(ctx, ticket); err != nil {
		t.Fatalf("expected JoinQueue to succeed by auto-resigning existing match, got error: %v", err)
	}

	// Verify previous game is cleanly finished
	activeAg, _ := gameSvc.GetActiveGame(ag.ID)
	if activeAg.Status != game.StatusFinished {
		t.Errorf("expected old game to be finished, got %s", activeAg.Status)
	}

	// Verify alice is in queue
	st, err := mmSvc.GetStatus(ctx, "alice")
	if err != nil || st == nil {
		t.Fatalf("expected alice to be in matchmaking queue, got %v", st)
	}
}
