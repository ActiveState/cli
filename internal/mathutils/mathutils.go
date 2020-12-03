package mathutils

import (
	"math"

	"github.com/ActiveState/cli/internal/sliceutils"
)

func MaxInt(ints ...int) int {
	i, _ := sliceutils.GetInt(ints, 0)
	for _, v := range ints {
		i = int(math.Max(float64(i), float64(v)))
	}
	return i
}

func MinInt(ints ...int) int {
	i, _ := sliceutils.GetInt(ints, 0)
	for _, v := range ints {
		i = int(math.Min(float64(i), float64(v)))
	}
	return i
}

func Total(ints ...int) int {
	i := 0
	for _, v := range ints {
		i += v
	}
	return i
}
