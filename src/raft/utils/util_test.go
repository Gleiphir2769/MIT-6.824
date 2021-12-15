package utils

import (
	"fmt"
	"testing"
)

func TestGetMiddleNum(t *testing.T) {
	a := []int32{0, -1, -1}
	fmt.Println(GetMiddleNum(a))
}
