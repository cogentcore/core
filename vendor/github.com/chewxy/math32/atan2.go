package math32

import "math"

func Atan2(x, y float32) float32 {
	return float32(math.Atan2(float64(x), float64(y)))
}
