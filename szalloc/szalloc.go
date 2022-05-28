// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package szalloc

import (
	"fmt"
	"image"
	"sort"

	"github.com/goki/mat32"
)

// SzAlloc manages allocation of sizes to a spec'd maximum number
// of groups.  Used for allocating texture images to image arrays
// under the severe constraints of only 16 images.
type SzAlloc struct {
	On        bool                `desc:"true if configured and ready to use"`
	MaxGps    int                 `desc:"maximum number of groups to allocate"`
	ItmSizes  []image.Point       `desc:"original list of item sizes to be allocated"`
	UniqSizes []image.Point       `desc:"list of all unique sizes -- operate on this for grouping"`
	UniqSzMap map[image.Point]int `desc:"map of all unique sizes, with count per"`
	GpSizes   []image.Point       `desc:"list of allocated group sizes"`
	GpAllocs  [][]int             `desc:"allocation of image indexes by size"`
	ItmIdxs   []*Idxs             `desc:"allocation image value indexes to image indexes"`
}

// SetSizes sets the max number of groups, and item sizes to organize
func (sa *SzAlloc) SetSizes(gps int, itms []image.Point) {
	sa.MaxGps = gps
	sa.ItmSizes = itms
}

func Area(sz image.Point) int {
	return sz.X * sz.Y
}

// Alloc allocates items as a function of size
func (sa *SzAlloc) Alloc() {
	ni := len(sa.ItmSizes)
	if ni == 0 {
		return
	}

	sa.UniqSz()
	nu := len(sa.UniqSizes)
	if ni <= sa.MaxGps { // all fits
		sa.AllocItms() // directly allocate existing items
		return
	}

	// separately divide X and Y sorted lists to create groups
	xorder := make([]int, nu)
	yorder := make([]int, nu)
	for i := range xorder {
		xorder[i] = i
		yorder[i] = i
	}

	sort.Slice(xorder, func(i, j int) bool {
		visz := sa.UniqSizes[xorder[i]]
		vjsz := sa.UniqSizes[xorder[j]]
		return visz.X < vjsz.X
	})
	sort.Slice(yorder, func(i, j int) bool {
		visz := sa.UniqSizes[yorder[i]]
		vjsz := sa.UniqSizes[yorder[j]]
		return visz.Y < vjsz.Y
	})

	nper := float32(nu) / float32(sa.MaxGps)
	if nper < 1 {
		nper = 1
	}

	sa.GpSizes = make([]image.Point, sa.MaxGps)
	for i := 0; i < sa.MaxGps; i++ {
		icut := int(mat32.Round(float32(i+1) * nper))
		if icut >= nu {
			icut = nu - 1
		}
		sa.GpSizes[i] = image.Point{sa.UniqSizes[xorder[icut]].X, sa.UniqSizes[yorder[icut]].Y}
	}
	// ensure last one is max
	sa.GpSizes[sa.MaxGps-1] = image.Point{sa.UniqSizes[xorder[nu-1]].X, sa.UniqSizes[xorder[nu-1]].Y}
	sa.AllocGps()
}

// UniqSz computes unique sizes
func (sa *SzAlloc) UniqSz() {
	ni := len(sa.ItmSizes)
	sa.UniqSizes = make([]image.Point, 0, ni)
	sa.UniqSzMap = make(map[image.Point]int, ni)
	for _, sz := range sa.ItmSizes {
		n, has := sa.UniqSzMap[sz]
		n++
		if !has {
			sa.UniqSizes = append(sa.UniqSizes, sz)
		}
		sa.UniqSzMap[sz] = n
	}
	// fmt.Printf("n uniq sizes: %d\n", len(sa.UniqSizes))
}

// AllocGps allocates groups based on final groupings
func (sa *SzAlloc) AllocGps() {
	ni := len(sa.ItmSizes)
	ng := len(sa.GpSizes)
	// fmt.Printf("gpsizes: %v\n", sa.GpSizes)
	li := 0
	gi := 0
	gsz := sa.GpSizes[0]
	sa.ItmIdxs = make([]*Idxs, ni)
	sa.GpAllocs = make([][]int, ng)
	for i, sz := range sa.ItmSizes {
		var j int
		for j, gsz = range sa.GpSizes {
			if sz.X <= gsz.X && sz.Y <= gsz.Y {
				gi = j
				break
			}
		}
		li = len(sa.GpAllocs[gi])
		sa.GpAllocs[gi] = append(sa.GpAllocs[gi], i)
		sa.ItmIdxs[i] = NewIdxs(gi, li, sz, gsz)
	}
	// sa.PrintGps()
	sa.On = true
}

// PrintGps prints the group allocations
func (sa *SzAlloc) PrintGps() {
	for j, ga := range sa.GpAllocs {
		fmt.Printf("idx: %2d  gsz: %v  n: %d\n", j, sa.GpSizes[j], len(ga))
	}
}

// AllocItms directly allocate items each to their own group -- all fits
func (sa *SzAlloc) AllocItms() {
	ni := len(sa.ItmSizes)
	sa.GpSizes = make([]image.Point, ni)
	sa.ItmIdxs = make([]*Idxs, ni)
	sa.GpAllocs = make([][]int, ni)
	for i, sz := range sa.ItmSizes {
		sa.GpAllocs[i] = append(sa.GpAllocs[i], i)
		sa.ItmIdxs[i] = NewIdxs(i, 0, sz, sz)
		sa.GpSizes[i] = sz
	}
	// sa.PrintGps()
	sa.On = true
}
