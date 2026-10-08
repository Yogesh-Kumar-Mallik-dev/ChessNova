package chess

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

type PGNMove struct {
	Ply  int    `json:"ply"`
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
	SAN  string `json:"san"`
	FEN  string `json:"fen"`
}

type PGNGame struct {
	Headers map[string]string `json:"headers"`
	Moves   []PGNMove         `json:"moves"`
	Result  string            `json:"result"`
}

func NewPGNGame() *PGNGame {
	return &PGNGame{
		Headers: make(map[string]string),
		Moves:   make([]PGNMove, 0),
		Result:  "*",
	}
}

func (pg *PGNGame) SetHeader(key, value string) {
	pg.Headers[key] = value
}

func (pg *PGNGame) AddMove(ply int, san, fen string) {
	pg.Moves = append(pg.Moves, PGNMove{Ply: ply, SAN: san, FEN: fen})
}

func (pg *PGNGame) AddMoveWithSquares(ply int, from, to, san, fen string) {
	pg.Moves = append(pg.Moves, PGNMove{Ply: ply, From: from, To: to, SAN: san, FEN: fen})
}

func (pg *PGNGame) Export() string {
	var sb strings.Builder

	// Seven Tag Roster default order
	standardTags := []string{"Event", "Site", "Date", "Round", "White", "Black", "Result"}
	seen := make(map[string]bool)

	for _, tag := range standardTags {
		val, ok := pg.Headers[tag]
		if !ok {
			if tag == "Result" {
				val = pg.Result
			} else {
				val = "?"
			}
		}
		sb.WriteString(fmt.Sprintf("[%s \"%s\"]\n", tag, val))
		seen[tag] = true
	}

	for k, v := range pg.Headers {
		if !seen[k] {
			sb.WriteString(fmt.Sprintf("[%s \"%s\"]\n", k, v))
		}
	}

	sb.WriteString("\n")

	// Format moves into lines of max ~80 chars
	var moveLine strings.Builder
	for i, m := range pg.Moves {
		moveStr := ""
		if i%2 == 0 {
			moveNum := (i / 2) + 1
			moveStr = fmt.Sprintf("%d. %s ", moveNum, m.SAN)
		} else {
			moveStr = fmt.Sprintf("%s ", m.SAN)
		}

		if moveLine.Len()+len(moveStr) > 80 {
			sb.WriteString(strings.TrimRight(moveLine.String(), " ") + "\n")
			moveLine.Reset()
		}
		moveLine.WriteString(moveStr)
	}

	resStr := pg.Result
	if resStr == "" {
		resStr = "*"
	}
	moveLine.WriteString(resStr)
	sb.WriteString(moveLine.String() + "\n")

	return sb.String()
}

var tagRegex = regexp.MustCompile(`^\[([A-Za-z0-9_]+)\s+"(.*)"\]$`)

func ParsePGN(content string) (*PGNGame, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	game := NewPGNGame()
	var moveTextBuilder strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			matches := tagRegex.FindStringSubmatch(line)
			if len(matches) == 3 {
				game.SetHeader(matches[1], matches[2])
				if matches[1] == "Result" {
					game.Result = matches[2]
				}
			}
		} else {
			moveTextBuilder.WriteString(line)
			moveTextBuilder.WriteString(" ")
		}
	}

	// Tokenize moveText, strip comments { ... }, recursive variations, etc.
	text := moveTextBuilder.String()
	commentRegex := regexp.MustCompile(`\{[^}]*\}`)
	text = commentRegex.ReplaceAllString(text, "")
	variationRegex := regexp.MustCompile(`\([^)]*\)`)
	text = variationRegex.ReplaceAllString(text, "")

	tokens := strings.Fields(text)
	pos := InitialPosition()
	if fen, ok := game.Headers["FEN"]; ok && strings.TrimSpace(fen) != "" {
		if customPos, err := FromFEN(fen); err == nil {
			pos = customPos
		}
	}

	ply := 1

	for _, token := range tokens {
		// Ignore NAGs ($1, $2, etc.)
		if strings.HasPrefix(token, "$") {
			continue
		}
		// check if token is result
		if token == "1-0" || token == "0-1" || token == "1/2-1/2" || token == "*" {
			game.Result = token
			break
		}
		// skip move numbers like "1.", "12..." or inline moves like "1.e4"
		if strings.Contains(token, ".") {
			parts := strings.Split(token, ".")
			token = parts[len(parts)-1]
			if token == "" {
				continue
			}
		}

		m, err := ParseSAN(pos, token)
		if err != nil {
			return nil, fmt.Errorf("failed to parse move '%s' at ply %d: %w", token, ply, err)
		}

		san, _ := MoveToSAN(pos, m)
		nextPos, err := ApplyMove(pos, m)
		if err != nil {
			return nil, err
		}
		game.AddMoveWithSquares(ply, m.From.String(), m.To.String(), san, ToFEN(nextPos))
		pos = nextPos
		ply++
	}

	return game, nil
}
