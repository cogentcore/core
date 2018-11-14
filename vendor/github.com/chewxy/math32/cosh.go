package math32

import "math"

func Cosh(x float32) float32 {
	return float32(math.Cosh(float64(x)))
}
