// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package minmax

import "math"

// notes: gonum/plot has function labelling: talbotLinHanrahan
// based on this algorithm: http://vis.stanford.edu/files/2010-TickLabels-InfoVis.pdf
// but it is goes beyond this basic functionality, and is not exported in any case..
// but could be accessed using DefaultTicks api.

// NiceRoundNumber returns the closest nice round number either above or below
// the given number, based on the observation that numbers 1, 2, 5
// at any power are "nice".
// This is used for choosing graph labels, and auto-scaling ranges to contain
// a given value.
// if below == true then returned number is strictly less than given number
// otherwise it is strictly larger.
func NiceRoundNumber(x float64, below bool) float64 {
	rn := x
	neg := false
	if x < 0 {
		neg = true
		below = !below // reverses..
	}
	abs := math.Abs(x)
	exp := int(math.Floor(math.Log10(abs)))
	order := math.Pow(10, float64(exp))
	f := abs / order // fraction between 1 and 10
	if below {
		switch {
		case f >= 5:
			rn = 5
		case f >= 2:
			rn = 2
		default:
			rn = 1
		}
	} else {
		switch {
		case f <= 1:
			rn = 1
		case f <= 2:
			rn = 2
		case f <= 5:
			rn = 5
		default:
			rn = 10
		}
	}
	if neg {
		return -rn * order
	}
	return rn * order
}
