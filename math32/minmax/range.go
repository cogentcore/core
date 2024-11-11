// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package minmax

// Range32 represents a range of values for plotting, where the min or max can optionally be fixed
type Range32 struct {
	F32

	// fix the minimum end of the range
	FixMin bool

	// fix the maximum end of the range
	FixMax bool
}

// SetMin sets a fixed min value
func (rr *Range32) SetMin(mn float32) *Range32 {
	rr.FixMin = true
	rr.Min = mn
	return rr
}

// SetMax sets a fixed max value
func (rr *Range32) SetMax(mx float32) *Range32 {
	rr.FixMax = true
	rr.Max = mx
	return rr
}

// Range returns Max - Min
func (rr *Range32) Range() float32 {
	return rr.Max - rr.Min
}

// Clamp returns min, max values clamped according to Fixed min / max of range.
func (rr *Range32) Clamp(mnIn, mxIn float32) (mn, mx float32) {
	mn, mx = mnIn, mxIn
	if rr.FixMin && rr.Min < mn {
		mn = rr.Min
	}
	if rr.FixMax && rr.Max > mx {
		mx = rr.Max
	}
	return
}

///////////////////////////////////////////////////////////////////////
//  64

// Range64 represents a range of values for plotting, where the min or max can optionally be fixed
type Range64 struct {
	F64

	// fix the minimum end of the range
	FixMin bool

	// fix the maximum end of the range
	FixMax bool
}

// SetMin sets a fixed min value
func (rr *Range64) SetMin(mn float64) *Range64 {
	rr.FixMin = true
	rr.Min = mn
	return rr
}

// SetMax sets a fixed max value
func (rr *Range64) SetMax(mx float64) *Range64 {
	rr.FixMax = true
	rr.Max = mx
	return rr
}

// Range returns Max - Min
func (rr *Range64) Range() float64 {
	return rr.Max - rr.Min
}

// Clamp returns min, max values clamped according to Fixed min / max of range.
func (rr *Range64) Clamp(mnIn, mxIn float64) (mn, mx float64) {
	mn, mx = mnIn, mxIn
	if rr.FixMin && rr.Min < mn {
		mn = rr.Min
	}
	if rr.FixMax && rr.Max > mx {
		mx = rr.Max
	}
	return
}
