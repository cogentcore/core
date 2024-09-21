// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"slices"
)

// todo: make a version of these functions that takes
// a tensor with n x 3 shape, where the 3 inner values
// specify the Range Start, End, Incr values, across n ranges.
// these would convert to the current Range-based format that does the impl,
// using the Range helper functions, which are also easier and more explicit
// to use in Go code.

// Range represents a range of values, for extracting slices of data,
// using standard for loop logic with a Start and exclusive End value,
// and an increment: for i := Start; i < End; i += Incr.
// The zero value means all values in the dimension.
type Range struct {
	// Starting value.
	Start int

	// End value. 0 default = size of relevant dimension.
	End int

	// Increment. must be positive, 1 or greater. 0 default = 1.
	Incr int
}

// EndActual is the actual end value given the size of the dimension.
func (rn *Range) EndActual(size int) int {
	if rn.End == 0 {
		return size
	}
	return min(rn.End, size) // preserves -1 for no values.
}

// IncrActual is the actual increment value.
func (rn *Range) IncrActual() int {
	return max(1, rn.Incr)
}

// Size is the number of elements in the actual range given
// size of the dimension.
func (rn *Range) Size(size int) int {
	e := rn.EndActual(size)
	if e <= rn.Start {
		return 0
	}
	i := rn.IncrActual()
	return (e - rn.Start) / i
}

// RangeSizes returns a set of sizes applying the ranges, in order, to
// the given dimension sizes. It is important that all dimensions
// are non-zero, otherwise nothing will be included.
// An error is returned if this is the case.
// Dimensions beyond the ranges specified are automatically included.
func RangeSizes(sizes []int, ranges ...Range) ([]int, error) {
	nsz := slices.Clone(sizes)
	mx := min(len(ranges), len(sizes))
	var zd []int
	for i := range mx {
		nsz[i] = ranges[i].Size(sizes[i])
		if nsz[i] == 0 {
			zd = append(zd, i)
		}
	}
	if len(zd) > 0 {
		return nsz, fmt.Errorf("tensor.Shape Slice has zero size for following dimensions: %v", zd)
	}
	return nsz, nil
}

func IntRangeTensor(start, end, incr int) *Int {
	rn := Range{Start: start, End: end, Incr: incr}
	tsr := NewInt(rn.Size(end))
	idx := 0
	for i := start; i < end; i += incr {
		tsr.Values[idx] = i
		idx++
	}
	return tsr
}
