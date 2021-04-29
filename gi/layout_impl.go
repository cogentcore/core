// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"

	"github.com/goki/gi/gist"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
)

// This file contains all the hairy implementation logic of layout
// Done as separate functions because not really needed in derived types

// LaySumDim returns whether we sum up elements along given dimension?  else use
// max for shared dimension.
func LaySumDim(ly Layouts, d mat32.Dims) bool {
	if (d == mat32.X && (ly == LayoutHoriz || ly == LayoutHorizFlow)) || (d == mat32.Y && (ly == LayoutVert || ly == LayoutVertFlow)) {
		return true
	}
	return false
}

// LaySummedDim returns the dimension along which layout is summing.
func LaySummedDim(ly Layouts) mat32.Dims {
	if ly == LayoutHoriz || ly == LayoutHorizFlow {
		return mat32.X
	}
	return mat32.Y
}

////////////////////////////////////////////////////////////////////////////////////////
//     Gather Sizes

// first depth-first Size2D pass: terminal concrete items compute their AllocSize
// we focus on Need: Max(Min, AllocSize), and Want: Max(Pref, AllocSize) -- Max is
// only used if we need to fill space, during final allocation
//
// second me-first Layout2D pass: each layout allocates AllocSize for its
// children based on aggregated size data, and so on down the tree

// GatherSizesSumMax gets basic sum and max data across all kiddos
func GatherSizesSumMax(ly *Layout) (sumPref, sumNeed, maxPref, maxNeed mat32.Vec2) {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		ni.LayState.UpdateSizes()
		sumNeed = sumNeed.Add(ni.LayState.Size.Need)
		sumPref = sumPref.Add(ni.LayState.Size.Pref)
		maxNeed = maxNeed.Max(ni.LayState.Size.Need)
		maxPref = maxPref.Max(ni.LayState.Size.Pref)

		if Layout2DTrace {
			fmt.Printf("Size:   %v Child: %v, need: %v, pref: %v\n", ly.Path(), ni.Nm, ni.LayState.Size.Need.Dim(LaySummedDim(ly.Lay)), ni.LayState.Size.Pref.Dim(LaySummedDim(ly.Lay)))
		}
	}
	return
}

// GatherSizes is size first pass: gather the size information from the children
func GatherSizes(ly *Layout) {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	sumPref, sumNeed, maxPref, maxNeed := GatherSizesSumMax(ly)

	prefSizing := false
	mvp := ly.ViewportSafe()
	if mvp != nil && mvp.HasFlag(int(VpFlagPrefSizing)) {
		prefSizing = ly.Sty.Layout.Overflow == gist.OverflowScroll // special case
	}

	for d := mat32.X; d <= mat32.Y; d++ {
		pref := ly.LayState.Size.Pref.Dim(d)
		if prefSizing || pref == 0 {
			if LaySumDim(ly.Lay, d) { // our layout now updated to sum
				ly.LayState.Size.Need.SetMaxDim(d, sumNeed.Dim(d))
				ly.LayState.Size.Pref.SetMaxDim(d, sumPref.Dim(d))
			} else { // use max for other dir
				ly.LayState.Size.Need.SetMaxDim(d, maxNeed.Dim(d))
				ly.LayState.Size.Pref.SetMaxDim(d, maxPref.Dim(d))
			}
		} else { // use target size from style
			if Layout2DTrace {
				fmt.Printf("Size:   %v pref nonzero, setting as need: %v\n", ly.Path(), pref)
			}
			ly.LayState.Size.Need.SetDim(d, pref)
		}
	}

	spc := ly.BoxSpace()
	ly.LayState.Size.Need.SetAddScalar(2.0 * spc)
	ly.LayState.Size.Pref.SetAddScalar(2.0 * spc)

	elspc := float32(0.0)
	if sz >= 2 {
		elspc = float32(sz-1) * ly.Spacing.Dots
	}
	if LaySumDim(ly.Lay, mat32.X) {
		ly.LayState.Size.Need.X += elspc
		ly.LayState.Size.Pref.X += elspc
	}
	if LaySumDim(ly.Lay, mat32.Y) {
		ly.LayState.Size.Need.Y += elspc
		ly.LayState.Size.Pref.Y += elspc
	}

	ly.LayState.UpdateSizes() // enforce max and normal ordering, etc
	if Layout2DTrace {
		fmt.Printf("Size:   %v gather sizes need: %v, pref: %v, elspc: %v\n", ly.Path(), ly.LayState.Size.Need, ly.LayState.Size.Pref, elspc)
	}
}

