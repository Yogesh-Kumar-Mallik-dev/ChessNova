package puzzle

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Puzzle struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FEN      string             `bson:"fen" json:"fen"`
	Solution []string           `bson:"solution" json:"solution"`
	Rating   int                `bson:"rating" json:"rating"`
	Themes   []string           `bson:"themes" json:"themes"`
}

type PublicPuzzle struct {
	ID         primitive.ObjectID `json:"id"`
	FEN        string             `json:"fen"`
	Rating     int                `json:"rating"`
	Themes     []string           `json:"themes"`
	TotalMoves int                `json:"totalMoves"`
}

func (p *Puzzle) ToPublic() PublicPuzzle {
	return PublicPuzzle{
		ID:         p.ID,
		FEN:        p.FEN,
		Rating:     p.Rating,
		Themes:     p.Themes,
		TotalMoves: len(p.Solution),
	}
}

type Repository interface {
	GetRandomPuzzle(ctx context.Context) (*Puzzle, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*Puzzle, error)
	SeedInitialPuzzles(ctx context.Context) error
}

type MongoRepository struct {
	collection *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		collection: db.Collection("puzzles"),
	}
}

func (r *MongoRepository) GetRandomPuzzle(ctx context.Context) (*Puzzle, error) {
	// Sample 1 puzzle randomly using mongo aggregation
	pipeline := mongo.Pipeline{
		{{Key: "$sample", Value: bson.D{{Key: "size", Value: 1}}}},
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var list []Puzzle
	if err := cursor.All(ctx, &list); err != nil || len(list) == 0 {
		return nil, mongo.ErrNoDocuments
	}
	return &list[0], nil
}

func (r *MongoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*Puzzle, error) {
	var p Puzzle
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *MongoRepository) SeedInitialPuzzles(ctx context.Context) error {
	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err == nil && count > 0 {
		return nil
	}

	samplePuzzles := []interface{}{
		Puzzle{
			FEN:      "r1bqk2r/pppp1ppp/2n5/4p3/2B1P1n1/3P1N2/PPP2KPP/RNBQ3R w kq - 1 6",
			Solution: []string{"f2g3", "d7d6", "h2h3"},
			Rating:   1250,
			Themes:   []string{"defense", "king_safety"},
		},
		Puzzle{
			FEN:      "6k1/5ppp/8/8/8/8/5PPP/4R1K1 w - - 0 1",
			Solution: []string{"e1e8"},
			Rating:   1000,
			Themes:   []string{"mateIn1", "backRankMate"},
		},
		Puzzle{
			FEN:      "r1b1k2r/ppppqppp/2n5/4n3/1bP2B2/5N2/PP1NPPPP/R2QKB1R w KQkq - 0 8",
			Solution: []string{"f3e5", "c6e5", "e2e3"},
			Rating:   1350,
			Themes:   []string{"opening", "skewer"},
		},
		Puzzle{
			FEN:      "r4rk1/ppp2ppp/2n5/3q4/3P4/5N2/PP1Q1PPP/R3R1K1 b - - 0 14",
			Solution: []string{"a8d8", "a1d1", "d8d6"},
			Rating:   1450,
			Themes:   []string{"middlegame", "positional"},
		},
		Puzzle{
			FEN:      "r1b2rk1/pp3ppp/8/1Bp5/3q4/8/PP3PPP/R2QR1K1 w - - 0 1",
			Solution: []string{"d1e2", "c8e6", "a1d1"},
			Rating:   1300,
			Themes:   []string{"middlegame", "tactic"},
		},
		Puzzle{
			FEN:      "3r2k1/5ppp/8/8/4N3/8/5PPP/1R4K1 w - - 0 1",
			Solution: []string{"b1b8"},
			Rating:   1100,
			Themes:   []string{"mateIn1", "backRankMate"},
		},
		Puzzle{
			FEN:      "r1bqk2r/pppp1ppp/8/4n3/1b2P3/2N5/PPP2PPP/R1BQKB1R w KQkq - 0 7",
			Solution: []string{"d1d4", "b4c3", "d4c3"},
			Rating:   1200,
			Themes:   []string{"fork", "doubleAttack"},
		},
		Puzzle{
			FEN:      "3r2k1/p4ppp/1p6/8/8/8/PP3PPP/2R3K1 w - - 0 1",
			Solution: []string{"c1c8"},
			Rating:   1050,
			Themes:   []string{"deflection", "backRankMate"},
		},
		Puzzle{
			FEN:      "r1b1kb1r/pppp1ppp/8/4q3/8/8/PPP2PPP/RNBQKB1R w KQkq - 0 8",
			Solution: []string{"f1e2", "f8e7", "e1g1"},
			Rating:   1150,
			Themes:   []string{"defense", "castling"},
		},
		Puzzle{
			FEN:      "r1bq1rk1/ppp2ppp/2n1pn2/3p4/2PP4/2N1PN2/PP3PPP/R1BQKB1R w KQ - 0 7",
			Solution: []string{"f1d3", "b7b6", "e1g1"},
			Rating:   1220,
			Themes:   []string{"opening", "development"},
		},
		Puzzle{
			FEN:      "r1b2rk1/2q1bppp/p2p1n2/np2p3/4P3/1BN1BN2/PPP2PPP/R2Q1RK1 w - - 0 13",
			Solution: []string{"c3d5", "f6d5", "b3d5"},
			Rating:   1400,
			Themes:   []string{"middlegame", "outpost"},
		},
		Puzzle{
			FEN:      "6k1/1p3p1p/p5p1/8/8/1P3qP1/P1Q2P1P/6K1 w - - 0 1",
			Solution: []string{"c2c8", "g8g7", "c8c3"},
			Rating:   1320,
			Themes:   []string{"endgame", "queenEndgame"},
		},
		Puzzle{
			FEN:      "r2q1rk1/1pp2ppp/p1np1n2/4p3/1bP1P3/2NP1B2/PP3PPP/R1BQ1RK1 w - - 0 10",
			Solution: []string{"c1g5", "h7h6", "g5f6"},
			Rating:   1250,
			Themes:   []string{"pin", "kingside"},
		},
		Puzzle{
			FEN:      "r4rk1/pp1b1ppp/1qn1pn2/2bp4/8/2NBPN2/PPPB1PPP/R2Q1RK1 w - - 0 10",
			Solution: []string{"c3a4", "b6c7", "a4c5"},
			Rating:   1380,
			Themes:   []string{"fork", "trappedPiece"},
		},
		Puzzle{
			FEN:      "5rk1/5ppp/8/8/8/8/5PPP/3RR1K1 w - - 0 1",
			Solution: []string{"d1d8", "f8d8", "e1e8"},
			Rating:   950,
			Themes:   []string{"mateIn2", "backRankMate"},
		},
	}

	_, err = r.collection.InsertMany(ctx, samplePuzzles, options.InsertMany())
	return err
}
