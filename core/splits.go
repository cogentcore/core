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
	// and the third with a long span along the second main axis line,
	// with a split between the two lines.  Visually, the splits form
	// an inverted T shape for a horizontal main axis.
	TileSecondLong

	// Plus is arranged like a plus sign + with the main split along
	// the main axis line, and then individual cross-axis splits
	// between the first two and next two elements.
	TilePlus
)

var (
	// tileNumElements is the number of elements per tile.
	// the number of splitter handles is n-1.
	tileNumElements = map[SplitsTiles]int{TileSpan: 1, TileSplit: 2, TileFirstLong: 3, TileSecondLong: 3, TilePlus: 4}

	// tileNumSubSplits is the number of SubSplits proportions per tile.
	// The Long cases require 2 pairs, first for the split along the cross axis
	// and second for the split along the main axis; Plus requires 3 pairs.
	tileNumSubSplits = map[SplitsTiles]int{TileSpan: 1, TileSplit: 2, TileFirstLong: 4, TileSecondLong: 4, TilePlus: 6}
)

// Splits allocates a certain proportion of its space to each of its children,
// organized along [styles.Style.Direction] as the main axis, and supporting
// [SplitsTiles] of 2D splits configurations along the cross axis.
// There is always a split between each Tile segment along the main axis,
// with the proportion of the total main axis space per Tile allocated
// according to normalized Splits factors.
// If all Tiles are Span then a 1D line is generated.  Children are allocated
// in order along the main axis, according to each of the Tiles,
// which consume 1 to 4 elements, and have 0 to 3 splits internally.
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

	// TileSplits is the proportion (0-1 normalized, enforced) of space
	// allocated to each Tile element along the main axis.
	// 0 indicates that an element should  be completely collapsed.
	// By default, each element gets the same amount of space.
	TileSplits []float32

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

	// handleDirs contains the target directions for each of the handles.
	// this is set by parent split in its style function, and consumed
	// by each handle in its own style function.
	handleDirs []styles.Directions
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
	sl.FinalStyler(func(s *styles.Style) {
		sl.styleSplits()
	})
	sl.SetOnChildAdded(func(n tree.Node) {
		if n != sl.Parts {
			AsWidget(n).Styler(func(s *styles.Style) {
				// splits elements must scroll independently and grow
				s.Overflow.Set(styles.OverflowAuto)
				s.Grow.Set(1, 1)
				s.Direction = styles.Column
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
			sl.evenSplits(sl.TileSplits)
			sl.NeedsLayout()

		} else if kn <= len(sl.Children) {
			e.SetHandled()
			if sl.TileSplits[kn-1] <= 0.01 {
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
		sl.styleSplits()
		addHand := func(hidx int) {
			tree.AddAt(p, "handle-"+strconv.Itoa(hidx), func(w *Handle) {
				w.OnChange(func(e events.Event) {
					sl.setHandlePos(w.IndexInParent(), w.Value())
					sl.SendChange()
				})
				w.Styler(func(s *styles.Style) {
					ix := w.IndexInParent()
					if len(sl.handleDirs) > ix {
						s.Direction = sl.handleDirs[ix]
					}
				})
			})
		}

		nt := len(sl.Tiles)
		for i := range nt - 1 {
			addHand(i)
		}
		hi := nt - 1
		for _, t := range sl.Tiles {
			switch t {
			case TileSpan:
			case TileSplit:
				addHand(hi)
				hi++
			case TileFirstLong, TileSecondLong:
				addHand(hi)     // long
				addHand(hi + 1) // sub
				hi += 2
			case TilePlus:
				addHand(hi)     // long
				addHand(hi + 1) // sub1
				addHand(hi + 2) // sub2
				hi += 3
			}
		}
	})
}

// SetSplits sets the split proportions for the children.
// In general you should pass the same number of args
// as there are children, though fewer could be passed.
func (sl *Splits) SetSplits(splits ...float32) *Splits {
	sl.updateSplits()
	_, hasNonSpans := sl.tilesTotal()
	if !hasNonSpans {
		nc := len(splits)
		sl.TileSplits = slicesx.SetLength(sl.TileSplits, nc)
		copy(sl.TileSplits, splits)
		sl.Tiles = slicesx.SetLength(sl.Tiles, nc)
		for i := range nc {
			sl.Tiles[i] = TileSpan
		}
		sl.updateSplits()
		return sl
	}
	for i, sp := range splits {
		sl.SetSplit(i, sp)
	}
	return sl
}

// SetSplit sets the split proportion of relevant display width
// specific to given child index.  Also updates other split values
// in proportion.
func (sl *Splits) SetSplit(idx int, val float32) {
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
			sl.TileSplits[i] = val
			sl.normOtherSplits(i, sl.TileSplits)
		case TileSplit:
			sl.SubSplits[i][ri] = val
			sl.normOtherSplits(ri, sl.SubSplits[i])
		case TileFirstLong:
			if ri == 0 {
				sl.SubSplits[i][0] = val
				sl.normOtherSplits(0, sl.SubSplits[i][:2])
			} else {
				sl.SubSplits[i][1+ri] = val
				sl.normOtherSplits(ri-1, sl.SubSplits[i][2:])
			}
		case TileSecondLong:
			if ri == 2 {
				sl.SubSplits[i][1] = val
				sl.normOtherSplits(1, sl.SubSplits[i][:2])
			} else {
				sl.SubSplits[i][2+ri] = val
				sl.normOtherSplits(ri, sl.SubSplits[i][2:])
			}
		case TilePlus:
			si := 2 + ri
			gi := (si / 2) * 2
			oi := 1 - (si % 2)
			sl.SubSplits[i][si] = val
			sl.normOtherSplits(oi, sl.SubSplits[i][gi:gi+2])
		}
		ci += tn
	}
}

// Splits returns the split proportion for each child element.
func (sl *Splits) Splits() []float32 {
	nc := len(sl.Children)
	sv := make([]float32, nc)
	for i := range nc {
		sv[i] = sl.Split(i)
	}
	return sv
}

// Split returns the split proportion for given child index
func (sl *Splits) Split(idx int) float32 {
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
			return sl.TileSplits[i]
		case TileSplit:
			return sl.SubSplits[i][ri]
		case TileFirstLong:
			if ri == 0 {
				return sl.SubSplits[i][0]
			}
			return sl.SubSplits[i][1+ri]
		case TileSecondLong:
			if ri == 2 {
				return sl.SubSplits[i][1]
			}
			return sl.SubSplits[i][2+ri]
		case TilePlus:
			si := 2 + ri
			return sl.SubSplits[i][si]
		}
		ci += tn
	}
	return 0
}

