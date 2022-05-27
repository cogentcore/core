// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package szalloc

import (
	"fmt"
	"image"
	"sort"

	"github.com/goki/ki/ints"
)

// SzAlloc manages allocation of sizes to a spec'd maximum number
// of groups.  Used for allocating texture images to image arrays
// under the severe constraints of only 16 images.
type SzAlloc struct {
	MaxGps    int                 `desc:"maximum number of groups to allocate"`
	ItmSizes  []image.Point       `desc:"original list of item sizes to be allocated"`
	GpSizes   []image.Point       `desc:"list of allocated group sizes"`
	GpAllocs  [][]int             `desc:"allocation of image indexes by size"`
	ItmIdxs   []*Idxs             `desc:"allocation image value indexes to image indexes"`
	UniqSizes map[image.Point]int `desc:"map of all unique sizes, with count per"`
}

// PctDiff returns the percent difference vs. max of two vals
func PctDiff(a, b int) float32 {
	mx := ints.MaxInt(a, b)
	if mx == 0 {
		return 0
	}
	d := ints.AbsInt(a - b)
	return float32(d) / float32(mx)
}

// SetSizes sets the max number of groups, and item sizes to organize
func (sa *SzAlloc) SetSizes(gps int, itms []image.Point) {
	sa.MaxGps = gps
	sa.ItmSizes = itms
}

// Alloc allocates items as a function of size
func (sa *SzAlloc) Alloc() {
	ni := len(sa.ItmSizes)
	if ni == 0 {
		return
	}
	order := make([]int, ni)
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool {
		visz := sa.ItmSizes[order[i]]
		vjsz := sa.ItmSizes[order[j]]
		pctdx := PctDiff(visz.X, vjsz.X)
		pctdy := PctDiff(visz.Y, vjsz.Y)
		if pctdx < .2 && pctdy < .2 { // if close, sort by X
			return visz.X < vjsz.X
		}
		iarea := visz.X * visz.Y
		jarea := vjsz.X * vjsz.Y
		return iarea < jarea
	})

	sa.GpSizes = make([]image.Point, 0, ni)
	sa.UniqSizes = make(map[image.Point]int, ni)
	for _, vi := range order {
		vsz := sa.ItmSizes[vi]
		n, has := sa.UniqSizes[vsz]
		n++
		if !has {
			sa.GpSizes = append(sa.GpSizes, vsz)
		}
		sa.UniqSizes[vsz] = n
	}
	fmt.Printf("gpsizes: %v\n", sa.GpSizes)
	if len(sa.UniqSizes) >= sa.MaxGps {
		prvalloc := make([]image.Point, len(sa.GpSizes), ni)
		copy(prvalloc, sa.GpSizes)
		sa.GpSizes = sa.GpSizes[:0]
		lstsz := prvalloc[0]
		mxsz := lstsz
		// group together by area, progresively until under thr
		areathr := float32(0.1)
		itr := 0
		for {
			for i, sz := range prvalloc {
				if i == 0 {
					continue
				}
				mxsz.X = ints.MaxInt(mxsz.X, sz.X)
				mxsz.Y = ints.MaxInt(mxsz.Y, sz.Y)
				pd := PctDiff(lstsz.X*lstsz.Y, sz.X*sz.Y)
				if pd < areathr {
					continue
				}
				lstsz = sz
				sa.GpSizes = append(sa.GpSizes, mxsz)
				mxsz = sz
			}
			lsz := prvalloc[len(prvalloc)-1]
			mxsz.X = ints.MaxInt(mxsz.X, lsz.X)
			mxsz.Y = ints.MaxInt(mxsz.Y, lsz.Y)
			if sa.GpSizes[len(sa.GpSizes)-1] != mxsz {
				sa.GpSizes = append(sa.GpSizes, mxsz)
			}
			fmt.Printf("itr: %d  gps: %d\n gpsizes: %v\n", itr, len(sa.GpSizes), sa.GpSizes)
			if len(sa.GpSizes) < sa.MaxGps {
				fmt.Printf("done!\n")
				break
			}
			prvalloc = prvalloc[0:len(sa.GpSizes)]
			copy(prvalloc, sa.GpSizes)
			sa.GpSizes = sa.GpSizes[:0]
			lstsz = prvalloc[0]
			areathr += .1
			itr++
		}
	}
	ng := len(sa.GpSizes)
	lsz := sa.ItmSizes[order[len(order)-1]]
	if sa.GpSizes[len(sa.GpSizes)-1] != lsz {
		sa.GpSizes = append(sa.GpSizes, lsz)
		ng++
	}
	fmt.Printf("gpsizes: %v\n", sa.GpSizes)
	gi := 0
	li := 0
	gsz := sa.GpSizes[0]
	sa.ItmIdxs = make([]*Idxs, ni)
	sa.GpAllocs = make([][]int, ng)
	for i, vi := range order {
		vsz := sa.ItmSizes[vi]
		if vsz.X <= gsz.X && vsz.Y <= gsz.Y {
			li = len(sa.GpAllocs[gi])
			sa.GpAllocs[gi] = append(sa.GpAllocs[gi], vi)
		} else {
			li = 0
			gi++
			gsz = sa.GpSizes[gi]
			sa.GpAllocs[gi] = append(sa.GpAllocs[gi], vi)
		}
		sa.ItmIdxs[vi] = NewIdxs(gi, li, vsz, gsz)
		fmt.Printf("idx: %2d  img: %2d  sz: %v  gsz: %v  gi: %2d  li: %2d\n", i, vi, vsz, gsz, gi, li)
	}
}
