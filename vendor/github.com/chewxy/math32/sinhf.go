package math32

import "math"

func Sinh(x float32) float32 {
	return float32(math.Sinh(float64(x)))
}
