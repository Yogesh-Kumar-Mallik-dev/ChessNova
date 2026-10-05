package chess

import "fmt"

type Square int8

const (
	A1 Square = 0; B1 Square = 1; C1 Square = 2; D1 Square = 3; E1 Square = 4; F1 Square = 5; G1 Square = 6; H1 Square = 7
	A2 Square = 8; B2 Square = 9; C2 Square = 10; D2 Square = 11; E2 Square = 12; F2 Square = 13; G2 Square = 14; H2 Square = 15
	A3 Square = 16; B3 Square = 17; C3 Square = 18; D3 Square = 19; E3 Square = 20; F3 Square = 21; G3 Square = 22; H3 Square = 23
	A4 Square = 24; B4 Square = 25; C4 Square = 26; D4 Square = 27; E4 Square = 28; F4 Square = 29; G4 Square = 30; H4 Square = 31
	A5 Square = 32; B5 Square = 33; C5 Square = 34; D5 Square = 35; E5 Square = 36; F5 Square = 37; G5 Square = 38; H5 Square = 39
	A6 Square = 40; B6 Square = 41; C6 Square = 42; D6 Square = 43; E6 Square = 44; F6 Square = 45; G6 Square = 46; H6 Square = 47
	A7 Square = 48; B7 Square = 49; C7 Square = 50; D7 Square = 51; E7 Square = 52; F7 Square = 53; G7 Square = 54; H7 Square = 55
	A8 Square = 56; B8 Square = 57; C8 Square = 58; D8 Square = 59; E8 Square = 60; F8 Square = 61; G8 Square = 62; H8 Square = 63
	NoSquare Square = -1
)

func NewSquare(file, rank int) Square {
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return NoSquare
	}
	return Square(rank*8 + file)
}

func (s Square) IsValid() bool {
	return s >= 0 && s < 64
}

func (s Square) File() int {
	return int(s % 8)
}

func (s Square) Rank() int {
	return int(s / 8)
}

func (s Square) String() string {
	if !s.IsValid() {
		return "-"
	}
	file := byte('a' + s.File())
	rank := byte('1' + s.Rank())
	return string([]byte{file, rank})
}

func ParseSquare(s string) (Square, error) {
	if len(s) != 2 {
		return NoSquare, fmt.Errorf("invalid square notation: %s", s)
	}
	file := int(s[0] - 'a')
	rank := int(s[1] - '1')
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return NoSquare, fmt.Errorf("square out of bounds: %s", s)
	}
	return NewSquare(file, rank), nil
}
