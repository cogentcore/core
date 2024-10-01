// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"reflect"
	"slices"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/metadata"
)

// Reshaped is a reshaping wrapper around another "source" [Tensor],
// that provides a length-preserving reshaped view onto the source Tensor.
// Reshaping by adding new size=1 dimensions (via [NewAxis] value) is
// often important for properly aligning two tensors in a computationally
// compatible manner; see the [AlignShapes] function.
// [Reshaped.AsValues] on this view returns a new [Values] with the view
// shape, calling [Clone] on the source tensor to get the values.
type Reshaped struct { //types:add

	// Tensor source that we are a masked view onto.
	Tensor Tensor

	// Reshape is the effective shape we use for access.
	// This must have the same Len() as the source Tensor.
	Reshape Shape
}

// NewReshaped returns a new [Reshaped] view of given tensor, with given shape
// sizes. If no such sizes are provided, the source shape is used.
// A single -1 value can be used to automatically specify the remaining tensor
// length, as long as the other sizes are an even multiple of the total length.
// A single -1 returns a 1D view of the entire tensor.
func NewReshaped(tsr Tensor, sizes ...int) *Reshaped {
	rs := &Reshaped{Tensor: tsr}
	if len(sizes) == 0 {
		rs.Reshape.CopyFrom(tsr.Shape())
	} else {
		errors.Log(rs.SetShapeSizes(sizes...))
	}
	return rs
}

// Reshape returns a view of the given tensor with given shape sizes.
// A single -1 value can be used to automatically specify the remaining tensor
// length, as long as the other sizes are an even multiple of the total length.
// A single -1 returns a 1D view of the entire tensor.
func Reshape(tsr Tensor, sizes ...int) Tensor {
	if len(sizes) == 0 {
		err := errors.New("tensor.Reshape: must pass shape sizes")
		errors.Log(err)
		return tsr
	}
	if len(sizes) == 1 {
		sz := sizes[0]
		if sz == -1 {
			return As1D(tsr)
		}
	}
	rs := &Reshaped{Tensor: tsr}
	errors.Log(rs.SetShapeSizes(sizes...))
	return rs
}

// Transpose returns a new [Reshaped] tensor with the strides
// switched so that rows and column dimensions are effectively
// reversed.
func Transpose(tsr Tensor) Tensor {
	rs := &Reshaped{Tensor: tsr}
	rs.Reshape.CopyFrom(tsr.Shape())
	rs.Reshape.Strides = ColumnMajorStrides(rs.Reshape.Sizes...)
	return rs
}

// NewRowCellsView returns a 2D [Reshaped] view onto the given tensor,
// with a single outer "row" dimension and a single inner "cells" dimension,
// with the given 'split' dimension specifying where the cells start.
// All dimensions prior to split are collapsed to form the new outer row dimension,
// and the remainder are collapsed to form the 1D cells dimension.
// This is useful for stats, metrics and other packages that operate
// on data in this shape.
func NewRowCellsView(tsr Tensor, split int) *Reshaped {
	sizes := tsr.ShapeSizes()
	rows := sizes[:split]
	cells := sizes[split:]
	nr := 1
	for _, r := range rows {
		nr *= r
	}
	nc := 1
	for _, c := range cells {
		nc *= c
	}
	return NewReshaped(tsr, nr, nc)
}

// AsReshaped returns the tensor as a [Reshaped] view.
// If it already is one, then it is returned, otherwise it is wrapped
// with an initial shape equal to the source tensor.
func AsReshaped(tsr Tensor) *Reshaped {
	if rs, ok := tsr.(*Reshaped); ok {
		return rs
	}
	return NewReshaped(tsr)
}

