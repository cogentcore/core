// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log/slog"

	"goki.dev/colors"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

var (
	// DefaultTopAppBar is the function that makes default elements for
	// The TopAppBar, called by most code.  Set to your own function
	// to customize, or set to nil to prevent any default.
	DefaultTopAppBar = DefaultTopAppBarStd
)

// DefaultTopAppBarStd is the standard impl for a [Scene.TopAppBar].
// It adds navigation buttons and an editable chooser bar,
// and calls AddDefaultOverflowMenu to provide default menu items,
// which will appear below any other OverflowMenu items added.
func DefaultTopAppBarStd(tb *TopAppBar) { //gti:add
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
	ch := NewChooser(tb, "nav-bar").SetEditable(true).SetType(ChooserOutlined)
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
	ch.Style(func(s *styles.Style) {
		s.Border.Radius = styles.BorderRadiusFull
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerHighest)
		if !s.Is(states.Focused) {
			s.Border.Width.Zero()
		}
	})
	tb.AddDefaultOverflowMenu()
}

// TopAppBar is a [Frame] that is useful for holding [Button]s that do things.
// It automatically moves items that do not fit into an overflow menu, and
// manages additional items that are always placed onto this overflow menu.
// Set the Scene.TopAppBar to a toolbar function
// In general it should be possible to use a single toolbar + overflow to
// manage all an app's functionality, in a way that is portable across
// mobile and desktop environments.
type TopAppBar struct { //goki:embedder
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

// TopAppBarer is an interface that types can satisfy to add a toolbar when they
// are displayed in the GUI. In the TopAppBar method, types typically add [goki.dev/gi/v2/giv.FuncButton]
// and [gi.Separator] objects to the toolbar that they are passed, although they can
// do anything they want. [TopAppBarFor] checks for implementation of this interface.
type TopAppBarer interface {
	TopAppBar(tb *TopAppBar)
}

// TopAppBarFor returns the TopAppBar function of the given value on the given toolbar,
// if the given value is implements the [TopAppBarer] interface, else nil.
func TopAppBarFor(val any) func(tb *TopAppBar) {
	tbr, ok := val.(TopAppBarer)
	if !ok {
		return nil
	}
	return tbr.TopAppBar
}

func (tb *TopAppBar) CopyFieldsFrom(frm any) {
	fr := frm.(*TopAppBar)
	tb.Frame.CopyFieldsFrom(&fr.Frame)
}

func (tb *TopAppBar) OnInit() {
	tb.TopAppBarStyles()
	tb.HandleLayoutEvents()
}

func (tb *TopAppBar) TopAppBarStyles() {
	ToolbarStyles(tb)
}

func (tb *TopAppBar) IsVisible() bool {
	// do not render toolbars with no buttons
	return tb.WidgetBase.IsVisible() && len(tb.Kids) > 0
}

func (tb *TopAppBar) SizeUp(sc *Scene) {
	tb.AllItemsToChildren(sc)
	tb.Frame.SizeUp(sc)
	ma := tb.Styles.MainAxis
	sz := &tb.Alloc.Size                    // reset for others
	sz.Content.SetDim(ma, sz.Space.Dim(ma)) // reset for others still
	sz.SetTotalFromContent()
}

// don't report any size at sizeup, etc

// AllItemsToChildren moves the overflow items back to the children,
// so the full set is considered for the next layout round,
// and ensures the overflow button is made and moves it
// to the end of the list.
func (tb *TopAppBar) AllItemsToChildren(sc *Scene) {
	if len(tb.OverflowItems) == 0 && !tb.HasChildren() {
		return
	}
	if len(tb.OverflowItems) > 0 {
		tb.Kids = append(tb.Kids, tb.OverflowItems...)
		tb.OverflowItems = nil
	}
	if tb.OverflowButton == nil {
		ic := icons.MoreVert
		if tb.Styles.MainAxis != mat32.X {
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

func (tb *TopAppBar) SizeDown(sc *Scene, iter int) bool {
	if iter == 0 || !tb.HasChildren() { // first do a normal layout to get everyone's target positions
		tb.Frame.SizeDown(sc, iter)
		ma := tb.Styles.MainAxis
		sz := &tb.Alloc.Size
		sz.Content.SetDim(ma, sz.Space.Dim(ma)) // reset for others still
		sz.SetTotalFromContent()
		return true // needs another iter
	}
	if iter == 1 {
		tb.MoveToOverflow(sc)
		tb.Frame.SizeDown(sc, iter)
		return true
	}
	return tb.Frame.SizeDown(sc, iter)
}

// MoveToOverflow moves overflow out of children to the OverflowItems list
func (tb *TopAppBar) MoveToOverflow(sc *Scene) {
	ma := tb.Styles.MainAxis
	// note: the ScBBox is intersected with parents actual display size
	_, pwb := tb.ParentWidget()
	psz := pwb.Alloc.Size.Content.Sub(tb.Alloc.Size.Space)
	avail := psz.Dim(ma)
	// fmt.Println(pwb, pwb.Alloc.Size)
	ovsz := tb.OverflowButton.Alloc.Size.Total.Dim(ma)
	avsz := avail - ovsz
	tb.Alloc.Size.Alloc.SetDim(ma, avail)
	tb.Alloc.Size.Total.SetDim(ma, avail)
	tb.Alloc.Size.Content.SetDim(ma, avail)
	tb.OverflowItems = nil
	n := len(tb.Kids)
	ovidx := n - 1
	hasOv := false
	szsum := float32(0)
	tb.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		if i >= n-1 {
			return ki.Break
		}
		ksz := kwb.Alloc.Size.Total.Dim(ma)
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
}

// OverflowMenu is the overflow menu function
func (tb *TopAppBar) OverflowMenu(m *Scene) {
	nm := len(tb.OverflowMenus)
	if len(tb.OverflowItems) > 0 {
		for _, k := range tb.OverflowItems {
			if k.This() == tb.OverflowButton.This() {
				continue
			}
			cl := k.This().Clone()
			m.AddChild(cl)
			cl.This().(Widget).Config(m)
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
func (tb *TopAppBar) AddOverflowMenu(fun func(m *Scene)) {
	tb.OverflowMenus = append(tb.OverflowMenus, fun)
}

// AddDefaultOverflowMenu adds the default menu function to overflow menu list,
// typically at the end.
func (tb *TopAppBar) AddDefaultOverflowMenu() {
	tb.OverflowMenus = append(tb.OverflowMenus, tb.DefaultOverflowMenu)
}

// DefaultOverflowMenu adds standard default overflow menu items.
// Typically you will want to add additional items and then call this function.
func (tb *TopAppBar) DefaultOverflowMenu(m *Scene) {
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
		NewButton(m).SetText("Focus next").SetIcon(icons.CenterFocusStrong).
			OnClick(func(e events.Event) {
				AllRenderWins.FocusNext()
			})
		NewButton(m).SetText("Minimize").SetIcon(icons.Minimize).
			OnClick(func(e events.Event) {
				win := tb.Sc.RenderWin()
				if win != nil {
					win.Minimize()
				}
			})
		NewSeparator(m)
		NewButton(m).SetText("Close Window").SetIcon(icons.Close).SetKey(keyfun.WinClose).
			OnClick(func(e events.Event) {
				win := tb.Sc.RenderWin()
				if win != nil {
					win.CloseReq()
				}
			})
		NewButton(m).SetText("Quit").SetIcon(icons.Close).SetShortcut("Command+Q").
			OnClick(func(e events.Event) {
				QuitReq()
			})
		NewSeparator(m)
		for _, w := range MainRenderWins {
			if w != nil {
				NewButton(m).SetText(w.Title).OnClick(func(e events.Event) {
					w.Raise()
				})
			}
		}
		if len(DialogRenderWins) > 0 {
			NewSeparator(m)
			for _, w := range DialogRenderWins {
				if w != nil {
					NewButton(m).SetText(w.Title).OnClick(func(e events.Event) {
						w.Raise()
					})
				}
			}
		}
	})
}

// SetShortcuts sets the shortcuts to window associated with TopAppBar
func (tb *TopAppBar) SetShortcuts() {
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

func (tb *TopAppBar) Destroy() {
	tb.DeleteShortcuts()
	tb.Frame.Destroy()
}

// DeleteShortcuts deletes the shortcuts -- called when destroyed
func (tb *TopAppBar) DeleteShortcuts() {
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
func (tb *TopAppBar) UpdateButtons() {
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
func (tb *TopAppBar) FindButtonByName(name string) (*Button, bool) {
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
