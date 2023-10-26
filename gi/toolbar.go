// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
)

// Toolbar is a [Frame] that is useful for holding [Button]s that do things.
type Toolbar struct { //goki:embedder
	Frame
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

// SetShortcuts sets the shortcuts to window associated with Toolbar
func (tb *Toolbar) SetShortcuts() {
	em := tb.EventMgr()
	if em == nil {
		return
	}
	for _, mi := range tb.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			em.AddShortcut(ac.Shortcut, ac)
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
