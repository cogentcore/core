// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log"
	"sync"

	"goki.dev/colors"
	"goki.dev/girl/gist"
	"goki.dev/girl/units"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/ki/v2"
)

type TabViewEmbedder interface {
	AsTabView() *TabView
}

func AsTabView(k ki.Ki) *TabView {
	if k == nil || k.This() == nil {
		return nil
	}
	if ac, ok := k.(TabViewEmbedder); ok {
		return ac.AsTabView()
	}
	return nil
}

func (ac *TabView) AsTabView() *TabView {
	return ac
}

// TabView switches among child widgets via tabs.  The selected widget gets
// the full allocated space avail after the tabs are accounted for.  The
// TabView is just a Vertical layout that manages two child widgets: a
// HorizFlow Layout for the tabs (which can flow across multiple rows as
// needed) and a Stacked Frame that actually contains all the children, and
// provides scrollbars as needed to any content within.  Typically should have
// max stretch and a set preferred size, so it expands.
type TabView struct {
	Layout

	// maximum number of characters to include in tab label -- elides labels that are longer than that
	MaxChars int `desc:"maximum number of characters to include in tab label -- elides labels that are longer than that"`

	// signal for tab widget -- see TabViewSignals for the types
	TabViewSig ki.Signal `copy:"-" json:"-" xml:"-" desc:"signal for tab widget -- see TabViewSignals for the types"`

	// show a new tab button at right of list of tabs
	NewTabButton bool `desc:"show a new tab button at right of list of tabs"`

	// if true, tabs are not user-deleteable
	NoDeleteTabs bool `desc:"if true, tabs are not user-deleteable"`

	// type of widget to create in a new tab via new tab button -- Frame by default
	NewTabType *gti.Type `desc:"type of widget to create in a new tab via new tab button -- Frame by default"`

	// [view: -] mutex protecting updates to tabs -- tabs can be driven programmatically and via user input so need extra protection
	Mu sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" desc:"mutex protecting updates to tabs -- tabs can be driven programmatically and via user input so need extra protection"`
}

// event functions for this type
var TabViewEventFuncs WidgetEvents

func (tv *TabView) OnInit() {
	tv.AddEvents(&TabViewEventFuncs)
	tv.AddStyler(func(w *WidgetBase, s *gist.Style) {
		// need border for separators (see RenderTabSeps)
		// TODO: maybe better solution for tab sep styles?
		s.Border.Style.Set(gist.BorderSolid)
		s.Border.Width.Set(units.Px(1))
		s.Border.Color.Set(colors.Scheme.OutlineVariant)
		s.BackgroundColor.SetSolid(colors.Scheme.Background)
		s.Color = colors.Scheme.OnBackground
		s.MaxWidth.SetPx(-1)
		s.MaxHeight.SetPx(-1)
	})
}

func (tv *TabView) OnChildAdded(child ki.Ki) {
	if _, wb := AsWidget(child); wb != nil {
		switch wb.Name() {
		case "tabs":
			wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.SetStretchMaxWidth()
				s.Height.SetEm(1.8)
				s.Overflow = gist.OverflowHidden // no scrollbars!
				s.Margin.Set()
				s.Padding.Set()
				// tabs.Spacing.SetPx(4 * Prefs.DensityMul())
				s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)

				// s.Border.Style.Set(gist.BorderNone)
				// s.Border.Style.Bottom = gist.BorderSolid
				// s.Border.Width.Bottom.SetPx(1)
				// s.Border.Color.Bottom = colors.Scheme.OutlineVariant
			})
		case "frame":
			frame := child.(*Frame)
			frame.StackTopOnly = true // key for allowing each tab to have its own size
			wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.SetMinPrefWidth(units.Em(10))
				s.SetMinPrefHeight(units.Em(6))
				s.SetStretchMax()
			})
		}
	}
}

func (tv *TabView) CopyFieldsFrom(frm any) {
	fr := frm.(*TabView)
	tv.Layout.CopyFieldsFrom(&fr.Layout)
	tv.MaxChars = fr.MaxChars
	tv.NewTabButton = fr.NewTabButton
	tv.NewTabType = fr.NewTabType
}

