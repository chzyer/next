package util

import "sync/atomic"

type AtomicInt int32

func (i *AtomicInt) Store(n int) {
	atomic.AddInt32((*int32)(i), int32(n))
}

func (i *AtomicInt) Add(n int) int {
	return int(atomic.AddInt32((*int32)(i), int32(n)))
}

func (i *AtomicInt) Val() int {
	return int(atomic.LoadInt32((*int32)(i)))
}

func EqualInts(i1, i2 []int) bool {
	if len(i1) != len(i2) {
		return false
	}
	for idx := range i1 {
		if i1[idx] != i2[idx] {
			return false
		}
	}
	return true
}

func InInts(a int, as []int) bool {
	for _, a2 := range as {
		if a == a2 {
			return true
		}
	}
	return false
}
