package math32

import "unsafe"

// Float32bits returns the IEEE 754 binary representation of f.
func Float32bits(f float32) uint32 { return *(*uint32)(unsafe.Pointer(&f)) }

// Float32frombits returns the floating point number corresponding
// to the IEEE 754 binary representation b.
func Float32frombits(b uint32) float32 { return *(*float32)(unsafe.Pointer(&b)) }

// Float64bits returns the IEEE 754 binary representation of f.
func Float64bits(f float64) uint64 { return *(*uint64)(unsafe.Pointer(&f)) }

// Float64frombits returns the floating point number corresponding
// the IEEE 754 binary representation b.
func Float64frombits(b uint64) float64 { return *(*float64)(unsafe.Pointer(&b)) }