func (tv *TabView) Disconnect() {
	tv.Layout.Disconnect()
	tv.TabViewSig.DisconnectAll()
}

// NTabs returns number of tabs
func (tv *TabView) NTabs() int {
	fr := tv.Frame()
	if fr == nil {
		return 0
	}
	return len(fr.Kids)
}

// CurTab returns currently-selected tab, and its index -- returns false none
func (tv *TabView) CurTab() (Widget, int, bool) {
	if tv.NTabs() == 0 {
		return nil, -1, false
	}
	tv.Mu.Lock()
	defer tv.Mu.Unlock()
	fr := tv.Frame()
	if fr.StackTop < 0 {
		return nil, -1, false
	}
	widg := fr.Child(fr.StackTop).(Widget)
	return widg, fr.StackTop, true
}

// AddTab adds a widget as a new tab, with given tab label, and returns the
// index of that tab
func (tv *TabView) AddTab(widg Widget, label string) int {
	fr := tv.Frame()
	idx := len(*fr.Children())
	tv.InsertTab(widg, label, idx)
	return idx
}

// InsertTabOnlyAt inserts just the tab at given index -- after panel has
// already been added to frame -- assumed to be wrapped in update.  Generally
// for internal use.
func (tv *TabView) InsertTabOnlyAt(widg Widget, label string, idx int) {
	tb := tv.Tabs()
	tb.SetChildAdded()
	tab := tb.InsertNewChild(TabButtonType, idx, label).(*TabButton)
	tab.Data = idx
	tab.Tooltip = label
	tab.NoDelete = tv.NoDeleteTabs
	tab.SetText(label)
	tab.ActionSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data any) {
		tvv := AsTabView(recv)
		act := send.(*TabButton)
		tabIdx := act.Data.(int)
		tvv.SelectTabIndexAction(tabIdx)
	})
	fr := tv.Frame()
	if len(fr.Kids) == 1 {
		fr.StackTop = 0
		tab.SetSelected(true)
	} else {
		widg.SetFlag(true, Invisible) // new tab is invisible until selected
	}
}

// InsertTab inserts a widget into given index position within list of tabs
func (tv *TabView) InsertTab(widg Widget, label string, idx int) {
	tv.Mu.Lock()
	fr := tv.Frame()
	updt := tv.UpdateStart()
	fr.SetChildAdded()
	fr.InsertChild(widg, idx)
	tv.InsertTabOnlyAt(widg, label, idx)
	tv.Mu.Unlock()
	tv.UpdateEndLayout(updt)
}

// NewTab adds a new widget as a new tab of given widget type, with given
// tab label, and returns the new widget
func (tv *TabView) NewTab(typ *gti.Type, label string) Widget {
	fr := tv.Frame()
	idx := len(*fr.Children())
	widg := tv.InsertNewTab(typ, label, idx)
	return widg
}

// NewTabLayout adds a new widget as a new tab of given widget type,
// with given tab label, and returns the new widget.
// A Layout is added first and the widget is added to that layout.
// The Layout has "-lay" suffix added to name.
func (tv *TabView) NewTabLayout(typ *gti.Type, label string) (Widget, *Layout) {
	ly := tv.NewTab(LayoutType, label).(*Layout)
	ly.SetName(label + "-lay")
	widg := ly.NewChild(typ, label).(Widget)
	return widg, ly
}

// NewTabFrame adds a new widget as a new tab of given widget type,
// with given tab label, and returns the new widget.
// A Frame is added first and the widget is added to that Frame.
// The Frame has "-frame" suffix added to name.
func (tv *TabView) NewTabFrame(typ *gti.Type, label string) (Widget, *Frame) {
	fr := tv.NewTab(FrameType, label).(*Frame)
	fr.SetName(label + "-frame")
	widg := fr.NewChild(typ, label).(Widget)
	return widg, fr
}

