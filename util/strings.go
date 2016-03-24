package util

import "strings"

func FillString(s string, n int, ch string) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(ch, n-len(s))
}
