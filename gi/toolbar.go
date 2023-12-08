// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// Toolbar is a [Frame] that is useful for holding [Button]s that do things.
// It automatically moves items that do not fit into an overflow menu, and
// manages additional items that are always placed onto this overflow menu.
// In general it should be possible to use a single toolbar + overflow to
// manage all an app's functionality, in a way that is portable across
// mobile and desktop environments.
// See [Widget.ConfigToolbar] for the standard toolbar config method for
// any given widget, and [Scene.AppBars] for [ToolbarFuncs] for [Scene]
// elements who should be represented in the main AppBar (e.g., TopAppBar).
type Toolbar struct { //goki:embedder
	Frame

	// items moved from the main toolbar, will be shown in the overflow menu
	OverflowItems ki.Slice `set:"-" json:"-" xml:"-"`

	// functions for overflow menu: use AddOverflowMenu to add.
	// These are processed in _reverse_ order (last in, first called)
	// so that the default items are added last.
	OverflowMenus []func(m *Scene) `set:"-" json:"-" xml:"-"`

	// This is the overflow button
	OverflowButton *Button
}

func (tb *Toolbar) CopyFieldsFrom(frm any) {
	fr := frm.(*Toolbar)
	tb.Frame.CopyFieldsFrom(&fr.Frame)
}

func (tb *Toolbar) OnInit() {
	tb.ToolbarStyles()
	tb.HandleLayoutEvents()
}

func (tb *Toolbar) ToolbarStyles() {
	ToolbarStyles(tb)
}

func (tb *Toolbar) IsVisible() bool {
	// do not render toolbars with no buttons
	return tb.WidgetBase.IsVisible() && len(tb.Kids) > 0
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

func (tb *Toolbar) SizeFromChildren(iter int, pass LayoutPasses) mat32.Vec2 {
	csz := tb.Frame.SizeFromChildren(iter, pass)
	if pass == SizeUpPass || (pass == SizeDownPass && iter == 0) {
		ovsz := tb.OverflowButton.Geom.Size.Actual.Total.X
		csz.X = ovsz // present the minimum size initially
		return csz
	}
	return csz
}

// AllItemsToChildren moves the overflow items back to the children,
// so the full set is considered for the next layout round,
// and ensures the overflow button is made and moves it
// to the end of the list.
func (tb *Toolbar) AllItemsToChildren() {
	if tb.OverflowButton == nil {
		ic := icons.MoreVert
		if tb.Styles.Direction != styles.Row {
			ic = icons.MoreHoriz
		}
		tb.OverflowButton = NewButton(tb, "overflow-menu").SetIcon(ic).
			SetTooltip("Overflow toolbar items and additional menu items")
		tb.OverflowButton.Menu = tb.OverflowMenu
	}
	if len(tb.OverflowItems) > 0 {
		tb.Kids = append(tb.Kids, tb.OverflowItems...)
		tb.OverflowItems = nil
	}
	ovi := -1
	for i, k := range tb.Kids {
		_, wb := AsWidget(k)
		if wb.This() == tb.OverflowButton.This() {
			ovi = i
			break
		}
	}
	if ovi >= 0 {
		tb.Kids.DeleteAtIndex(ovi)
	}
	tb.Kids = append(tb.Kids, tb.OverflowButton.This())
	tb.OverflowButton.Update()
}

func (tb *Toolbar) ParentSize() float32 {
	ma := tb.Styles.Direction.Dim()
	_, pwb := tb.ParentWidget()
	psz := pwb.Geom.Size.Alloc.Content.Sub(tb.Geom.Size.Space)
	avail := psz.Dim(ma)
	return avail
}

// MoveToOverflow moves overflow out of children to the OverflowItems list
func (tb *Toolbar) MoveToOverflow() {
	ma := tb.Styles.Direction.Dim()
	avail := tb.ParentSize()
	ovsz := tb.OverflowButton.Geom.Size.Actual.Total.Dim(ma)
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
			return ki.Break
		}
		ksz := kwb.Geom.Size.Alloc.Total.Dim(ma)
		szsum += ksz
		if szsum > avsz {
			if !hasOv {
				ovidx = i
				hasOv = true
			}
			tb.OverflowItems = append(tb.OverflowItems, kwi)
		}
		return ki.Continue
	})
	if ovidx != n-1 {
		tb.Kids.Move(n-1, ovidx)
		tb.Kids = tb.Kids[:ovidx+1]
	}
	tb.OverflowButton.Update()
}

