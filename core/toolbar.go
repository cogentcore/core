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

	// OverflowMenus are functions for the overflow menu; use [Toolbar.AddOverflowMenu] to add.
	// These are processed in reverse order (last in, first called)
	// so that the default items are added last.
	OverflowMenus []func(m *Scene) `set:"-" json:"-" xml:"-"`

	// overflowItems are items moved from the main toolbar that will be shown in the overflow menu.
	overflowItems tree.Slice

	// overflowButton is the widget to pull up the overflow menu.
	overflowButton *Button
}

func (tb *Toolbar) OnInit() {
	tb.Frame.OnInit()
	ToolbarStyles(tb)

	AddChildAt(tb, "overflow-menu", func(w *Button) {
		tb.overflowButton = w
		ic := icons.MoreVert
		if tb.Styles.Direction != styles.Row {
			ic = icons.MoreHoriz
		}
		w.SetIcon(ic).SetTooltip("Additional menu items")
		w.Builder(func() {
			tb, ok := w.Parent().(*Toolbar)
			if ok {
				w.Menu = tb.OverflowMenu
			}
		})
	})
}

func (tb *Toolbar) IsVisible() bool {
	// do not render toolbars with no buttons
	return tb.WidgetBase.IsVisible() && len(tb.Kids) > 0
}

// AppChooser returns the app [Chooser] used for searching for
// items. It will only be non-nil if this toolbar has been configured
// with an app chooser, which typically only happens for app bars.
func (tb *Toolbar) AppChooser() *Chooser {
	ch, _ := tb.ChildByName("app-chooser").(*Chooser)
	return ch
}

func (tb *Toolbar) SizeUp() {
	tb.AllItemsToChildren()
	tb.Frame.SizeUp()
}

// todo: try doing move to overflow in Final

func (tb *Toolbar) SizeDown(iter int) bool {
	redo := tb.Frame.SizeDown(iter)
	if iter == 0 {
		return true // ensure a second pass
	}
	tb.MoveToOverflow()
	return redo
}

func (tb *Toolbar) SizeFromChildren(iter int, pass LayoutPasses) math32.Vector2 {
	csz := tb.Frame.SizeFromChildren(iter, pass)
	if pass == SizeUpPass || (pass == SizeDownPass && iter == 0) {
		dim := tb.Styles.Direction.Dim()
		ovsz := tb.overflowButton.Geom.Size.Actual.Total.Dim(dim)
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
	if len(tb.overflowItems) > 0 {
		tb.Kids = append(tb.Kids, tb.overflowItems...)
		tb.overflowItems = nil
	}
	ovi := -1
	for i, k := range tb.Kids {
		_, wb := AsWidget(k)
		if wb == nil {
			continue
		}
		if wb.This() == tb.overflowButton.This() {
			ovi = i
			break
		}
	}
	if ovi >= 0 {
		tb.Kids.DeleteAtIndex(ovi)
	}
	tb.Kids = append(tb.Kids, tb.overflowButton.This())
	tb.overflowButton.Update()
}

func (tb *Toolbar) ParentSize() float32 {
	ma := tb.Styles.Direction.Dim()
	psz := tb.ParentWidget().Geom.Size.Alloc.Content.Sub(tb.Geom.Size.Space)
	avail := psz.Dim(ma)
	return avail
}

// MoveToOverflow moves overflow out of children to the OverflowItems list
func (tb *Toolbar) MoveToOverflow() {
	ma := tb.Styles.Direction.Dim()
	avail := tb.ParentSize()
	ovsz := tb.overflowButton.Geom.Size.Actual.Total.Dim(ma)
	avsz := avail - ovsz
	sz := &tb.Geom.Size
	sz.Alloc.Total.SetDim(ma, avail)
	sz.SetContentFromTotal(&sz.Alloc)
	n := len(tb.Kids)
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
			tb.overflowItems = append(tb.overflowItems, kwi)
		}
		return tree.Continue
	})
	if ovidx != n-1 {
		tb.Kids.Move(n-1, ovidx)
		tb.Kids = tb.Kids[:ovidx+1]
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
	if len(tb.overflowItems) > 0 {
		for _, k := range tb.overflowItems {
			if k.This() == tb.overflowButton.This() {
				continue
			}
			cl := k.This().Clone()
			m.AddChild(cl)
			cl.(Widget).AsWidget().Build()
		}
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
func (tb *Toolbar) AddOverflowMenu(fun func(m *Scene)) {
	tb.OverflowMenus = append(tb.OverflowMenus, fun)
}

// ToolbarStyles styles the given widget to have standard toolbar styling.
func ToolbarStyles(w Widget) {
	w.Style(func(s *styles.Style) {
		s.Border.Radius = styles.BorderRadiusFull
		s.Background = colors.C(colors.Scheme.SurfaceContainer)
		s.Gap.Zero()
		s.Align.Items = styles.Center
	})
	w.AsWidget().StyleFinal(func(s *styles.Style) {
		if s.Direction == styles.Row {
			s.Grow.Set(1, 0)
			s.Padding.SetHorizontal(units.Dp(16))
		} else {
			s.Grow.Set(0, 1)
			s.Padding.SetVertical(units.Dp(16))
		}
	})
	w.OnWidgetAdded(func(w Widget) { // TODO(config)
		if bt := AsButton(w); bt != nil {
			bt.Type = ButtonAction
			return
		}
		if sp, ok := w.(*Separator); ok {
			sp.Style(func(s *styles.Style) {
				s.Direction = w.AsWidget().Styles.Direction.Other()
			})
		}
	})
}

// BasicBar is a [Frame] that automatically has [ToolbarStyles] applied but does
// not have the more advanced features of a [Toolbar].
type BasicBar struct {
	Frame
}

func (tb *BasicBar) OnInit() {
	tb.Frame.OnInit()
	ToolbarStyles(tb)
}
