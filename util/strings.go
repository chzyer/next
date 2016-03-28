package util

import "strings"

func FillString(s string, n int, ch string) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(ch, n-len(s))
}

func IntIn(s int, ss []int) bool {
	for _, sss := range ss {
		if sss == s {
			return true
		}
	}
	return false
}

func In(s string, ss []string) bool {
	for _, sss := range ss {
		if sss == s {
			return true
		}
	}
	return false
}