// OverflowMenu is the overflow menu function
func (tb *Toolbar) OverflowMenu(m *Scene) {
	nm := len(tb.OverflowMenus)
	if len(tb.OverflowItems) > 0 {
		for _, k := range tb.OverflowItems {
			if k.This() == tb.OverflowButton.This() {
				continue
			}
			cl := k.This().Clone()
			m.AddChild(cl)
			cl.This().(Widget).Config()
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

// AddOverflowMenu adds given menu function to overflow menu list
func (tb *Toolbar) AddOverflowMenu(fun func(m *Scene)) {
	tb.OverflowMenus = append(tb.OverflowMenus, fun)
}

// SetShortcuts sets the shortcuts to window associated with Toolbar
func (tb *Toolbar) SetShortcuts() {
	em := tb.EventMgr()
	if em == nil {
		return
	}
	for _, mi := range tb.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			bt := AsButton(mi)
			em.AddShortcut(bt.Shortcut, bt)
		}
	}
}

func (tb *Toolbar) Destroy() {
	tb.DeleteShortcuts()
	tb.Frame.Destroy()
}

// DeleteShortcuts deletes the shortcuts -- called when destroyed
func (tb *Toolbar) DeleteShortcuts() {
	em := tb.EventMgr()
	if em == nil {
		return
	}
	for _, mi := range tb.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			em.DeleteShortcut(ac.Shortcut, ac)
		}
	}
}

// UpdateButtons calls UpdateFunc on all buttons in toolbar.
// individual menus are automatically generated at popup time.
func (tb *Toolbar) UpdateButtons() {
	if tb == nil {
		return
	}
	updt := tb.UpdateStart()
	defer tb.UpdateEndRender(updt)

	for _, mi := range tb.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			ac.UpdateButtons()
		}
	}
}

// TODO(kai/menu): figure out what to do here
/*
// FindButtonByName finds an button on the toolbar, or any sub-menu, with
// given name (exact match) -- this is not the Text label but the Name of the
// element (for AddButton items, this is the same as Label or Icon (if Label
// is empty)) -- returns false if not found
func (tb *Toolbar) FindButtonByName(name string) (*Button, bool) {
	for _, mi := range tb.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			if ac.Name() == name {
				return ac, true
			}
			if ac.Menu != nil {
				if sac, ok := ac.Menu.FindButtonByName(name); ok {
					return sac, ok
				}
			}
		}
	}
	return nil, false
}
*/

//////////////////////////////////////////////////////////////////////////////
// 	ToolbarFuncs

// ToolbarFuncs are functions for creating control bars,
// attached to different sides of a Scene (e.g., TopAppBar at Top,
// NavBar at Bottom, etc).  Functions are called in forward order
// so first added are called first.
type ToolbarFuncs []func(tb *Toolbar)

// Add adds the given function for configuring a toolbar
func (bf *ToolbarFuncs) Add(fun func(tb *Toolbar)) *ToolbarFuncs {
	*bf = append(*bf, fun)
	return bf
}

// Call calls all the functions for configuring given toolbar
func (bf *ToolbarFuncs) Call(tb *Toolbar) {
	for _, fun := range *bf {
		fun(tb)
	}
}

// IsEmpty returns true if there are no functions added
func (bf *ToolbarFuncs) IsEmpty() bool {
	return len(*bf) == 0
}

//////////////////////////////////////////////////////////////////////////////
// 	ToolbarStyles

// ToolbarStyles can be applied to any layout (e.g., Frame) to achieve
// standard toolbar styling.
func ToolbarStyles(ly Layouter) {
	lb := ly.AsLayout()
	ly.Style(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Border.Radius = styles.BorderRadiusFull
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)
		s.Padding.SetHoriz(units.Dp(16))
		s.Gap.Zero()
	})
	ly.OnWidgetAdded(func(w Widget) {
		if bt := AsButton(w); bt != nil {
			bt.Type = ButtonAction
			return
		}
		if sp, ok := w.(*Separator); ok {
			sp.Horiz = lb.Styles.Direction != styles.Row
		}
	})
}
