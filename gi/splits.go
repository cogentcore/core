// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/mat32/v2"
)

// Config notes: only needs config when number of kids changes
// otherwise just needs new layout

// Splits allocates a fixed proportion of space to each child, along given
// dimension, always using only the available space given to it by its parent
// (i.e., it will force its children, which should be layouts (typically
// Frame's), to have their own scroll bars as necessary).  It should
// generally be used as a main outer-level structure within a window,
// providing a framework for inner elements -- it allows individual child
// elements to update independently and thus is important for speeding update
// performance.  It uses the Widget Parts to hold the splitter widgets
// separately from the children that contain the rest of the scenegraph to be
// displayed within each region.
type Splits struct { //goki:embedder
	WidgetBase

	// size of the handle region in the middle of each split region, where the splitter can be dragged -- other-dimension size is 2x of this
	HandleSize units.Value `xml:"handle-size"`

	// proportion (0-1 normalized, enforced) of space allocated to each element -- can enter 0 to collapse a given element
	Splits []float32 `set:"-"`

	// A saved version of the splits which can be restored -- for dynamic collapse / expand operations
	SavedSplits []float32 `set:"-"`

	// dimension along which to split the space
	Dim mat32.Dims
}

func (sl *Splits) CopyFieldsFrom(frm any) {
	fr := frm.(*Splits)
	sl.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	sl.HandleSize = fr.HandleSize
	mat32.CopyFloat32s(&sl.Splits, fr.Splits)
	mat32.CopyFloat32s(&sl.SavedSplits, fr.SavedSplits)
	sl.Dim = fr.Dim
}

func (sl *Splits) OnInit() {
	sl.HandleSplitsEvents()
	sl.SplitsStyles()
}

func (sl *Splits) SplitsStyles() {
	sl.Style(func(s *styles.Style) {
		sl.HandleSize.SetDp(10)
		s.MaxWidth.SetDp(-1)
		s.MaxHeight.SetDp(-1)
		s.Margin.Set()
		s.Padding.Set()
	})
	sl.OnWidgetAdded(func(w Widget) {
		if hl, ok := w.(*Handle); ok {
			// hl.ThumbSize = sl.HandleSize
			hl.On(events.Change, func(e events.Event) {
				ip, _ := hl.IndexInParent()
				sl.SetSplitAction(ip, hl.Value())
			})
		}
	})
}

// UpdateSplits updates the splits to be same length as number of children,
// and normalized
func (sl *Splits) UpdateSplits() {
	sz := len(sl.Kids)
	if sz == 0 {
		return
	}
	if sl.Splits == nil || len(sl.Splits) != sz {
		sl.Splits = make([]float32, sz)
	}
	sum := float32(0.0)
	for _, sp := range sl.Splits {
		sum += sp
	}
	if sum == 0 { // set default even splits
		sl.EvenSplits()
		sum = 1.0
	} else {
		norm := 1.0 / sum
		for i := range sl.Splits {
			sl.Splits[i] *= norm
		}
	}
}

// EvenSplits splits space evenly across all panels
func (sl *Splits) EvenSplits() {
	updt := sl.UpdateStart()
	sz := len(sl.Kids)
	if sz == 0 {
		return
	}
	even := 1.0 / float32(sz)
	for i := range sl.Splits {
		sl.Splits[i] = even
	}
	sl.UpdateEndLayout(updt)
}

// SetSplits sets the split proportions -- can use 0 to hide / collapse a
// child entirely.
func (sl *Splits) SetSplits(splits ...float32) *Splits {
	sl.UpdateSplits()
	sz := len(sl.Kids)
	mx := min(sz, len(splits))
	for i := 0; i < mx; i++ {
		sl.Splits[i] = splits[i]
	}
	sl.UpdateSplits()
	return sl
}

// SetSplitsList sets the split proportions using a list (slice) argument,
// instead of variable args -- e.g., for Python or other external users.
// can use 0 to hide / collapse a child entirely -- just does the basic local
// update start / end -- use SetSplitsAction to trigger full rebuild
// which is typically required
func (sl *Splits) SetSplitsList(splits []float32) *Splits {
	return sl.SetSplits(splits...)
}

// SetSplitsAction sets the split proportions -- can use 0 to hide / collapse a
// child entirely -- does full rebuild at level of scene
func (sl *Splits) SetSplitsAction(splits ...float32) *Splits {
	updt := sl.UpdateStart()
	sl.SetSplits(splits...)
	sl.UpdateEndLayout(updt)
	return sl
}

