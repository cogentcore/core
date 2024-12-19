// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"math"
	"reflect"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/reflectx"
)

// Masked is a filtering wrapper around another "source" [Tensor],
// that provides a bit-masked view onto the Tensor defined by a [Bool] [Values]
// tensor with a matching shape. If the bool mask has a 'false'
// then the corresponding value cannot be Set, and Float access returns
// NaN indicating missing data (other type access returns the zero value).
// A new Masked view defaults to a full transparent view of the source tensor.
// To produce a new [Values] tensor with only the 'true' cases,
// (i.e., the copy function of numpy), call [Masked.AsValues].
type Masked struct { //types:add

	// Tensor source that we are a masked view onto.
	Tensor Tensor

	// Bool tensor with same shape as source tensor, providing mask.
	Mask *Bool
}

// NewMasked returns a new [Masked] view of given tensor,
// with given [Bool] mask values. If no mask is provided,
// a default full transparent (all bool values = true) mask is used.
func NewMasked(tsr Tensor, mask ...*Bool) *Masked {
	ms := &Masked{Tensor: tsr}
	if len(mask) == 1 {
		ms.Mask = mask[0]
		ms.SyncShape()
	} else {
		ms.Mask = NewBoolShape(tsr.Shape())
		ms.Mask.SetTrue()
	}
	return ms
}

// Mask is the general purpose masking function, which checks
// if the mask arg is a Bool and uses if so.
// Otherwise, it logs an error.
func Mask(tsr, mask Tensor) Tensor {
	if mb, ok := mask.(*Bool); ok {
		return NewMasked(tsr, mb)
	}
	errors.Log(errors.New("tensor.Mask: provided tensor is not a Bool tensor"))
	return tsr
}

// AsMasked returns the tensor as a [Masked] view.
// If it already is one, then it is returned, otherwise it is wrapped
// with an initially fully transparent mask.
func AsMasked(tsr Tensor) *Masked {
	if ms, ok := tsr.(*Masked); ok {
		return ms
	}
	return NewMasked(tsr)
}

// SetTensor sets the given source tensor. If the shape does not match
// the current Mask, then a new transparent mask is established.
func (ms *Masked) SetTensor(tsr Tensor) {
	ms.Tensor = tsr
	ms.SyncShape()
}

// SyncShape ensures that [Masked.Mask] shape is the same as source tensor.
// If the Mask does not exist or is a different shape from the source,
// then it is created or reshaped, and all values set to true ("transparent").
func (ms *Masked) SyncShape() {
	if ms.Mask == nil {
		ms.Mask = NewBoolShape(ms.Tensor.Shape())
		ms.Mask.SetTrue()
		return
	}
	if !ms.Mask.Shape().IsEqual(ms.Tensor.Shape()) {
		SetShapeFrom(ms.Mask, ms.Tensor)
		ms.Mask.SetTrue()
	}
}

func (ms *Masked) Label() string            { return label(metadata.Name(ms), ms.Shape()) }
func (ms *Masked) String() string           { return Sprintf("", ms, 0) }
func (ms *Masked) Metadata() *metadata.Data { return ms.Tensor.Metadata() }
func (ms *Masked) IsString() bool           { return ms.Tensor.IsString() }
func (ms *Masked) DataType() reflect.Kind   { return ms.Tensor.DataType() }
func (ms *Masked) ShapeSizes() []int        { return ms.Tensor.ShapeSizes() }
func (ms *Masked) Shape() *Shape            { return ms.Tensor.Shape() }
func (ms *Masked) Len() int                 { return ms.Tensor.Len() }
func (ms *Masked) NumDims() int             { return ms.Tensor.NumDims() }
func (ms *Masked) DimSize(dim int) int      { return ms.Tensor.DimSize(dim) }

