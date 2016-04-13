package util

import (
	"fmt"
	"strings"
	"sync/atomic"

	"strconv"
)

const (
	B Unit = 1 << (iota * 10)
	KB
	MB
	GB
	PB
)

type Unit int64
type Size Unit

func (u Unit) String() string {
	val := float64(u)
	var unit string
	if u >= PB {
		val /= float64(PB)
		unit = "PB"
	} else if u > GB {
		val /= float64(GB)
		unit = "GB"
	} else if u > MB {
		val /= float64(MB)
		unit = "MB"
	} else if u > KB {
		val /= float64(KB)
		unit = "KB"
	} else {
		unit = "B"
	}
	n := fmt.Sprintf("%.2f", val)
	n = strings.TrimRight(n, "0.")
	if n == "" {
		n = "0"
	}
	return n + unit
}

func (u *Unit) Add(n Unit) {
	atomic.AddInt64((*int64)(u), int64(n))
}

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
