package math32

import "math"

func J1(x float32) float32 {
	return float32(math.J1(float64(x)))
}
