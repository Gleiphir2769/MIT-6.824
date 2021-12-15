package utils

import (
	"log"
	"sort"
)

// Debugging
const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

func GetMiddleNum(src []int32) int32 {
	sorted := make([]int, len(src))
	for i, num := range src {
		sorted[i] = int(num)
	}
	sort.Ints(sorted)
	return int32(sorted[len(src)/2])
}
