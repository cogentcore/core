// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"math"

	"cogentcore.org/core/math32"
)

///////////////////////////////////////////
//  Tolerance

// Tolerance32 sets a = b for any element where |a-b| <= tol.
// This can be called prior to any metric function.
func Tolerance32(a, b []float32, tol float32) {
	if len(a) != len(b) {
		panic("metric: slice lengths do not match")
	}
	for i, av := range a {
		bv := b[i]
		if math32.IsNaN(av) || math32.IsNaN(bv) {
			continue
		}
		if math32.Abs(av-bv) <= tol {
			a[i] = bv
		}
	}
}

// Tolerance64 sets a = b for any element where |a-b| <= tol.
// This can be called prior to any metric function.
func Tolerance64(a, b []float64, tol float64) {
	if len(a) != len(b) {
		panic("metric: slice lengths do not match")
	}
	for i, av := range a {
		bv := b[i]
		if math.IsNaN(av) || math.IsNaN(bv) {
			continue
		}
		if math.Abs(av-bv) <= tol {
			a[i] = bv
		}
	}
}
