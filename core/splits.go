// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"log/slog"
	"strconv"
	"strings"

	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
)

// SplitsTiles specifies 2D tiles for organizing elements within the [Splits] Widget.
// The [styles.Style.Direction] defines the main axis, and the cross axis is orthogonal
// to that, which is organized into chunks of 2 cross-axis "rows".  In the case of a
// 1D pattern, only Span is relevant, indicating a single element per split.
type SplitsTiles int32 //enums:enum -trim-prefix Tile

const (
	// Span has a single element spanning the cross dimension, i.e.,
	// a vertical span for a horizontal main axis, or a horizontal
	// span for a vertical main axis.  It is the only valid value
	// for 1D Splits, where it specifies a single element per split.
	// If all tiles are Span, then a 1D line is generated.
	TileSpan SplitsTiles = iota

	// Split has a split between elements along the cross dimension,
	// with the first of 2 elements in the first main axis line and
	// the second in the second line.
	TileSplit

	// FirstLong has a long span of first element along the first
	// main axis line and a split between the next two elements
	// along the second line, with a split between the two lines.
	// Visually, the splits form a T shape for a horizontal main axis.
	TileFirstLong

	// SecondLong has the first two elements split along the first line,
	//	and the third with a long span along the second main axis line,
	// with a split between the two lines.  Visually, the splits form
	// an inverted T shape for a horizontal main axis.
	TileSecondLong
)

var (
	// tileNumElements is the number of elements per tile.
	// the number of splitter handles is n-1.
	tileNumElements = map[SplitsTiles]int{TileSpan: 1, TileSplit: 2, TileFirstLong: 3, TileSecondLong: 3}

	// tileNumSubSplits is the number of SubSplits proportions per tile.
	// The Long cases require 2 pairs, first for the split along the cross axis
	// and second for the split along the main axis.
	tileNumSubSplits = map[SplitsTiles]int{TileSpan: 1, TileSplit: 2, TileFirstLong: 4, TileSecondLong: 4}
)

// Splits allocates a certain proportion of its space to each of its children,
// organized along [styles.Style.Direction] as the main axis, and supporting
// [SplitsTiles] of 2D splits configurations along the cross axis.
// There is always a split between each Tile segment along the main axis,
// with the proportion of the total main axis space per Tile allocated
// according to normalized Splits factors.
// If all Tiles are Span then a 1D line is generated.  Children are allocated
// in order along the main axis, according to each of the Tiles,
// which consume 1, 2 or 3 elements, and have 0, 1 or 2 splits internally.
// The internal split proportion are stored separately in SubSplits.
// A [Handle] widget is added to the Parts for each split, allowing the user
// to drag the relative size of each splits region.
// If more complex geometries are required, use nested Splits.
type Splits struct {
	Frame

	// Tiles specifies the 2D layout of elements along the [styles.Style.Direction]
	// main axis and the orthogonal cross axis.  If all Tiles are TileSpan, then
	// a 1D line is generated.  There is always a split between each Tile segment,
	// and different tiles consume different numbers of elements in order, and
	// have different numbers of SubSplits.  Because each Tile can represent a
	// different number of elements, care must be taken to ensure that the full
	// set of tiles corresponds to the actual number of children.  A default
	// 1D configuration will be imposed if there is a mismatch.
	Tiles []SplitsTiles

	// Splits is the proportion (0-1 normalized, enforced) of space
	// allocated to each Tile element along the main axis.
	// 0 indicates that an element should  be completely collapsed.
	// By default, each element gets the same amount of space.
	Splits []float32

	// SubSplits contains splits proportions for each Tile element, with
	// a variable number depending on the Tile.  For the First and Second Long
	// elements, there are 2 subsets of sub-splits, with 4 total subsplits.
	SubSplits [][]float32

	// savedSplits is a saved version of the Splits that can be restored
	// for dynamic collapse/expand operations.
	savedSplits []float32

	// savedSubSplits is a saved version of the SubSplits that can be restored
	// for dynamic collapse/expand operations.
	savedSubSplits [][]float32
}

