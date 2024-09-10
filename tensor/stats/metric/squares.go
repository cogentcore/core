// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

/*

///////////////////////////////////////////
//  Cosine

// Cosine64 computes the cosine of the angle between two vectors (-1..1),
// as the normalized inner product: inner product / sqrt(ssA * ssB).
// If vectors are mean-normalized = Correlation.
// Skips NaN's and panics if lengths are not equal.
func Cosine64(a, b []float64) float64 {
	if len(a) != len(b) {
		panic("metric: slice lengths do not match")
	}
	ss := float64(0)
	var ass, bss float64
	for i, av := range a {
		bv := b[i]
		if math.IsNaN(av) || math.IsNaN(bv) {
			continue
		}
		ss += av * bv  // between
		ass += av * av // within
		bss += bv * bv
	}
	vp := math.Sqrt(ass * bss)
	if vp > 0 {
		ss /= vp
	}
	return ss
}

///////////////////////////////////////////
//  InvCosine

// InvCosine32 computes 1 - cosine of the angle between two vectors (-1..1),
// as the normalized inner product: inner product / sqrt(ssA * ssB).
// If vectors are mean-normalized = Correlation.
// Skips NaN's and panics if lengths are not equal.
func InvCosine64(a, b []float64) float64 {
	return 1 - Cosine64(a, b)
}


*/
