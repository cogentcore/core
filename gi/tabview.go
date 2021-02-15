// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log"
	"reflect"
	"sync"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// TabView switches among child widgets via tabs.  The selected widget gets
// the full allocated space avail after the tabs are accounted for.  The
// TabView is just a Vertical layout that manages two child widgets: a
// HorizFlow Layout for the tabs (which can flow across multiple rows as
// needed) and a Stacked Frame that actually contains all the children, and
// provides scrollbars as needed to any content within.  Typically should have
// max stretch and a set preferred size, so it expands.
type TabView struct {
	Layout
	MaxChars     int          `desc:"maximum number of characters to include in tab label -- elides labels that are longer than that"`
	TabViewSig   ki.Signal    `copy:"-" json:"-" xml:"-" desc:"signal for tab widget -- see TabViewSignals for the types"`
	NewTabButton bool         `desc:"show a new tab button at right of list of tabs"`
	NoDeleteTabs bool         `desc:"if true, tabs are not user-deleteable"`
	NewTabType   reflect.Type `desc:"type of widget to create in a new tab via new tab button -- Frame by default"`
	Mu           sync.Mutex   `copy:"-" json:"-" xml:"-" view:"-" desc:"mutex protecting updates to tabs -- tabs can be driven programmatically and via user input so need extra protection"`
}

var KiT_TabView = kit.Types.AddType(&TabView{}, TabViewProps)

// AddNewTabView adds a new tabview to given parent node, with given name.
func AddNewTabView(parent ki.Ki, name string) *TabView {
	return parent.AddNewChild(KiT_TabView, name).(*TabView)
}

func (tv *TabView) CopyFieldsFrom(frm interface{}) {
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

var TabViewProps = ki.Props{
	"EnumType:Flag":    KiT_NodeFlags,
	"border-color":     &Prefs.Colors.Border,
	"border-width":     units.NewPx(2),
	"background-color": &Prefs.Colors.Background,
	"color":            &Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
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
func (tv *TabView) CurTab() (Node2D, int, bool) {
	if tv.NTabs() == 0 {
		return nil, -1, false
	}
	tv.Mu.Lock()
	defer tv.Mu.Unlock()
	fr := tv.Frame()
	if fr.StackTop < 0 {
		return nil, -1, false
	}
	widg := fr.Child(fr.StackTop).(Node2D)
	return widg, fr.StackTop, true
}

// AddTab adds a widget as a new tab, with given tab label, and returns the
// index of that tab
func (tv *TabView) AddTab(widg Node2D, label string) int {
	fr := tv.Frame()
	idx := len(*fr.Children())
	tv.InsertTab(widg, label, idx)
	return idx
}

// InsertTabOnlyAt inserts just the tab at given index -- after panel has
// already been added to frame -- assumed to be wrapped in update.  Generally
// for internal use.
func (tv *TabView) InsertTabOnlyAt(widg Node2D, label string, idx int) {
	tb := tv.Tabs()
	tb.SetChildAdded()
	tab := tb.InsertNewChild(KiT_TabButton, idx, label).(*TabButton)
	tab.Data = idx
	tab.Tooltip = label
	tab.NoDelete = tv.NoDeleteTabs
	tab.SetText(label)
	tab.ActionSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TabView).(*TabView)
		act := send.Embed(KiT_TabButton).(*TabButton)
		tabIdx := act.Data.(int)
		tvv.SelectTabIndexAction(tabIdx)
	})
	fr := tv.Frame()
	if len(fr.Kids) == 1 {
		fr.StackTop = 0
		tab.SetSelectedState(true)
	} else {
		widg.AsNode2D().SetInvisible() // new tab is invisible until selected
	}
}

// InsertTab inserts a widget into given index position within list of tabs
func (tv *TabView) InsertTab(widg Node2D, label string, idx int) {
	tv.Mu.Lock()
	fr := tv.Frame()
	updt := tv.UpdateStart()
	tv.SetFullReRender()
	fr.SetChildAdded()
	fr.InsertChild(widg, idx)
	tv.InsertTabOnlyAt(widg, label, idx)
	tv.Mu.Unlock()
	tv.UpdateEnd(updt)
}

// AddNewTab adds a new widget as a new tab of given widget type, with given
// tab label, and returns the new widget
func (tv *TabView) AddNewTab(typ reflect.Type, label string) Node2D {
	fr := tv.Frame()
	idx := len(*fr.Children())
	widg := tv.InsertNewTab(typ, label, idx)
	return widg
}

