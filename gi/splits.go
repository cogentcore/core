// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"strconv"
	"strings"

	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// Config notes: only needs config when number of kids changes
// otherwise just needs new layout

// Splits allocates a fixed proportion of space to each child, along given
// dimension, always using only the available space given to it by its parent
// (i.e., it will force its children, which should be layouts (typically
// Frame's), to have their own scroll bars as necessary).
// Do not set the Grow factor on these children of the splits, because
// it uses this to set the split proportions.
// It should generally be used as a main outer-level structure within a window,
// providing a framework for inner elements -- it allows individual child
// elements to update independently and thus is important for speeding update
// performance.  It uses the Widget Parts to hold the splitter widgets
// separately from the children that contain the rest of the scenegraph to be
// displayed within each region.
type Splits struct { //goki:embedder
	Layout

	// dimension along which to split the space
	Dim mat32.Dims

	// proportion (0-1 normalized, enforced) of space allocated to each element.
	// Enter 0 to collapse a given element
	Splits []float32 `set:"-"`

	// A saved version of the splits which can be restored.
	// For dynamic collapse / expand operations
	SavedSplits []float32 `set:"-"`
}

func (sl *Splits) CopyFieldsFrom(frm any) {
	fr := frm.(*Splits)
	sl.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
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
		s.Grow.Set(1, 1)
		s.Margin.Zero()
		s.Padding.Zero()
		s.Gap.SetDim(sl.Dim, units.Dp(14))
		s.Gap.SetDim(sl.Dim.Other(), units.Dp(0))
	})
	sl.OnWidgetAdded(func(w Widget) {
		if hl, ok := w.(*Handle); ok {
			// hl.ThumbSize = sl.HandleSize
			hl.On(events.Change, func(e events.Event) {
				ip, _ := hl.IndexInParent()
				sl.SetSplitAction(ip, hl.Value())
			})
			w.Style(func(s *styles.Style) {

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

// SetSplitAction sets the new splitter value, for given splitter.
// New value is 0..1 value of position of that splitter.
// It is a sum of all the positions up to that point.
// Splitters are updated to ensure that selected position is achieved,
// while dividing remainder appropriately.
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
	sl.UpdateEndLayout(updt)
}

func (sl *Splits) ConfigWidget(sc *Scene) {
	sl.UpdateSplits()
	sl.ConfigSplitters(sc)
}

func (sl *Splits) ConfigSplitters(sc *Scene) {
	parts := sl.NewParts()
	sz := len(sl.Kids)
	mods, updt := parts.SetNChildren(sz-1, HandleType, "handle-")
	for _, hlk := range *sl.Parts.Children() {
		hl := hlk.(*Handle)
		// hl.SplitterNo = i
		// hl.Icon = spicon
		hl.Dim = sl.Dim
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

func (sl *Splits) ApplyStyle(sc *Scene) {
	sl.StyMu.Lock()

	sl.UpdateSplits()
	sl.ApplyStyleWidget(sc)
	sl.StyMu.Unlock()

	sl.ConfigSplitters(sc)
}

func (sl *Splits) SizeDownSetAllocs(sc *Scene, iter int) {
	sz := &sl.Geom.Size
	csz := sz.Alloc.Content
	// fmt.Println(sl, sz.String())
	od := sl.Dim.Other()
	cszd := csz.Dim(sl.Dim)
	cszod := csz.Dim(od)
	sl.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		sw := mat32.Floor(sl.Splits[i] * cszd)
		ksz := &kwb.Geom.Size
		ksz.Alloc.Total.SetDim(sl.Dim, sw)
		ksz.Alloc.Total.SetDim(od, cszod)
		ksz.SetContentFromTotal(&ksz.Alloc)
		return ki.Continue
	})
}

func (sl *Splits) Position(sc *Scene) {
	sl.UpdateSplits()
	sl.ConfigScrolls(sc)
	sl.PositionSplits(sc)
	sl.PositionChildren(sc)
}

func (sl *Splits) PositionSplits(sc *Scene) {
	if sl.Parts != nil {
		sl.Parts.Geom.Size = sl.Geom.Size // inherit: allows bbox to include handle
	}
	od := sl.Dim.Other()
	csz := sl.Geom.Size.Actual.Content // excludes gaps
	cszd := csz.Dim(sl.Dim)
	pos := float32(0)
	gap := mat32.Floor(sl.Styles.Gap.Dim(sl.Dim).Dots)

	mid := .5 * csz.Dim(od)
	hand := sl.Parts.Child(0).(*Handle)
	hwd := hand.Geom.Size.Actual.Total.Dim(sl.Dim)
	hmrg := hand.Styles.Margin.Dots()
	if sl.Dim == mat32.X {
		hwd -= hmrg.Left
	} else {
		hwd -= hmrg.Top
	}
	hht := hand.Geom.Size.Actual.Total.Dim(od)
	nhand := float32(len(*sl.Parts.Children()))
	sod := mid - .5*nhand*hht

	sl.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwb.Geom.RelPos.SetZero()
		if i == 0 {
			return ki.Continue
		}
		sw := mat32.Floor(sl.Splits[i-1] * cszd)
		pos += sw + gap
		kwb.Geom.RelPos.SetDim(sl.Dim, pos)
		hl := sl.Parts.Child(i - 1).(*Handle)
		hl.Geom.RelPos.SetDim(sl.Dim, pos-hwd-2) // todo: Kai -- -2 is needed but not sure why..
		hl.Geom.RelPos.SetDim(od, sod+float32(i-1)*hht)
		hl.Min = 0
		hl.Max = cszd
		hl.Pos = pos
		return ki.Continue
	})
}

func (sl *Splits) Render(sc *Scene) {
	if sl.PushBounds(sc) {
		sl.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
			sp := sl.Splits[i]
			if sp <= 0.01 {
				kwb.SetState(true, states.Invisible)
			} else {
				kwb.SetState(false, states.Invisible)
			}
			kwi.Render(sc)
			return ki.Continue
		})
		sl.RenderParts(sc)
		sl.PopBounds(sc)
	}
}

// func (sl *Splits) StateIs(states.Focused) bool {
// 	return sl.ContainsFocus() // anyone within us gives us focus..
// }
