package math32

import "math"

func Asin(x float32) float32 {
	return float32(math.Asin(float64(x)))
}