// SaveSplits saves the current set of splits in SavedSplits, for a later RestoreSplits
func (sl *Splits) SaveSplits() {
	sz := len(sl.Splits)
	if sz == 0 {
		return
	}
	if sl.SavedSplits == nil || len(sl.SavedSplits) != sz {
		sl.SavedSplits = make([]float32, sz)
	}
	copy(sl.SavedSplits, sl.Splits)
}

// RestoreSplits restores a previously-saved set of splits (if it exists), does an update
func (sl *Splits) RestoreSplits() {
	if sl.SavedSplits == nil {
		return
	}
	sl.SetSplitsAction(sl.SavedSplits...)
}

// CollapseChild collapses given child(ren) (sets split proportion to 0),
// optionally saving the prior splits for later Restore function -- does an
// Update -- triggered by double-click of splitter
func (sl *Splits) CollapseChild(save bool, idxs ...int) {
	updt := sl.UpdateStart()
	if save {
		sl.SaveSplits()
	}
	sz := len(sl.Kids)
	for _, idx := range idxs {
		if idx >= 0 && idx < sz {
			sl.Splits[idx] = 0
		}
	}
	sl.UpdateSplits()
	sl.UpdateEndLayout(updt)
}

// RestoreChild restores given child(ren) -- does an Update
func (sl *Splits) RestoreChild(idxs ...int) {
	updt := sl.UpdateStart()
	sz := len(sl.Kids)
	for _, idx := range idxs {
		if idx >= 0 && idx < sz {
			sl.Splits[idx] = 1.0 / float32(sz)
		}
	}
	sl.UpdateSplits()
	sl.UpdateEndLayout(updt)
}

// IsCollapsed returns true if given split number is collapsed
func (sl *Splits) IsCollapsed(idx int) bool {
	sz := len(sl.Kids)
	if idx >= 0 && idx < sz {
		return sl.Splits[idx] < 0.01
	}
	return false
}

// SetSplitAction sets the new splitter value, for given splitter -- new
// value is 0..1 value of position of that splitter -- it is a sum of all the
// positions up to that point.  Splitters are updated to ensure that selected
// position is achieved, while dividing remainder appropriately.
func (sl *Splits) SetSplitAction(idx int, nwval float32) {
	updt := sl.UpdateStart()
	sz := len(sl.Splits)
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
	if idx < sz-1 {
		oldrmdr := 1 - oldsum
		if oldrmdr <= 0 {
			if rmdr > 0 {
				dper := rmdr / float32((sz-1)-idx)
				for i := idx + 1; i < sz; i++ {
					sl.Splits[i] = dper
				}
			}
		} else {
			for i := idx + 1; i < sz; i++ {
				curval := sl.Splits[i]
				sl.Splits[i] = rmdr * (curval / oldrmdr) // proportional
			}
		}
	}
	sl.Splits[idx] = uval
	// fmt.Printf("splits: %v value: %v  splts: %v\n", idx, nwval, sl.Splits)
	sl.UpdateSplits()
	// fmt.Printf("splits: %v\n", sl.Splits)
	sl.UpdateEndRender(updt)
}

func (sl *Splits) ConfigWidget(sc *Scene) {
	sl.UpdateSplits()
	sl.ConfigSplitters(sc)
}

func (sl *Splits) ConfigSplitters(sc *Scene) {
	fmt.Println("cfg sl")
	parts := sl.NewParts(LayoutNil)
	sz := len(sl.Kids)
	fmt.Println(sz, sl.Kids)
	mods, updt := parts.SetNChildren(sz-1, HandleType, "handle-")
	odim := mat32.OtherDim(sl.Dim)
	spc := sl.BoxSpace()
	size := sl.LayState.Alloc.Size.Dim(sl.Dim) - spc.Size().Dim(sl.Dim)
	handsz := sl.HandleSize.Dots
	mid := 0.5 * (sl.LayState.Alloc.Size.Dim(odim) - spc.Size().Dim(odim))
	fmt.Println(parts.Kids)
	for i, hlk := range *parts.Children() {
		hl := hlk.(*Handle)
		// hl.SplitterNo = i
		// hl.Icon = spicon
		hl.Dim = sl.Dim
		fmt.Println(size, handsz*2)
		hl.LayState.Alloc.Size.SetDim(sl.Dim, size)
		hl.LayState.Alloc.Size.SetDim(odim, handsz*2)
		hl.LayState.Alloc.SizeOrig = hl.LayState.Alloc.Size
		hl.LayState.Alloc.PosRel.SetDim(sl.Dim, 0)
		hl.LayState.Alloc.PosRel.SetDim(odim, mid-handsz+float32(i)*handsz*4)
		hl.LayState.Alloc.PosOrig = hl.LayState.Alloc.PosRel
		hl.Min = sl.LayState.Alloc.Pos.Dim(sl.Dim)
		hl.Max = sl.LayState.Alloc.Size.Sub(sl.LayState.Alloc.Pos).Dim(sl.Dim)
		// hl.Snap = false
		// hl.ThumbSize = sl.HandleSize
	}
	if mods {
		parts.Update()
		parts.UpdateEnd(updt)
	}
}