// NewTabAction adds a new widget as a new tab of given widget type, with given
// tab label, and returns the new widget -- emits TabAdded signal
func (tv *TabView) NewTabAction(typ *gti.Type, label string) Widget {
	updt := tv.UpdateStart()
	widg := tv.NewTab(typ, label)
	fr := tv.Frame()
	idx := len(*fr.Children()) - 1
	tv.TabViewSig.Emit(tv.This(), int64(TabAdded), idx)
	tv.UpdateEndLayout(updt)
	return widg
}

// InsertNewTab inserts a new widget of given type into given index position
// within list of tabs, and returns that new widget
func (tv *TabView) InsertNewTab(typ *gti.Type, label string, idx int) Widget {
	updt := tv.UpdateStart()
	fr := tv.Frame()
	fr.SetChildAdded()
	widg := fr.InsertNewChild(typ, idx, label).(Widget)
	tv.InsertTabOnlyAt(widg, label, idx)
	tv.UpdateEndLayout(updt)
	return widg
}

// TabAtIndex returns content widget and tab button at given index, false if
// index out of range (emits log message)
func (tv *TabView) TabAtIndex(idx int) (Widget, *TabButton, bool) {
	tv.Mu.Lock()
	defer tv.Mu.Unlock()

	fr := tv.Frame()
	tb := tv.Tabs()
	sz := len(*fr.Children())
	if idx < 0 || idx >= sz {
		log.Printf("giv.TabView: index %v out of range for number of tabs: %v\n", idx, sz)
		return nil, nil, false
	}
	tab := tb.Child(idx).(*TabButton)
	widg := fr.Child(idx).(Widget)
	return widg, tab, true
}

// SelectTabIndex selects tab at given index, returning it -- returns false if
// index is invalid
func (tv *TabView) SelectTabIndex(idx int) (Widget, bool) {
	widg, tab, ok := tv.TabAtIndex(idx)
	if !ok {
		return nil, false
	}
	fr := tv.Frame()
	if fr.StackTop == idx {
		return widg, true
	}
	tv.Mu.Lock()
	updt := tv.UpdateStart()
	tv.UnselectOtherTabs(idx)
	tab.SetSelected(true)
	fr.StackTop = idx
	tv.Mu.Unlock()
	tv.UpdateEndLayout(updt)
	return widg, true
}

// SelectTabIndexAction selects tab at given index and emits selected signal,
// with the index of the selected tab -- this is what is called when a tab is
// clicked
func (tv *TabView) SelectTabIndexAction(idx int) {
	_, ok := tv.SelectTabIndex(idx)
	if ok {
		tv.TabViewSig.Emit(tv.This(), int64(TabSelected), idx)
	}
}

// TabByName returns tab with given name (nil if not found -- see TabByNameTry)
func (tv *TabView) TabByName(label string) Widget {
	t, _ := tv.TabByNameTry(label)
	return t
}

// TabByNameTry returns tab with given name, and an error if not found.
func (tv *TabView) TabByNameTry(label string) (Widget, error) {
	tv.Mu.Lock()
	defer tv.Mu.Unlock()

	tb := tv.Tabs()
	idx, ok := tb.Children().IndexByName(label, 0)
	if !ok {
		return nil, fmt.Errorf("gi.TabView: Tab named %v not found in %v", label, tv.Path())
	}
	fr := tv.Frame()
	widg := fr.Child(idx).(Widget)
	return widg, nil
}

// TabIndexByName returns tab index for given tab name, and an error if not found.
func (tv *TabView) TabIndexByName(label string) (int, error) {
	tv.Mu.Lock()
	defer tv.Mu.Unlock()

	tb := tv.Tabs()
	idx, ok := tb.Children().IndexByName(label, 0)
	if !ok {
		return -1, fmt.Errorf("gi.TabView: Tab named %v not found in %v", label, tv.Path())
	}
	return idx, nil
}

// TabName returns tab name at given index
func (tv *TabView) TabName(idx int) string {
	tv.Mu.Lock()
	defer tv.Mu.Unlock()

	tb := tv.Tabs()
	tbut, err := tb.ChildTry(idx)
	if err != nil {
		return ""
	}
	return tbut.Name()
}

