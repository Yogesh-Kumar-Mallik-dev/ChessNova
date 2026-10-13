package game

import (
	"context"
	"testing"

	"chess-platform/server/internal/chess"
	"chess-platform/server/internal/rating"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type mockGameRepo struct {
	games map[string]*GameDocument
}

func newMockGameRepo() *mockGameRepo {
	return &mockGameRepo{games: make(map[string]*GameDocument)}
}

func (m *mockGameRepo) Save(ctx context.Context, doc *GameDocument) error {
	m.games[doc.ID] = doc
	return nil
}

func (m *mockGameRepo) FindByID(ctx context.Context, id string) (*GameDocument, error) {
	if g, ok := m.games[id]; ok {
		return g, nil
	}
	return nil, ErrGameNotFound
}

func (m *mockGameRepo) FindUserGames(ctx context.Context, userID string, limit, skip int) ([]GameDocument, error) {
	var list []GameDocument
	for _, g := range m.games {
		if g.White.UserID == userID || g.Black.UserID == userID {
			list = append(list, *g)
		}
	}
	return list, nil
}

func (m *mockGameRepo) FindRecentGames(ctx context.Context, limit int) ([]GameDocument, error) {
	var list []GameDocument
	for _, g := range m.games {
		list = append(list, *g)
	}
	return list, nil
}

func (m *mockGameRepo) EnsureIndexes(ctx context.Context) error {
	return nil
}

type mockRatingRepo struct {
	ratings map[string]*rating.UserRating
}

func newMockRatingRepo() *mockRatingRepo {
	return &mockRatingRepo{ratings: make(map[string]*rating.UserRating)}
}

func (m *mockRatingRepo) GetRating(ctx context.Context, userID primitive.ObjectID, cat rating.Category) (*rating.UserRating, error) {
	key := userID.Hex() + string(cat)
	if r, ok := m.ratings[key]; ok {
		return r, nil
	}
	return &rating.UserRating{
		UserID:   userID,
		Category: cat,
		Rating:   rating.DefaultRating,
	}, nil
}

func (m *mockRatingRepo) UpsertRating(ctx context.Context, ur *rating.UserRating) error {
	key := ur.UserID.Hex() + string(ur.Category)
	m.ratings[key] = ur
	return nil
}

func (m *mockRatingRepo) GetLeaderboard(ctx context.Context, cat rating.Category, limit int) ([]rating.UserRating, error) {
	return nil, nil
}

func (m *mockRatingRepo) EnsureIndexes(ctx context.Context) error {
	return nil
}

func TestGameServiceLifecycle(t *testing.T) {
	gRepo := newMockGameRepo()
	rRepo := newMockRatingRepo()
	svc := NewService(gRepo, rRepo)
	ctx := context.Background()

	white := PlayerInfo{UserID: primitive.NewObjectID().Hex(), Username: "whitePlayer", Rating: 450}
	black := PlayerInfo{UserID: primitive.NewObjectID().Hex(), Username: "blackPlayer", Rating: 480}
	tc := TimeControl{InitialSeconds: 300, Increment: 0}

	// 1. Create Game
	ag, err := svc.CreateGame(white, black, tc)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}
	if ag.ID == "" {
		t.Error("expected non-empty game ID")
	}

	// 2. Fetch Active Game
	fetched, err := svc.GetActiveGame(ag.ID)
	if err != nil {
		t.Fatalf("failed to fetch active game: %v", err)
	}
	if fetched.White.Username != "whitePlayer" {
		t.Errorf("expected whitePlayer, got %s", fetched.White.Username)
	}

	// 3. Make Move (1. e4)
	e2, _ := chess.ParseSquare("e2")
	e4, _ := chess.ParseSquare("e4")
	moveEvent, err := svc.MakeMove(ctx, ag.ID, white.UserID, chess.Move{From: e2, To: e4})
	if err != nil {
		t.Fatalf("failed to make move e4: %v", err)
	}
	if moveEvent.SAN != "e4" {
		t.Errorf("expected SAN e4, got %s", moveEvent.SAN)
	}

	// 4. Try making move out of turn (White plays again) -> Should fail
	e4Sq, _ := chess.ParseSquare("e4")
	e5Sq, _ := chess.ParseSquare("e5")
	_, err = svc.MakeMove(ctx, ag.ID, white.UserID, chess.Move{From: e4Sq, To: e5Sq})
	if err != ErrNotPlayerTurn {
		t.Errorf("expected ErrNotPlayerTurn, got %v", err)
	}

	// 5. Black replies 1... e5
	e7, _ := chess.ParseSquare("e7")
	e5, _ := chess.ParseSquare("e5")
	moveEvent2, err := svc.MakeMove(ctx, ag.ID, black.UserID, chess.Move{From: e7, To: e5})
	if err != nil {
		t.Fatalf("failed to make move e5: %v", err)
	}
	if moveEvent2.SAN != "e5" {
		t.Errorf("expected SAN e5, got %s", moveEvent2.SAN)
	}

	// 6. Draw Negotiation
	offeredColor, err := svc.OfferDraw(ag.ID, white.UserID)
	if err != nil {
		t.Fatalf("failed to offer draw: %v", err)
	}
	if offeredColor != chess.White {
		t.Errorf("expected offeredColor White, got %v", offeredColor)
	}

	// Accept Draw
	doc, err := svc.AcceptDraw(ctx, ag.ID, black.UserID)
	if err != nil {
		t.Fatalf("failed to accept draw: %v", err)
	}
	if doc.Result != "1/2-1/2" {
		t.Errorf("expected result 1/2-1/2, got %s", doc.Result)
	}
	if doc.Status != StatusFinished {
		t.Errorf("expected status finished, got %v", doc.Status)
	}
}

func TestGameServiceResignation(t *testing.T) {
	gRepo := newMockGameRepo()
	rRepo := newMockRatingRepo()
	svc := NewService(gRepo, rRepo)
	ctx := context.Background()

	white := PlayerInfo{UserID: primitive.NewObjectID().Hex(), Username: "whitePlayer", Rating: 450}
	black := PlayerInfo{UserID: primitive.NewObjectID().Hex(), Username: "blackPlayer", Rating: 480}
	tc := TimeControl{InitialSeconds: 180, Increment: 0}

	ag, err := svc.CreateGame(white, black, tc)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Black resigns
	doc, err := svc.Resign(ctx, ag.ID, black.UserID)
	if err != nil {
		t.Fatalf("failed to resign: %v", err)
	}
	if doc.Result != "1-0" {
		t.Errorf("expected result 1-0 on black resignation, got %s", doc.Result)
	}
}