// AddNewTabAction adds a new widget as a new tab of given widget type, with given
// tab label, and returns the new widget -- emits TabAdded signal
func (tv *TabView) AddNewTabAction(typ reflect.Type, label string) Node2D {
	widg := tv.AddNewTab(typ, label)
	fr := tv.Frame()
	idx := len(*fr.Children()) - 1
	tv.TabViewSig.Emit(tv.This(), int64(TabAdded), idx)
	return widg
}

// InsertNewTab inserts a new widget of given type into given index position
// within list of tabs, and returns that new widget
func (tv *TabView) InsertNewTab(typ reflect.Type, label string, idx int) Node2D {
	fr := tv.Frame()
	updt := tv.UpdateStart()
	tv.SetFullReRender()
	fr.SetChildAdded()
	widg := fr.InsertNewChild(typ, idx, label).(Node2D)
	tv.InsertTabOnlyAt(widg, label, idx)
	tv.UpdateEnd(updt)
	return widg
}

// TabAtIndex returns content widget and tab button at given index, false if
// index out of range (emits log message)
func (tv *TabView) TabAtIndex(idx int) (Node2D, *TabButton, bool) {
	tv.Mu.Lock()
	defer tv.Mu.Unlock()

	fr := tv.Frame()
	tb := tv.Tabs()
	sz := len(*fr.Children())
	if idx < 0 || idx >= sz {
		log.Printf("giv.TabView: index %v out of range for number of tabs: %v\n", idx, sz)
		return nil, nil, false
	}
	tab := tb.Child(idx).Embed(KiT_TabButton).(*TabButton)
	widg := fr.Child(idx).(Node2D)
	return widg, tab, true
}