// SelectTabByName selects tab by name, returning it.
func (tv *TabView) SelectTabByName(label string) Widget {
	idx, err := tv.TabIndexByName(label)
	if err == nil {
		tv.SelectTabIndex(idx)
		fr := tv.Frame()
		return fr.Child(idx).(Widget)
	}
	return nil
}

// SelectTabByNameTry selects tab by name, returning it.  Returns error if not found.
func (tv *TabView) SelectTabByNameTry(label string) (Widget, error) {
	idx, err := tv.TabIndexByName(label)
	if err == nil {
		tv.SelectTabIndex(idx)
		fr := tv.Frame()
		return fr.Child(idx).(Widget), nil
	}
	return nil, err
}

// RecycleTab returns a tab with given name, first by looking for an existing one,
// and if not found, making a new one with widget of given type.
// If sel, then select it.  returns widget for tab.
func (tv *TabView) RecycleTab(label string, typ *gti.Type, sel bool) Widget {
	widg, err := tv.TabByNameTry(label)
	if err == nil {
		if sel {
			tv.SelectTabByName(label)
		}
		return widg
	}
	widg = tv.NewTab(typ, label)
	if sel {
		tv.SelectTabByName(label)
	}
	return widg
}

// DeleteTabIndex deletes tab at given index, optionally calling destroy on
// tab contents -- returns widget if destroy == false, tab name, and bool success
func (tv *TabView) DeleteTabIndex(idx int, destroy bool) (Widget, string, bool) {
	widg, _, ok := tv.TabAtIndex(idx)
	if !ok {
		return nil, "", false
	}

	tnm := tv.TabName(idx)
	tv.Mu.Lock()
	fr := tv.Frame()
	sz := len(*fr.Children())
	tb := tv.Tabs()
	updt := tv.UpdateStart()
	nxtidx := -1
	if fr.StackTop == idx {
		if idx > 0 {
			nxtidx = idx - 1
		} else if idx < sz-1 {
			nxtidx = idx
		}
	}
	fr.DeleteChildAtIndex(idx, destroy)
	tb.DeleteChildAtIndex(idx, ki.DestroyKids) // always destroy -- we manage
	tv.RenumberTabs()
	tv.Mu.Unlock()
	if nxtidx >= 0 {
		tv.SelectTabIndex(nxtidx)
	}
	tv.UpdateEndLayout(updt)
	if destroy {
		return nil, tnm, true
	} else {
		return widg, tnm, true
	}
}

// DeleteTabIndexAction deletes tab at given index using destroy flag, and
// emits TabDeleted signal with name of deleted tab
// this is called by the delete button on the tab
func (tv *TabView) DeleteTabIndexAction(idx int) {
	_, tnm, ok := tv.DeleteTabIndex(idx, true)
	if ok {
		tv.TabViewSig.Emit(tv.This(), int64(TabDeleted), tnm)
	}
}

// ConfigNewTabButton configures the new tab + button at end of list of tabs
func (tv *TabView) ConfigNewTabButton(sc *Scene) bool {
	sz := tv.NTabs()
	tb := tv.Tabs()
	ntb := len(tb.Kids)
	if tv.NewTabButton {
		if ntb == sz+1 {
			return false
		}
		if tv.NewTabType == nil {
			tv.NewTabType = FrameType
		}
		tab := tb.InsertNewChild(ActionType, ntb, "new-tab").(*Action)
		tab.Data = -1
		tab.SetIcon(icons.Add)
		tab.ActionSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data any) {
			tvv := AsTabView(recv)
			tvv.NewTabAction(tvv.NewTabType, "New Tab")
			tvv.SelectTabIndex(len(*tvv.Frame().Children()) - 1)
		})
		return true
	} else {
		if ntb == sz {
			return false
		}
		tb.DeleteChildAtIndex(ntb-1, ki.DestroyKids) // always destroy -- we manage
		return true
	}
}

// TabViewSignals are signals that the TabView can send
type TabViewSignals int64

const (
	// TabSelected indicates tab was selected -- data is the tab index
	TabSelected TabViewSignals = iota

	// TabAdded indicates tab was added -- data is the tab index
	TabAdded

	// TabDeleted indicates tab was deleted -- data is the tab name
	TabDeleted

	TabViewSignalsN
)

