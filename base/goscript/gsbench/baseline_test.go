// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gsbench

import "testing"

// The Baseline benchmarks use only basic types, operators and control flow,
// which JavaScript and WebAssembly represent directly, to measure the best
// performance that can be expected from code compiled to JavaScript relative
// to WebAssembly and native code.

var (
	baseSinkI   int
	baseSinkF   float64
	baseSinkF32 float32
)

func BenchmarkBaselineIntArith(b *testing.B) {
	for range b.N {
		s := 0
		for i := range 10000 {
			s += (i * 7) ^ (s >> 3)
			s -= i / 3
			s &= 0xffffff
		}
		baseSinkI = s
	}
}

func BenchmarkBaselineFloatArith(b *testing.B) {
	for range b.N {
		s := 0.0
		for i := range 10000 {
			x := float64(i)
			s += x*0.5 - s*0.001 + x/(s+1)
		}
		baseSinkF = s
	}
}

func BenchmarkBaselineFloat32Arith(b *testing.B) {
	for range b.N {
		var s float32
		for i := range 10000 {
			x := float32(i)
			s += x*0.5 - s*0.001 + x/(s+1)
		}
		baseSinkF32 = s
	}
}

func BenchmarkBaselineBranches(b *testing.B) {
	for range b.N {
		s := 0
		for i := range 10000 {
			if i%3 == 0 {
				s += i
			} else if i%5 == 0 {
				s -= i / 2
			} else if i > 5000 && s < 100000 {
				s *= 2
			} else {
				s++
			}
		}
		baseSinkI = s
	}
}

func BenchmarkBaselineSwitch(b *testing.B) {
	for range b.N {
		s := 0
		for i := range 10000 {
			switch i % 8 {
			case 0:
				s += 3
			case 1, 2:
				s -= 1
			case 3:
				s ^= i
			case 4, 5, 6:
				s += i & 7
			default:
				s >>= 1
			}
		}
		baseSinkI = s
	}
}

func BenchmarkBaselineNestedLoops(b *testing.B) {
	for range b.N {
		s := 0
		for i := range 100 {
			for j := range 100 {
				if (i+j)%2 == 0 {
					s += i * j
				}
			}
		}
		baseSinkI = s
	}
}

//go:noinline
func baselineAdd(a, b int) int {
	if a > b {
		return a - b
	}
	return a + b
}

func BenchmarkBaselineFuncCall(b *testing.B) {
	for range b.N {
		s := 0
		for i := range 10000 {
			s = baselineAdd(s, i) & 0xffff
		}
		baseSinkI = s
	}
}

func BenchmarkBaselineFloatSlice(b *testing.B) {
	xs := make([]float64, 1000)
	for i := range xs {
		xs[i] = float64(i) * 0.25
	}
	for range b.N {
		s := 0.0
		for _, x := range xs {
			s += x * x
		}
		for i := range xs {
			xs[i] = xs[i]*0.999 + 0.001
		}
		baseSinkF = s
	}
}
