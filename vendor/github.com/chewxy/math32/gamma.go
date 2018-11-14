package math32

import "math"

func Gamma(x float32) float32 {
	return float32(math.Gamma(float64(x)))
}
