package math32

import "math"

func Logb(x float32) float32 {
	return float32(math.Logb(float64(x)))
}

func Ilogb(x float32) int {
	return math.Ilogb(float64(x))
}
