// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"math"

	"cogentcore.org/core/math32"
)

///////////////////////////////////////////
//  Abs

// Abs32 computes the sum of absolute value of differences (L1 Norm).
// Skips NaN's and panics if lengths are not equal.
func Abs32(a, b []float32) float32 {
	if len(a) != len(b) {
		panic("metric: slice lengths do not match")
	}
	ss := float32(0)
	for i, av := range a {
		bv := b[i]
		if math32.IsNaN(av) || math32.IsNaN(bv) {
			continue
		}
		ss += math32.Abs(av - bv)
	}
	return ss
}

// Abs64 computes the sum of absolute value of differences (L1 Norm).
// Skips NaN's and panics if lengths are not equal.
func Abs64(a, b []float64) float64 {
	if len(a) != len(b) {
		panic("metric: slice lengths do not match")
	}
	ss := float64(0)
	for i, av := range a {
		bv := b[i]
		if math.IsNaN(av) || math.IsNaN(bv) {
			continue
		}
		ss += math.Abs(av - bv)
	}
	return ss
}

///////////////////////////////////////////
//  Hamming

// Hamming32 computes the sum of 1's for every element that is different
// (city block).
// Skips NaN's and panics if lengths are not equal.
func Hamming32(a, b []float32) float32 {
	if len(a) != len(b) {
		panic("metric: slice lengths do not match")
	}
	ss := float32(0)
	for i, av := range a {
		bv := b[i]
		if math32.IsNaN(av) || math32.IsNaN(bv) {
			continue
		}
		if av != bv {
			ss += 1
		}
	}
	return ss
}

// Hamming64 computes the sum of absolute value of differences (L1 Norm).
// Skips NaN's and panics if lengths are not equal.
func Hamming64(a, b []float64) float64 {
	if len(a) != len(b) {
		panic("metric: slice lengths do not match")
	}
	ss := float64(0)
	for i, av := range a {
		bv := b[i]
		if math.IsNaN(av) || math.IsNaN(bv) {
			continue
		}
		if av != bv {
			ss += 1
		}
	}
	return ss
}
