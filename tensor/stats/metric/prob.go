// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"math"

	"cogentcore.org/core/math32"
)

///////////////////////////////////////////
//  CrossEntropy

// CrossEntropy32 computes cross-entropy between the two vectors.
// Skips NaN's and panics if lengths are not equal.
func CrossEntropy32(a, b []float32) float32 {
	if len(a) != len(b) {
		panic("metric: slice lengths do not match")
	}
	ss := float32(0)
	for i, av := range a {
		bv := b[i]
		if math32.IsNaN(av) || math32.IsNaN(bv) {
			continue
		}
		bv = math32.Max(bv, 0.000001)
		bv = math32.Min(bv, 0.999999)
		if av >= 1.0 {
			ss += -math32.Log(bv)
		} else if av <= 0.0 {
			ss += -math32.Log(1.0 - bv)
		} else {
			ss += av*math32.Log(av/bv) + (1-av)*math32.Log((1-av)/(1-bv))
		}
	}
	return ss
}

// CrossEntropy64 computes the cross-entropy between the two vectors.
// Skips NaN's and panics if lengths are not equal.
func CrossEntropy64(a, b []float64) float64 {
	if len(a) != len(b) {
		panic("metric: slice lengths do not match")
	}
	ss := float64(0)
	for i, av := range a {
		bv := b[i]
		if math.IsNaN(av) || math.IsNaN(bv) {
			continue
		}
		bv = math.Max(bv, 0.000001)
		bv = math.Min(bv, 0.999999)
		if av >= 1.0 {
			ss += -math.Log(bv)
		} else if av <= 0.0 {
			ss += -math.Log(1.0 - bv)
		} else {
			ss += av*math.Log(av/bv) + (1-av)*math.Log((1-av)/(1-bv))
		}
	}
	return ss
}
