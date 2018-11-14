package math32

import "math"

func Asinh(x float32) float32 {
	return float32(math.Asinh(float64(x)))
}
