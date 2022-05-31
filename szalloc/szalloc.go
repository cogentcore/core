// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package szalloc

import (
	"fmt"
	"image"
	"math/rand"

	"github.com/goki/ki/ints"
	"github.com/goki/ki/sliceclone"
)

// MaxIters is maximum number of iterations for adapting sizes to fit constraints
const MaxIters = 100

// SzAlloc manages allocation of sizes to a spec'd maximum number
// of groups.  Used for allocating texture images to image arrays
// under the severe constraints of only 16 images.
type SzAlloc struct {
	On           bool                `desc:"true if configured and ready to use"`
	MaxGps       image.Point         `desc:"maximum number of groups in X and Y dimensions"`
	MaxNGps      int                 `desc:"maximum number of groups = X * Y"`
	MaxItmsPerGp int                 `desc:"maximum number of items per group -- constraint is enforced in addition to MaxGps"`
	ItmSizes     []image.Point       `desc:"original list of item sizes to be allocated"`
	UniqSizes    []image.Point       `desc:"list of all unique sizes -- operate on this for grouping"`
	UniqSzMap    map[image.Point]int `desc:"map of all unique sizes, with count per"`
	GpSizes      []image.Point       `desc:"list of allocated group sizes"`
	GpAllocs     [][]int             `desc:"allocation of image indexes by size"`
	ItmIdxs      []*Idxs             `desc:"allocation image value indexes to image indexes"`
	XSizes       []int               `desc:"sorted list of all unique sizes"`
	YSizes       []int               `desc:"sorted list of all unique sizes"`
	GpNs         image.Point         `desc:"number of items in each dimension group (X, Y)"`
	XGpIdxs      []int               `desc:"list of x group indexes"`
	YGpIdxs      []int               `desc:"list of y group indexes"`
}