func (sl *Splits) Init() {
	sl.Frame.Init()
	sl.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Margin.Zero()
		s.Padding.Zero()
		s.Min.Y.Em(10)

		if sl.SizeClass() == SizeCompact {
			s.Direction = styles.Column
		} else {
			s.Direction = styles.Row
		}
	})
	sl.SetOnChildAdded(func(n tree.Node) {
		if n != sl.Parts {
			_, wb := AsWidget(n)
			wb.Styler(func(s *styles.Style) {
				// splits elements must scroll independently and grow
				s.Overflow.Set(styles.OverflowAuto)
				s.Grow.Set(1, 1)
			})
		}
	})

	sl.OnKeyChord(func(e events.Event) {
		kc := string(e.KeyChord())
		mod := "Control+"
		if TheApp.Platform() == system.MacOS {
			mod = "Meta+"
		}
		if !strings.HasPrefix(kc, mod) {
			return
		}
		kns := kc[len(mod):]

		knc, err := strconv.Atoi(kns)
		if err != nil {
			return
		}
		kn := int(knc)
		if kn == 0 {
			e.SetHandled()
			sl.evenSplits(sl.Splits)
			sl.NeedsLayout()

		} else if kn <= len(sl.Children) {
			e.SetHandled()
			if sl.Splits[kn-1] <= 0.01 {
				sl.restoreChild(kn - 1)
			} else {
				sl.collapseSplit(true, kn-1)
			}
		}
	})

	sl.Updater(func() {
		sl.updateSplits()
	})
	parts := sl.newParts()
	parts.Maker(func(p *tree.Plan) {
		// handles are organized first between tiles, then within tiles.
		addHand := func(hidx int, hdir styles.Directions) {
			tree.AddAt(p, "handle-"+strconv.Itoa(hidx), func(w *Handle) {
				w.OnChange(func(e events.Event) {
					sl.setSplitPos(w.IndexInParent(), w.Value())
				})
				w.Styler(func(s *styles.Style) {
					dir := sl.Styles.Direction
					odir := dir.Other()
					if hdir == styles.Column {
						s.Direction = dir
					} else {
						s.Direction = odir
					}
				})
			})
		}

		nt := len(sl.Tiles)
		for i := range nt - 1 {
			addHand(i, styles.Column)
		}
		hi := nt - 1
		for _, t := range sl.Tiles {
			switch t {
			case TileSpan:
			case TileSplit:
				addHand(hi, styles.Row)
				hi++
			case TileFirstLong, TileSecondLong:
				addHand(hi, styles.Row)      // long
				addHand(hi+1, styles.Column) // sub
				hi += 2
			}
		}
	})
}

// tilesTotal returns the total number of child elements associated
// with the current set of Tiles elements, and whether there are any
// non-TileSpan elements, which has implications for error handling
// if the total does not match the actual number of children in the Splits.
func (sl *Splits) tilesTotal() (total int, hasNonSpans bool) {
	for _, t := range sl.Tiles {
		total += tileNumElements[t]
		if t != TileSpan {
			hasNonSpans = true
		}
	}
	return
}

// updateSplits ensures the Tiles, Splits and SubSplits are all configured properly,
// given the number of children in the splits.
func (sl *Splits) updateSplits() *Splits {
	nc := len(sl.Children)
	if nc == 0 {
		return sl
	}
	ntc, hasNonSpans := sl.tilesTotal()
	if ntc != nc {
		if ntc != 0 && hasNonSpans {
			slog.Error("core.Splits: number of children for current Tiles != number of actual children, reverting to 1D", "children", nc, "tiles", ntc)
		}
		sl.Tiles = slicesx.SetLength(sl.Tiles, nc)
		for i := range nc {
			sl.Tiles[i] = TileSpan
		}
	}
	nt := len(sl.Tiles)
	sl.Splits = slicesx.SetLength(sl.Splits, nt)
	sl.normSplits(sl.Splits)
	sl.SubSplits = slicesx.SetLength(sl.SubSplits, nt)
	for i, t := range sl.Tiles {
		ssn := tileNumSubSplits[t]
		ss := sl.SubSplits[i]
		ss = slicesx.SetLength(ss, ssn)
		switch t {
		case TileSpan:
			ss[0] = 1
		case TileSplit:
			sl.normSplits(ss)
		case TileFirstLong, TileSecondLong:
			sl.normSplits(ss[:2]) // first is cross-axis
			sl.normSplits(ss[2:])
		}
		sl.SubSplits[i] = ss
	}
	return sl
}

// normSplits normalizes the given splits proportions,
// using evenSplits if all zero
func (sl *Splits) normSplits(s []float32) {
	sum := float32(0)
	for _, sp := range s {
		sum += sp
	}
	if sum == 0 { // set default even splits
		sl.evenSplits(s)
	} else {
		norm := 1 / sum
		for i := range sl.Splits {
			sl.Splits[i] *= norm
		}
	}
}

// evenSplits splits space evenly across all elements
func (sl *Splits) evenSplits(s []float32) {
	n := len(s)
	if n == 0 {
		return
	}
	even := 1.0 / float32(n)
	for i := range s {
		s[i] = even
	}
}

