package puzzle

import (
	"context"
	"time"

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
	Category  string             `bson:"category,omitempty" json:"category,omitempty"` // mate, capture, fork, pin, skewer, position
	Tier      int                `bson:"tier,omitempty" json:"tier,omitempty"`         // 1 to 5 progressive level
	IsDaily   bool               `bson:"isDaily,omitempty" json:"isDaily,omitempty"`
	DailyDate string             `bson:"dailyDate,omitempty" json:"dailyDate,omitempty"`
	WeekIndex int                `bson:"weekIndex,omitempty" json:"weekIndex,omitempty"` // 1-7 for weekly rotation
}

type PublicPuzzle struct {
	ID         primitive.ObjectID `json:"id"`
	LichessID  string             `json:"lichessId,omitempty"`
	FEN        string             `json:"fen"`
	Rating     int                `json:"rating"`
	Themes     []string           `json:"themes"`
	Category   string             `json:"category"`
	Tier       int                `json:"tier"`
	TotalMoves int                `json:"totalMoves"`
	IsDaily    bool               `json:"isDaily,omitempty"`
	DailyDate  string             `json:"dailyDate,omitempty"`
	WeekIndex  int                `json:"weekIndex,omitempty"`
}

func DetermineCategory(themes []string) string {
	tSet := make(map[string]bool, len(themes))
	for _, t := range themes {
		tSet[t] = true
	}
	if tSet["mate"] || tSet["mateIn1"] || tSet["mateIn2"] || tSet["mateIn3"] ||
		tSet["backRankMate"] || tSet["smotheredMate"] || tSet["bodenMate"] ||
		tSet["cornerMate"] || tSet["operaMate"] {
		return "mate"
	}
	if tSet["fork"] {
		return "fork"
	}
	if tSet["pin"] {
		return "pin"
	}
	if tSet["skewer"] {
		return "skewer"
	}
	if tSet["hangingPiece"] || tSet["advantage"] || tSet["crushing"] {
		if tSet["quietMove"] || tSet["defensiveMove"] || tSet["zugzwang"] {
			return "position"
		}
		return "capture"
	}
	return "position"
}

func DetermineTier(rating int) int {
	switch {
	case rating < 900:
		return 1 // Beginner / Novice
	case rating < 1300:
		return 2 // Intermediate
	case rating < 1650:
		return 3 // Advanced
	case rating < 2000:
		return 4 // Expert
	default:
		return 5 // Master
	}
}

func (p *Puzzle) ToPublic() PublicPuzzle {
	cat := p.Category
	if cat == "" {
		cat = DetermineCategory(p.Themes)
	}
	tier := p.Tier
	if tier == 0 {
		tier = DetermineTier(p.Rating)
	}
	return PublicPuzzle{
		ID:         p.ID,
		LichessID:  p.LichessID,
		FEN:        p.FEN,
		Rating:     p.Rating,
		Themes:     p.Themes,
		Category:   cat,
		Tier:       tier,
		TotalMoves: len(p.Solution),
		IsDaily:    p.IsDaily,
		DailyDate:  p.DailyDate,
		WeekIndex:  p.WeekIndex,
	}
}

type QueryFilter struct {
	Category  string
	Tier      int
	MinRating int
	MaxRating int
}

type Repository interface {
	GetRandomPuzzle(ctx context.Context, filter QueryFilter) (*Puzzle, error)
	GetDailyPuzzle(ctx context.Context) (*Puzzle, error)
	GetWeeklyPuzzles(ctx context.Context) ([]Puzzle, error)
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

func (r *MongoRepository) GetRandomPuzzle(ctx context.Context, filter QueryFilter) (*Puzzle, error) {
	match := bson.M{}
	if filter.Category != "" && filter.Category != "all" {
		match["category"] = filter.Category
	}
	if filter.Tier > 0 {
		match["tier"] = filter.Tier
	}
	if filter.MinRating > 0 || filter.MaxRating > 0 {
		rFilter := bson.M{}
		if filter.MinRating > 0 {
			rFilter["$gte"] = filter.MinRating
		}
		if filter.MaxRating > 0 {
			rFilter["$lte"] = filter.MaxRating
		}
		match["rating"] = rFilter
	}

	pipeline := mongo.Pipeline{}
	if len(match) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: match}})
	}
	pipeline = append(pipeline, bson.D{{Key: "$sample", Value: bson.D{{Key: "size", Value: 1}}}})

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var list []Puzzle
	if err := cursor.All(ctx, &list); err != nil || len(list) == 0 {
		// If specific filter had no results, fallback to unconstrained random sample
		if len(match) > 0 {
			fallbackPipe := mongo.Pipeline{
				{{Key: "$sample", Value: bson.D{{Key: "size", Value: 1}}}},
			}
			fbCursor, fbErr := r.collection.Aggregate(ctx, fallbackPipe)
			if fbErr == nil {
				defer fbCursor.Close(ctx)
				if err := fbCursor.All(ctx, &list); err == nil && len(list) > 0 {
					return &list[0], nil
				}
			}
		}
		return nil, mongo.ErrNoDocuments
	}
	return &list[0], nil
}

func (r *MongoRepository) GetDailyPuzzle(ctx context.Context) (*Puzzle, error) {
	todayStr := time.Now().UTC().Format("2006-01-02")
	var p Puzzle
	err := r.collection.FindOne(ctx, bson.M{"dailyDate": todayStr}).Decode(&p)
	if err == nil {
		return &p, nil
	}

	// Try any marked as daily
	err = r.collection.FindOne(ctx, bson.M{"isDaily": true}).Decode(&p)
	if err == nil {
		return &p, nil
	}

	// Fallback to random
	return r.GetRandomPuzzle(ctx, QueryFilter{})
}

func (r *MongoRepository) GetWeeklyPuzzles(ctx context.Context) ([]Puzzle, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"weekIndex": bson.M{"$gte": 1, "$lte": 7}}, options.Find().SetSort(bson.M{"weekIndex": 1}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var list []Puzzle
	if err := cursor.All(ctx, &list); err != nil || len(list) == 0 {
		// Sample 7 random puzzles if weekly set is empty
		pipeline := mongo.Pipeline{
			{{Key: "$sample", Value: bson.D{{Key: "size", Value: 7}}}},
		}
		c2, err := r.collection.Aggregate(ctx, pipeline)
		if err == nil {
			defer c2.Close(ctx)
			_ = c2.All(ctx, &list)
		}
	}
	return list, nil
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
	todayStr := time.Now().UTC().Format("2006-01-02")

	// Upsert all curated Lichess puzzles with assigned Category and Tier
	for i, p := range CuratedLichessPuzzles {
		category := DetermineCategory(p.Themes)
		tier := DetermineTier(p.Rating)
		isDaily := (i == 0) // First puzzle is designated daily
		dailyDate := ""
		if isDaily {
			dailyDate = todayStr
		}
		weekIndex := 0
		if i >= 1 && i <= 7 {
			weekIndex = i // Next 7 are weekly challenges 1..7
		}

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
				"category":  category,
				"tier":      tier,
				"isDaily":   isDaily,
				"dailyDate": dailyDate,
				"weekIndex": weekIndex,
			},
		}
		_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
		if err != nil {
			return err
		}
	}
	return nil
}
