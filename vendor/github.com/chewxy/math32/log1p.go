package math32

import "math"

func Log1p(x float32) float32 {
	return float32(math.Log1p(float64(x)))
}
