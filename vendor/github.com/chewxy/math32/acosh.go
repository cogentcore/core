package math32

import "math"

func Acosh(x float32) float32 {
	return float32(math.Acosh(float64(x)))
}