// ChildIsCollapsed returns true if the split proportion
// for given child index is 0.  Also checks the overall tile
// splits for the child.
func (sl *Splits) ChildIsCollapsed(idx int) bool {
	if sl.Split(idx) < 0.01 {
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
		if sl.TileSplits[i] < 0.01 {
			return true
		}
		// extra consideration for long split onto subs:
		switch t {
		case TileFirstLong:
			if ri > 0 && sl.SubSplits[i][1] < 0.01 {
				return true
			}
		case TileSecondLong:
			if ri < 2 && sl.SubSplits[i][0] < 0.01 {
				return true
			}
		case TilePlus:
			if ri < 2 {
				return sl.SubSplits[i][0] < 0.01
			}
			return sl.SubSplits[i][1] < 0.01
		}
		return false
	}
	return false
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

// updateSplits ensures the Tiles, TileSplits and SubSplits
// are all configured properly, given the number of children.
func (sl *Splits) updateSplits() *Splits {
	nc := len(sl.Children)
	ntc, hasNonSpans := sl.tilesTotal()
	if nc == 0 && ntc == 0 {
		return sl
	}
	if nc > 0 && ntc != nc {
		if ntc != 0 && hasNonSpans {
			slog.Error("core.Splits: number of children for current Tiles != number of actual children, reverting to 1D", "children", nc, "tiles", ntc)
		}
		sl.Tiles = slicesx.SetLength(sl.Tiles, nc)
		for i := range nc {
			sl.Tiles[i] = TileSpan
		}
	}
	nt := len(sl.Tiles)
	sl.TileSplits = slicesx.SetLength(sl.TileSplits, nt)
	sl.normSplits(sl.TileSplits)
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
		case TilePlus:
			for j := range 3 {
				sl.normSplits(ss[2*j : 2*j+2])
			}
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
		return
	}
	norm := 1 / sum
	for i := range s {
		s[i] *= norm
	}
}

