// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/icons"
)

// TODO(kai): support AutoOverflowMenu for Toolbar

// Toolbar is a [Frame] that is useful for holding [Button]s that do things.
type Toolbar struct { //goki:embedder
	Frame
}

// Toolbarer is an interface that types can satisfy to add a toolbar when they
// are displayed in the GUI. In the Toolbar method, types typically add [goki.dev/gi/v2/giv.FuncButton]
// and [gi.Separator] objects to the toolbar that they are passed, although they can
// do anything they want. [ToolbarFor] checks for implementation of this interface.
type Toolbarer interface {
	Toolbar(tb *Toolbar)
}

// ToolbarFor calls the Toolbar function of the given value on the given toolbar,
// if the given value is implements the [Toolbarer] interface. Otherwise, it does
// nothing. It returns whether the given value implements that interface.
func ToolbarFor(val any, tb *Toolbar) bool {
	tbr, ok := val.(Toolbarer)
	if !ok {
		return false
	}
	tbr.Toolbar(tb)
	return true
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
	tb.Style(func(s *styles.Style) {
		s.SetStretchMaxWidth()
		s.Border.Radius = styles.BorderRadiusFull
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)
		s.Margin.Set(units.Dp(4))
		s.Padding.SetHoriz(units.Dp(16))
	})
	tb.OnWidgetAdded(func(w Widget) {
		if bt := AsButton(w); bt != nil {
			bt.Type = ButtonAction
			return
		}
		if sp, ok := w.(*Separator); ok {
			sp.Horiz = tb.Lay != LayoutHoriz
		}
	})
}

func (tb *Toolbar) IsVisible() bool {
	// do not render toolbars with no buttons
	return tb.WidgetBase.IsVisible() && len(tb.Kids) > 0
}

// OverflowMenu returns the overflow menu element, a button on the end of the toolbar
// that has a menu containing all of the toolbar buttons that either don't fit or are
// too low-frequency to go in the main toolbar. If the overflow menu button doesn't
// already exist, it makes it and a separator separating it from the rest of the toolbar.
// OverflowMenu is designed to be used by end-user code; for example:
//
//	tb.OverflowMenu().SetMenu(func(m *gi.Scene) {
//		giv.NewFuncButton(m, me.SomethingLowFrequency)
//	})
func (tb *Toolbar) OverflowMenu() *Button {
	if om, ok := tb.ChildByName("overflow-menu").(*Button); ok {
		return om
	}
	NewSeparator(tb, "overflow-menu-separator")
	ic := icons.MoreVert
	if tb.Lay != LayoutHoriz {
		ic = icons.MoreHoriz
	}
	return NewButton(tb, "overflow-menu").SetIcon(ic).SetTooltip("More")
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

// UpdateButtons calls UpdateFunc on all buttons in toolbar -- individual
// menus are automatically updated just prior to menu popup
func (tb *Toolbar) UpdateButtons() {
	if tb == nil {
		return
	}
	updt := tb.UpdateStart()
	defer tb.UpdateEnd(updt)
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