// ConfigWidget initializes the tab widget children if it hasn't been done yet.
// only the 2 primary children (Frames) need to be configured.
// no re-config needed when adding / deleting tabs -- just new layout.
func (tv *TabView) ConfigWidget(sc *Scene) {
	if len(tv.Kids) != 0 {
		return
	}
	tv.Lay = LayoutVert
	tv.SetFlag(true, ReRenderAnchor)

	frame := NewFrame(tv, "tabs")
	frame.Lay = LayoutHorizFlow

	frame = NewFrame(tv, "frame")
	frame.Lay = LayoutStacked
	frame.SetFlag(true, ReRenderAnchor)

	tv.ConfigNewTabButton(sc)
}

// Tabs returns the layout containing the tabs -- the first element within us
func (tv *TabView) Tabs() *Frame {
	return tv.Child(0).(*Frame)
}

// Frame returns the stacked frame layout -- the second element
func (tv *TabView) Frame() *Frame {
	return tv.Child(1).(*Frame)
}

// UnselectOtherTabs turns off all the tabs except given one
func (tv *TabView) UnselectOtherTabs(idx int) {
	sz := tv.NTabs()
	tbs := tv.Tabs()
	for i := 0; i < sz; i++ {
		if i == idx {
			continue
		}
		tb := tbs.Child(i).(*TabButton)
		if tb.IsSelected() {
			tb.SetSelected(false)
		}
	}
}

// RenumberTabs assigns proper index numbers to each tab
func (tv *TabView) RenumberTabs() {
	sz := tv.NTabs()
	tbs := tv.Tabs()
	for i := 0; i < sz; i++ {
		tb := tbs.Child(i).(*TabButton)
		tb.Data = i
	}
}

// RenderTabSeps renders the separators between tabs
func (tv *TabView) RenderTabSeps(sc *Scene) {
	rs, pc, st := tv.RenderLock(sc)
	defer tv.RenderUnlock(rs)

	// just like with standard separator, use top width like CSS
	// (see https://www.w3schools.com/howto/howto_css_dividers.asp)
	pc.StrokeStyle.Width = st.Border.Width.Top
	pc.StrokeStyle.SetColor(&st.Border.Color.Top)
	bw := st.Border.Width.Dots()

	tbs := tv.Tabs()
	sz := len(tbs.Kids)
	for i := 1; i < sz; i++ {
		tb := tbs.Child(i).(Widget)
		ni := tb.AsWidget()

		pos := ni.LayState.Alloc.Pos
		sz := ni.LayState.Alloc.Size.Sub(st.EffMargin().Size())
		pc.DrawLine(rs, pos.X-bw.Pos().X, pos.Y, pos.X-bw.Pos().X, pos.Y+sz.Y)
	}
	pc.FillStrokeClear(rs)
}

func (tv *TabView) FilterEvents() {
	tv.Events.CopyFrom(&TabViewEventFuncs)
}

