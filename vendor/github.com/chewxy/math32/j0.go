package math32

import "math"

func J0(x float32) float32 {
	return float32(math.J0(float64(x)))
}
