package math32

import "math"

func Log2(x float32) float32 {
	return float32(math.Log2(float64(x)))
}
