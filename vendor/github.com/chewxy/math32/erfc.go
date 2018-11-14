package math32

import "math"

func Erfc(x float32) float32 {
	return float32(math.Erfc(float64(x)))
}
