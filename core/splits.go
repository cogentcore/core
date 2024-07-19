// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
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

// Splits allocates a certain proportion of its space to each of its children,
// organized according to Order and [styles.Style.Columns], where Columns
// can be an even divisor of the Order length to create 2D layouts,
// or 1 to specify a vertical instead of horizontal layout.
// It adds [Handle] widgets to its parts that allow the user to customize
// the amount of space allocated to each child.
type Splits struct {
	Frame

	// Splits is the proportion (0-1 normalized, enforced) of space
	// allocated to each element. 0 indicates that an element should
	// be completely collapsed. By default, each element gets the
	// same amount of space.
	Splits []float32

	// Order is the organization of the splits content, using indexes 0..n-1
	// to specify what goes where, in generally increasing seqeuential order.
	// This is used to specify 2D layouts when [styles.Style.Columns] is an
	// even divisor of the Order length, e.g., 0012 with Columns = 2
	// specifies the first element spanning the top row, with the next two
	// elements splitting the bottom row.
	Order []int

	// savedSplits is a saved version of the splits that can be restored
	// for dynamic collapse/expand operations.
	savedSplits []float32
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
			AsWidget(n).Styler(func(s *styles.Style) {
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
			sl.evenSplits()
		} else if kn <= len(sl.Children) {
			e.SetHandled()
			if sl.Splits[kn-1] <= 0.01 {
				sl.restoreChild(kn - 1)
			} else {
				sl.collapseChild(true, kn-1)
			}
		}
	})

	sl.Updater(func() {
		sl.updateSplits()
	})
	parts := sl.newParts()
	parts.Maker(func(p *tree.Plan) {
		for i := range len(sl.Children) - 1 { // one fewer handle than children
			tree.AddAt(p, "handle-"+strconv.Itoa(i), func(w *Handle) {
				w.OnChange(func(e events.Event) {
					sl.setSplit(w.IndexInParent(), w.Value())
				})
				w.Styler(func(s *styles.Style) {
					s.Direction = sl.Styles.Direction
				})
			})
		}
	})
}

// updateSplits normalizes the splits and ensures that there are as
// many split proportions as children.
func (sl *Splits) updateSplits() *Splits {
	n := len(sl.Children)
	if n == 0 {
		return sl
	}
	sl.Splits = slicesx.SetLength(sl.Splits, n)
	sum := float32(0)
	for _, sp := range sl.Splits {
		sum += sp
	}
	if sum == 0 { // set default even splits
		sl.evenSplits()
		sum = 1
	} else {
		norm := 1 / sum
		for i := range sl.Splits {
			sl.Splits[i] *= norm
		}
	}
	return sl
}

// evenSplits splits space evenly across all panels
func (sl *Splits) evenSplits() {
	n := len(sl.Children)
	if n == 0 {
		return
	}
	even := 1.0 / float32(n)
	for i := range sl.Splits {
		sl.Splits[i] = even
	}
	sl.NeedsLayout()
}

// saveSplits saves the current set of splits in SavedSplits, for a later RestoreSplits
func (sl *Splits) saveSplits() {
	n := len(sl.Splits)
	if n == 0 {
		return
	}
	if sl.savedSplits == nil || len(sl.savedSplits) != n {
		sl.savedSplits = make([]float32, n)
	}
	copy(sl.savedSplits, sl.Splits)
}

// restoreSplits restores a previously saved set of splits (if it exists), does an update
func (sl *Splits) restoreSplits() {
	if sl.savedSplits == nil {
		return
	}
	sl.SetSplits(sl.savedSplits...).NeedsLayout()
}

// collapseChild collapses given child(ren) (sets split proportion to 0),
// optionally saving the prior splits for later Restore function -- does an
// Update -- triggered by double-click of splitter
func (sl *Splits) collapseChild(save bool, idxs ...int) {
	if save {
		sl.saveSplits()
	}
	n := len(sl.Children)
	for _, idx := range idxs {
		if idx >= 0 && idx < n {
			sl.Splits[idx] = 0
		}
	}
	sl.updateSplits()
	sl.NeedsLayout()
}

// restoreChild restores given child(ren) -- does an Update
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

// isCollapsed returns true if given split number is collapsed
func (sl *Splits) isCollapsed(idx int) bool {
	n := len(sl.Children)
	if idx >= 0 && idx < n {
		return sl.Splits[idx] < 0.01
	}
	return false
}

// setSplit sets the new splitter value, for given splitter.
// New value is 0..1 value of position of that splitter.
// It is a sum of all the positions up to that point.
// Splitters are updated to ensure that selected position is achieved,
// while dividing remainder appropriately.
func (sl *Splits) setSplit(idx int, nwval float32) {
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
	sl.updateSplits()
	sz := &sl.Geom.Size
	csz := sz.Alloc.Content.Sub(sz.InnerSpace)
	dim := sl.Styles.Direction.Dim()
	od := dim.Other()
	cszd := csz.Dim(dim)
	cszod := csz.Dim(od)
	hand := sl.Parts.Child(0).(*Handle)
	hwd := hand.Geom.Size.Actual.Total.Dim(dim)
	cszd -= float32(len(sl.Splits)-1) * hwd
	sl.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		sw := math32.Round(sl.Splits[i] * cszd)
		ksz := &cwb.Geom.Size
		ksz.Alloc.Total.SetDim(dim, sw)
		ksz.Alloc.Total.SetDim(od, cszod)
		ksz.setContentFromTotal(&ksz.Alloc)
		// ksz.Actual = ksz.Alloc
		return tree.Continue
	})
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

func (sl *Splits) positionSplits() {
	if sl.NumChildren() <= 1 {
		return
	}
	if sl.Parts != nil {
		sl.Parts.Geom.Size = sl.Geom.Size // inherit: allows bbox to include handle
	}
	sz := &sl.Geom.Size
	dim := sl.Styles.Direction.Dim()
	od := dim.Other()
	csz := sz.Alloc.Content.Sub(sz.InnerSpace)
	cszd := csz.Dim(dim)
	pos := float32(0)

	hand := sl.Parts.Child(0).(*Handle)
	hwd := hand.Geom.Size.Actual.Total.Dim(dim)
	hht := hand.Geom.Size.Actual.Total.Dim(od)
	mid := (csz.Dim(od) - hht) / 2
	cszd -= float32(len(sl.Splits)-1) * hwd

	sl.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cwb.Geom.RelPos.SetZero()
		if i == 0 {
			return tree.Continue
		}
		sw := math32.Round(sl.Splits[i-1] * cszd)
		pos += sw + hwd
		cwb.Geom.RelPos.SetDim(dim, pos)
		hl := sl.Parts.Child(i - 1).(*Handle)
		hl.Geom.RelPos.SetDim(dim, pos-hwd)
		hl.Geom.RelPos.SetDim(od, mid)
		hl.Min = 0
		hl.Max = cszd
		hl.Pos = pos
		return tree.Continue
	})
}

func (sl *Splits) RenderWidget() {
	if sl.PushBounds() {
		sl.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
			sp := sl.Splits[i]
			if sp <= 0.01 {
				cwb.SetState(true, states.Invisible)
			} else {
				cwb.SetState(false, states.Invisible)
			}
			cw.RenderWidget()
			return tree.Continue
		})
		sl.renderParts()
		sl.PopBounds()
	}
}
