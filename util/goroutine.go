package util

import (
	"runtime"
	"strings"
)

func GetRuntimeStackInfo() []byte {
	stack := make([]byte, 1024)
	n := 0
	for {
		n = runtime.Stack(stack, true)
		if n == cap(stack) {
			stack = make([]byte, cap(stack)*2)
			continue
		}
		break
	}
	return stack[:n]
}

func GetRuntimeStack() []string {
	sp := strings.Split(string(GetRuntimeStackInfo()), "\n\n")
	return sp
}

func FindRuntimeStack(s string) []string {
	stacks := GetRuntimeStack()
	s = strings.ToLower(s)
	var ret []string
	for _, stack := range stacks {
		if strings.Contains(strings.ToLower(stack), s) {
			ret = append(ret, stack)
		}
	}
	return ret
}
