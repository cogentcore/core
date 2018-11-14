package math32

func Sqrt(x float32) float32

// TODO: add assembly for !build noasm
func sqrt(x float32) float32 {
	// special cases
	switch {
	case x == 0 || IsNaN(x) || IsInf(x, 1):
		return x
	case x < 0:
		return NaN()
	}
	ix := Float32bits(x)

	// normalize x
	exp := int((ix >> shift) & mask)
	if exp == 0 { // subnormal x
		for ix&(1<<shift) == 0 {
			ix <<= 1
			exp--
		}
		exp++
	}
	exp -= bias // unbias exponent
	ix &^= mask << shift
	ix |= 1 << shift
	if exp&1 == 1 { // odd exp, double x to make it even
		ix <<= 1
	}
	exp >>= 1 // exp = exp/2, exponent of square root
	// generate sqrt(x) bit by bit
	ix <<= 1
	var q, s uint32               // q = sqrt(x)
	r := uint32(1 << (shift + 1)) // r = moving bit from MSB to LSB
	for r != 0 {
		t := s + r
		if t <= ix {
			s = t + r
			ix -= t
			q += r
		}
		ix <<= 1
		r >>= 1
	}
	// final rounding
	if ix != 0 { // remainder, result not exact
		q += q & 1 // round according to extra bit
	}
	ix = q>>1 + uint32(exp-1+bias)<<shift // significand + biased exponent
	return Float32frombits(ix)

}
