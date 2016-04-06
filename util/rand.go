package util

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func Randn(start, end int) int {
	return start + rand.Intn(end-start)
}
