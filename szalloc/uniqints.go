// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package szalloc

import (
	"image"
	"sort"

	"github.com/goki/mat32"
	"goki.dev/ki/v2/ints"
)

// UniqSortedInts returns the ints in sorted order with only unique vals
func UniqSortedInts(vals []int) []int {
	sort.Ints(vals)
	sz := len(vals)
	lst := vals[0]
	uvals := make([]int, 0, sz)
	uvals = append(uvals, vals[0])
	for i := 1; i < sz; i++ {
		v := vals[i]
		if v != lst {
			uvals = append(uvals, v)
		}
	}
	return uvals
}

// SizeGroups returns evenly-spaced size groups of max N -- could be less
func SizeGroups(sizes []int, maxN int) []int {
	ns := len(sizes)
	mxgp := ints.MinInt(ns, maxN)
	nper := float32(ns) / float32(mxgp)

	idxs := make([]int, mxgp)
	for i := 0; i < mxgp; i++ {
		cut := int(mat32.Round(float32(i+1) * nper))
		if cut >= ns {
			cut = ns - 1
		}
		idxs[i] = cut
	}
	idxs[mxgp-1] = ns - 1
	return idxs
}

// PointsClone returns clone of []image.Point list
func PointsClone(pts []image.Point) []image.Point {
	np := len(pts)
	cpts := make([]image.Point, np)
	copy(cpts, pts)
	return cpts
}
