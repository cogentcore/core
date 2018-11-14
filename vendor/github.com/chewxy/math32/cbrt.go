package math32

import "math"

func Cbrt(x float32) float32 {
	return float32(math.Cbrt(float64(x)))
}