func (tv *TabView) Render(sc *Scene) {
	wi := tv.This().(Widget)
	if tv.PushBounds(sc) {
		wi.FilterEvents()
		tv.RenderScrolls(sc)
		tv.RenderChildren(sc)
		tv.RenderTabSeps(sc)
		tv.PopBounds(sc)
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// TabButton

// TabButton is a larger select action and a small close action. Indicator
// icon is used for close icon.
type TabButton struct {
	Action

	// if true, this tab does not have the delete button avail
	NoDelete bool `desc:"if true, this tab does not have the delete button avail"`
}

// event functions for this type
var TabButtonEventFuncs WidgetEvents

func (tb *TabButton) OnInit() {
	tb.AddEvents(&TabButtonEventFuncs)
	tb.AddStyler(func(w *WidgetBase, s *gist.Style) {
		// s.Cursor = cursor.HandPointing
		s.MinWidth.SetCh(8)
		s.MaxWidth.SetPx(500)
		s.MinHeight.SetEm(1.6)

		// s.Border.Style.Right = gist.BorderSolid
		// s.Border.Width.Right.SetPx(1)

		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
		s.Color = colors.Scheme.OnSurface

		s.Border.Radius.Set()
		s.Text.Align = gist.AlignCenter
		s.Margin.Set()
		s.Padding.Set(units.Px(8 * Prefs.DensityMul()))

		// s.Border.Style.Set(gist.BorderNone)
		// if tb.IsSelected() {
		// 	s.Border.Style.Bottom = gist.BorderSolid
		// 	s.Border.Width.Bottom.SetPx(2)
		// 	s.Border.Color.Bottom = colors.Scheme.Primary
		// }

		if tb.HasFlag(Hovered) {
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerHighest)
		}
		if w.HasFocus() {
			s.Border.Style.Set(gist.BorderSolid)
			s.Border.Width.Set(units.Px(2))
			s.Border.Color.Set(colors.Scheme.Outline)
		}
	})
}

func (tb *TabButton) OnChildAdded(child ki.Ki) {
	if _, wb := AsWidget(child); wb != nil {
		switch wb.Name() {
		case "Parts":
			wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Overflow = gist.OverflowHidden // no scrollbars!
			})
		case "icon":
			wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Width.SetEm(1)
				s.Height.SetEm(1)
				s.Margin.Set()
				s.Padding.Set()
			})
		case "label":
			label := child.(*Label)
			label.Type = LabelTitleSmall
			wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Margin.Set()
				s.Padding.Set()
			})
		case "close-stretch":
			wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Width.SetCh(1)
			})
		case "close":
			wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Width.SetEx(0.5)
				s.Height.SetEx(0.5)
				s.Margin.Set()
				s.Padding.Set()
				s.AlignV = gist.AlignMiddle
				s.Border.Radius = gist.BorderRadiusFull
				s.BackgroundColor.SetSolid(colors.Transparent)
			})
		case "sc-stretch":
			wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.MinWidth.SetCh(2)
			})
		case "shortcut":
			wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Margin.Set()
				s.Padding.Set()
			})
		}
	}
}

func (tb *TabButton) TabView() *TabView {
	tv := tb.ParentByType(TabViewType, ki.Embeds)
	if tv == nil {
		return nil
	}
	return AsTabView(tv)
}

func (tb *TabButton) FilterEvents() {
	tb.Events.CopyFrom(&TabButtonEventFuncs)
}

func (tb *TabButton) ConfigParts(sc *Scene) {
	if !tb.NoDelete {
		tb.ConfigPartsDeleteButton(sc)
		return
	}
	tb.Action.ConfigParts(sc) // regular
}

func (tb *TabButton) ConfigPartsDeleteButton(sc *Scene) {
	config := ki.TypeAndNameList{}
	icIdx, lbIdx := tb.ConfigPartsIconLabel(&config, tb.Icon, tb.Text)
	config.Add(StretchType, "close-stretch")
	clsIdx := len(config)
	config.Add(ActionType, "close")
	mods, updt := tb.Parts.ConfigChildren(config)
	tb.ConfigPartsSetIconLabel(tb.Icon, tb.Text, icIdx, lbIdx)
	if mods {
		cls := tb.Parts.Child(clsIdx).(*Action)
		if tb.Indicator.IsNil() {
			tb.Indicator = icons.Close
		}

		icnm := tb.Indicator
		cls.SetIcon(icnm)
		cls.SetProp("no-focus", true)
		cls.ActionSig.ConnectOnly(tb.This(), func(recv, send ki.Ki, sig int64, data any) {
			tbb := tb
			tabIdx := tbb.Data.(int)
			tvv := tb.TabView()
			if tvv != nil {
				if !Prefs.Params.OnlyCloseActiveTab || tbb.IsSelected() { // only process delete when already selected if OnlyCloseActiveTab is on
					tvv.DeleteTabIndexAction(tabIdx)
				} else {
					tvv.SelectTabIndexAction(tabIdx) // otherwise select
				}
			}
		})
		tb.UpdateEnd(updt)
	}
}

func (tb *TabButton) ConfigWidget(sc *Scene) {
	tb.ConfigParts(sc)
}