func (sl *Splits) HandleSplitsKeys() {
	sl.OnKeyChord(func(e events.Event) {
		kc := string(e.KeyChord())
		mod := "Control+"
		if goosi.TheApp.Platform() == goosi.MacOS {
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
		// fmt.Printf("kc: %v kns: %v kn: %v\n", kc, kns, kn)
		if kn == 0 {
			e.SetHandled()
			sl.EvenSplits()
		} else if kn <= len(sl.Kids) {
			e.SetHandled()
			if sl.Splits[kn-1] <= 0.01 {
				sl.RestoreChild(kn - 1)
			} else {
				sl.CollapseChild(true, kn-1)
			}
		}
	})
}

func (sl *Splits) HandleSplitsEvents() {
	sl.HandleSplitsKeys()
}

func (sl *Splits) StyleSplits(sc *Scene) {
	sl.ApplyStyleWidget(sc)
	sl.HandleSize.ToDots(&sl.Styles.UnContext)
}

func (sl *Splits) ApplyStyle(sc *Scene) {
	sl.StyMu.Lock()

	sl.StyleSplits(sc)
	sl.UpdateSplits()
	sl.StyMu.Unlock()

	sl.ConfigSplitters(sc)
}

func (sl *Splits) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	sl.DoLayoutBase(sc, parBBox, iter)
	sl.DoLayoutParts(sc, parBBox, iter)
	sl.UpdateSplits()

	handsz := sl.HandleSize.Dots
	// fmt.Printf("handsz: %v\n", handsz)
	sz := len(sl.Kids)
	odim := mat32.OtherDim(sl.Dim)
	spc := sl.BoxSpace()
	size := sl.LayState.Alloc.Size.Dim(sl.Dim) - spc.Size().Dim(sl.Dim)
	avail := size - handsz*float32(sz-1)
	// fmt.Printf("avail: %v\n", avail)
	osz := sl.LayState.Alloc.Size.Dim(odim) - spc.Size().Dim(odim)
	pos := float32(0.0)

	spsum := float32(0)
	for i, sp := range sl.Splits {
		_, wb := AsWidget(sl.Kids[i])
		if wb == nil {
			continue
		}
		isz := sp * avail
		wb.LayState.Alloc.Size.SetDim(sl.Dim, isz)
		wb.LayState.Alloc.Size.SetDim(odim, osz)
		wb.LayState.Alloc.SizeOrig = wb.LayState.Alloc.Size
		wb.LayState.Alloc.PosRel.SetDim(sl.Dim, pos)
		wb.LayState.Alloc.PosRel.SetDim(odim, spc.Pos().Dim(odim))
		// fmt.Printf("spl: %v sp: %v size: %v alloc: %v  pos: %v\n", i, sp, isz, wb.LayState.Alloc.SizeOrig, wb.LayState.Alloc.PosRel)

		pos += isz + handsz

		spsum += sp
		if i < sz-1 {
			hl := sl.Parts.Child(i).(*Handle)
			hl.Pos = spsum
			// hl.UpdatePosFromValue(hl.Value)
		}
	}

	return sl.DoLayoutChildren(sc, iter)
}

func (sl *Splits) Render(sc *Scene) {
	if sl.PushBounds(sc) {
		for i, kid := range sl.Kids {
			wi, wb := AsWidget(kid)
			if wb == nil {
				continue
			}
			sp := sl.Splits[i]
			if sp <= 0.01 {
				wb.SetState(true, states.Invisible)
			} else {
				wb.SetState(false, states.Invisible)
			}
			wi.Render(sc) // needs to disconnect using invisible
		}
		fmt.Println(sl.Parts.Kids)
		sl.RenderParts(sc)
		sl.PopBounds(sc)
	}
}

// func (sl *Splits) StateIs(states.Focused) bool {
// 	return sl.ContainsFocus() // anyone within us gives us focus..
// }