// ChildrenUpdateSizes calls UpdateSizes on all children -- layout must at least call this
func (ly *Layout) ChildrenUpdateSizes() {
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		ni.LayState.UpdateSizes()
	}
}

// GatherSizesFlow is size first pass: gather the size information from the children
func GatherSizesFlow(ly *Layout, iter int) {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	if iter > 0 {
		ly.ChildrenUpdateSizes() // essential to call this
		prv := ly.ChildSize
		ly.LayState.Size.Need = prv
		ly.LayState.Size.Pref = prv
		ly.LayState.Alloc.Size = prv
		ly.LayState.UpdateSizes() // enforce max and normal ordering, etc
		if Layout2DTrace {
			fmt.Printf("Size:   %v iter 1 fix size: %v\n", ly.Path(), prv)
		}
		return
	}

	sumPref, sumNeed, maxPref, maxNeed := GatherSizesSumMax(ly)

	// for flow, the need is always *maxNeed* (i.e., a single item)
	// and the pref is based on styled pref estimate

	sdim := LaySummedDim(ly.Lay)
	odim := mat32.OtherDim(sdim)
	pref := ly.LayState.Size.Pref.Dim(sdim)
	// opref := ly.LayState.Size.Pref.Dim(odim) // not using

	if pref == 0 {
		pref = 6 * maxNeed.Dim(sdim) // 6 items preferred backup default
	}
	if pref == 0 {
		pref = 200 // final backstop
	}

	if Layout2DTrace {
		fmt.Printf("Size:   %v flow pref start: %v\n", ly.Path(), pref)
	}

	sNeed := sumNeed.Dim(sdim)
	nNeed := float32(1)
	tNeed := sNeed
	oNeed := maxNeed.Dim(odim)
	if sNeed > pref {
		tNeed = pref
		nNeed = mat32.Ceil(sNeed / tNeed)
		oNeed *= nNeed
	}

	sPref := sumPref.Dim(sdim)
	nPref := float32(1)
	tPref := sPref
	oPref := maxPref.Dim(odim)
	if sPref > pref {
		if sNeed > pref {
			tPref = pref
			nPref = mat32.Ceil(sPref / tPref)
			oPref *= nPref
		} else {
			tPref = sNeed // go with need
		}
	}

	ly.LayState.Size.Need.SetMaxDim(sdim, maxNeed.Dim(sdim)) // Need min = max of single item
	ly.LayState.Size.Pref.SetMaxDim(sdim, tPref)
	ly.LayState.Size.Need.SetMaxDim(odim, oNeed)
	ly.LayState.Size.Pref.SetMaxDim(odim, oPref)

	spc := ly.BoxSpace()
	ly.LayState.Size.Need.SetAddScalar(2.0 * spc)
	ly.LayState.Size.Pref.SetAddScalar(2.0 * spc)

	elspc := float32(0.0)
	if sz >= 2 {
		elspc = float32(sz-1) * ly.Spacing.Dots
	}
	if LaySumDim(ly.Lay, mat32.X) {
		ly.LayState.Size.Need.X += elspc
		ly.LayState.Size.Pref.X += elspc
	}
	if LaySumDim(ly.Lay, mat32.Y) {
		ly.LayState.Size.Need.Y += elspc
		ly.LayState.Size.Pref.Y += elspc
	}

	ly.LayState.UpdateSizes() // enforce max and normal ordering, etc
	if Layout2DTrace {
		fmt.Printf("Size:   %v gather sizes need: %v, pref: %v, elspc: %v\n", ly.Path(), ly.LayState.Size.Need, ly.LayState.Size.Pref, elspc)
	}
}

// todo: grid does not process spans at all yet -- assumes = 1

