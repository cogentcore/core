package math32

import "math"

func Atan(x float32) float32 {
	return float32(math.Atan(float64(x)))
}
