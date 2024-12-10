// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from github.com/gonum/plot:
// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"log/slog"
	"math"
	"strconv"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32/minmax"
)

// data defines the main data interfaces for plotting
// and the different Roles for data.

var (
	ErrInfinity = errors.New("plotter: infinite data point")
	ErrNoData   = errors.New("plotter: no data points")
)

// Data is a map of Roles and Data for that Role, providing the
// primary way of passing data to a Plotter
type Data map[Roles]Valuer

// Valuer is the data interface for plotting, supporting either
// float64 or string representations. It is satisfied by the tensor.Tensor
// interface, so a tensor can be used directly for plot Data.
type Valuer interface {
	// Len returns the number of values.
	Len() int

	// Float1D(i int) returns float64 value at given index.
	Float1D(i int) float64

	// String1D(i int) returns string value at given index.
	String1D(i int) string
}

// Roles are the roles that a given set of data values can play,
// designed to be sufficiently generalizable across all different
// types of plots, even if sometimes it is a bit of a stretch.
type Roles int32 //enums:enum

const (
	// NoRole is the default no-role specified case.
	NoRole Roles = iota

	// X axis
	X

	// Y axis
	Y

	// Z axis
	Z

	// U is the X component of a vector or first quartile in Box plot, etc.
	U

	// V is the Y component of a vector or third quartile in a Box plot, etc.
	V

	// W is the Z component of a vector
	W

	// Low is a lower error bar or region.
	Low

	// High is an upper error bar or region.
	High

	// Size controls the size of points etc.
	Size

	// Color controls the color of points or other elements.
	Color

	// Label renders a label, typically from string data, but can also be used for values.
	Label
)

// CheckFloats returns an error if any of the arguments are Infinity.
// or if there are no non-NaN data points available for plotting.
func CheckFloats(fs ...float64) error {
	n := 0
	for _, f := range fs {
		switch {
		case math.IsNaN(f):
		case math.IsInf(f, 0):
			return ErrInfinity
		default:
			n++
		}
	}
	if n == 0 {
		return ErrNoData
	}
	return nil
}

// CheckNaNs returns true if any of the floats are NaN
func CheckNaNs(fs ...float64) bool {
	for _, f := range fs {
		if math.IsNaN(f) {
			return true
		}
	}
	return false
}

// Range updates given Range with values from data.
func Range(data Valuer, rng *minmax.F64) {
	for i := 0; i < data.Len(); i++ {
		v := data.Float1D(i)
		if math.IsNaN(v) {
			continue
		}
		rng.FitValInRange(v)
	}
}

// RangeClamp updates the given axis Min, Max range values based
// on the range of values in the given [Data], and the given style range.
func RangeClamp(data Valuer, axisRng *minmax.F64, styleRng *minmax.Range64) {
	Range(data, axisRng)
	axisRng.Min, axisRng.Max = styleRng.Clamp(axisRng.Min, axisRng.Max)
}

// CheckLengths checks that all the data elements have the same length.
// Logs and returns an error if not.
func (dt Data) CheckLengths() error {
	n := 0
	for _, v := range dt {
		if n == 0 {
			n = v.Len()
		} else {
			if v.Len() != n {
				err := errors.New("plot.Data has inconsistent lengths -- all data elements must have the same length -- plotting aborted")
				return errors.Log(err)
			}
		}
	}
	return nil
}

// Values provides a minimal implementation of the Data interface
// using a slice of float64.
type Values []float64

func (vs Values) Len() int {
	return len(vs)
}

func (vs Values) Float1D(i int) float64 {
	return vs[i]
}

func (vs Values) String1D(i int) string {
	return strconv.FormatFloat(vs[i], 'g', -1, 64)
}

// CopyValues returns a Values that is a copy of the values
// from Data, or an error if there are no values, or if one of
// the copied values is a Infinity.
// NaN values are skipped in the copying process.
func CopyValues(data Valuer) (Values, error) {
	if data == nil {
		return nil, ErrNoData
	}
	cpy := make(Values, 0, data.Len())
	for i := 0; i < data.Len(); i++ {
		v := data.Float1D(i)
		if math.IsNaN(v) {
			continue
		}
		if err := CheckFloats(v); err != nil {
			return nil, err
		}
		cpy = append(cpy, v)
	}
	return cpy, nil
}

// MustCopyRole returns Values copy of given role from given data map,
// logging an error and returning nil if not present.
func MustCopyRole(data Data, role Roles) Values {
	d, ok := data[role]
	if !ok {
		slog.Error("plot Data role not present, but is required", "role:", role)
		return nil
	}
	return errors.Log1(CopyValues(d))
}

// CopyRole returns Values copy of given role from given data map,
// returning nil if role not present.
func CopyRole(data Data, role Roles) Values {
	d, ok := data[role]
	if !ok {
		return nil
	}
	v, _ := CopyValues(d)
	return v
}

// PlotX returns plot pixel X coordinate values for given data.
func PlotX(plt *Plot, data Valuer) []float32 {
	px := make([]float32, data.Len())
	for i := range px {
		px[i] = plt.PX(data.Float1D(i))
	}
	return px
}

// PlotY returns plot pixel Y coordinate values for given data.
func PlotY(plt *Plot, data Valuer) []float32 {
	py := make([]float32, data.Len())
	for i := range py {
		py[i] = plt.PY(data.Float1D(i))
	}
	return py
}

//////// Labels

// Labels provides a minimal implementation of the Data interface
// using a slice of string. It always returns 0 for Float1D.
type Labels []string

func (lb Labels) Len() int {
	return len(lb)
}

func (lb Labels) Float1D(i int) float64 {
	return 0
}

func (lb Labels) String1D(i int) string {
	return lb[i]
}
