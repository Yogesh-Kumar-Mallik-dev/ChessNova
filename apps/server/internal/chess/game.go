package chess

import (
	"errors"
)

type GameOutcome string

const (
	OutcomePlaying              GameOutcome = "playing"
	OutcomeCheckmate            GameOutcome = "checkmate"
	OutcomeStalemate            GameOutcome = "stalemate"
	OutcomeInsufficientMaterial GameOutcome = "insufficient_material"
	OutcomeFiftyMoves           GameOutcome = "fifty_moves"
	OutcomeThreefoldRepetition  GameOutcome = "threefold_repetition"
	OutcomeResignation          GameOutcome = "resignation"
	OutcomeTimeout              GameOutcome = "timeout"
	OutcomeDrawAgreed           GameOutcome = "draw_agreed"
	OutcomeAborted              GameOutcome = "aborted"
)

type RecordedMove struct {
	Ply       int    `json:"ply"`
	Move      Move   `json:"move"`
	SAN       string `json:"san"`
	FENBefore string `json:"fenBefore"`
	FENAfter  string `json:"fenAfter"`
}

type Game struct {
	InitialPosition Position
	CurrentPosition Position
	Moves           []RecordedMove
	Repetitions     map[string]int
	Outcome         GameOutcome
	Winner          *Color
	DrawOfferedBy   *Color
}

func NewGame() *Game {
	initPos := InitialPosition()
	g := &Game{
		InitialPosition: initPos,
		CurrentPosition: initPos.Clone(),
		Moves:           make([]RecordedMove, 0),
		Repetitions:     make(map[string]int),
		Outcome:         OutcomePlaying,
	}
	g.Repetitions[initPos.RepetitionKey()] = 1
	return g
}

func NewGameFromFEN(fen string) (*Game, error) {
	pos, err := FromFEN(fen)
	if err != nil {
		return nil, err
	}
	g := &Game{
		InitialPosition: pos,
		CurrentPosition: pos.Clone(),
		Moves:           make([]RecordedMove, 0),
		Repetitions:     make(map[string]int),
		Outcome:         OutcomePlaying,
	}
	g.Repetitions[pos.RepetitionKey()] = 1
	g.updateGameStatus()
	return g, nil
}

func (g *Game) MakeMove(m Move) (RecordedMove, error) {
	if g.Outcome != OutcomePlaying {
		return RecordedMove{}, errors.New("game is already finished")
	}

	san, err := MoveToSAN(g.CurrentPosition, m)
	if err != nil {
		return RecordedMove{}, err
	}

	fenBefore := ToFEN(g.CurrentPosition)
	nextPos, err := ApplyMove(g.CurrentPosition, m)
	if err != nil {
		return RecordedMove{}, err
	}

	fenAfter := ToFEN(nextPos)
	ply := len(g.Moves) + 1
	rec := RecordedMove{
		Ply:       ply,
		Move:      m,
		SAN:       san,
		FENBefore: fenBefore,
		FENAfter:  fenAfter,
	}

	g.Moves = append(g.Moves, rec)
	g.CurrentPosition = nextPos

	// Track threefold repetition
	repKey := nextPos.RepetitionKey()
	g.Repetitions[repKey]++

	// Reset draw offer on move
	g.DrawOfferedBy = nil

	g.updateGameStatus()

	return rec, nil
}

func (g *Game) MakeMoveSAN(san string) (RecordedMove, error) {
	m, err := ParseSAN(g.CurrentPosition, san)
	if err != nil {
		return RecordedMove{}, err
	}
	return g.MakeMove(m)
}

func (g *Game) updateGameStatus() {
	if IsCheckmate(g.CurrentPosition) {
		g.Outcome = OutcomeCheckmate
		winner := g.CurrentPosition.SideToMove.Other()
		g.Winner = &winner
		return
	}

	if IsStalemate(g.CurrentPosition) {
		g.Outcome = OutcomeStalemate
		return
	}

	if IsInsufficientMaterial(g.CurrentPosition) {
		g.Outcome = OutcomeInsufficientMaterial
		return
	}

	if IsFiftyMoveRule(g.CurrentPosition) {
		g.Outcome = OutcomeFiftyMoves
		return
	}

	if g.CanClaimThreefoldRepetition() {
		// In online chess, automatic threefold claim or available claim
		if g.Repetitions[g.CurrentPosition.RepetitionKey()] >= 3 {
			g.Outcome = OutcomeThreefoldRepetition
			return
		}
	}
}

func (g *Game) CanClaimThreefoldRepetition() bool {
	return g.Repetitions[g.CurrentPosition.RepetitionKey()] >= 3
}

func (g *Game) IsOver() bool {
	return g.Outcome != OutcomePlaying
}

func (g *Game) CanClaimFiftyMoves() bool {
	return IsFiftyMoveRule(g.CurrentPosition)
}

func (g *Game) Resign(color Color) error {
	if g.Outcome != OutcomePlaying {
		return errors.New("game is already finished")
	}
	g.Outcome = OutcomeResignation
	winner := color.Other()
	g.Winner = &winner
	return nil
}

func (g *Game) Timeout(color Color) error {
	if g.Outcome != OutcomePlaying {
		return errors.New("game is already finished")
	}
	g.Outcome = OutcomeTimeout
	winner := color.Other()
	g.Winner = &winner
	return nil
}

func (g *Game) OfferDraw(by Color) error {
	if g.Outcome != OutcomePlaying {
		return errors.New("game is already finished")
	}
	g.DrawOfferedBy = &by
	return nil
}

func (g *Game) AcceptDraw(by Color) error {
	if g.Outcome != OutcomePlaying {
		return errors.New("game is already finished")
	}
	if g.DrawOfferedBy == nil || *g.DrawOfferedBy == by {
		return errors.New("no valid draw offer to accept")
	}
	g.Outcome = OutcomeDrawAgreed
	g.DrawOfferedBy = nil
	return nil
}

func (g *Game) DeclineDraw(by Color) error {
	if g.DrawOfferedBy == nil || *g.DrawOfferedBy == by {
		return errors.New("no valid draw offer to decline")
	}
	g.DrawOfferedBy = nil
	return nil
}

func (g *Game) Abort() error {
	if g.Outcome != OutcomePlaying {
		return errors.New("game is already finished")
	}
	if len(g.Moves) > 2 {
		return errors.New("cannot abort game after move 1")
	}
	g.Outcome = OutcomeAborted
	return nil
}

func (g *Game) Result() string {
	if g.Outcome == OutcomePlaying {
		return "*"
	}
	if g.Winner != nil {
		if *g.Winner == White {
			return "1-0"
		}
		return "0-1"
	}
	if g.Outcome == OutcomeAborted {
		return "*"
	}
	return "1/2-1/2"
}

func (g *Game) ToPGN(headers map[string]string) string {
	pgn := NewPGNGame()
	for k, v := range headers {
		pgn.SetHeader(k, v)
	}
	pgn.Result = g.Result()
	for _, m := range g.Moves {
		pgn.AddMoveWithSquares(m.Ply, m.Move.From.String(), m.Move.To.String(), m.SAN, m.FENAfter)
	}
	return pgn.Export()
}