// SelectTabIndex selects tab at given index, returning it -- returns false if
// index is invalid
func (tv *TabView) SelectTabIndex(idx int) (Node2D, bool) {
	widg, tab, ok := tv.TabAtIndex(idx)
	if !ok {
		return nil, false
	}
	fr := tv.Frame()
	if fr.StackTop == idx {
		return widg, true
	}
	tv.Mu.Lock()
	// tv.Viewport.BlockUpdates() // not needed for this apparently
	updt := tv.UpdateStart()
	tv.UnselectOtherTabs(idx)
	tab.SetSelectedState(true)
	fr.StackTop = idx
	fr.SetFullReRender()
	tv.WinFullReRender() // tell window to do a full redraw
	// tv.Viewport.UnblockUpdates()
	tv.Mu.Unlock()
	tv.UpdateEnd(updt)
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
func (tv *TabView) TabByName(label string) Node2D {
	t, _ := tv.TabByNameTry(label)
	return t
}

// TabByNameTry returns tab with given name, and an error if not found.
func (tv *TabView) TabByNameTry(label string) (Node2D, error) {
	tv.Mu.Lock()
	defer tv.Mu.Unlock()

	tb := tv.Tabs()
	idx, ok := tb.Children().IndexByName(label, 0)
	if !ok {
		return nil, fmt.Errorf("gi.TabView: Tab named %v not found in %v", label, tv.Path())
	}
	fr := tv.Frame()
	widg := fr.Child(idx).(Node2D)
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
func (tv *TabView) SelectTabByName(label string) Node2D {
	idx, err := tv.TabIndexByName(label)
	if err == nil {
		tv.SelectTabIndex(idx)
		fr := tv.Frame()
		return fr.Child(idx).(Node2D)
	}
	return nil
}

// SelectTabByNameTry selects tab by name, returning it.  Returns error if not found.
func (tv *TabView) SelectTabByNameTry(label string) (Node2D, error) {
	idx, err := tv.TabIndexByName(label)
	if err == nil {
		tv.SelectTabIndex(idx)
		fr := tv.Frame()
		return fr.Child(idx).(Node2D), nil
	}
	return nil, err
}

// RecycleTab returns a tab with given name, first by looking for an existing one,
// and if not found, making a new one with widget of given type.
// If sel, then select it.  returns widget for tab.
func (tv *TabView) RecycleTab(label string, typ reflect.Type, sel bool) Node2D {
	widg, err := tv.TabByNameTry(label)
	if err == nil {
		if sel {
			tv.SelectTabByName(label)
		}
		return widg
	}
	widg = tv.AddNewTab(typ, label)
	if sel {
		tv.SelectTabByName(label)
	}
	return widg
}

// DeleteTabIndex deletes tab at given index, optionally calling destroy on
// tab contents -- returns widget if destroy == false, tab name, and bool success
func (tv *TabView) DeleteTabIndex(idx int, destroy bool) (Node2D, string, bool) {
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
	tv.SetFullReRender()
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
	tv.UpdateEnd(updt)
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
func (tv *TabView) ConfigNewTabButton() bool {
	sz := tv.NTabs()
	tb := tv.Tabs()
	ntb := len(tb.Kids)
	if tv.NewTabButton {
		if ntb == sz+1 {
			return false
		}
		if tv.NewTabType == nil {
			tv.NewTabType = KiT_Frame
		}
		tab := tb.InsertNewChild(KiT_Action, ntb, "new-tab").(*Action)
		tab.Data = -1
		tab.SetIcon("plus")
		tab.ActionSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			tvv := recv.Embed(KiT_TabView).(*TabView)
			tvv.SetFullReRender()
			tvv.AddNewTabAction(tvv.NewTabType, "New Tab")
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

//go:generate stringer -type=TabViewSignals

// Config initializes the tab widget children if it hasn't been done yet
func (tv *TabView) Config() {
	if len(tv.Kids) != 0 {
		return
	}
	tv.StyMu.RLock()
	needSty := tv.Sty.Font.Size.Val == 0
	tv.StyMu.RUnlock()
	if needSty {
		tv.StyleLayout()
	}

	updt := tv.UpdateStart()
	tv.Lay = LayoutVert
	tv.SetReRenderAnchor()

	tabs := AddNewFrame(tv, "tabs", LayoutHorizFlow)
	tabs.SetStretchMaxWidth()
	// tabs.SetStretchMaxHeight()
	// tabs.SetMinPrefWidth(units.NewEm(10))
	tabs.SetProp("height", units.NewEm(1.8))
	tabs.SetProp("overflow", gist.OverflowHidden) // no scrollbars!
	tabs.SetProp("padding", units.NewPx(0))
	tabs.SetProp("margin", units.NewPx(0))
	tabs.SetProp("spacing", units.NewPx(4))
	tabs.SetProp("background-color", "linear-gradient(pref(Control), highlight-10)")

	frame := AddNewFrame(tv, "frame", LayoutStacked)
	frame.SetMinPrefWidth(units.NewEm(10))
	frame.SetMinPrefHeight(units.NewEm(7))
	frame.StackTopOnly = true // key for allowing each tab to have its own size
	frame.SetStretchMax()
	frame.SetReRenderAnchor()

	tv.ConfigNewTabButton()

	tv.UpdateEnd(updt)
}

// Tabs returns the layout containing the tabs -- the first element within us
func (tv *TabView) Tabs() *Frame {
	tv.Config()
	return tv.Child(0).(*Frame)
}

// Frame returns the stacked frame layout -- the second element
func (tv *TabView) Frame() *Frame {
	tv.Config()
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
		tb := tbs.Child(i).Embed(KiT_TabButton).(*TabButton)
		if tb.IsSelected() {
			tb.SetSelectedState(false)
		}
	}
}

// RenumberTabs assigns proper index numbers to each tab
func (tv *TabView) RenumberTabs() {
	sz := tv.NTabs()
	tbs := tv.Tabs()
	for i := 0; i < sz; i++ {
		tb := tbs.Child(i).Embed(KiT_TabButton).(*TabButton)
		tb.Data = i
	}
}

func (tv *TabView) Style2D() {
	tv.Config()
	tv.Layout.Style2D()
}

// RenderTabSeps renders the separators between tabs
func (tv *TabView) RenderTabSeps() {
	rs, pc, st := tv.RenderLock()
	defer tv.RenderUnlock(rs)

	pc.StrokeStyle.Width = st.Border.Width
	pc.StrokeStyle.SetColor(&st.Border.Color)
	bw := st.Border.Width.Dots

	tbs := tv.Tabs()
	sz := len(tbs.Kids)
	for i := 1; i < sz; i++ {
		tb := tbs.Child(i).(Node2D)
		ni := tb.AsWidget()

		pos := ni.LayState.Alloc.Pos
		sz := ni.LayState.Alloc.Size.AddScalar(-2.0 * st.Layout.Margin.Dots)
		pc.DrawLine(rs, pos.X-bw, pos.Y, pos.X-bw, pos.Y+sz.Y)
	}
	pc.FillStrokeClear(rs)
}

func (tv *TabView) Render2D() {
	if tv.FullReRenderIfNeeded() {
		return
	}
	if tv.PushBounds() {
		tv.This().(Node2D).ConnectEvents2D()
		tv.RenderScrolls()
		tv.Render2DChildren()
		tv.RenderTabSeps()
		tv.PopBounds()
	} else {
		tv.DisconnectAllEvents(AllPris) // uses both Low and Hi
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// TabButton

// TabButton is a larger select action and a small close action. Indicator
// icon is used for close icon.
type TabButton struct {
	Action
	NoDelete bool `desc:"if true, this tab does not have the delete button avail"`
}

var KiT_TabButton = kit.Types.AddType(&TabButton{}, TabButtonProps)

// TabButtonMinWidth is the minimum width of the tab button, in Ch units
var TabButtonMinWidth = float32(8)

var TabButtonProps = ki.Props{
	"EnumType:Flag":    KiT_ButtonFlags,
	"min-width":        units.NewCh(TabButtonMinWidth),
	"min-height":       units.NewEm(1.6),
	"border-width":     units.NewPx(0),
	"border-radius":    units.NewPx(0),
	"border-color":     &Prefs.Colors.Border,
	"text-align":       gist.AlignCenter,
	"background-color": &Prefs.Colors.Control,
	"color":            &Prefs.Colors.Font,
	"padding":          units.NewPx(4), // we go to edge of bar
	"margin":           units.NewPx(0),
	"indicator":        "close",
	"#icon": ki.Props{
		"width":   units.NewEm(1),
		"height":  units.NewEm(1),
		"margin":  units.NewPx(0),
		"padding": units.NewPx(0),
		"fill":    &Prefs.Colors.Icon,
		"stroke":  &Prefs.Colors.Font,
	},
	"#label": ki.Props{
		"margin":  units.NewPx(0),
		"padding": units.NewPx(0),
	},
	"#close-stretch": ki.Props{
		"width": units.NewCh(1),
	},
	"#close": ki.Props{
		"width":          units.NewEx(0.5),
		"height":         units.NewEx(0.5),
		"margin":         units.NewPx(0),
		"padding":        units.NewPx(0),
		"vertical-align": gist.AlignBottom,
	},
	"#shortcut": ki.Props{
		"margin":  units.NewPx(0),
		"padding": units.NewPx(0),
	},
	"#sc-stretch": ki.Props{
		"min-width": units.NewCh(2),
	},
	ButtonSelectors[ButtonActive]: ki.Props{
		"background-color": "linear-gradient(lighter-0, highlight-10)",
	},
	ButtonSelectors[ButtonInactive]: ki.Props{
		"border-color": "lighter-50",
		"color":        "lighter-50",
	},
	ButtonSelectors[ButtonHover]: ki.Props{
		"background-color": "linear-gradient(highlight-10, highlight-10)",
	},
	ButtonSelectors[ButtonFocus]: ki.Props{
		"border-width":     units.NewPx(2),
		"background-color": "linear-gradient(samelight-50, highlight-10)",
	},
	ButtonSelectors[ButtonDown]: ki.Props{
		"color":            "lighter-90",
		"background-color": "linear-gradient(highlight-30, highlight-10)",
	},
	ButtonSelectors[ButtonSelected]: ki.Props{
		"background-color": "linear-gradient(pref(Select), highlight-10)",
	},
}

func (tb *TabButton) TabView() *TabView {
	tv := tb.ParentByType(KiT_TabView, ki.Embeds)
	if tv == nil {
		return nil
	}
	return tv.Embed(KiT_TabView).(*TabView)
}

func (tb *TabButton) ConfigParts() {
	tb.Parts.SetProp("overflow", gist.OverflowHidden) // no scrollbars!
	if !tb.NoDelete {
		tb.ConfigPartsDeleteButton()
		return
	}
	tb.Action.ConfigParts() // regular
}

func (tb *TabButton) ConfigPartsDeleteButton() {
	config := kit.TypeAndNameList{}
	clsIdx := 0
	config.Add(KiT_Action, "close")
	config.Add(KiT_Stretch, "close-stretch")
	icIdx, lbIdx := tb.ConfigPartsIconLabel(&config, string(tb.Icon), tb.Text)
	mods, updt := tb.Parts.ConfigChildren(config)
	tb.ConfigPartsSetIconLabel(string(tb.Icon), tb.Text, icIdx, lbIdx)
	if mods {
		cls := tb.Parts.Child(clsIdx).(*Action)
		if tb.Indicator.IsNil() {
			tb.Indicator = "close"
		}
		tb.StylePart(Node2D(cls))

		icnm := string(tb.Indicator)
		cls.SetIcon(icnm)
		cls.SetProp("no-focus", true)
		cls.ActionSig.ConnectOnly(tb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			tbb := recv.Embed(KiT_TabButton).(*TabButton)
			tabIdx := tbb.Data.(int)
			tvv := tb.TabView()
			if tvv != nil {
				if tbb.IsSelected() { // only process delete when already selected
					tvv.DeleteTabIndexAction(tabIdx)
				} else {
					tvv.SelectTabIndexAction(tabIdx) // otherwise select
				}
			}
		})
		tb.UpdateEnd(updt)
	}
}
