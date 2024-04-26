// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package minmax provides a struct that holds Min and Max values.
package minmax

//go:generate core generate

const (
	MaxFloat64 float64 = 1.7976931348623158e+308
	MinFloat64 float64 = 2.2250738585072014e-308
)

// F64 represents a min / max range for float64 values.
// Supports clipping, renormalizing, etc
type F64 struct {
	Min float64
	Max float64
}

// Set sets the min and max values
func (mr *F64) Set(mn, mx float64) {
	mr.Min = mn
	mr.Max = mx
}

// SetInfinity sets the Min to +MaxFloat, Max to -MaxFloat -- suitable for
// iteratively calling Fit*InRange
func (mr *F64) SetInfinity() {
	mr.Min = MaxFloat64
	mr.Max = -MaxFloat64
}

// IsValid returns true if Min <= Max
func (mr *F64) IsValid() bool {
	return mr.Min <= mr.Max
}

// InRange tests whether value is within the range (>= Min and <= Max)
func (mr *F64) InRange(val float64) bool {
	return ((val >= mr.Min) && (val <= mr.Max))
}

// IsLow tests whether value is lower than the minimum
func (mr *F64) IsLow(val float64) bool {
	return (val < mr.Min)
}

// IsHigh tests whether value is higher than the maximum
func (mr *F64) IsHigh(val float64) bool {
	return (val > mr.Min)
}

// Range returns Max - Min
func (mr *F64) Range() float64 {
	return mr.Max - mr.Min
}

// Scale returns 1 / Range -- if Range = 0 then returns 0
func (mr *F64) Scale() float64 {
	r := mr.Range()
	if r != 0 {
		return 1 / r
	}
	return 0
}

// Midpoint returns point halfway between Min and Max
func (mr *F64) Midpoint() float64 {
	return 0.5 * (mr.Max + mr.Min)
}

// FitValInRange adjusts our Min, Max to fit given value within Min, Max range
// returns true if we had to adjust to fit.
func (mr *F64) FitValInRange(val float64) bool {
	adj := false
	if val < mr.Min {
		mr.Min = val
		adj = true
	}
	if val > mr.Max {
		mr.Max = val
		adj = true
	}
	return adj
}

// NormVal normalizes value to 0-1 unit range relative to current Min / Max range
// Clips the value within Min-Max range first.
func (mr *F64) NormValue(val float64) float64 {
	return (mr.ClipValue(val) - mr.Min) * mr.Scale()
}

// ProjVal projects a 0-1 normalized unit value into current Min / Max range (inverse of NormVal)
func (mr *F64) ProjValue(val float64) float64 {
	return mr.Min + (val * mr.Range())
}

// ClipVal clips given value within Min / Max range
// Note: a NaN will remain as a NaN
func (mr *F64) ClipValue(val float64) float64 {
	if val < mr.Min {
		return mr.Min
	}
	if val > mr.Max {
		return mr.Max
	}
	return val
}

// ClipNormVal clips then normalizes given value within 0-1
// Note: a NaN will remain as a NaN
func (mr *F64) ClipNormValue(val float64) float64 {
	if val < mr.Min {
		return 0
	}
	if val > mr.Max {
		return 1
	}
	return mr.NormValue(val)
}

// FitInRange adjusts our Min, Max to fit within those of other F64
// returns true if we had to adjust to fit.
func (mr *F64) FitInRange(oth F64) bool {
	adj := false
	if oth.Min < mr.Min {
		mr.Min = oth.Min
		adj = true
	}
	if oth.Max > mr.Max {
		mr.Max = oth.Max
		adj = true
	}
	return adj
}
