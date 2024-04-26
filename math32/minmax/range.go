// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package minmax

// Range32 represents a range of values for plotting, where the min or max can optionally be fixed
type Range32 struct {

	// Min and Max range values
	F32

	// fix the minimum end of the range
	FixMin bool

	// fix the maximum end of the range
	FixMax bool
}

// SetMin sets a fixed min value
func (rr *Range32) SetMin(mn float32) {
	rr.FixMin = true
	rr.Min = mn
}

// SetMax sets a fixed max value
func (rr *Range32) SetMax(mx float32) {
	rr.FixMax = true
	rr.Max = mx
}

// Range returns Max - Min
func (rr *Range32) Range() float32 {
	return rr.Max - rr.Min
}

///////////////////////////////////////////////////////////////////////
//  64

// Range64 represents a range of values for plotting, where the min or max can optionally be fixed
type Range64 struct {

	// Min and Max range values
	F64

	// fix the minimum end of the range
	FixMin bool

	// fix the maximum end of the range
	FixMax bool
}

// SetMin sets a fixed min value
func (rr *Range64) SetMin(mn float64) {
	rr.FixMin = true
	rr.Min = mn
}

// SetMax sets a fixed max value
func (rr *Range64) SetMax(mx float64) {
	rr.FixMax = true
	rr.Max = mx
}

// Range returns Max - Min
func (rr *Range64) Range() float64 {
	return rr.Max - rr.Min
}
