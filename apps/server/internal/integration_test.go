package internal_test

import (
	"context"
	"testing"
	"time"

	"chess-platform/server/internal/auth"
	"chess-platform/server/internal/chess"
	"chess-platform/server/internal/game"
	"chess-platform/server/internal/rating"
	"chess-platform/server/internal/user"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCompleteGameIntegrationFlow(t *testing.T) {
	// Connect to local MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("skipping integration test, MongoDB not reachable")
		return
	}
	if err := client.Ping(ctx, nil); err != nil {
		t.Skip("skipping integration test, MongoDB ping failed")
		return
	}

	testDB := client.Database("chess_test")
	defer func() {
		_ = testDB.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	}()

	userRepo := user.NewMongoRepository(testDB)
	_ = userRepo.EnsureIndexes(ctx)

	gameRepo := game.NewMongoRepository(testDB)
	_ = gameRepo.EnsureIndexes(ctx)

	ratingRepo := rating.NewMongoRepository(testDB)
	_ = ratingRepo.EnsureIndexes(ctx)

	authService := auth.NewService(userRepo, "test-access-secret-12345", "test-refresh-secret-67890")
	gameService := game.NewService(gameRepo, ratingRepo)

	// Step 1: Register Player A and Player B
	userA, tokensA, err := authService.Register(ctx, "magnus", "magnus@example.com", "superpass123")
	if err != nil {
		t.Fatalf("failed to register player A: %v", err)
	}
	if tokensA.AccessToken == "" {
		t.Fatalf("expected access token for player A")
	}

	userB, tokensB, err := authService.Register(ctx, "hikaru", "hikaru@example.com", "superpass456")
	if err != nil {
		t.Fatalf("failed to register player B: %v", err)
	}
	if tokensB.AccessToken == "" {
		t.Fatalf("expected access token for player B")
	}

	// Step 2: Create game between Player A (White) and Player B (Black)
	tc := game.TimeControl{InitialSeconds: 180, Increment: 2} // 3+2 blitz
	whitePlayer := game.PlayerInfo{UserID: userA.ID.Hex(), Username: userA.Username, Rating: rating.DefaultRating}
	blackPlayer := game.PlayerInfo{UserID: userB.ID.Hex(), Username: userB.Username, Rating: rating.DefaultRating}

	ag, err := gameService.CreateGame(whitePlayer, blackPlayer, tc)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}
	if ag.ID == "" {
		t.Fatalf("expected non-empty game ID")
	}

	// Step 3: Play Scholar's Mate moves
	// 1. e4 e5
	// 2. Bc4 Nc6
	// 3. Qh5 Nf6
	// 4. Qxf7#
	gameMoves := []struct {
		userID string
		from   chess.Square
		to     chess.Square
		san    string
	}{
		{userA.ID.Hex(), chess.E2, chess.E4, "e4"},
		{userB.ID.Hex(), chess.E7, chess.E5, "e5"},
		{userA.ID.Hex(), chess.F1, chess.C4, "Bc4"},
		{userB.ID.Hex(), chess.B8, chess.C6, "Nc6"},
		{userA.ID.Hex(), chess.D1, chess.H5, "Qh5"},
		{userB.ID.Hex(), chess.G8, chess.F6, "Nf6"},
		{userA.ID.Hex(), chess.H5, chess.F7, "Qxf7#"},
	}

	for i, m := range gameMoves {
		res, err := gameService.MakeMove(ctx, ag.ID, m.userID, chess.NewMove(m.from, m.to, nil))
		if err != nil {
			t.Fatalf("move %d (%s) failed: %v", i+1, m.san, err)
		}
		if res.SAN != m.san {
			t.Fatalf("move %d expected SAN %s, got %s", i+1, m.san, res.SAN)
		}
	}

	// Verify game ended in checkmate with White winning
	if ag.Status != game.StatusFinished {
		t.Fatalf("expected game status to be finished, got %s", ag.Status)
	}
	if ag.Engine.Outcome != chess.OutcomeCheckmate {
		t.Fatalf("expected outcome checkmate, got %s", ag.Engine.Outcome)
	}
	if ag.Engine.Result() != "1-0" {
		t.Fatalf("expected result 1-0, got %s", ag.Engine.Result())
	}

	// Step 4: Verify Game was persisted in MongoDB
	savedDoc, err := gameRepo.FindByID(ctx, ag.ID)
	if err != nil {
		t.Fatalf("failed to find saved game in MongoDB: %v", err)
	}
	if savedDoc.Result != "1-0" {
		t.Fatalf("expected saved result 1-0, got %s", savedDoc.Result)
	}
	if len(savedDoc.Moves) != 7 {
		t.Fatalf("expected 7 saved moves, got %d", len(savedDoc.Moves))
	}
	if savedDoc.PGN == "" {
		t.Fatalf("expected saved PGN to be non-empty")
	}

	// Step 5: Verify Ratings updated in MongoDB
	ratingA, err := ratingRepo.GetRating(ctx, userA.ID, rating.Blitz)
	if err != nil {
		t.Fatalf("failed to get rating for player A: %v", err)
	}
	if ratingA.Rating <= rating.DefaultRating {
		t.Fatalf("expected winner rating to increase above %d, got %d", rating.DefaultRating, ratingA.Rating)
	}
	if ratingA.Wins != 1 {
		t.Fatalf("expected 1 win for player A, got %d", ratingA.Wins)
	}

	ratingB, err := ratingRepo.GetRating(ctx, userB.ID, rating.Blitz)
	if err != nil {
		t.Fatalf("failed to get rating for player B: %v", err)
	}
	if ratingB.Rating >= rating.DefaultRating {
		t.Fatalf("expected loser rating to decrease below %d, got %d", rating.DefaultRating, ratingB.Rating)
	}
	if ratingB.Losses != 1 {
		t.Fatalf("expected 1 loss for player B, got %d", ratingB.Losses)
	}

	// Step 6: Verify user game history retrieval
	userGames, err := gameRepo.FindUserGames(ctx, userA.ID.Hex(), 10, 0)
	if err != nil {
		t.Fatalf("failed to find user games: %v", err)
	}
	if len(userGames) != 1 {
		t.Fatalf("expected 1 user game in history, got %d", len(userGames))
	}
}
