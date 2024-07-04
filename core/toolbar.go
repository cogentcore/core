// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// Toolbar is a [Frame] that is useful for holding [Button]s that do things.
// It automatically moves items that do not fit into an overflow menu, and
// manages additional items that are always placed onto this overflow menu.
// Use [Body.AddAppBar] to add to the default toolbar at the top of an app.
type Toolbar struct {
	Frame

	// OverflowMenus are functions for configuring the overflow menu of the
	// toolbar. You can use [Toolbar.AddOverflowMenu] to add them.
	// These are processed in reverse order (last in, first called)
	// so that the default items are added last.
	OverflowMenus []func(m *Scene) `set:"-" json:"-" xml:"-"`

	// allItemsPlan has all the items, during layout sizing
	allItemsPlan *tree.Plan

	// overflowItems are items moved from the main toolbar that will be
	// shown in the overflow menu.
	overflowItems []*tree.PlanItem

	// overflowButton is the widget to pull up the overflow menu.
	overflowButton *Button
}

func (tb *Toolbar) Init() {
	tb.Frame.Init()
	ToolbarStyles(tb)
	tb.FinalMaker(func(p *tree.Plan) { // must go at end
		tree.AddAt(p, "overflow-menu", func(w *Button) {
			ic := icons.MoreVert
			if tb.Styles.Direction != styles.Row {
				ic = icons.MoreHoriz
			}
			w.SetIcon(ic).SetTooltip("Additional menu items")
			w.Updater(func() {
				tb, ok := w.Parent.(*Toolbar)
				if ok {
					w.Menu = tb.OverflowMenu
				}
			})
		})
	})
}

func (tb *Toolbar) IsVisible() bool {
	// do not render toolbars with no buttons
	return tb.WidgetBase.IsVisible() && len(tb.Children) > 0
}

func (tb *Toolbar) SizeUp() {
	tb.AllItemsToChildren()
	tb.Frame.SizeUp()
}

func (tb *Toolbar) SizeDown(iter int) bool {
	redo := tb.Frame.SizeDown(iter)
	if iter == 0 {
		return true // ensure a second pass
	}
	if tb.Scene.showIter > 0 {
		tb.MoveToOverflow()
	}
	return redo
}

func (tb *Toolbar) SizeFromChildren(iter int, pass LayoutPasses) math32.Vector2 {
	csz := tb.Frame.SizeFromChildren(iter, pass)
	if pass == SizeUpPass || (pass == SizeDownPass && iter == 0) {
		dim := tb.Styles.Direction.Dim()
		ovsz := tb.Styles.UnitContext.FontEm * 2
		if tb.overflowButton != nil {
			ovsz = tb.overflowButton.Geom.Size.Actual.Total.Dim(dim)
		}
		csz.SetDim(dim, ovsz) // present the minimum size initially
		return csz
	}
	return csz
}

// AllItemsToChildren moves the overflow items back to the children,
// so the full set is considered for the next layout round,
// and ensures the overflow button is made and moves it
// to the end of the list.
func (tb *Toolbar) AllItemsToChildren() {
	tb.overflowItems = nil
	tb.allItemsPlan = &tree.Plan{Parent: tb.This}
	tb.Make(tb.allItemsPlan)
	np := len(tb.allItemsPlan.Children)
	if tb.NumChildren() != np {
		tb.Scene.RenderWidget()
		tb.Update() // todo: needs one more redraw here
	}
}

func (tb *Toolbar) ParentSize() float32 {
	ma := tb.Styles.Direction.Dim()
	psz := tb.ParentWidget().Geom.Size.Alloc.Content.Sub(tb.Geom.Size.Space)
	avail := psz.Dim(ma)
	return avail
}

// MoveToOverflow moves overflow out of children to the OverflowItems list
func (tb *Toolbar) MoveToOverflow() {
	if !tb.HasChildren() {
		return
	}
	ma := tb.Styles.Direction.Dim()
	avail := tb.ParentSize()
	li := tb.Children[tb.NumChildren()-1]
	tb.overflowButton = nil
	if li != nil {
		if ob, ok := li.(*Button); ok {
			tb.overflowButton = ob
		}
	}
	if tb.overflowButton == nil {
		return
	}
	ovsz := tb.overflowButton.Geom.Size.Actual.Total.Dim(ma)
	avsz := avail - ovsz
	sz := &tb.Geom.Size
	sz.Alloc.Total.SetDim(ma, avail)
	sz.SetContentFromTotal(&sz.Alloc)
	n := len(tb.Children)
	pn := len(tb.allItemsPlan.Children)
	ovidx := n - 1
	hasOv := false
	szsum := float32(0)
	tb.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		if i >= n-1 {
			return tree.Break
		}
		ksz := kwb.Geom.Size.Alloc.Total.Dim(ma)
		szsum += ksz
		if szsum > avsz {
			if !hasOv {
				ovidx = i
				hasOv = true
			}
			pi := tb.allItemsPlan.Children[i]
			tb.overflowItems = append(tb.overflowItems, pi)
		}
		return tree.Continue
	})
	if hasOv {
		p := &tree.Plan{Parent: tb.This}
		p.Children = tb.allItemsPlan.Children[:ovidx]
		p.Children = append(p.Children, tb.allItemsPlan.Children[pn-1]) // ovm
		p.Update(tb)
	}
	if len(tb.overflowItems) == 0 && len(tb.OverflowMenus) == 0 {
		tb.overflowButton.SetState(true, states.Invisible)
	} else {
		tb.overflowButton.SetState(false, states.Invisible)
		tb.overflowButton.Update()
	}
}

// OverflowMenu adds the overflow menu to the given Scene.
func (tb *Toolbar) OverflowMenu(m *Scene) {
	nm := len(tb.OverflowMenus)
	ni := len(tb.overflowItems)
	if ni > 0 {
		p := &tree.Plan{Parent: tb.This}
		p.Children = tb.overflowItems
		p.Update(m)
		if nm > 1 { // default includes sep
			NewSeparator(m)
		}
	}
	// reverse order so defaults are last
	for i := nm - 1; i >= 0; i-- {
		fn := tb.OverflowMenus[i]
		fn(m)
	}
}

// AddOverflowMenu adds the given menu function to the overflow menu list.
// These functions are called in reverse order such that the last added function
// is called first when constructing the menu.
func (tb *Toolbar) AddOverflowMenu(fun func(m *Scene)) *Toolbar {
	tb.OverflowMenus = append(tb.OverflowMenus, fun)
	return tb
}

// ToolbarStyles styles the given widget to have standard toolbar styling.
func ToolbarStyles(w Widget) {
	w.AsWidget().Styler(func(s *styles.Style) {
		s.Border.Radius = styles.BorderRadiusFull
		s.Background = colors.Scheme.SurfaceContainer
		s.Gap.Zero()
		s.Align.Items = styles.Center
	})
	w.AsWidget().FinalStyler(func(s *styles.Style) {
		if s.Direction == styles.Row {
			s.Grow.Set(1, 0)
			s.Padding.SetHorizontal(units.Dp(16))
		} else {
			s.Grow.Set(0, 1)
			s.Padding.SetVertical(units.Dp(16))
		}
	})
	w.AsWidget().OnWidgetAdded(func(w Widget) {
		if bt := AsButton(w); bt != nil {
			bt.Type = ButtonAction
			return
		}
		if sp, ok := w.(*Separator); ok {
			sp.Styler(func(s *styles.Style) {
				s.Direction = w.AsWidget().Styles.Direction.Other()
			})
		}
	})
}