// saveSplits saves the current set of splits in SavedSplits, for a later RestoreSplits
func (sl *Splits) saveSplits() {
	n := len(sl.Splits)
	if n == 0 {
		return
	}
	sl.savedSplits = slicesx.SetLength(sl.savedSplits, n)
	copy(sl.savedSplits, sl.Splits)
	sl.savedSubSplits = slicesx.SetLength(sl.savedSubSplits, n)
	for i, ss := range sl.SubSplits {
		sv := sl.savedSubSplits[i]
		sv = slicesx.SetLength(sv, len(ss))
		copy(sv, ss)
		sl.savedSubSplits[i] = sv
	}
}

// restoreSplits restores a previously saved set of splits (if it exists), does an update
func (sl *Splits) restoreSplits() {
	if len(sl.savedSplits) != len(sl.Splits) {
		return
	}
	sl.SetSplits(sl.savedSplits...)
	for i, ss := range sl.SubSplits {
		sv := sl.savedSubSplits[i]
		if len(sv) == len(ss) {
			copy(ss, sv)
		}
	}
	sl.NeedsLayout()
}

// setSplit sets given proportional "Splits" space to given value.
// Splits are indexed first by Tiles (major splits) and then
// within tiles, where TileSplit has 2 and the Long cases,
// have the long element first followed by the two smaller ones.
// Calls updateSplits after to ensure renormalization and
// NeedsLayout to ensure layout is updated.
func (sl *Splits) setSplit(idx int, val float32) {
	nt := len(sl.Tiles)
	if nt == 0 {
		return
	}
	if idx < nt {
		sl.Splits[idx] = val
		return
	}
	ci := nt
	for i, t := range sl.Tiles {
		tn := tileNumElements[t]
		ri := idx - ci
		if ri < 0 {
			break
		}
		switch t {
		case TileSpan:
		case TileSplit:
			if ri < 2 {
				sl.SubSplits[i][ri] = val
			}
		case TileFirstLong, TileSecondLong:
			if ri < 3 {
				sl.SubSplits[i][ri] = val
			}
		}
		ci += tn
	}
	sl.updateSplits()
	sl.NeedsLayout()
}

// collapseSplit collapses the splitter region(s) at given index(es),
// by setting splits value to 0.
// optionally saving the prior splits for later Restore function.
func (sl *Splits) collapseSplit(save bool, idxs ...int) {
	if save {
		sl.saveSplits()
	}
	for _, idx := range idxs {
		sl.setSplit(idx, 0)
	}
}

// setSplitPos sets given splitter position to given 0-1 normalized value.
// Calls updateSplits after to ensure renormalization and
// NeedsLayout to ensure layout is updated.
func (sl *Splits) setSplitPos(idx int, val float32) {
	ci := 0
	for i, t := range sl.Tiles {
		tn := tileNumElements[t]
		if idx < ci || idx >= ci+tn {
			ci += tn
			continue
		}
		ri := idx - ci
		switch t {
		case TileSpan:
			sl.Splits[idx] = val
		case TileSplit:
			sl.SubSplits[i][ri] = val
		case TileFirstLong:
			if ri == 0 {
				sl.SubSplits[i][0] = val
			} else {
				sl.SubSplits[i][2+ri] = val
			}
		case TileSecondLong:
			if ri == 2 {
				sl.SubSplits[i][0] = val
			} else {
				sl.SubSplits[i][ri] = val
			}
		}
		ci += tn
	}
	sl.updateSplits()
	sl.NeedsLayout()
}

// restoreChild restores given child(ren)
// todo: not clear if this makes sense anymore
func (sl *Splits) restoreChild(idxs ...int) {
	n := len(sl.Children)
	for _, idx := range idxs {
		if idx >= 0 && idx < n {
			sl.Splits[idx] = 1.0 / float32(n)
		}
	}
	sl.updateSplits()
	sl.NeedsLayout()
}

// splitValue returns the split proportion for given child index
func (sl *Splits) splitValue(idx int) float32 {
	ci := 0
	for i, t := range sl.Tiles {
		tn := tileNumElements[t]
		if idx < ci || idx >= ci+tn {
			ci += tn
			continue
		}
		ri := idx - ci
		switch t {
		case TileSpan:
			return sl.Splits[ri]
		case TileSplit:
			return sl.SubSplits[i][ri]
		case TileFirstLong:
			if ri == 0 {
				return sl.SubSplits[i][0]
			} else {
				return sl.SubSplits[i][2+ri-1]
			}
		case TileSecondLong:
			if ri == 2 {
				return sl.SubSplits[i][0]
			} else {
				return sl.SubSplits[i][2+ri-1]
			}
		}
		ci += tn
	}
	return 0
}

