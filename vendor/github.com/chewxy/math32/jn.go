package math32

import "math"

func Jn(n int, x float32) float32 {
	return float32(math.Jn(n, float64(x)))
}

func Yn(n int, x float32) float32 {
	return float32(math.Yn(n, float64(x)))
}
