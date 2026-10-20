package friend

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type mockFriendRepo struct {
	friendships map[string]*Friendship
	challenges  map[string]*Challenge
}

func newMockFriendRepo() *mockFriendRepo {
	return &mockFriendRepo{
		friendships: make(map[string]*Friendship),
		challenges:  make(map[string]*Challenge),
	}
}

func (m *mockFriendRepo) SendFriendRequest(ctx context.Context, f *Friendship) error {
	f.ID = primitive.NewObjectID()
	f.CreatedAt = time.Now()
	f.UpdatedAt = time.Now()
	f.Status = FriendshipPending
	m.friendships[f.ID.Hex()] = f
	return nil
}

func (m *mockFriendRepo) RespondFriendRequest(ctx context.Context, id, addresseeID primitive.ObjectID, accept bool) (*Friendship, error) {
	f, ok := m.friendships[id.Hex()]
	if !ok || f.AddresseeID != addresseeID {
		return nil, primitive.ErrInvalidHex
	}
	if accept {
		f.Status = FriendshipAccepted
	} else {
		f.Status = FriendshipDeclined
	}
	f.UpdatedAt = time.Now()
	return f, nil
}

func (m *mockFriendRepo) RemoveFriend(ctx context.Context, userA, userB primitive.ObjectID) error {
	for k, f := range m.friendships {
		if (f.RequesterID == userA && f.AddresseeID == userB) || (f.RequesterID == userB && f.AddresseeID == userA) {
			delete(m.friendships, k)
			return nil
		}
	}
	return nil
}

func (m *mockFriendRepo) ListFriends(ctx context.Context, userID primitive.ObjectID) ([]Friendship, error) {
	var list []Friendship
	for _, f := range m.friendships {
		if f.Status == FriendshipAccepted && (f.RequesterID == userID || f.AddresseeID == userID) {
			list = append(list, *f)
		}
	}
	return list, nil
}

func (m *mockFriendRepo) ListPendingRequests(ctx context.Context, userID primitive.ObjectID) ([]Friendship, []Friendship, error) {
	var in, out []Friendship
	for _, f := range m.friendships {
		if f.Status == FriendshipPending {
			if f.AddresseeID == userID {
				in = append(in, *f)
			} else if f.RequesterID == userID {
				out = append(out, *f)
			}
		}
	}
	return in, out, nil
}

func (m *mockFriendRepo) GetFriendship(ctx context.Context, userA, userB primitive.ObjectID) (*Friendship, error) {
	for _, f := range m.friendships {
		if (f.RequesterID == userA && f.AddresseeID == userB) || (f.RequesterID == userB && f.AddresseeID == userA) {
			return f, nil
		}
	}
	return nil, nil
}

func (m *mockFriendRepo) CreateChallenge(ctx context.Context, c *Challenge) error {
	c.ID = primitive.NewObjectID()
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	c.Status = ChallengePending
	m.challenges[c.ID.Hex()] = c
	return nil
}

func (m *mockFriendRepo) GetChallenge(ctx context.Context, id primitive.ObjectID) (*Challenge, error) {
	c, ok := m.challenges[id.Hex()]
	if !ok {
		return nil, primitive.ErrInvalidHex
	}
	return c, nil
}

func (m *mockFriendRepo) UpdateChallenge(ctx context.Context, id primitive.ObjectID, status ChallengeStatus, gameID string) error {
	c, ok := m.challenges[id.Hex()]
	if !ok {
		return primitive.ErrInvalidHex
	}
	c.Status = status
	if gameID != "" {
		c.GameID = gameID
	}
	c.UpdatedAt = time.Now()
	return nil
}

func (m *mockFriendRepo) ListPendingChallenges(ctx context.Context, userID primitive.ObjectID) ([]*Challenge, []*Challenge, error) {
	var in, out []*Challenge
	for _, c := range m.challenges {
		if c.Status == ChallengePending {
			if c.ChallengedID == userID {
				in = append(in, c)
			} else if c.ChallengerID == userID {
				out = append(out, c)
			}
		}
	}
	return in, out, nil
}

func (m *mockFriendRepo) EnsureIndexes(ctx context.Context) error {
	return nil
}

func TestFriendAndChallengeFlow(t *testing.T) {
	repo := newMockFriendRepo()
	ctx := context.Background()

	aliceID := primitive.NewObjectID()
	bobID := primitive.NewObjectID()

	// 1. Alice sends friend request to Bob
	f := &Friendship{
		RequesterID:       aliceID,
		RequesterUsername: "alice",
		AddresseeID:       bobID,
		AddresseeUsername: "bob",
	}
	if err := repo.SendFriendRequest(ctx, f); err != nil {
		t.Fatalf("Failed to send friend request: %v", err)
	}

	// 2. Check pending requests
	in, out, err := repo.ListPendingRequests(ctx, bobID)
	if err != nil || len(in) != 1 || len(out) != 0 {
		t.Fatalf("Expected 1 incoming request for Bob, got %d in, %d out", len(in), len(out))
	}

	// 3. Bob accepts friend request
	updated, err := repo.RespondFriendRequest(ctx, f.ID, bobID, true)
	if err != nil || updated.Status != FriendshipAccepted {
		t.Fatalf("Expected accepted status, got %v (err: %v)", updated, err)
	}

	// 4. Verify in friends list
	friends, err := repo.ListFriends(ctx, aliceID)
	if err != nil || len(friends) != 1 {
		t.Fatalf("Expected 1 friend for Alice, got %d", len(friends))
	}

	// 5. Challenge creation: Rated vs Casual
	chRated := &Challenge{
		ChallengerID:       aliceID,
		ChallengerUsername: "alice",
		ChallengedID:       bobID,
		ChallengedUsername: "bob",
		InitialSeconds:     300,
		Increment:          0,
		Rated:              true,
		ColorPref:          "random",
	}
	if err := repo.CreateChallenge(ctx, chRated); err != nil {
		t.Fatalf("Failed to create rated challenge: %v", err)
	}
	if !chRated.Rated {
		t.Error("Expected challenge to be rated")
	}

	chCasual := &Challenge{
		ChallengerID:       bobID,
		ChallengerUsername: "bob",
		ChallengedID:       aliceID,
		ChallengedUsername: "alice",
		InitialSeconds:     180,
		Increment:          2,
		Rated:              false, // Casual / without ELO
		ColorPref:          "white",
	}
	if err := repo.CreateChallenge(ctx, chCasual); err != nil {
		t.Fatalf("Failed to create casual challenge: %v", err)
	}
	if chCasual.Rated {
		t.Error("Expected challenge to be casual / unrated")
	}

	// Accept casual challenge with game ID
	err = repo.UpdateChallenge(ctx, chCasual.ID, ChallengeAccepted, "game-12345")
	if err != nil {
		t.Fatalf("Failed to accept challenge: %v", err)
	}

	storedCh, err := repo.GetChallenge(ctx, chCasual.ID)
	if err != nil || storedCh.Status != ChallengeAccepted || storedCh.GameID != "game-12345" {
		t.Fatalf("Expected accepted challenge with game ID, got %v", storedCh)
	}
}