// isCollapsed returns true if the split proportion
// for given child index is 0.  Also checks the overall tile
// splits for the child.
func (sl *Splits) isCollapsed(idx int) bool {
	if sl.splitValue(idx) < 0.01 {
		return true
	}
	ci := 0
	for i, t := range sl.Tiles {
		tn := tileNumElements[t]
		if idx < ci || idx >= ci+tn {
			ci += tn
			continue
		}
		ri := idx - ci
		if sl.Splits[i] < 0.01 {
			return true
		}
		switch t {
		case TileFirstLong:
			if ri > 0 && sl.SubSplits[i][1] < 0.01 {
				return true
			}
		case TileSecondLong:
			if ri < 2 && sl.SubSplits[i][1] < 0.01 {
				return true
			}
		}
	}
	return false
}

// setSplit sets the new splitter value, for given splitter.
// New value is 0..1 value of position of that splitter.
// It is a sum of all the positions up to that point.
// Splitters are updated to ensure that selected position is achieved,
// while dividing remainder appropriately.
func (sl *Splits) setSplitOld(idx int, nwval float32) {
	n := len(sl.Splits)
	oldsum := float32(0)
	for i := 0; i <= idx; i++ {
		oldsum += sl.Splits[i]
	}
	delta := nwval - oldsum
	oldval := sl.Splits[idx]
	uval := oldval + delta
	if uval < 0 {
		uval = 0
		delta = -oldval
		nwval = oldsum + delta
	}
	rmdr := 1 - nwval
	if idx < n-1 {
		oldrmdr := 1 - oldsum
		if oldrmdr <= 0 {
			if rmdr > 0 {
				dper := rmdr / float32((n-1)-idx)
				for i := idx + 1; i < n; i++ {
					sl.Splits[i] = dper
				}
			}
		} else {
			for i := idx + 1; i < n; i++ {
				curval := sl.Splits[i]
				sl.Splits[i] = rmdr * (curval / oldrmdr) // proportional
			}
		}
	}
	sl.Splits[idx] = uval
	sl.updateSplits()
	sl.NeedsLayout()
}

func (sl *Splits) SizeDownSetAllocs(iter int) {
	if sl.NumChildren() <= 1 {
		return
	}
	sl.updateSplits()
	sz := &sl.Geom.Size
	// note: InnerSpace is computed based on n children -- not accurate!
	csz := sz.Alloc.Content
	dim := sl.Styles.Direction.Dim()
	odim := dim.Other()
	cszd := csz.Dim(dim)
	cszo := csz.Dim(odim)
	gap := sl.Styles.Gap.Dots().Floor()
	gapd := gap.Dim(dim)
	gapo := gap.Dim(odim)
	hand := sl.Parts.Child(0).(*Handle)
	hwd := hand.Geom.Size.Actual.Total.Dim(dim)
	cszd -= float32(len(sl.Splits)-1) * (hwd + gapd)

	setCsz := func(idx int, szm, szc float32) {
		_, kwb := AsWidget(sl.Child(idx))
		ksz := &kwb.Geom.Size
		ksz.Alloc.Total.SetDim(dim, szm)
		ksz.Alloc.Total.SetDim(odim, szc)
		ksz.setContentFromTotal(&ksz.Alloc)
	}

	ci := 0
	for i, t := range sl.Tiles {
		szt := math32.Round(sl.Splits[i] * cszd) // tile size, main axis
		szcs := cszo - hwd - gapo                // cross axis spilt
		szs := szt - hwd - gapd
		tn := tileNumElements[t]
		switch t {
		case TileSpan:
			setCsz(ci, szt, cszo)
		case TileSplit:
			setCsz(ci, szt, math32.Round(szcs*sl.SubSplits[i][0]))
			setCsz(ci+1, szt, math32.Round(szcs*sl.SubSplits[i][1]))
		case TileFirstLong:
			fcht := math32.Round(szcs * sl.SubSplits[i][0])
			scht := math32.Round(szcs * sl.SubSplits[i][1])
			setCsz(ci, szt, fcht)
			setCsz(ci+1, math32.Round(szs*sl.SubSplits[i][2]), scht)
			setCsz(ci+2, math32.Round(szs*sl.SubSplits[i][3]), scht)
		case TileSecondLong:
			fcht := math32.Round(szcs * sl.SubSplits[i][0])
			scht := math32.Round(szcs * sl.SubSplits[i][1])
			setCsz(ci+2, szt, fcht)
			setCsz(ci, math32.Round(szs*sl.SubSplits[i][2]), scht)
			setCsz(ci+1, math32.Round(szs*sl.SubSplits[i][3]), scht)
		}
		ci += tn
	}
}