// normOtherSplits normalizes the given splits proportions,
// while keeping the one at the given index at its current value.
func (sl *Splits) normOtherSplits(idx int, s []float32) {
	n := len(s)
	if n == 1 {
		return
	}
	val := s[idx]
	sum := float32(0)
	even := (1 - val) / float32(n-1)
	for i, sp := range s {
		if i != idx {
			if sp == 0 {
				s[i], sp = even, even
			}
			sum += sp
		}
	}
	norm := (1 - val) / sum
	nsum := float32(0)
	for i := range s {
		if i != idx {
			s[i] *= norm
		}
		nsum += s[i]
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
	n := len(sl.TileSplits)
	if n == 0 {
		return
	}
	sl.savedSplits = slicesx.SetLength(sl.savedSplits, n)
	copy(sl.savedSplits, sl.TileSplits)
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
	if len(sl.savedSplits) != len(sl.TileSplits) {
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

// setSplitIndex sets given proportional "Splits" space to given value.
// Splits are indexed first by Tiles (major splits) and then
// within tiles, where TileSplit has 2 and the Long cases,
// have the long element first followed by the two smaller ones.
// Calls updateSplits after to ensure renormalization and
// NeedsLayout to ensure layout is updated.
func (sl *Splits) setSplitIndex(idx int, val float32) {
	nt := len(sl.Tiles)
	if nt == 0 {
		return
	}
	if idx < nt {
		sl.TileSplits[idx] = val
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
			if ri == 0 {
				sl.SubSplits[i][ri] = val
			} else {
				sl.SubSplits[i][2+ri-1] = val
			}
		case TilePlus:
			sl.SubSplits[i][2+ri] = val
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
		sl.setSplitIndex(idx, 0)
	}
}

// setHandlePos sets given splits handle position to given 0-1 normalized value.
// Handles are indexed 0..Tiles-1 for main tiles handles, then sequentially
// for any additional child sub-splits depending on tile config.
// Calls updateSplits after to ensure renormalization and
// NeedsLayout to ensure layout is updated.
func (sl *Splits) setHandlePos(idx int, val float32) {
	val = math32.Clamp(val, 0, 1)

	update := func(idx int, nw float32, s []float32) {
		n := len(s)
		old := s[idx]
		sumTo := float32(0)
		for i := range idx + 1 {
			sumTo += s[i]
		}
		delta := nw - sumTo
		uval := old + delta
		if uval < 0 {
			uval = 0
			delta = -old
			nw = sumTo + delta
		}
		rmdr := 1 - nw
		oldrmdr := 1 - sumTo
		if oldrmdr <= 0 {
			if rmdr > 0 {
				dper := rmdr / float32((n-1)-idx)
				for i := idx + 1; i < n; i++ {
					s[i] = dper
				}
			}
		} else {
			for i := idx + 1; i < n; i++ {
				cur := s[i]
				s[i] = rmdr * (cur / oldrmdr) // proportional
			}
		}
		s[idx] = uval
	}

	nt := len(sl.Tiles)
	if idx < nt-1 {
		update(idx, val, sl.TileSplits)
		sl.updateSplits()
		sl.NeedsLayout()
		return
	}
	ci := nt - 1
	for i, t := range sl.Tiles {
		tn := tileNumElements[t] - 1
		if tn == 0 {
			continue
		}
		if idx < ci || idx >= ci+tn {
			ci += tn
			continue
		}
		ri := idx - ci
		switch t {
		case TileSplit:
			update(0, val, sl.SubSplits[i])
		case TileFirstLong, TileSecondLong:
			if ri == 0 {
				update(0, val, sl.SubSplits[i][:2])
			} else {
				update(0, val, sl.SubSplits[i][2:])
			}
		case TilePlus:
			if ri == 0 {
				update(0, val, sl.SubSplits[i][:2])
			} else {
				gi := ri * 2
				update(0, val, sl.SubSplits[i][gi:gi+2])
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
			sl.TileSplits[idx] = 1.0 / float32(n)
		}
	}
	sl.updateSplits()
	sl.NeedsLayout()
}

func (sl *Splits) styleSplits() {
	nt := len(sl.Tiles)
	if nt == 0 {
		return
	}
	nh := nt - 1
	for _, t := range sl.Tiles {
		nh += tileNumElements[t] - 1
	}
	sl.handleDirs = slicesx.SetLength(sl.handleDirs, nh)
	dir := sl.Styles.Direction
	odir := dir.Other()
	hi := nt - 1 // extra handles

	for i, t := range sl.Tiles {
		if i > 0 {
			sl.handleDirs[i-1] = dir
		}
		switch t {
		case TileSpan:
		case TileSplit:
			sl.handleDirs[hi] = odir
			hi++
		case TileFirstLong, TileSecondLong:
			sl.handleDirs[hi] = odir
			sl.handleDirs[hi+1] = dir
			hi += 2
		case TilePlus:
			sl.handleDirs[hi] = odir
			sl.handleDirs[hi+1] = dir
			sl.handleDirs[hi+2] = dir
			hi += 3
		}
	}
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
	cszd -= float32(len(sl.TileSplits)-1) * (hwd + gapd)

	setCsz := func(idx int, szm, szc float32) {
		cwb := AsWidget(sl.Child(idx))
		ksz := &cwb.Geom.Size
		ksz.Alloc.Total.SetDim(dim, szm)
		ksz.Alloc.Total.SetDim(odim, szc)
		ksz.setContentFromTotal(&ksz.Alloc)
	}

	ci := 0
	for i, t := range sl.Tiles {
		szt := math32.Round(sl.TileSplits[i] * cszd) // tile size, main axis
		szcs := cszo - hwd - gapo                    // cross axis spilt
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
			fcht := math32.Round(szcs * sl.SubSplits[i][1])
			scht := math32.Round(szcs * sl.SubSplits[i][0])
			setCsz(ci, math32.Round(szs*sl.SubSplits[i][2]), scht)
			setCsz(ci+1, math32.Round(szs*sl.SubSplits[i][3]), scht)
			setCsz(ci+2, szt, fcht)
		case TilePlus:
			fcht := math32.Round(szcs * sl.SubSplits[i][0])
			scht := math32.Round(szcs * sl.SubSplits[i][1])
			setCsz(ci, math32.Round(szs*sl.SubSplits[i][2]), fcht)
			setCsz(ci+1, math32.Round(szs*sl.SubSplits[i][3]), fcht)
			setCsz(ci+2, math32.Round(szs*sl.SubSplits[i][4]), scht)
			setCsz(ci+3, math32.Round(szs*sl.SubSplits[i][5]), scht)
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
	cszd -= float32(len(sl.TileSplits)-1) * (hwd + gapd)
	hwdg := hwd + 0.5*gapd

	setChildPos := func(idx int, dpos, opos float32) {
		cwb := AsWidget(sl.Child(idx))
		cwb.Geom.RelPos.SetDim(dim, dpos)
		cwb.Geom.RelPos.SetDim(odim, opos)
	}
	setHandlePos := func(idx int, dpos, opos, lpos, mn, mx float32) {
		hl := sl.Parts.Child(idx).(*Handle)
		hl.Geom.RelPos.SetDim(dim, dpos)
		hl.Geom.RelPos.SetDim(odim, opos)
		hl.Pos = lpos
		hl.Min = mn
		hl.Max = mx
	}

	tpos := float32(0) // tile position
	ci := 0
	nt := len(sl.Tiles)
	hi := nt - 1 // extra handles

	for i, t := range sl.Tiles {
		szt := math32.Round(sl.TileSplits[i] * cszd) // tile size, main axis
		szcs := cszo - hwd - gapo                    // cross axis spilt
		szs := szt - hwd - gapd
		tn := tileNumElements[t]
		if i > 0 {
			setHandlePos(i-1, tpos-hwdg, .5*(cszo-hht), tpos, 0, cszd)
		}
		switch t {
		case TileSpan:
			setChildPos(ci, tpos, 0)
		case TileSplit:
			fcht := math32.Round(szcs * sl.SubSplits[i][0])
			setHandlePos(hi, tpos+.5*(szt-hht), fcht+0.5*gapo, fcht, 0, szcs)
			hi++
			setChildPos(ci, tpos, 0)
			setChildPos(ci+1, tpos, fcht+hwd+gapo)
		case TileFirstLong, TileSecondLong:
			fcht := math32.Round(szcs * sl.SubSplits[i][0])
			scht := math32.Round(szcs * sl.SubSplits[i][1])
			swd := math32.Round(szs * sl.SubSplits[i][2])
			bot := fcht + hwd + gapo
			setHandlePos(hi, tpos+.5*(szt-hht), fcht+0.5*gapo, fcht, 0, szcs) // long
			if t == TileFirstLong {
				setHandlePos(hi+1, tpos+swd+0.5*gapd, bot+0.5*(scht-hht), tpos+swd, tpos, tpos+szs)
				setChildPos(ci, tpos, 0)
				setChildPos(ci+1, tpos, bot)
				setChildPos(ci+2, tpos+swd+hwd+gapd, bot)
			} else {
				setHandlePos(hi+1, tpos+swd+0.5*gapd, 0.5*(fcht-hht), tpos+swd, tpos, tpos+szs)
				setChildPos(ci, tpos, 0)
				setChildPos(ci+1, tpos+swd+hwd+gapd, 0)
				setChildPos(ci+2, tpos, bot)
			}
			hi += 2
		case TilePlus:
			fcht := math32.Round(szcs * sl.SubSplits[i][0])
			scht := math32.Round(szcs * sl.SubSplits[i][1])
			bot := fcht + hwd + gapo
			setHandlePos(hi, tpos+.5*(szt-hht), fcht+0.5*gapo, fcht, 0, szcs) // long
			swd1 := math32.Round(szs * sl.SubSplits[i][2])
			swd2 := math32.Round(szs * sl.SubSplits[i][4])
			setHandlePos(hi+1, tpos+swd1+0.5*gapd, 0.5*(fcht-hht), tpos+swd1, tpos, tpos+szs)
			setHandlePos(hi+2, tpos+swd2+0.5*gapd, bot+0.5*(scht-hht), tpos+swd2, tpos, tpos+szs)
			setChildPos(ci, tpos, 0)
			setChildPos(ci+1, tpos+swd1+hwd+gapd, 0)
			setChildPos(ci+2, tpos, bot)
			setChildPos(ci+3, tpos+swd2+hwd+gapd, bot)
			hi += 3
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
	if sl.StartRender() {
		sl.ForWidgetChildren(func(i int, kwi Widget, cwb *WidgetBase) bool {
			cwb.SetState(sl.ChildIsCollapsed(i), states.Invisible)
			kwi.RenderWidget()
			return tree.Continue
		})
		sl.renderParts()
		sl.EndRender()
	}
}
