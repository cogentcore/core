// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package minmax

//go:generate core generate

import "fmt"

//gosl: start minmax

// F32 represents a min / max range for float32 values.
// Supports clipping, renormalizing, etc
type F32 struct {
	Min float32
	Max float32

	pad, pad1 int32 // for gpu use
}

// Set sets the min and max values
func (mr *F32) Set(mn, mx float32) {
	mr.Min = mn
	mr.Max = mx
}

// SetInfinity sets the Min to +MaxFloat, Max to -MaxFloat -- suitable for
// iteratively calling Fit*InRange
func (mr *F32) SetInfinity() {
	mr.Min = MaxFloat32
	mr.Max = -MaxFloat32
}

// IsValid returns true if Min <= Max
func (mr *F32) IsValid() bool {
	return mr.Min <= mr.Max
}

// InRange tests whether value is within the range (>= Min and <= Max)
func (mr *F32) InRange(val float32) bool {
	return ((val >= mr.Min) && (val <= mr.Max))
}

// IsLow tests whether value is lower than the minimum
func (mr *F32) IsLow(val float32) bool {
	return (val < mr.Min)
}

// IsHigh tests whether value is higher than the maximum
func (mr *F32) IsHigh(val float32) bool {
	return (val > mr.Min)
}

// Range returns Max - Min
func (mr *F32) Range() float32 {
	return mr.Max - mr.Min
}

// Scale returns 1 / Range -- if Range = 0 then returns 0
func (mr *F32) Scale() float32 {
	r := mr.Range()
	if r != 0 {
		return 1.0 / r
	}
	return 0
}

// Midpoint returns point halfway between Min and Max
func (mr *F32) Midpoint() float32 {
	return 0.5 * (mr.Max + mr.Min)
}

// FitValInRange adjusts our Min, Max to fit given value within Min, Max range
// returns true if we had to adjust to fit.
func (mr *F32) FitValInRange(val float32) bool {
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
func (mr *F32) NormValue(val float32) float32 {
	return (mr.ClipValue(val) - mr.Min) * mr.Scale()
}

// ProjVal projects a 0-1 normalized unit value into current Min / Max range (inverse of NormVal)
func (mr *F32) ProjValue(val float32) float32 {
	return mr.Min + (val * mr.Range())
}

// ClipVal clips given value within Min / Max range
// Note: a NaN will remain as a NaN
func (mr *F32) ClipValue(val float32) float32 {
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
func (mr *F32) ClipNormValue(val float32) float32 {
	if val < mr.Min {
		return 0
	}
	if val > mr.Max {
		return 1
	}
	return mr.NormValue(val)
}

//gosl: end minmax

func (mr *F32) String() string {
	return fmt.Sprintf("{%g %g}", mr.Min, mr.Max)
}

// FitInRange adjusts our Min, Max to fit within those of other F32
// returns true if we had to adjust to fit.
func (mr *F32) FitInRange(oth F32) bool {
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
