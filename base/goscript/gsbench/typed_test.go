// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gsbench

import "testing"

// Benchmarks of slices of fixed-size numbers, which goscript holds in
// JavaScript typed arrays, in the ways that mesh and GPU code uses them.

var sinkU32 uint32

// BenchmarkFloat32Append appends the coordinates of 1000 vertices to a nil
// float32 slice.
func BenchmarkFloat32Append(b *testing.B) {
	for range b.N {
		var s []float32
		for i := range 1000 {
			x := float32(i)
			s = append(s, x, x*0.5, x*0.25)
		}
		sinkF = s[len(s)-1]
	}
}

// BenchmarkFloat32Fill writes the coordinates of 1000 vertices into a
// float32 slice made to hold them, as mesh generators do.
func BenchmarkFloat32Fill(b *testing.B) {
	s := make([]float32, 3000)
	for range b.N {
		for i := range 1000 {
			x := float32(i)
			s[3*i] = x
			s[3*i+1] = x * 0.5
			s[3*i+2] = x * 0.25
		}
		sinkF = s[2999]
	}
}

// BenchmarkUint32Sum reads 1000 elements of a uint32 slice, with a range
// loop and by index.
func BenchmarkUint32Sum(b *testing.B) {
	s := make([]uint32, 1000)
	for i := range s {
		s[i] = uint32(i) * 7
	}
	for range b.N {
		var t uint32
		for _, v := range s {
			t += v
		}
		for i := range len(s) {
			t ^= s[i]
		}
		sinkU32 = t
	}
}

// BenchmarkSmallFloat32Slices makes 1000 small float32 slices, as code
// that returns vectors in slices does.
func BenchmarkSmallFloat32Slices(b *testing.B) {
	for range b.N {
		var t float32
		for i := range 1000 {
			v := []float32{float32(i), 1, 2}
			t += v[0] + v[2]
		}
		sinkF = t
	}
}
