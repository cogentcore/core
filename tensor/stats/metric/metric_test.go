// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"math"
	"testing"

	"cogentcore.org/core/math32"
)

func TestAll(t *testing.T) {
	a64 := []float64{.5, .2, .1, .7, math.NaN(), .5}
	b64 := []float64{.2, .5, .1, .7, 0, .2}

	a32 := []float32{.5, .2, .1, .7, math32.NaN(), .5}
	b32 := []float32{.2, .5, .1, .7, 0, .2}

	ss := SumSquares64(a64, b64)
	if ss != 0.27 {
		t.Errorf("SumSquares64: %g\n", ss)
	}
	ss32 := SumSquares32(a32, b32)
	if ss32 != float32(ss) {
		t.Errorf("SumSquares32: %g\n", ss32)
	}

	ec := Euclidean64(a64, b64)
	if math.Abs(ec-math.Sqrt(0.27)) > 1.0e-10 {
		t.Errorf("Euclidean64: %g  vs. %g\n", ec, math.Sqrt(0.27))
	}
	ec32 := Euclidean32(a32, b32)
	if ec32 != float32(ec) {
		t.Errorf("Euclidean32: %g\n", ec32)
	}

	cv := Covariance64(a64, b64)
	if cv != 0.023999999999999994 {
		t.Errorf("Covariance64: %g\n", cv)
	}
	cv32 := Covariance32(a32, b32)
	if cv32 != float32(cv) {
		t.Errorf("Covariance32: %g\n", cv32)
	}

	cr := Correlation64(a64, b64)
	if cr != 0.47311118871909136 {
		t.Errorf("Correlation64: %g\n", cr)
	}
	cr32 := Correlation32(a32, b32)
	if cr32 != 0.47311115 {
		t.Errorf("Correlation32: %g\n", cr32)
	}

	cs := Cosine64(a64, b64)
	if cs != 0.861061697819235 {
		t.Errorf("Cosine64: %g\n", cs)
	}
	cs32 := Cosine32(a32, b32)
	if cs32 != 0.86106175 {
		t.Errorf("Cosine32: %g\n", cs32)
	}

	ab := Abs64(a64, b64)
	if ab != 0.8999999999999999 {
		t.Errorf("Abs64: %g\n", ab)
	}
	ab32 := Abs32(a32, b32)
	if ab32 != 0.90000004 {
		t.Errorf("Abs32: %g\n", ab32)
	}

	hm := Hamming64(a64, b64)
	if hm != 3 {
		t.Errorf("Hamming64: %g\n", hm)
	}
	hm32 := Hamming32(a32, b32)
	if hm32 != 3 {
		t.Errorf("Hamming32: %g\n", hm32)
	}
}