// AsValues returns a copy of this tensor as raw [Values].
// This "renders" the Masked view into a fully contiguous
// and optimized memory representation of that view.
// Because the masking pattern is unpredictable, only a 1D shape is possible.
func (ms *Masked) AsValues() Values {
	dt := ms.Tensor.DataType()
	n := ms.Len()
	switch {
	case ms.Tensor.IsString():
		vals := make([]string, 0, n)
		for i := range n {
			if !ms.Mask.Bool1D(i) {
				continue
			}
			vals = append(vals, ms.Tensor.String1D(i))
		}
		return NewStringFromValues(vals...)
	case reflectx.KindIsFloat(dt):
		vals := make([]float64, 0, n)
		for i := range n {
			if !ms.Mask.Bool1D(i) {
				continue
			}
			vals = append(vals, ms.Tensor.Float1D(i))
		}
		return NewFloat64FromValues(vals...)
	default:
		vals := make([]int, 0, n)
		for i := range n {
			if !ms.Mask.Bool1D(i) {
				continue
			}
			vals = append(vals, ms.Tensor.Int1D(i))
		}
		return NewIntFromValues(vals...)
	}
}

// SourceIndexes returns a flat [Int] tensor of the mask values
// that match the given getTrue argument state.
// These can be used as indexes in the [Indexed] view, for example.
// The resulting tensor is 2D with inner dimension = number of source
// tensor dimensions, to hold the indexes, and outer dimension = number
// of indexes.
func (ms *Masked) SourceIndexes(getTrue bool) *Int {
	n := ms.Len()
	nd := ms.Tensor.NumDims()
	idxs := make([]int, 0, n*nd)
	for i := range n {
		if ms.Mask.Bool1D(i) != getTrue {
			continue
		}
		ix := ms.Tensor.Shape().IndexFrom1D(i)
		idxs = append(idxs, ix...)
	}
	it := NewIntFromValues(idxs...)
	it.SetShapeSizes(len(idxs)/nd, nd)
	return it
}

////////  Floats

func (ms *Masked) Float(i ...int) float64 {
	if !ms.Mask.Bool(i...) {
		return math.NaN()
	}
	return ms.Tensor.Float(i...)
}

func (ms *Masked) SetFloat(val float64, i ...int) {
	if !ms.Mask.Bool(i...) {
		return
	}
	ms.Tensor.SetFloat(val, i...)
}

func (ms *Masked) Float1D(i int) float64 {
	if !ms.Mask.Bool1D(i) {
		return math.NaN()
	}
	return ms.Tensor.Float1D(i)
}

func (ms *Masked) SetFloat1D(val float64, i int) {
	if !ms.Mask.Bool1D(i) {
		return
	}
	ms.Tensor.SetFloat1D(val, i)
}

////////  Strings

func (ms *Masked) StringValue(i ...int) string {
	if !ms.Mask.Bool(i...) {
		return ""
	}
	return ms.Tensor.StringValue(i...)
}

func (ms *Masked) SetString(val string, i ...int) {
	if !ms.Mask.Bool(i...) {
		return
	}
	ms.Tensor.SetString(val, i...)
}

func (ms *Masked) String1D(i int) string {
	if !ms.Mask.Bool1D(i) {
		return ""
	}
	return ms.Tensor.String1D(i)
}

func (ms *Masked) SetString1D(val string, i int) {
	if !ms.Mask.Bool1D(i) {
		return
	}
	ms.Tensor.SetString1D(val, i)
}

////////  Ints

func (ms *Masked) Int(i ...int) int {
	if !ms.Mask.Bool(i...) {
		return 0
	}
	return ms.Tensor.Int(i...)
}

func (ms *Masked) SetInt(val int, i ...int) {
	if !ms.Mask.Bool(i...) {
		return
	}
	ms.Tensor.SetInt(val, i...)
}

func (ms *Masked) Int1D(i int) int {
	if !ms.Mask.Bool1D(i) {
		return 0
	}
	return ms.Tensor.Int1D(i)
}

// SetInt1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (ms *Masked) SetInt1D(val int, i int) {
	if !ms.Mask.Bool1D(i) {
		return
	}
	ms.Tensor.SetInt1D(val, i)
}

// Filter sets the mask values using given Filter function.
// The filter function gets the 1D index into the source tensor.
func (ms *Masked) Filter(filterer func(tsr Tensor, idx int) bool) {
	n := ms.Tensor.Len()
	for i := range n {
		ms.Mask.SetBool1D(filterer(ms.Tensor, i), i)
	}
}

// check for interface impl
var _ Tensor = (*Masked)(nil)
