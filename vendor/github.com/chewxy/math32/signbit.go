package math32

// Signbit returns true if x is negative or negative zero.
func Signbit(x float32) bool {
	return Float32bits(x)&(1<<31) != 0
}
