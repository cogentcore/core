// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"slices"
)

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

// Slice returns a new shape applying the ranges, in order, to
// the dimensions. It is important that all dimensions are non-zero,
// otherwise nothing will be included.  An error is returned if this
// is the case.  Dimensions beyond the ranges specified are
// automatically included.
func (sh *Shape) Slice(ranges ...Range) ([]int, error) {
	nsz := slices.Clone(sh.Sizes)
	mx := min(len(ranges), len(sh.Sizes))
	var zd []int
	for i := range mx {
		nsz[i] = ranges[i].Size(sh.Sizes[i])
		if nsz[i] == 0 {
			zd = append(zd, i)
		}
	}
	if len(zd) > 0 {
		return nsz, fmt.Errorf("tensor.Shape Slice has zero size for following dimensions: %v", zd)
	}
	return nsz, nil
}

// Slice extracts a subset of values from the given tensor into the
// output tensor, according to the provided ranges.
// Dimensions beyond the ranges specified are automatically included.
func Slice(tsr, out Tensor, ranges ...Range) error {
	nsz, err := tsr.Shape().Slice(ranges...)
	if err != nil {
		return err
	}
	ndim := len(nsz)
	out.SetShape(nsz...)
	nl := out.Len()
	oc := make([]int, ndim) // orig coords
	nr := len(ranges)
	for ni := range nl {
		nc := out.Shape().Index(ni)
		for i := range ndim {
			c := nc[i]
			if i < nr {
				r := ranges[i]
				oc[i] = r.Start + c*r.IncrActual()
			} else {
				oc[i] = c
			}
		}
		oi := tsr.Shape().Offset(oc...)
		if out.IsString() {
			out.SetString1D(tsr.String1D(oi), ni)
		} else {
			out.SetFloat1D(tsr.Float1D(oi), ni)
		}
	}
	return nil
}
