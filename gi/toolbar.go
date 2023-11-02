// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log/slog"

	"goki.dev/colors"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// DefaultTopAppBar is the default value for [Scene.TopAppBar].
// It adds navigation buttons and an editable chooser bar.
func DefaultTopAppBar(tb *Toolbar) { //gti:add
	NewButton(tb).SetIcon(icons.ArrowBack).OnClick(func(e events.Event) {
		stg := tb.Sc.MainStage()
		mm := stg.StageMgr
		if mm == nil {
			slog.Error("Top app bar has no MainMgr")
			return
		}
		// if we are down to the last window, we don't
		// let people close it with the back button
		if mm.Stack.Len() <= 1 {
			return
		}
		if stg.NewWindow {
			mm.RenderWin.CloseReq()
			return
		}
		mm.PopDeleteType(stg.Type)
	})
	// NewButton(tb).SetIcon(icons.ArrowForward)
	ch := NewChooser(tb).SetEditable(true)
	ch.SetItemsFunc(func() {
		stg := tb.Sc.MainStage()
		mm := stg.StageMgr
		if mm == nil {
			slog.Error("Top app bar has no MainMgr")
			return
		}
		ch.Items = make([]any, mm.Stack.Len())
		for i, kv := range mm.Stack.Order {
			ch.Items[i] = kv.Val.AsBase().Scene.Name()
			if kv.Val == stg {
				ch.SetCurIndex(i)
			}
		}
	})
	ch.OnChange(func(e events.Event) {
		stg := tb.Sc.MainStage()
		mm := stg.StageMgr
		if mm == nil {
			slog.Error("Top app bar has no MainMgr")
			return
		}
		// TODO: optimize this?
		kv := mm.Stack.Order[ch.CurIndex]
		mm.Stack.DeleteIdx(ch.CurIndex, ch.CurIndex+1)
		mm.Stack.InsertAtIdx(mm.Stack.Len(), kv.Key, kv.Val)
	})
	NewSeparator(tb)
}

// Toolbar is a [Frame] that is useful for holding [Button]s that do things.
type Toolbar struct { //goki:embedder
	Frame

	// items moved from the main toolbar
	OverflowItems ki.Slice `set:"-" json:"-" xml:"-"`

	// menu functions for overflow
	OverflowMenus []func(m *Scene) `set:"-" json:"-" xml:"-"`

	// This is the overflow button
	OverflowButton *Button
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

// DoLayoutAlloc moves overflow to the end of children for layout
func (tb *Toolbar) DoLayoutAlloc(sc *Scene, iter int) bool {
	if !tb.HasChildren() {
		return tb.Frame.DoLayoutAlloc(sc, iter)
	}
	if iter == 0 {
		tb.ToolbarLayoutIter0(sc)
		tb.Frame.DoLayoutAlloc(sc, iter)
		return true // needs another iter
	} else {
		tb.ToolbarLayoutIter1(sc)
		tb.Frame.DoLayoutAlloc(sc, iter)
		return false
	}
}

func (tb *Toolbar) GetSize(sc *Scene, iter int) {
	if iter == 0 {
		if len(tb.OverflowItems) > 0 {
			tb.Kids = append(tb.Kids, tb.OverflowItems...)
			tb.OverflowItems = nil
		}
	}
	tb.Frame.GetSize(sc, iter)
}

func (tb *Toolbar) ToolbarLayoutIter0(sc *Scene) {
	if tb.OverflowButton == nil {
		ic := icons.MoreVert
		if tb.Lay != LayoutHoriz {
			ic = icons.MoreHoriz
		}
		tb.OverflowButton = NewButton(tb, "overflow-menu").SetIcon(ic).
			SetTooltip("Overflow toolbar items and additional menu items")
		tb.OverflowButton.Menu = tb.OverflowMenu
		tb.OverflowButton.Config(sc)
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
}

func (tb *Toolbar) ToolbarLayoutIter1(sc *Scene) {
	ldim := LaySummedDim(tb.Lay)
	avail := tb.ScBBox.Max
	ovsz := tb.OverflowButton.BBox.Size()
	avsz := avail.Sub(ovsz)
	dmx := float32(avsz.X)
	if ldim == mat32.Y {
		dmx = float32(avsz.Y)
	}

	tb.OverflowItems = nil
	n := len(tb.Kids)
	ovidx := n - 1
	hasOv := false
	for i := 0; i < n-1; i++ {
		k := tb.Kids[i]
		_, wb := AsWidget(k)
		wbbm := mat32.NewVec2FmPoint(wb.BBox.Max)
		wdmx := wbbm.Dim(ldim)
		ov := wdmx > dmx
		if ov {
			if !hasOv {
				ovidx = i
				hasOv = true
			}
			tb.OverflowItems = append(tb.OverflowItems, k)
		}
	}
	if ovidx != n-1 {
		tb.Kids.Move(n-1, ovidx)
		tb.Kids = tb.Kids[:ovidx+1]
	}
}

// ManageOverflow processes any overflow according to overflow settings.
func (tb *Toolbar) ManageOverflow(sc *Scene) {
	tb.SetFlag(true, LayoutScrollsOff)
	tb.ExtraSize.SetScalar(0)
	for d := mat32.X; d <= mat32.Y; d++ {
		tb.HasScroll[d] = false
	}
	// todo: move others out of range
}

// OverflowMenu is the overflow menu function
func (tb *Toolbar) OverflowMenu(m *Scene) {
	if len(tb.OverflowItems) > 0 {
		for _, k := range tb.OverflowItems {
			if k.This() == tb.OverflowButton.This() {
				continue
			}
			cl := k.This().Clone()
			m.AddChild(cl)
			cl.This().(Widget).Config(m)
		}
		NewSeparator(m)
	}
	for _, fn := range tb.OverflowMenus {
		fn(m)
	}
}

// AddOverflowMenu adds given menu function to overflow menu list
func (tb *Toolbar) AddOverflowMenu(fun func(m *Scene)) {
	tb.OverflowMenus = append(tb.OverflowMenus, fun)
}

// AddDefaultOverflowMenu adds the default menu function to overflow menu list,
// typically at the end.
func (tb *Toolbar) AddDefaultOverflowMenu() {
	tb.OverflowMenus = append(tb.OverflowMenus, tb.DefaultOverflowMenu)
}

func (tb *Toolbar) DefaultOverflowMenu(m *Scene) {
	NewSeparator(m)
	NewButton(m).SetText("System preferences").SetIcon(icons.Settings).SetKey(keyfun.Prefs).
		OnClick(func(e events.Event) {
			TheViewIFace.PrefsView(&Prefs)
		})
	NewButton(m).SetText("Inspect").SetIcon(icons.Edit).SetKey(keyfun.GoGiEditor).
		OnClick(func(e events.Event) {
			TheViewIFace.GoGiEditor(tb.Sc)
		})
	NewButton(m).SetText("Edit").SetMenu(func(m *Scene) {
		NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keyfun.Copy)
		NewButton(m).SetText("Cut").SetIcon(icons.ContentCut).SetKey(keyfun.Cut)
		NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).SetKey(keyfun.Paste)
	})
	NewButton(m).SetText("Window").SetMenu(func(m *Scene) {
		NewButton(m).SetText("Focus next").SetIcon(icons.CenterFocusStrong)
		NewButton(m).SetText("Minimize").SetIcon(icons.Minimize)
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