// SetSizes sets the max number of groups along each dimension (X, Y),
// so total number of groups is X*Y, and max items per group,
// and item sizes to organize
func (sa *SzAlloc) SetSizes(gps image.Point, itmsPerGp int, itms []image.Point) {
	sa.MaxGps = gps
	sa.MaxNGps = Area(gps)
	sa.MaxItmsPerGp = itmsPerGp
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
	if ni <= sa.MaxNGps { // all fits
		sa.AllocItemsNoGps() // directly allocate existing items
		return
	}

	sa.XSizes = make([]int, nu)
	sa.YSizes = make([]int, nu)
	for i, usz := range sa.UniqSizes {
		sa.XSizes[i] = usz.X
		sa.YSizes[i] = usz.Y
	}
	sa.XSizes = UniqSortedInts(sa.XSizes)
	sa.YSizes = UniqSortedInts(sa.YSizes)
	sa.XGpIdxs = SizeGroups(sa.XSizes, sa.MaxGps.X)
	sa.YGpIdxs = SizeGroups(sa.YSizes, sa.MaxGps.Y)

	maxItms := 0
	sa.GpAllocs, maxItms = sa.AllocGps(sa.XGpIdxs, sa.YGpIdxs)
	if maxItms > sa.MaxItmsPerGp {
		sa.LimitGpNs()
	}
	sa.GpSizes = sa.SizesFmIdxs(sa.XGpIdxs, sa.YGpIdxs)
	sa.GpAllocs, _ = sa.AllocGps(sa.XGpIdxs, sa.YGpIdxs) // final updates
	sa.AllocGpItems()
	sa.UpdateGpMaxSz()

	sa.On = true
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

// XYSizeFmIdx returns X,Y sizes from X,Y indexes in image.Point
// into XSizes, YSizes
func (sa *SzAlloc) XYSizeFmIdx(idx image.Point) image.Point {
	return image.Point{sa.XSizes[idx.X], sa.YSizes[idx.Y]}
}

// XYFmGpi returns x, y indexes from gp index
func XYfmGpi(gi, nxi int) (xi, yi int) {
	xi = gi % nxi
	yi = gi / nxi
	return
}

// SizesFmIdxs returns X,Y sizes from X,Y indexes in image.Point
// into XSizes, YSizes arrays
func (sa *SzAlloc) SizesFmIdxs(xgpi, ygpi []int) []image.Point {
	ng := len(xgpi) * len(ygpi)
	szs := make([]image.Point, ng)
	for yi, ygi := range ygpi {
		ysz := sa.YSizes[ygi]
		for xi, xgi := range xgpi {
			xsz := sa.XSizes[xgi]
			gi := yi*len(xgpi) + xi
			szs[gi] = image.Point{xsz, ysz}
		}
	}
	return szs
}

// AllocGps allocates groups based on given indexes into XSizes, YSizes.
// returns allocs = indexes of items per each group,
// and max number of items per group
func (sa *SzAlloc) AllocGps(xgpi, ygpi []int) (allocs [][]int, maxItms int) {
	ng := len(xgpi) * len(ygpi)
	maxItms = 0
	gi := 0
	allocs = make([][]int, ng)
	for i, sz := range sa.ItmSizes {
		for yi, ygi := range ygpi {
			ysz := sa.YSizes[ygi]
			if sz.Y > ysz {
				continue
			}
			for xi, xgi := range xgpi {
				xsz := sa.XSizes[xgi]
				if sz.X > xsz {
					continue
				}
				gi = yi*len(xgpi) + xi
				break
			}
			break
		}
		allocs[gi] = append(allocs[gi], i)
		nitm := len(allocs[gi])
		maxItms = ints.MaxInt(nitm, maxItms)
	}
	return
}

// AllocGpItems allocates items in groups based on final GpAllocs
func (sa *SzAlloc) AllocGpItems() {
	ni := len(sa.ItmSizes)
	sa.ItmIdxs = make([]*Idxs, ni)
	for gi, ga := range sa.GpAllocs {
		gsz := sa.GpSizes[gi]
		for i, li := range ga {
			sz := sa.ItmSizes[li]
			sa.ItmIdxs[li] = NewIdxs(gi, i, sz, gsz)
		}
	}
}

// UpdateGpMaxSz updates the group sizes based on actual max sizes of items
func (sa *SzAlloc) UpdateGpMaxSz() {
	for j, ga := range sa.GpAllocs {
		na := len(ga)
		if na == 0 {
			continue
		}
		sz := sa.ItmSizes[ga[0]]
		// fmt.Printf("j: %2d  sz: %v\n", j, sz)
		for i := 1; i < na; i++ {
			isz := sa.ItmSizes[ga[i]]
			// fmt.Printf("\ti: %2d  itm: %3d  isz: %v\n", i, ga[i], isz)
			sz.X = ints.MaxInt(sz.X, isz.X)
			sz.Y = ints.MaxInt(sz.Y, isz.Y)
		}
		sa.GpSizes[j] = sz
	}
}

// LimitGpNs updates group sizes to ensure that the MaxItmsPerGp limit
// is not exceeded.
func (sa *SzAlloc) LimitGpNs() {
	nxi := len(sa.XGpIdxs)

	xidxs := sliceclone.Int(sa.XGpIdxs)
	yidxs := sliceclone.Int(sa.YGpIdxs)
	gpallocs, bestmax := sa.AllocGps(xidxs, yidxs)

	avg := len(sa.ItmSizes) / sa.MaxNGps
	low := (avg * 3) / 4

	bestXidxs := sliceclone.Int(sa.XGpIdxs)
	bestYidxs := sliceclone.Int(sa.YGpIdxs)

	itr := 0
	for itr = 0; itr < MaxIters; itr++ {
		chg := false
		for j, ga := range gpallocs {
			xi, yi := XYfmGpi(j, nxi)
			na := len(ga)
			if na <= low {
				if rand.Intn(2) == 0 {
					if xidxs[xi] < len(sa.XSizes)-1 {
						xidxs[xi] = xidxs[xi] + 1
					}
				} else {
					if yidxs[yi] < len(sa.YSizes)-1 {
						yidxs[yi] = yidxs[yi] + 1
					}
				}
				chg = true
			} else if na > sa.MaxItmsPerGp {
				if rand.Intn(2) == 0 {
					if xidxs[xi] > 0 {
						xidxs[xi] = xidxs[xi] - 1
					}
				} else {
					if yidxs[yi] > 0 {
						yidxs[yi] = yidxs[yi] - 1
					}
				}
				chg = true
			}
		}
		if !chg {
			// fmt.Printf("itr: %d  no change, bailing\n", itr)
			break
		}
		maxItms := 0
		gpallocs, maxItms = sa.AllocGps(xidxs, yidxs)
		if maxItms < bestmax {
			bestmax = maxItms
			bestXidxs = sliceclone.Int(xidxs)
			bestYidxs = sliceclone.Int(yidxs)
		}
		// gps := sa.SizesFmIdxs(xidxs, yidxs)
		// fmt.Printf("itr: %d  maxi: %d  gps: %v\n", itr, maxItms, gps)
		if maxItms <= sa.MaxItmsPerGp {
			break
		}
	}
	sa.XGpIdxs = bestXidxs
	sa.YGpIdxs = bestYidxs
	// _, maxItms := sa.AllocGps(sa.XGpIdxs, sa.YGpIdxs)
	// fmt.Printf("itrs: %d  maxItms: %d\n", itr, maxItms)
	// edgps := sa.SizesFmIdxs(sa.XGpIdxs, sa.YGpIdxs)
	// fmt.Printf("ending gps: %v\n", edgps)
}

// PrintGps prints the group allocations
func (sa *SzAlloc) PrintGps() {
	for j, ga := range sa.GpAllocs {
		fmt.Printf("idx: %2d  gsz: %v  n: %d\n", j, sa.GpSizes[j], len(ga))
	}
}

// AllocItemsNoGps directly allocate items each to their own group -- all fits
func (sa *SzAlloc) AllocItemsNoGps() {
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
