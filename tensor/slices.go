// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

// SlicesMagic are special elements in slice expressions, including
// NewAxis, FullAxis, and Ellipsis in [NewSliced] expressions.
type SlicesMagic int //enums:enum

const (
	// FullAxis indicates that the full existing axis length should be used.
	// This is equivalent to Slice{}, but is more semantic. In NumPy it is
	// equivalent to a single : colon.
	FullAxis SlicesMagic = iota

	// NewAxis creates a new singleton (length=1) axis, used to to reshape
	// without changing the size. Can also be used in [Reshaped].
	NewAxis

	// Ellipsis (...) is used in [NewSliced] expressions to produce
	// a flexibly-sized stretch of FullAxis dimensions, which automatically
	// aligns the remaining slice elements based on the source dimensionality.
	Ellipsis
)

// Slice represents a slice of index values, for extracting slices of data,
// along a dimension of a given size, which is provided separately as an argument.
// Uses standard 'for' loop logic with a Start and _exclusive_ Stop value,
// and a Step increment: for i := Start; i < Stop; i += Step.
// The values stored in this struct are the _inputs_ for computing the actual
// slice values based on the actual size parameter for the dimension.
// Negative numbers count back from the end (i.e., size + val), and
// the zero value results in a list of all values in the dimension, with Step = 1 if 0.
// The behavior is identical to the NumPy slice.
type Slice struct {
	// Start is the starting value. If 0 and Step < 0, = size-1;
	// If negative, = size+Start.
	Start int

	// Stop value. If 0 and Step >= 0, = size;
	// If 0 and Step < 0, = -1, to include whole range.
	// If negative = size+Stop.
	Stop int

	// Step increment. If 0, = 1; if negative then Start must be > Stop
	// to produce anything.
	Step int
}

// NewSlice returns a new Slice with given srat, stop, step values.
func NewSlice(start, stop, step int) Slice {
	return Slice{Start: start, Stop: stop, Step: step}
}

// GetStart is the actual start value given the size of the dimension.
func (sl Slice) GetStart(size int) int {
	if sl.Start == 0 && sl.Step < 0 {
		return size - 1
	}
	if sl.Start < 0 {
		return size + sl.Start
	}
	return sl.Start
}

// GetStop is the actual end value given the size of the dimension.
func (sl Slice) GetStop(size int) int {
	if sl.Stop == 0 && sl.Step >= 0 {
		return size
	}
	if sl.Stop == 0 && sl.Step < 0 {
		return -1
	}
	if sl.Stop < 0 {
		return size + sl.Stop
	}
	return min(sl.Stop, size)
}

// GetStep is the actual increment value.
func (sl Slice) GetStep() int {
	if sl.Step == 0 {
		return 1
	}
	return sl.Step
}

// Len is the number of elements in the actual slice given
// size of the dimension.
func (sl Slice) Len(size int) int {
	s := sl.GetStart(size)
	e := sl.GetStop(size)
	i := sl.GetStep()
	n := max((e-s)/i, 0)
	pe := s + n*i
	if i < 0 {
		if pe > e {
			n++
		}
	} else {
		if pe < e {
			n++
		}
	}
	return n
}

// ToIntSlice writes values to given []int slice, with given size parameter
// for the dimension being sliced. If slice is wrong size to hold values,
// not all are written: allocate ints using Len(size) to fit.
func (sl Slice) ToIntSlice(size int, ints []int) {
	n := len(ints)
	if n == 0 {
		return
	}
	s := sl.GetStart(size)
	e := sl.GetStop(size)
	inc := sl.GetStep()
	idx := 0
	if inc < 0 {
		for i := s; i > e; i += inc {
			ints[idx] = i
			idx++
			if idx >= n {
				break
			}
		}
	} else {
		for i := s; i < e; i += inc {
			ints[idx] = i
			idx++
			if idx >= n {
				break
			}
		}
	}
}

// IntSlice returns []int slice with slice index values, up to given actual size.
func (sl Slice) IntSlice(size int) []int {
	n := sl.Len(size)
	if n == 0 {
		return nil
	}
	ints := make([]int, n)
	sl.ToIntSlice(size, ints)
	return ints
}

// IntTensor returns an [Int] [Tensor] for slice, using actual size.
func (sl Slice) IntTensor(size int) *Int {
	n := sl.Len(size)
	if n == 0 {
		return nil
	}
	tsr := NewInt(n)
	sl.ToIntSlice(size, tsr.Values)
	return tsr
}
