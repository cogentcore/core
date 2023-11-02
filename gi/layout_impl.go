// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"

	"goki.dev/girl/styles"
	"goki.dev/mat32/v2"
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

// first depth-first GetSize pass: terminal concrete items compute their AllocSize
// we focus on Need: Max(Min, AllocSize), and Want: Max(Pref, AllocSize) -- Max is
// only used if we need to fill space, during final allocation
//
// second me-first DoLayout pass: each layout allocates AllocSize for its
// children based on aggregated size data, and so on down the tree

// GatherSizesSumMax gets basic sum and max data across all kiddos
func GatherSizesSumMax(ly *Layout) (sumPref, sumNeed, maxPref, maxNeed mat32.Vec2) {
	sz := len(ly.Kids)
	if sz == 0 {
		return
	}

	if ly.Lay == LayoutStacked && ly.Is(LayoutStackTopOnly) {
		sn, err := ly.ChildTry(ly.StackTop)
		if err != nil {
			return
		}
		ni := sn.(Widget).AsWidget()
		if ni == nil {
			return
		}
		sumNeed = sumNeed.Add(ni.LayState.Size.Need)
		sumPref = sumPref.Add(ni.LayState.Size.Pref)
		maxNeed = maxNeed.Max(ni.LayState.Size.Need)
		maxPref = maxPref.Max(ni.LayState.Size.Pref)
		return
	}

	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		wi, ok := c.(Widget)
		if !ok {
			fmt.Printf("c is %+v\n", c)
			continue
		}
		wb := wi.AsWidget()
		if wb == nil {
			continue
		}
		wb.LayState.UpdateSizes()
		sumNeed = sumNeed.Add(wb.LayState.Size.Need)
		sumPref = sumPref.Add(wb.LayState.Size.Pref)
		maxNeed = maxNeed.Max(wb.LayState.Size.Need)
		maxPref = maxPref.Max(wb.LayState.Size.Pref)

		if LayoutTrace {
			fmt.Printf("Size:   %v Child: %v, need: %v, pref: %v\n", ly.Path(), wb.Nm, wb.LayState.Size.Need.Dim(LaySummedDim(ly.Lay)), wb.LayState.Size.Pref.Dim(LaySummedDim(ly.Lay)))
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
	sc := ly.Sc
	if sc != nil && sc.Is(ScPrefSizing) {
		// fmt.Println("pref!")
		prefSizing = ly.Styles.Overflow == styles.OverflowScroll // special case
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
			if LayoutTrace {
				fmt.Printf("Size:   %v pref nonzero, setting as need: %v\n", ly.Path(), pref)
			}
			ly.LayState.Size.Need.SetDim(d, pref)
		}
	}

	spc := ly.BoxSpace()
	ly.LayState.Size.Need.SetAdd(spc.Size())
	ly.LayState.Size.Pref.SetAdd(spc.Size())

	elspc := float32(0.0)
	if sz >= 2 {
		elspc = float32(sz-1) * ly.Styles.Spacing.Dots
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
	if LayoutTrace {
		fmt.Printf("Size:   %v gather sizes need: %v, pref: %v, elspc: %v\n", ly.Path(), ly.LayState.Size.Need, ly.LayState.Size.Pref, elspc)
	}
}

// ChildrenUpdateSizes calls UpdateSizes on all children -- layout must at least call this
func (ly *Layout) ChildrenUpdateSizes() {
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Widget).AsWidget()
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
		if LayoutTrace {
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

	if LayoutTrace {
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
	ly.LayState.Size.Need.SetAdd(spc.Size())
	ly.LayState.Size.Pref.SetAdd(spc.Size())

	elspc := float32(0.0)
	if sz >= 2 {
		elspc = float32(sz-1) * ly.Styles.Spacing.Dots
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
	if LayoutTrace {
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

	cols := ly.Styles.Columns
	rows := 0

	sz := len(ly.Kids)
	// collect overall size
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Widget).AsWidget()
		if ni == nil {
			continue
		}
		ni.StyMu.RLock()
		st := &ni.Styles
		ni.StyMu.RUnlock()
		if st.Col > 0 {
			cols = max(cols, st.Col+st.ColSpan)
		}
		if st.Row > 0 {
			rows = max(rows, st.Row+st.RowSpan)
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
		ni := c.(Widget).AsWidget()
		if ni == nil {
			continue
		}
		ni.LayState.UpdateSizes()
		ni.StyMu.RLock()
		st := &ni.Styles
		ni.StyMu.RUnlock()
		if st.Col > 0 {
			col = st.Col
		}
		if st.Row > 0 {
			row = st.Row
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
	sc := ly.Sc
	if sc != nil && sc.Is(ScPrefSizing) {
		prefSizing = ly.Styles.Overflow == styles.OverflowScroll // special case
	}

	// if there aren't existing prefs, we need to compute size
	if prefSizing || ly.LayState.Size.Pref.X == 0 || ly.LayState.Size.Pref.Y == 0 {
		sbw := ly.Styles.ScrollBarWidth.Dots
		maxRow := len(ly.GridData[Row])
		maxCol := len(ly.GridData[Col])
		if prefSizing {
			maxRow = min(LayoutPrefMaxRows, maxRow)
			maxCol = min(LayoutPrefMaxCols, maxCol)
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
	ly.LayState.Size.Need.SetAdd(spc.Size())
	ly.LayState.Size.Pref.SetAdd(spc.Size())

	ly.LayState.Size.Need.X += float32(cols-1) * ly.Styles.Spacing.Dots
	ly.LayState.Size.Pref.X += float32(cols-1) * ly.Styles.Spacing.Dots
	ly.LayState.Size.Need.Y += float32(rows-1) * ly.Styles.Spacing.Dots
	ly.LayState.Size.Pref.Y += float32(rows-1) * ly.Styles.Spacing.Dots

	ly.LayState.UpdateSizes() // enforce max and normal ordering, etc
	if LayoutTrace {
		fmt.Printf("Size:   %v gather sizes grid need: %v, pref: %v\n", ly.Path(), ly.LayState.Size.Need, ly.LayState.Size.Pref)
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//     Layout children

// LayoutSharedDim implements calculations to layout for the shared dimension
// (i.e., Vertical for Horizontal layout). Returns pos and size.
func LayoutSharedDimImpl(ly *Layout, avail, need, pref, max float32, spc styles.SideFloats, al styles.Align) (pos, size float32) {
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

	if ly.Lay == LayoutHoriz || ly.Lay == LayoutHorizFlow {
		pos = spc.Pos().Dim(mat32.Y)
	} else {
		pos = spc.Pos().Dim(mat32.X)
	}

	size = need
	if usePref {
		size = pref
	}
	if stretchMax || stretchNeed {
		size += extra
	} else {
		if styles.IsAlignMiddle(al) {
			pos += 0.5 * extra
		} else if styles.IsAlignEnd(al) {
			pos += extra
		} else if al == styles.AlignJustify { // treat justify as stretch
			size += extra
		}
	}

	// if LayoutTrace {
	// 	fmt.Printf("ly %v avail: %v targ: %v, extra %v, strMax: %v, strNeed: %v, pos: %v size: %v spc: %v\n", ly.Nm, avail, targ, extra, stretchMax, stretchNeed, pos, size, spc)
	// }

	return
}

// LayoutSharedDim lays out items along a shared dimension, where all elements
// share the same space, e.g., Horiz for a Vert layout, and vice-versa.
func LayoutSharedDim(ly *Layout, dim mat32.Dims) {
	spc := ly.BoxSpace()
	avail := ly.LayState.Alloc.Size.Dim(dim) - spc.Size().Dim(dim)
	for i, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Widget).AsWidget()
		if ni == nil {
			continue
		}
		if ly.Lay == LayoutStacked && ly.Is(LayoutStackTopOnly) && i != ly.StackTop {
			continue
		}
		ni.StyMu.RLock()
		al := ni.Styles.AlignDim(dim)
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

	elspc := float32(sz-1) * ly.Styles.Spacing.Dots
	al := ly.Styles.AlignDim(dim)
	spc := ly.BoxSpace()
	exspc := spc.Size().Dim(dim) + elspc
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
			ni := c.(Widget).AsWidget()
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
			ni := c.(Widget).AsWidget()
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
	if sz > 1 && extra > 0.0 && al == styles.AlignJustify && !stretchNeed && !stretchMax {
		addSpace = true
		// if neither, then just distribute as spacing for justify
		extraSpace = extra / float32(sz-1)
	}

	// now arrange everyone
	pos := spc.Pos().Dim(dim)

	// todo: need a direction setting too
	if styles.IsAlignEnd(al) && !stretchNeed && !stretchMax {
		pos += extra
	}

	if LayoutTrace {
		fmt.Printf("Layout: %v Along dim %v, avail: %v elspc: %v need: %v pref: %v targ: %v, extra %v, strMax: %v, strNeed: %v, nstr %v, strTot %v\n", ly.Path(), dim, avail, elspc, need, pref, targ, extra, stretchMax, stretchNeed, nstretch, stretchTot)
	}

	for i, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Widget).AsWidget()
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
		if LayoutTrace {
			fmt.Printf("Layout: %v Child: %v, pos: %v, size: %v, need: %v, pref: %v\n", ly.Path(), ni.Nm, pos, size, ni.LayState.Size.Need.Dim(dim), ni.LayState.Size.Pref.Dim(dim))
		}
		pos += size + ly.Styles.Spacing.Dots
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

	elspc := float32(sz-1) * ly.Styles.Spacing.Dots
	spc := ly.BoxSpace()
	exspc := spc.Size().Dim(dim) + elspc

	avail := ly.LayState.Alloc.Size.Dim(dim) - exspc
	odim := mat32.OtherDim(dim)

	// SidesTODO: might be odim
	pos := spc.Pos().Dim(dim)
	for i, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Widget).AsWidget()
		if ni == nil {
			continue
		}
		size := ni.LayState.Size.Need.Dim(dim)
		if pos+size > avail {
			ly.FlowBreaks = append(ly.FlowBreaks, i)
			pos = spc.Pos().Dim(dim)
		}
		ni.LayState.Alloc.Size.SetDim(dim, size)
		ni.LayState.Alloc.PosRel.SetDim(dim, pos)
		if LayoutTrace {
			fmt.Printf("Layout: %v Child: %v, pos: %v, size: %v, need: %v, pref: %v\n", ly.Path(), ni.Nm, pos, size, ni.LayState.Size.Need.Dim(dim), ni.LayState.Size.Pref.Dim(dim))
		}
		pos += size + ly.Styles.Spacing.Dots
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
			ni := c.(Widget).AsWidget()
			if ni == nil {
				continue
			}
			ni.StyMu.RLock()
			al := ni.Styles.AlignDim(odim)
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
		rpos += rmax + ly.Styles.Spacing.Dots
		ci = bi
	}
	ly.LayState.Size.Need = nsz
	ly.LayState.Size.Pref = nsz
	if LayoutTrace {
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
	elspc := float32(sz-1) * ly.Styles.Spacing.Dots
	al := ly.Styles.AlignDim(dim)
	spc := ly.BoxSpace()
	exspc := spc.Size().Dim(dim) + elspc
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
	if sz > 1 && extra > 0.0 && al == styles.AlignJustify && !stretchNeed && !stretchMax {
		addSpace = true
		// if neither, then just distribute as spacing for justify
		extraSpace = extra / float32(sz-1)
	}

	// now arrange everyone
	pos := spc.Pos().Dim(dim)

	// todo: need a direction setting too
	if styles.IsAlignEnd(al) && !stretchNeed && !stretchMax {
		pos += extra
	}

	if LayoutTrace {
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
		if LayoutTrace {
			fmt.Printf("Grid %v pos: %v, size: %v\n", rowcol, pos, size)
		}
		pos += size + ly.Styles.Spacing.Dots
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
		ni := c.(Widget).AsWidget()
		if ni == nil {
			continue
		}

		ni.StyMu.RLock()
		st := &ni.Styles
		ni.StyMu.RUnlock()
		if st.Col > 0 {
			col = st.Col
		}
		if st.Row > 0 {
			row = st.Row
		}

		{ // col, X dim
			dim := mat32.X
			gd := ly.GridData[Col][col]
			avail := gd.AllocSize
			al := st.AlignDim(dim)
			pref := ni.LayState.Size.Pref.Dim(dim)
			need := ni.LayState.Size.Need.Dim(dim)
			max := ni.LayState.Size.Max.Dim(dim)
			pos, size := LayoutSharedDimImpl(ly, avail, need, pref, max, styles.SideFloats{}, al)
			ni.LayState.Alloc.Size.SetDim(dim, size)
			ni.LayState.Alloc.PosRel.SetDim(dim, pos+gd.AllocPosRel)

		}
		{ // row, Y dim
			dim := mat32.Y
			gd := ly.GridData[Row][row]
			avail := gd.AllocSize
			al := st.AlignDim(dim)
			pref := ni.LayState.Size.Pref.Dim(dim)
			need := ni.LayState.Size.Need.Dim(dim)
			max := ni.LayState.Size.Max.Dim(dim)
			pos, size := LayoutSharedDimImpl(ly, avail, need, pref, max, styles.SideFloats{}, al)
			ni.LayState.Alloc.Size.SetDim(dim, size)
			ni.LayState.Alloc.PosRel.SetDim(dim, pos+gd.AllocPosRel)
		}

		if LayoutTrace {
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
	if ly.Lay == LayoutStacked && ly.Is(LayoutStackTopOnly) {
		sn, err := ly.ChildTry(ly.StackTop)
		if err != nil {
			return
		}
		ni := sn.(Widget).AsWidget()
		if ni == nil {
			return
		}
		ly.ChildSize.SetMax(ni.LayState.Alloc.PosRel.Add(ni.LayState.Alloc.Size))
		ni.LayState.Alloc.SizeOrig = ni.LayState.Alloc.Size
		return
	}
	for _, c := range ly.Kids {
		if c == nil {
			continue
		}
		ni := c.(Widget).AsWidget()
		if ni == nil {
			continue
		}
		ly.ChildSize.SetMax(ni.LayState.Alloc.PosRel.Add(ni.LayState.Alloc.Size))
		ni.LayState.Alloc.SizeOrig = ni.LayState.Alloc.Size
	}
}
