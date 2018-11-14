package math32

import "math"

func Y0(x float32) float32 {
	return float32(math.Y0(float64(x)))
}
