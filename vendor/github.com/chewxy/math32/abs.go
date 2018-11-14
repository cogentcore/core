package math32

// Abs returns the absolute value of x.
//
// Special cases are:
//	Abs(±Inf) = +Inf
//	Abs(NaN) = NaN
func Abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	if x == 0 {
		return 0 // return correctly abs(-0)
	}
	return x

	// asUint := Float32bits(x) & uint32(0x7FFFFFFF)
	// return Float32frombits(asUint)
}
