// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package path

// Epsilon is the smallest number below which we assume the value to be zero.
// This is to avoid numerical floating point issues.
var Epsilon = float32(1e-10)

// Precision is the number of significant digits at which floating point
// value will be printed to output formats.
var Precision = 8

// Equal returns true if a and b are equal within an absolute
// tolerance of Epsilon.
func Equal(a, b float32) bool {
	// avoid math.Abs
	if a < b {
		return b-a <= Epsilon
	}
	return a-b <= Epsilon
}
