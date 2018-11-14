package math32

import "math"

func Acos(x float32) float32 {
	return float32(math.Acos(float64(x)))
}