// GatherSizesGrid is size first pass: gather the size information from the
// children, grid version
func GatherSizesGrid(ly *Layout) {
	if len(ly.Kids) == 0 {
		return
	}

	cols := ly.Sty.Layout.Columns
	rows := 0

	sz := len(ly.Kids)
	// collect overall size
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		ni.StyMu.RLock()
		lst := ni.Sty.Layout
		ni.StyMu.RUnlock()
		if lst.Col > 0 {
			cols = ints.MaxInt(cols, lst.Col+lst.ColSpan)
		}
		if lst.Row > 0 {
			rows = ints.MaxInt(rows, lst.Row+lst.RowSpan)
		}
	}

	if cols == 0 {
		cols = int(mat32.Sqrt(float32(sz))) // whatever -- not well defined
	}
	if rows == 0 {
		rows = sz / cols
	}
	for rows*cols < sz { // not defined to have multiple items per cell -- make room for everyone
		rows++
	}

	ly.GridSize.X = cols
	ly.GridSize.Y = rows

	if len(ly.GridData[Row]) != rows {
		ly.GridData[Row] = make([]GridData, rows)
	}
	if len(ly.GridData[Col]) != cols {
		ly.GridData[Col] = make([]GridData, cols)
	}

	for i := range ly.GridData[Row] {
		gd := &ly.GridData[Row][i]
		gd.SizeNeed = 0
		gd.SizePref = 0
	}
	for i := range ly.GridData[Col] {
		gd := &ly.GridData[Col][i]
		gd.SizeNeed = 0
		gd.SizePref = 0
	}

	col := 0
	row := 0
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		ni.LayState.UpdateSizes()
		ni.StyMu.RLock()
		lst := ni.Sty.Layout
		ni.StyMu.RUnlock()
		if lst.Col > 0 {
			col = lst.Col
		}
		if lst.Row > 0 {
			row = lst.Row
		}
		// r   0   1   col X = max(ea in col) (Y = not used)
		//   +--+---+
		// 0 |  |   |  row Y = max(ea in row) (X = not used)
		//   +--+---+
		// 1 |  |   |
		//   +--+---+

		rgd := &(ly.GridData[Row][row])
		cgd := &(ly.GridData[Col][col])

		// todo: need to deal with span in sums..
		mat32.SetMax(&(rgd.SizeNeed), ni.LayState.Size.Need.Y)
		mat32.SetMax(&(rgd.SizePref), ni.LayState.Size.Pref.Y)
		mat32.SetMax(&(cgd.SizeNeed), ni.LayState.Size.Need.X)
		mat32.SetMax(&(cgd.SizePref), ni.LayState.Size.Pref.X)

		// for max: any -1 stretch dominates, else accumulate any max
		if rgd.SizeMax >= 0 {
			if ni.LayState.Size.Max.Y < 0 { // stretch
				rgd.SizeMax = -1
			} else {
				mat32.SetMax(&(rgd.SizeMax), ni.LayState.Size.Max.Y)
			}
		}
		if cgd.SizeMax >= 0 {
			if ni.LayState.Size.Max.Y < 0 { // stretch
				cgd.SizeMax = -1
			} else {
				mat32.SetMax(&(cgd.SizeMax), ni.LayState.Size.Max.X)
			}
		}

		col++
		if col >= cols { // todo: really only works if NO items specify row,col or ALL do..
			col = 0
			row++
			if row >= rows { // wrap-around.. no other good option
				row = 0
			}
		}
	}

	prefSizing := false
	mvp := ly.ViewportSafe()
	if mvp != nil && mvp.HasFlag(int(VpFlagPrefSizing)) {
		prefSizing = ly.Sty.Layout.Overflow == gist.OverflowScroll // special case
	}

	// if there aren't existing prefs, we need to compute size
	if prefSizing || ly.LayState.Size.Pref.X == 0 || ly.LayState.Size.Pref.Y == 0 {
		sbw := ly.Sty.Layout.ScrollBarWidth.Dots
		maxRow := len(ly.GridData[Row])
		maxCol := len(ly.GridData[Col])
		if prefSizing {
			maxRow = ints.MinInt(LayoutPrefMaxRows, maxRow)
			maxCol = ints.MinInt(LayoutPrefMaxCols, maxCol)
		}

		// Y = sum across rows which have max's
		var sumPref, sumNeed mat32.Vec2
		for i := 0; i < maxRow; i++ {
			gd := ly.GridData[Row][i]
			sumNeed.SetAddDim(mat32.Y, gd.SizeNeed)
			sumPref.SetAddDim(mat32.Y, gd.SizePref)
		}
		// X = sum across cols which have max's
		for i := 0; i < maxCol; i++ {
			gd := ly.GridData[Col][i]
			sumNeed.SetAddDim(mat32.X, gd.SizeNeed)
			sumPref.SetAddDim(mat32.X, gd.SizePref)
		}

		if prefSizing {
			sumNeed.X += sbw
			sumNeed.Y += sbw
			sumPref.X += sbw
			sumPref.Y += sbw
		}

		if ly.LayState.Size.Pref.X == 0 || prefSizing {
			ly.LayState.Size.Need.X = mat32.Max(ly.LayState.Size.Need.X, sumNeed.X)
			ly.LayState.Size.Pref.X = mat32.Max(ly.LayState.Size.Pref.X, sumPref.X)
		} else { // use target size from style otherwise
			ly.LayState.Size.Need.X = ly.LayState.Size.Pref.X
		}
		if ly.LayState.Size.Pref.Y == 0 || prefSizing {
			ly.LayState.Size.Need.Y = mat32.Max(ly.LayState.Size.Need.Y, sumNeed.Y)
			ly.LayState.Size.Pref.Y = mat32.Max(ly.LayState.Size.Pref.Y, sumPref.Y)
		} else { // use target size from style otherwise
			ly.LayState.Size.Need.Y = ly.LayState.Size.Pref.Y
		}
	} else { // neither are zero so we use those directly
		ly.LayState.Size.Need = ly.LayState.Size.Pref
	}

	spc := ly.BoxSpace()
	ly.LayState.Size.Need.SetAddScalar(2.0 * spc)
	ly.LayState.Size.Pref.SetAddScalar(2.0 * spc)

	ly.LayState.Size.Need.X += float32(cols-1) * ly.Spacing.Dots
	ly.LayState.Size.Pref.X += float32(cols-1) * ly.Spacing.Dots
	ly.LayState.Size.Need.Y += float32(rows-1) * ly.Spacing.Dots
	ly.LayState.Size.Pref.Y += float32(rows-1) * ly.Spacing.Dots

	ly.LayState.UpdateSizes() // enforce max and normal ordering, etc
	if Layout2DTrace {
		fmt.Printf("Size:   %v gather sizes grid need: %v, pref: %v\n", ly.Path(), ly.LayState.Size.Need, ly.LayState.Size.Pref)
	}
}

