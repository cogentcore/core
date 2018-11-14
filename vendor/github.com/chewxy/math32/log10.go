package math32

import "math"

func Log10(x float32) float32 {
	return float32(math.Log10(float64(x)))
}