// SetShapeSizes sets our shape sizes to the given values, which must result in
// the same length as the source tensor. An error is returned if not.
// If a different subset of content is desired, use another view such as [Sliced].
// Note that any number of size = 1 dimensions can be added without affecting
// the length, and the [NewAxis] value can be used to semantically
// indicate when such a new dimension is being inserted. This is often useful
// for aligning two tensors to achieve a desired computation; see [AlignShapes]
// function. A single -1 can be used to specify a dimension size that takes the
// remaining length, as long as the other sizes are an even multiple of the length.
// A single -1 indicates to use the full length.
func (rs *Reshaped) SetShapeSizes(sizes ...int) error {
	sln := rs.Tensor.Len()
	if sln == 0 {
		return nil
	}
	if sln == 1 {
		sz := sizes[0]
		if sz < 0 {
			rs.Reshape.SetShapeSizes(sln)
			return nil
		}
	}
	sz := slices.Clone(sizes)
	ln := 1
	negIdx := -1
	for i, s := range sz {
		if s < 0 {
			negIdx = i
		} else {
			ln *= s
		}
	}
	if negIdx >= 0 {
		if sln%ln != 0 {
			return errors.New("tensor.Reshaped SetShapeSizes: -1 cannot be used because the remaining dimensions are not an even multiple of the source tensor length")
		}
		sz[negIdx] = sln / ln
	}
	rs.Reshape.SetShapeSizes(sz...)
	if rs.Reshape.Len() != sln {
		return errors.New("tensor.Reshaped SetShapeSizes: new length is different from source tensor; use Sliced or other views to change view content")
	}
	return nil
}

func (rs *Reshaped) Label() string            { return label(rs.Metadata().Name(), rs.Shape()) }
func (rs *Reshaped) String() string           { return Sprintf("", rs, 0) }
func (rs *Reshaped) Metadata() *metadata.Data { return rs.Tensor.Metadata() }
func (rs *Reshaped) IsString() bool           { return rs.Tensor.IsString() }
func (rs *Reshaped) DataType() reflect.Kind   { return rs.Tensor.DataType() }
func (rs *Reshaped) ShapeSizes() []int        { return rs.Reshape.Sizes }
func (rs *Reshaped) Shape() *Shape            { return &rs.Reshape }
func (rs *Reshaped) Len() int                 { return rs.Reshape.Len() }
func (rs *Reshaped) NumDims() int             { return rs.Reshape.NumDims() }
func (rs *Reshaped) DimSize(dim int) int      { return rs.Reshape.DimSize(dim) }

// AsValues returns a copy of this tensor as raw [Values], with
// the same shape as our view.  This calls [Clone] on the source
// tensor to get the Values and then sets our shape sizes to it.
func (rs *Reshaped) AsValues() Values {
	vals := Clone(rs.Tensor)
	vals.SetShapeSizes(rs.Reshape.Sizes...)
	return vals
}

/////////////////////  Floats

func (rs *Reshaped) Float(i ...int) float64 {
	return rs.Tensor.Float1D(rs.Reshape.IndexTo1D(i...))
}

func (rs *Reshaped) SetFloat(val float64, i ...int) {
	rs.Tensor.SetFloat1D(val, rs.Reshape.IndexTo1D(i...))
}

func (rs *Reshaped) Float1D(i int) float64         { return rs.Tensor.Float1D(i) }
func (rs *Reshaped) SetFloat1D(val float64, i int) { rs.Tensor.SetFloat1D(val, i) }

/////////////////////  Strings

func (rs *Reshaped) StringValue(i ...int) string {
	return rs.Tensor.String1D(rs.Reshape.IndexTo1D(i...))
}

func (rs *Reshaped) SetString(val string, i ...int) {
	rs.Tensor.SetString1D(val, rs.Reshape.IndexTo1D(i...))
}

func (rs *Reshaped) String1D(i int) string         { return rs.Tensor.String1D(i) }
func (rs *Reshaped) SetString1D(val string, i int) { rs.Tensor.SetString1D(val, i) }

/////////////////////  Ints

func (rs *Reshaped) Int(i ...int) int {
	return rs.Tensor.Int1D(rs.Reshape.IndexTo1D(i...))
}

func (rs *Reshaped) SetInt(val int, i ...int) {
	rs.Tensor.SetInt1D(val, rs.Reshape.IndexTo1D(i...))
}

func (rs *Reshaped) Int1D(i int) int         { return rs.Tensor.Int1D(i) }
func (rs *Reshaped) SetInt1D(val int, i int) { rs.Tensor.SetInt1D(val, i) }

// check for interface impl
var _ Tensor = (*Reshaped)(nil)