// LayAllocFromParent: if we are not a child of a layout, then get allocation
// from a parent obj that has a layout size
func LayAllocFromParent(ly *Layout) {
	mvp := ly.ViewportSafe()
	if ly.Par == nil || mvp == nil || !ly.LayState.Alloc.Size.IsNil() {
		return
	}
	if ly.Par != mvp.This() {
		// note: zero alloc size happens all the time with non-visible tabs!
		// fmt.Printf("Layout: %v has zero allocation but is not a direct child of viewport -- this is an error -- every level must provide layout for the next! laydata:\n%+v\n", ly.Path(), ly.LayState)
		return
	}
	pni, _ := KiToNode2D(ly.Par)
	lyp := pni.AsLayout2D()
	if lyp == nil {
		ly.FuncUpParent(0, ly.This(), func(k ki.Ki, level int, d interface{}) bool {
			pni, _ := KiToNode2D(k)
			if pni == nil {
				return ki.Break
			}
			pg := pni.AsWidget()
			if pg == nil {
				return ki.Break
			}
			if !pg.LayState.Alloc.Size.IsNil() {
				ly.LayState.Alloc.Size = pg.LayState.Alloc.Size
				if Layout2DTrace {
					fmt.Printf("Layout: %v got parent alloc: %v from %v\n", ly.Path(), ly.LayState.Alloc.Size, pg.Path())
				}
				return ki.Break
			}
			return ki.Continue
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//     Layout children

// LayoutSharedDim implements calculations to layout for the shared dimension
// (i.e., Vertical for Horizontal layout). Returns pos and size.
func LayoutSharedDimImpl(ly *Layout, avail, need, pref, max, spc float32, al gist.Align) (pos, size float32) {
	usePref := true
	targ := pref
	extra := avail - targ
	if extra < -0.1 { // not fitting in pref, go with min
		usePref = false
		targ = need
		extra = avail - targ
	}
	extra = mat32.Max(extra, 0.0) // no negatives

	stretchNeed := false // stretch relative to need
	stretchMax := false  // only stretch Max = neg

	if usePref && extra > 0.0 { // have some stretch extra
		if max < 0.0 {
			stretchMax = true // only stretch those marked as infinitely stretchy
		}
	} else if extra > 0.0 { // extra relative to Need
		stretchNeed = true // stretch relative to need
	}

	pos = spc
	size = need
	if usePref {
		size = pref
	}
	if stretchMax || stretchNeed {
		size += extra
	} else {
		if gist.IsAlignMiddle(al) {
			pos += 0.5 * extra
		} else if gist.IsAlignEnd(al) {
			pos += extra
		} else if al == gist.AlignJustify { // treat justify as stretch
			size += extra
		}
	}

	// if Layout2DTrace {
	// 	fmt.Printf("ly %v avail: %v targ: %v, extra %v, strMax: %v, strNeed: %v, pos: %v size: %v spc: %v\n", ly.Nm, avail, targ, extra, stretchMax, stretchNeed, pos, size, spc)
	// }

	return
}

// LayoutSharedDim lays out items along a shared dimension, where all elements
// share the same space, e.g., Horiz for a Vert layout, and vice-versa.
func LayoutSharedDim(ly *Layout, dim mat32.Dims) {
	spc := ly.BoxSpace()
	avail := ly.LayState.Alloc.Size.Dim(dim) - 2.0*spc
	for i, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		if ly.Lay == LayoutStacked && ly.StackTopOnly && i != ly.StackTop {
			continue
		}
		ni.StyMu.RLock()
		al := ni.Sty.Layout.AlignDim(dim)
		ni.StyMu.RUnlock()
		pref := ni.LayState.Size.Pref.Dim(dim)
		need := ni.LayState.Size.Need.Dim(dim)
		max := ni.LayState.Size.Max.Dim(dim)
		pos, size := LayoutSharedDimImpl(ly, avail, need, pref, max, spc, al)
		ni.LayState.Alloc.Size.SetDim(dim, size)
		ni.LayState.Alloc.PosRel.SetDim(dim, pos)
	}
}

// LayoutAlongDim lays out all children along given dim -- only affects that dim --
// e.g., use LayoutSharedDim for other dim.
func LayoutAlongDim(ly *Layout, dim mat32.Dims) {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	elspc := float32(sz-1) * ly.Spacing.Dots
	al := ly.Sty.Layout.AlignDim(dim)
	spc := ly.BoxSpace()
	exspc := 2.0*spc + elspc
	avail := ly.LayState.Alloc.Size.Dim(dim) - exspc
	pref := ly.LayState.Size.Pref.Dim(dim) - exspc
	need := ly.LayState.Size.Need.Dim(dim) - exspc

	targ := pref
	usePref := true
	extra := avail - targ
	if extra < -0.1 { // not fitting in pref, go with need
		usePref = false
		targ = need
		extra = avail - targ
	}
	extra = mat32.Max(extra, 0.0) // no negatives

	nstretch := 0
	stretchTot := float32(0.0)
	stretchNeed := false        // stretch relative to need
	stretchMax := false         // only stretch Max = neg
	addSpace := false           // apply extra toward spacing -- for justify
	if usePref && extra > 0.0 { // have some stretch extra
		for _, c := range ly.Kids {
			if c == nil {
				continue
			}
			ni := c.(Node2D).AsWidget()
			if ni == nil {
				continue
			}
			if ni.LayState.Size.HasMaxStretch(dim) { // negative = stretch
				nstretch++
				stretchTot += ni.LayState.Size.Pref.Dim(dim)
			}
		}
		if nstretch > 0 {
			stretchMax = true // only stretch those marked as infinitely stretchy
		}
	} else if extra > 0.0 { // extra relative to Need
		for _, c := range ly.Kids {
			if c == nil {
				continue
			}
			ni := c.(Node2D).AsWidget()
			if ni == nil {
				continue
			}
			if ni.LayState.Size.HasMaxStretch(dim) || ni.LayState.Size.CanStretchNeed(dim) {
				nstretch++
				stretchTot += ni.LayState.Size.Pref.Dim(dim)
			}
		}
		if nstretch > 0 {
			stretchNeed = true // stretch relative to need
		}
	}

	extraSpace := float32(0.0)
	if sz > 1 && extra > 0.0 && al == gist.AlignJustify && !stretchNeed && !stretchMax {
		addSpace = true
		// if neither, then just distribute as spacing for justify
		extraSpace = extra / float32(sz-1)
	}

	// now arrange everyone
	pos := spc

	// todo: need a direction setting too
	if gist.IsAlignEnd(al) && !stretchNeed && !stretchMax {
		pos += extra
	}

	if Layout2DTrace {
		fmt.Printf("Layout: %v Along dim %v, avail: %v elspc: %v need: %v pref: %v targ: %v, extra %v, strMax: %v, strNeed: %v, nstr %v, strTot %v\n", ly.Path(), dim, avail, elspc, need, pref, targ, extra, stretchMax, stretchNeed, nstretch, stretchTot)
	}

	for i, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		size := ni.LayState.Size.Need.Dim(dim)
		if usePref {
			size = ni.LayState.Size.Pref.Dim(dim)
		}
		if stretchMax { // negative = stretch
			if ni.LayState.Size.HasMaxStretch(dim) { // in proportion to pref
				size += extra * (ni.LayState.Size.Pref.Dim(dim) / stretchTot)
			}
		} else if stretchNeed {
			if ni.LayState.Size.HasMaxStretch(dim) || ni.LayState.Size.CanStretchNeed(dim) {
				size += extra * (ni.LayState.Size.Pref.Dim(dim) / stretchTot)
			}
		} else if addSpace { // implies align justify
			if i > 0 {
				pos += extraSpace
			}
		}

		ni.LayState.Alloc.Size.SetDim(dim, size)
		ni.LayState.Alloc.PosRel.SetDim(dim, pos)
		if Layout2DTrace {
			fmt.Printf("Layout: %v Child: %v, pos: %v, size: %v, need: %v, pref: %v\n", ly.Path(), ni.Nm, pos, size, ni.LayState.Size.Need.Dim(dim), ni.LayState.Size.Pref.Dim(dim))
		}
		pos += size + ly.Spacing.Dots
	}
}

// LayoutFlow manages the flow layout along given dimension
// returns true if needs another iteration (only if iter == 0)
func LayoutFlow(ly *Layout, dim mat32.Dims, iter int) bool {
	ly.FlowBreaks = nil
	sz := len(ly.Kids)
	if sz == 0 {
		return false
	}

	elspc := float32(sz-1) * ly.Spacing.Dots
	spc := ly.BoxSpace()
	exspc := 2.0*spc + elspc

	avail := ly.LayState.Alloc.Size.Dim(dim) - exspc
	odim := mat32.OtherDim(dim)

	pos := spc
	for i, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		size := ni.LayState.Size.Need.Dim(dim)
		if pos+size > avail {
			ly.FlowBreaks = append(ly.FlowBreaks, i)
			pos = spc
		}
		ni.LayState.Alloc.Size.SetDim(dim, size)
		ni.LayState.Alloc.PosRel.SetDim(dim, pos)
		if Layout2DTrace {
			fmt.Printf("Layout: %v Child: %v, pos: %v, size: %v, need: %v, pref: %v\n", ly.Path(), ni.Nm, pos, size, ni.LayState.Size.Need.Dim(dim), ni.LayState.Size.Pref.Dim(dim))
		}
		pos += size + ly.Spacing.Dots
	}
	ly.FlowBreaks = append(ly.FlowBreaks, len(ly.Kids))

	nrows := len(ly.FlowBreaks)
	oavail := ly.LayState.Alloc.Size.Dim(odim) - exspc
	oavPerRow := oavail / float32(nrows)
	ci := 0
	rpos := float32(0)
	var nsz mat32.Vec2
	for _, bi := range ly.FlowBreaks {
		rmax := float32(0)
		for i := ci; i < bi; i++ {
			c := ly.Kids[i]
			if c == nil {
				continue
			}
			ni := c.(Node2D).AsWidget()
			if ni == nil {
				continue
			}
			ni.StyMu.RLock()
			al := ni.Sty.Layout.AlignDim(odim)
			ni.StyMu.RUnlock()
			pref := ni.LayState.Size.Pref.Dim(odim)
			need := ni.LayState.Size.Need.Dim(odim)
			max := ni.LayState.Size.Max.Dim(odim)
			pos, size := LayoutSharedDimImpl(ly, oavPerRow, need, pref, max, spc, al)
			ni.LayState.Alloc.Size.SetDim(odim, size)
			ni.LayState.Alloc.PosRel.SetDim(odim, rpos+pos)
			rmax = mat32.Max(rmax, size)
			nsz.X = mat32.Max(nsz.X, ni.LayState.Alloc.PosRel.X+ni.LayState.Alloc.Size.X)
			nsz.Y = mat32.Max(nsz.Y, ni.LayState.Alloc.PosRel.Y+ni.LayState.Alloc.Size.Y)
		}
		rpos += rmax + ly.Spacing.Dots
		ci = bi
	}
	ly.LayState.Size.Need = nsz
	ly.LayState.Size.Pref = nsz
	if Layout2DTrace {
		fmt.Printf("Layout: %v Flow final size: %v\n", ly.Path(), nsz)
	}
	// if nrows == 1 {
	// 	return false
	// }
	return true
}

// LayoutGridDim lays out grid data along each dimension (row, Y; col, X),
// same as LayoutAlongDim.  For cols, X has width prefs of each -- turn that
// into an actual allocated width for each column, and likewise for rows.
func LayoutGridDim(ly *Layout, rowcol RowCol, dim mat32.Dims) {
	gds := ly.GridData[rowcol]
	sz := len(gds)
	if sz == 0 {
		return
	}
	elspc := float32(sz-1) * ly.Spacing.Dots
	al := ly.Sty.Layout.AlignDim(dim)
	spc := ly.BoxSpace()
	exspc := 2.0*spc + elspc
	avail := ly.LayState.Alloc.Size.Dim(dim) - exspc
	pref := ly.LayState.Size.Pref.Dim(dim) - exspc
	need := ly.LayState.Size.Need.Dim(dim) - exspc

	targ := pref
	usePref := true
	extra := avail - targ
	if extra < -0.1 { // not fitting in pref, go with need
		usePref = false
		targ = need
		extra = avail - targ
	}
	extra = mat32.Max(extra, 0.0) // no negatives

	nstretch := 0
	stretchTot := float32(0.0)
	stretchNeed := false        // stretch relative to need
	stretchMax := false         // only stretch Max = neg
	addSpace := false           // apply extra toward spacing -- for justify
	if usePref && extra > 0.0 { // have some stretch extra
		for _, gd := range gds {
			if gd.SizeMax < 0 { // stretch
				nstretch++
				stretchTot += gd.SizePref
			}
		}
		if nstretch > 0 {
			stretchMax = true // only stretch those marked as infinitely stretchy
		}
	} else if extra > 0.0 { // extra relative to Need
		for _, gd := range gds {
			if gd.SizeMax < 0 || gd.SizePref > gd.SizeNeed {
				nstretch++
				stretchTot += gd.SizePref
			}
		}
		if nstretch > 0 {
			stretchNeed = true // stretch relative to need
		}
	}

	extraSpace := float32(0.0)
	if sz > 1 && extra > 0.0 && al == gist.AlignJustify && !stretchNeed && !stretchMax {
		addSpace = true
		// if neither, then just distribute as spacing for justify
		extraSpace = extra / float32(sz-1)
	}

	// now arrange everyone
	pos := spc

	// todo: need a direction setting too
	if gist.IsAlignEnd(al) && !stretchNeed && !stretchMax {
		pos += extra
	}

	if Layout2DTrace {
		fmt.Printf("Layout Grid Dim: %v All on dim %v, avail: %v need: %v pref: %v targ: %v, extra %v, strMax: %v, strNeed: %v, nstr %v, strTot %v\n", ly.Path(), dim, avail, need, pref, targ, extra, stretchMax, stretchNeed, nstretch, stretchTot)
	}

	for i := range gds {
		gd := &gds[i]
		size := gd.SizeNeed
		if usePref {
			size = gd.SizePref
		}
		if stretchMax { // negative = stretch
			if gd.SizeMax < 0 { // in proportion to pref
				size += extra * (gd.SizePref / stretchTot)
			}
		} else if stretchNeed {
			if gd.SizeMax < 0 || gd.SizePref > gd.SizeNeed {
				size += extra * (gd.SizePref / stretchTot)
			}
		} else if addSpace { // implies align justify
			if i > 0 {
				pos += extraSpace
			}
		}

		gd.AllocSize = size
		gd.AllocPosRel = pos
		if Layout2DTrace {
			fmt.Printf("Grid %v pos: %v, size: %v\n", rowcol, pos, size)
		}
		pos += size + ly.Spacing.Dots
	}
}

// LayoutGridLay manages overall grid layout of children
func LayoutGridLay(ly *Layout) {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	LayoutGridDim(ly, Row, mat32.Y)
	LayoutGridDim(ly, Col, mat32.X)

	col := 0
	row := 0
	cols := ly.GridSize.X
	rows := ly.GridSize.Y

	if cols*rows != ly.NumChildren() {
		GatherSizesGrid(ly)
	}

	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}

		ni.StyMu.RLock()
		lst := ni.Sty.Layout
		ni.StyMu.RUnlock()
		if lst.Col > 0 {
			col = lst.Col
		}
		if lst.Row > 0 {
			row = lst.Row
		}

		{ // col, X dim
			dim := mat32.X
			gd := ly.GridData[Col][col]
			avail := gd.AllocSize
			al := lst.AlignDim(dim)
			pref := ni.LayState.Size.Pref.Dim(dim)
			need := ni.LayState.Size.Need.Dim(dim)
			max := ni.LayState.Size.Max.Dim(dim)
			pos, size := LayoutSharedDimImpl(ly, avail, need, pref, max, 0, al)
			ni.LayState.Alloc.Size.SetDim(dim, size)
			ni.LayState.Alloc.PosRel.SetDim(dim, pos+gd.AllocPosRel)

		}
		{ // row, Y dim
			dim := mat32.Y
			gd := ly.GridData[Row][row]
			avail := gd.AllocSize
			al := lst.AlignDim(dim)
			pref := ni.LayState.Size.Pref.Dim(dim)
			need := ni.LayState.Size.Need.Dim(dim)
			max := ni.LayState.Size.Max.Dim(dim)
			pos, size := LayoutSharedDimImpl(ly, avail, need, pref, max, 0, al)
			ni.LayState.Alloc.Size.SetDim(dim, size)
			ni.LayState.Alloc.PosRel.SetDim(dim, pos+gd.AllocPosRel)
		}

		if Layout2DTrace {
			fmt.Printf("Layout: %v grid col: %v row: %v pos: %v size: %v\n", ly.Path(), col, row, ni.LayState.Alloc.PosRel, ni.LayState.Alloc.Size)
		}

		col++
		if col >= cols { // todo: really only works if NO items specify row,col or ALL do..
			col = 0
			row++
			if row >= rows { // wrap-around.. no other good option
				row = 0
			}
		}
	}
}

// FinalizeLayout is final pass through children to finalize the layout,
// computing summary size stats
func (ly *Layout) FinalizeLayout() {
	ly.ChildSize = mat32.Vec2Zero
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Node2D).AsWidget()
		if ni == nil {
			continue
		}
		ly.ChildSize.SetMax(ni.LayState.Alloc.PosRel.Add(ni.LayState.Alloc.Size))
		ni.LayState.Alloc.SizeOrig = ni.LayState.Alloc.Size
	}
}