func (sl *Splits) positionSplits() {
	if sl.NumChildren() <= 1 {
		return
	}
	if sl.Parts != nil {
		sl.Parts.Geom.Size = sl.Geom.Size // inherit: allows bbox to include handle
	}
	sz := &sl.Geom.Size
	dim := sl.Styles.Direction.Dim()
	odim := dim.Other()
	csz := sz.Alloc.Content
	cszd := csz.Dim(dim)
	cszo := csz.Dim(odim)
	gap := sl.Styles.Gap.Dots().Floor()
	gapd := gap.Dim(dim)
	gapo := gap.Dim(odim)

	hand := sl.Parts.Child(0).(*Handle)
	hwd := hand.Geom.Size.Actual.Total.Dim(dim)
	hht := hand.Geom.Size.Actual.Total.Dim(odim)
	mid := (csz.Dim(odim) - hht) / 2
	cszd -= float32(len(sl.Splits)-1) * (hwd + gapd)
	hwdg := hwd + 0.5*gapd

	setChildPos := func(idx int, dpos, opos float32) {
		_, kwb := AsWidget(sl.Child(idx))
		kwb.Geom.RelPos.SetDim(dim, dpos)
		kwb.Geom.RelPos.SetDim(odim, opos)
	}
	setHandlePos := func(idx int, dpos, opos, max, lpos float32) {
		hl := sl.Parts.Child(idx).(*Handle)
		hl.Geom.RelPos.SetDim(dim, dpos)
		hl.Geom.RelPos.SetDim(odim, opos)
		hl.Min = 0
		hl.Max = max
		hl.Pos = lpos
	}

	tpos := float32(0) // tile position
	ci := 0
	nt := len(sl.Tiles)
	hi := nt - 1 // extra handles

	for i, t := range sl.Tiles {
		szt := math32.Round(sl.Splits[i] * cszd) // tile size, main axis
		szcs := cszo - hwd - gapo                // cross axis spilt
		szs := szt - hwd - gapd
		tn := tileNumElements[t]
		if i > 0 {
			setHandlePos(i-1, tpos-hwdg, mid, cszd, tpos)
		}
		switch t {
		case TileSpan:
			setChildPos(ci, tpos, 0)
		case TileSplit:
			fcht := math32.Round(szcs * sl.SubSplits[i][0])
			setHandlePos(hi, tpos+.5*(szt-hht), fcht+0.5*gapo, cszo, fcht)
			hi++
			setChildPos(ci, tpos, 0)
			setChildPos(ci+1, tpos, fcht+hwd+gapo)
		case TileFirstLong, TileSecondLong:
			fcht := math32.Round(szcs * sl.SubSplits[i][0])
			scht := math32.Round(szcs * sl.SubSplits[i][1])
			swd := math32.Round(szs * sl.SubSplits[i][2])
			bot := fcht + hwd + gapo
			setHandlePos(hi, tpos+.5*(szt-hht), fcht+0.5*gapo, cszo, fcht) // long
			if t == TileFirstLong {
				setHandlePos(hi+1, tpos+swd+0.5*gapd, bot+0.5*(scht-hht), szt, swd)
				setChildPos(ci, tpos, 0)
				setChildPos(ci+1, tpos, bot)
				setChildPos(ci+2, tpos+swd+hwd+gapd, bot)
			} else {
				setHandlePos(hi+1, tpos+swd+0.5*gapd, 0.5*(fcht-hht), szt, swd)
				setChildPos(ci+2, tpos, bot)
				setChildPos(ci, tpos, 0)
				setChildPos(ci+1, tpos+swd+hwd+gapd, 0)
			}
			hi += 2
		}
		ci += tn
		tpos += szt + hwd + gapd
	}
}

func (sl *Splits) Position() {
	if !sl.HasChildren() {
		sl.Frame.Position()
		return
	}
	sl.updateSplits()
	sl.ConfigScrolls()
	sl.positionSplits()
	sl.positionChildren()
}

func (sl *Splits) RenderWidget() {
	if sl.PushBounds() {
		sl.ForWidgetChildren(func(i int, kwi Widget, kwb *WidgetBase) bool {
			kwb.SetState(sl.isCollapsed(i), states.Invisible)
			kwi.RenderWidget()
			return tree.Continue
		})
		sl.renderParts()
		sl.PopBounds()
	}
}
