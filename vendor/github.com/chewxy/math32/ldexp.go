package math32

// Ldexp is the inverse of Frexp.
// It returns frac × 2**exp.
//
// Special cases are:
//	Ldexp(±0, exp) = ±0
//	Ldexp(±Inf, exp) = ±Inf
//	Ldexp(NaN, exp) = NaN
func Ldexp(frac float32, exp int) float32 {
	return ldexp(frac, exp)
}

func ldexp(frac float32, exp int) float32 {
	// special cases
	switch {
	case frac == 0:
		return frac // correctly return -0
	case IsInf(frac, 0) || IsNaN(frac):
		return frac
	}
	frac, e := normalize(frac)
	exp += e
	x := Float32bits(frac)
	exp += int(x>>shift)&mask - bias
	if exp < -149 {
		return Copysign(0, frac) // underflow
	}
	if exp > 127 { // overflow
		if frac < 0 {
			return Inf(-1)
		}
		return Inf(1)
	}
	var m float32 = 1
	if exp < -(127 - 1) { // denormal
		exp += shift
		m = 1.0 / (1 << 23) // 1/(2**-23)
	}
	x &^= mask << shift
	x |= uint32(exp+bias) << shift
	return m * Float32frombits(x)
}
