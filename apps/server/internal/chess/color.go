package chess

import "fmt"

type Color uint8

const (
	White Color = 0
	Black Color = 1
)

func (c Color) Other() Color {
	if c == White {
		return Black
	}
	return White
}

func (c Color) String() string {
	if c == White {
		return "white"
	}
	return "black"
}

func ParseColor(s string) (Color, error) {
	switch s {
	case "w", "white", "White":
		return White, nil
	case "b", "black", "Black":
		return Black, nil
	default:
		return White, fmt.Errorf("invalid color: %s", s)
	}
}
