package math32

import "math"

func Sincos(x float32) (sin, cos float32) {
	sin64, cos64 := math.Sincos(float64(x))
	sin = float32(sin64)
	cos = float32(cos64)
	return
}
