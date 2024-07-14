// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from github.com/gonum/plot:
// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"errors"

	"cogentcore.org/core/math32"
)

// data defines the main data interfaces for plotting.
// Other more specific types of plots define their own interfaces.
// unlike gonum/plot, NaN values are treated as missing data here.

var (
	ErrInfinity = errors.New("plotter: infinite data point")
	ErrNoData   = errors.New("plotter: no data points")
)

// CheckFloats returns an error if any of the arguments are Infinity.
// or if there are no non-NaN data points available for plotting.
func CheckFloats(fs ...float32) error {
	n := 0
	for _, f := range fs {
		switch {
		case math32.IsNaN(f):
		case math32.IsInf(f, 0):
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
func CheckNaNs(fs ...float32) bool {
	for _, f := range fs {
		if math32.IsNaN(f) {
			return true
		}
	}
	return false
}

//////////////////////////////////////////////////
// 	Valuer

// Valuer provides an interface for a list of scalar values
type Valuer interface {
	// Len returns the number of values.
	Len() int

	// Value returns a value.
	Value(i int) float32
}

// Range returns the minimum and maximum values.
func Range(vs Valuer) (min, max float32) {
	min = math32.Inf(1)
	max = math32.Inf(-1)
	for i := 0; i < vs.Len(); i++ {
		v := vs.Value(i)
		if math32.IsNaN(v) {
			continue
		}
		min = math32.Min(min, v)
		max = math32.Max(max, v)
	}
	return
}

// Values implements the Valuer interface.
type Values []float32

func (vs Values) Len() int {
	return len(vs)
}

func (vs Values) Value(i int) float32 {
	return vs[i]
}

// CopyValues returns a Values that is a copy of the values
// from a Valuer, or an error if there are no values, or if one of
// the copied values is a Infinity.
// NaN values are skipped in the copying process.
func CopyValues(vs Valuer) (Values, error) {
	if vs.Len() == 0 {
		return nil, ErrNoData
	}
	cpy := make(Values, 0, vs.Len())
	for i := 0; i < vs.Len(); i++ {
		v := vs.Value(i)
		if math32.IsNaN(v) {
			continue
		}
		if err := CheckFloats(v); err != nil {
			return nil, err
		}
		cpy = append(cpy, v)
	}
	return cpy, nil
}

//////////////////////////////////////////////////
// 	XYer

// XYer provides an interface for a list of X,Y data pairs
type XYer interface {
	// Len returns the number of x, y pairs.
	Len() int

	// XY returns an x, y pair.
	XY(i int) (x, y float32)
}

// XYRange returns the minimum and maximum
// x and y values.
func XYRange(xys XYer) (xmin, xmax, ymin, ymax float32) {
	xmin, xmax = Range(XValues{xys})
	ymin, ymax = Range(YValues{xys})
	return
}

// XYs implements the XYer interface.
type XYs []math32.Vector2

func (xys XYs) Len() int {
	return len(xys)
}

func (xys XYs) XY(i int) (float32, float32) {
	return xys[i].X, xys[i].Y
}

// CopyXYs returns an XYs that is a copy of the x and y values from
// an XYer, or an error if one of the data points contains a NaN or
// Infinity.
func CopyXYs(data XYer) (XYs, error) {
	if data.Len() == 0 {
		return nil, ErrNoData
	}
	cpy := make(XYs, 0, data.Len())
	for i := range data.Len() {
		x, y := data.XY(i)
		if CheckNaNs(x, y) {
			continue
		}
		if err := CheckFloats(x, y); err != nil {
			return nil, err
		}
		cpy = append(cpy, math32.Vec2(x, y))
	}
	return cpy, nil
}

// PlotXYs returns plot coordinates for given set of XYs
func PlotXYs(plt *Plot, data XYs) XYs {
	ps := make(XYs, len(data))
	for i := range data {
		ps[i].X, ps[i].Y = plt.PX(data[i].X), plt.PY(data[i].Y)
	}
	return ps
}

// XValues implements the Valuer interface,
// returning the x value from an XYer.
type XValues struct {
	XYer
}

func (xs XValues) Value(i int) float32 {
	x, _ := xs.XY(i)
	return x
}

// YValues implements the Valuer interface,
// returning the y value from an XYer.
type YValues struct {
	XYer
}

func (ys YValues) Value(i int) float32 {
	_, y := ys.XY(i)
	return y
}

//////////////////////////////////////////////////
// 	XYer

// XYZer provides an interface for a list of X,Y,Z data triples.
// It also satisfies the XYer interface for the X,Y pairs.
type XYZer interface {
	// Len returns the number of x, y, z triples.
	Len() int

	// XYZ returns an x, y, z triple.
	XYZ(i int) (float32, float32, float32)

	// XY returns an x, y pair.
	XY(i int) (float32, float32)
}

// XYZs implements the XYZer interface using a slice.
type XYZs []XYZ

// XYZ is an x, y and z value.
type XYZ struct{ X, Y, Z float32 }

// Len implements the Len method of the XYZer interface.
func (xyz XYZs) Len() int {
	return len(xyz)
}

// XYZ implements the XYZ method of the XYZer interface.
func (xyz XYZs) XYZ(i int) (float32, float32, float32) {
	return xyz[i].X, xyz[i].Y, xyz[i].Z
}

// XY implements the XY method of the XYer interface.
func (xyz XYZs) XY(i int) (float32, float32) {
	return xyz[i].X, xyz[i].Y
}

// CopyXYZs copies an XYZer.
func CopyXYZs(data XYZer) (XYZs, error) {
	if data.Len() == 0 {
		return nil, ErrNoData
	}
	cpy := make(XYZs, 0, data.Len())
	for i := range data.Len() {
		x, y, z := data.XYZ(i)
		if CheckNaNs(x, y, z) {
			continue
		}
		if err := CheckFloats(x, y, z); err != nil {
			return nil, err
		}
		cpy = append(cpy, XYZ{X: x, Y: y, Z: z})
	}
	return cpy, nil
}

// XYValues implements the XYer interface, returning
// the x and y values from an XYZer.
type XYValues struct{ XYZer }

// XY implements the XY method of the XYer interface.
func (xy XYValues) XY(i int) (float32, float32) {
	x, y, _ := xy.XYZ(i)
	return x, y
}

//////////////////////////////////////////////////
// 	Labeler

// Labeler provides an interface for a list of string labels
type Labeler interface {
	// Label returns a label.
	Label(i int) string
}
