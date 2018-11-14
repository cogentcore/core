package math32

import "math"

func Lgamma(x float32) (lgamma float32, sign int) {
	lg, sign := math.Lgamma(float64(x))
	return float32(lg), sign
}
