package math32

import "math"

func Atanh(x float32) float32 {
	return float32(math.Atanh(float64(x)))
}
