package puzzle

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Puzzle struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	LichessID string             `bson:"lichessId,omitempty" json:"lichessId,omitempty"`
	FEN       string             `bson:"fen" json:"fen"`
	Solution  []string           `bson:"solution" json:"solution"`
	Rating    int                `bson:"rating" json:"rating"`
	Themes    []string           `bson:"themes" json:"themes"`
}

type PublicPuzzle struct {
	ID         primitive.ObjectID `json:"id"`
	LichessID  string             `json:"lichessId,omitempty"`
	FEN        string             `json:"fen"`
	Rating     int                `json:"rating"`
	Themes     []string           `json:"themes"`
	TotalMoves int                `json:"totalMoves"`
}

func (p *Puzzle) ToPublic() PublicPuzzle {
	return PublicPuzzle{
		ID:         p.ID,
		LichessID:  p.LichessID,
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
	if err == nil && count >= int64(len(CuratedLichessPuzzles)) {
		return nil
	}

	// Insert or replace with rich curated Lichess tactical puzzle set
	for _, p := range CuratedLichessPuzzles {
		filter := bson.M{"lichessId": p.LichessID}
		if p.LichessID == "" {
			filter = bson.M{"fen": p.FEN}
		}
		update := bson.M{
			"$set": bson.M{
				"lichessId": p.LichessID,
				"fen":       p.FEN,
				"solution":  p.Solution,
				"rating":    p.Rating,
				"themes":    p.Themes,
			},
		}
		_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
		if err != nil {
			return err
		}
	}
	return nil
}
