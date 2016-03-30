package util

import (
	"fmt"
	"strings"

	"strconv"
)

const (
	B Unit = 1 << (iota * 10)
	KB
	MB
	GB
	PB
)

var unitmap = map[string]int64{}

type Unit int64
type Size Unit

func ParseUnit(s string) (Unit, error) {
	s = strings.ToLower(s)
	switch s {
	case "b", "":
		return B, nil
	case "kb", "k":
		return KB, nil
	case "mb", "m":
		return MB, nil
	case "gb", "g":
		return GB, nil
	case "pb", "p":
		return PB, nil
	default:
		return B, fmt.Errorf("unknown unit: %v", s)
	}
}

func ParseSize(s string) (Size, error) {
	runes := []rune(s)
	idx := -1
	for i, r := range runes {
		if !((r >= '0' && r <= '9') || r == '.') {
			idx = i
			break
		}
	}
	unit := ""
	number := s
	if idx > 0 {
		number = string(runes[:idx])
		unit = string(runes[idx:])
	}
	n, err := strconv.ParseFloat(number, 64)
	if err != nil {
		return 0, err
	}
	u, err := ParseUnit(unit)
	if err != nil {
		return 0, err
	}

	return Size(n * float64(u)), nil

}
