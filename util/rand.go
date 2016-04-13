package util

import (
	"math/rand"
	"time"
)

var (
	letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

func init() {
	rand.Seed(time.Now().Unix())
}

func RandChoiseInt(i []int) int {
	return i[rand.Intn(len(i))]
}

func Randn(start, end int) int {
	return start + rand.Intn(end-start)
}

func RandStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
