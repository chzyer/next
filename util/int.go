package util

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
