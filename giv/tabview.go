// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"log"
	"reflect"

	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
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
	gi.Layout
	MaxChars   int       `desc:"maximum number of characters to include in tab label -- elides labels that are longer than that"`
	TabViewSig ki.Signal `json:"-" xml:"-" desc:"signal for tab widget -- see TabViewSignals for the types"`
}

var KiT_TabView = kit.Types.AddType(&TabView{}, TabViewProps)

var TabViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
	"width":            units.NewValue(10, units.Em),
	"height":           units.NewValue(10, units.Em),
}

// NTabs returns number of tabs
func (tv *TabView) NTabs() int {
	fr := tv.Frame()
	if fr == nil {
		return 0
	}
	return len(fr.Kids)
}

// AddTab adds a widget as a new tab, with given tab label, and returns the
// index of that tab
func (tv *TabView) AddTab(widg gi.Node2D, label string) int {
	fr := tv.Frame()
	idx := len(*fr.Children())
	tv.InsertTab(widg, label, idx)
	return idx
}

// InsertTab inserts a widget into given index position within list of tabs
func (tv *TabView) InsertTab(widg gi.Node2D, label string, idx int) {
	fr := tv.Frame()
	tb := tv.Tabs()
	updt := tv.UpdateStart()
	tv.SetFullReRender()
	fr.InsertChild(widg, idx)
	tab := tb.InsertNewChild(KiT_TabButton, idx, label).(*TabButton)
	tab.Data = idx
	tab.SetText(label)
	tab.ActionSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TabView).(*TabView)
		act := send.Embed(KiT_TabButton).(*TabButton)
		tabIdx := act.Data.(int)
		tvv.SelectTabIndex(tabIdx)
	})
	tv.UpdateEnd(updt)
}

// AddNewTab adds a new widget as a new tab of given widget type, with given
// tab label, and returns the new widget and its tab index
func (tv *TabView) AddNewTab(typ reflect.Type, label string) (gi.Node2D, int) {
	fr := tv.Frame()
	idx := len(*fr.Children())
	widg := tv.InsertNewTab(typ, label, idx)
	return widg, idx
}

// InsertNewTab inserts a new widget of given type into given index position
// within list of tabs, and returns that new widget
func (tv *TabView) InsertNewTab(typ reflect.Type, label string, idx int) gi.Node2D {
	fr := tv.Frame()
	tb := tv.Tabs()
	updt := tv.UpdateStart()
	tv.SetFullReRender()
	widg := fr.InsertNewChild(typ, idx, label).(gi.Node2D)
	tab := tb.InsertNewChild(KiT_TabButton, idx, label).(*TabButton)
	tab.Data = idx
	tab.SetText(label)
	tab.ActionSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TabView).(*TabView)
		act := send.Embed(KiT_TabButton).(*TabButton)
		tabIdx := act.Data.(int)
		tvv.SelectTabIndexAction(tabIdx)
	})
	tv.UpdateEnd(updt)
	return widg
}

// TabAtIndex returns content widget and tab button at given index, false if
// index out of range (emits log message)
func (tv *TabView) TabAtIndex(idx int) (gi.Node2D, *TabButton, bool) {
	fr := tv.Frame()
	tb := tv.Tabs()
	sz := len(*fr.Children())
	if idx < 0 || idx >= sz {
		log.Printf("giv.TabView: index %v out of range for number of tabs: %v\n", idx, sz)
		return nil, nil, false
	}
	tab := tb.KnownChild(idx).Embed(KiT_TabButton).(*TabButton)
	widg := fr.KnownChild(idx).(gi.Node2D)
	return widg, tab, true
}

// SelectTabIndex selects tab at given index, returning it -- returns false if
// index is invalid
func (tv *TabView) SelectTabIndex(idx int) (gi.Node2D, bool) {
	widg, tab, ok := tv.TabAtIndex(idx)
	if !ok {
		return nil, false
	}
	fr := tv.Frame()
	updt := tv.UpdateStart()
	tv.UnselectAllTabs()
	tab.SetSelectedState(true)
	fr.StackTop = idx
	tv.UpdateEnd(updt)
	return widg, true
}

// SelectTabIndexAction selects tab at given index and emits selected signal,
// with the index of the selected tab -- this is what is called when a tab is
// clicked
func (tv *TabView) SelectTabIndexAction(idx int) {
	_, ok := tv.SelectTabIndex(idx)
	if ok {
		tv.TabViewSig.Emit(tv.This, int64(TabSelected), idx)
	}
}

// TabByName returns tab with given name, and its index -- returns false if
// not found
func (tv *TabView) TabByName(label string) (gi.Node2D, int, bool) {
	tb := tv.Tabs()
	idx, ok := tb.Children().IndexByName(label, 0)
	if !ok {
		return nil, -1, false
	}
	fr := tv.Frame()
	widg := fr.KnownChild(idx).(gi.Node2D)
	return widg, idx, true
}

// SelectTabName selects tab by name, returning it -- returns false if not
// found
func (tv *TabView) SelectTabByName(label string) (gi.Node2D, int, bool) {
	widg, idx, ok := tv.TabByName(label)
	if ok {
		tv.SelectTabIndex(idx)
	}
	return widg, idx, ok
}

// DeleteTabIndex deletes tab at given index, optionally calling destroy on
// tab contents -- returns widget if destroy == false and bool success
func (tv *TabView) DeleteTabIndex(idx int, destroy bool) (gi.Node2D, bool) {
	widg, _, ok := tv.TabAtIndex(idx)
	if !ok {
		return nil, false
	}
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
	tb.DeleteChildAtIndex(idx, true) // always destroy -- we manage
	tv.RenumberTabs()
	if nxtidx >= 0 {
		tv.SelectTabIndex(nxtidx)
	}
	tv.UpdateEnd(updt)
	if destroy {
		return nil, true
	} else {
		return widg, true
	}
}

// DeleteTabIndexAction deletes tab at given index using destroy flag, and
// emits TabDeleted signal -- this is called by the delete button on the tab
func (tv *TabView) DeleteTabIndexAction(idx int) {
	_, ok := tv.DeleteTabIndex(idx, true)
	if ok {
		tv.TabViewSig.Emit(tv.This, int64(TabDeleted), idx)
	}
}

// TabViewSignals are signals that the TabView can send
type TabViewSignals int64

const (
	// TabSelected indicates tab was selected -- data is the tab index
	TabSelected TabViewSignals = iota

	// TabDeleted indicates tab was deleted -- data is the tab index
	TabDeleted

	TabViewSignalsN
)

//go:generate stringer -type=TabViewSignals

// InitTabView initializes the tab widget children if it hasn't been done yet
func (tv *TabView) InitTabView() {
	if len(tv.Kids) == 2 {
		return
	}
	if tv.Sty.Font.Size.Val == 0 { // not yet styled
		tv.StyleLayout()
	}
	updt := tv.UpdateStart()
	tv.Lay = gi.LayoutVert
	tv.SetReRenderAnchor()

	tabs := tv.AddNewChild(gi.KiT_Frame, "tabs").(*gi.Frame)
	tabs.Lay = gi.LayoutHoriz
	tabs.SetStretchMaxWidth()
	// tabs.SetStretchMaxHeight()
	tabs.SetMinPrefWidth(units.NewValue(10, units.Em))
	tabs.SetProp("height", units.NewValue(1.8, units.Em))
	tabs.SetProp("overflow", "hidden") // no scrollbars!
	tabs.SetProp("padding", units.NewValue(0, units.Px))
	tabs.SetProp("margin", units.NewValue(0, units.Px))
	tabs.SetProp("spacing", units.NewValue(4, units.Px))
	tabs.SetProp("background-color", "linear-gradient(pref(Control), highlight-10)")

	frame := tv.AddNewChild(gi.KiT_Frame, "frame").(*gi.Frame)
	frame.Lay = gi.LayoutStacked
	frame.SetMinPrefWidth(units.NewValue(10, units.Em))
	frame.SetMinPrefHeight(units.NewValue(7, units.Em))
	frame.SetStretchMaxWidth()
	frame.SetStretchMaxHeight()

	tv.UpdateEnd(updt)
}

// Tabs returns the layout containing the tabs -- the first element within us
func (tv *TabView) Tabs() *gi.Frame {
	tv.InitTabView()
	return tv.KnownChild(0).(*gi.Frame)
}

// Frame returns the stacked frame layout -- the second element
func (tv *TabView) Frame() *gi.Frame {
	tv.InitTabView()
	return tv.KnownChild(1).(*gi.Frame)
}

// UnselectAllTabs turns off all the tabs
func (tv *TabView) UnselectAllTabs() {
	tb := tv.Tabs()
	for _, tbk := range tb.Kids {
		tb := tbk.Embed(KiT_TabButton).(*TabButton)
		if tb.IsSelected() {
			tb.SetSelectedState(false)
		}
	}
}

// RenumberTabs assigns proper index numbers to each tab
func (tv *TabView) RenumberTabs() {
	tb := tv.Tabs()
	for idx, tbk := range tb.Kids {
		tb := tbk.Embed(KiT_TabButton).(*TabButton)
		tb.Data = idx
	}
}

func (tv *TabView) Style2D() {
	tv.InitTabView()
	tv.Layout.Style2D()
}

func (tv *TabView) Render2D() {
	tv.Layout.Render2D()
}

////////////////////////////////////////////////////////////////////////////////////////
// TabButton

// TabButton is a larger select action and a small close action. Indicator
// icon is used for close icon.
type TabButton struct {
	gi.Action
}

var KiT_TabButton = kit.Types.AddType(&TabButton{}, TabButtonProps)

var TabButtonProps = ki.Props{
	"border-width":     units.NewValue(0, units.Px),
	"border-radius":    units.NewValue(0, units.Px),
	"border-color":     &gi.Prefs.Colors.Border,
	"border-style":     gi.BorderSolid,
	"box-shadow.color": &gi.Prefs.Colors.Shadow,
	"text-align":       gi.AlignCenter,
	"background-color": &gi.Prefs.Colors.Control,
	"color":            &gi.Prefs.Colors.Font,
	"padding":          units.NewValue(4, units.Px), // we go to edge of bar
	"margin":           units.NewValue(0, units.Px),
	"indicator":        "close",
	"#icon": ki.Props{
		"width":   units.NewValue(1, units.Em),
		"height":  units.NewValue(1, units.Em),
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
		"fill":    &gi.Prefs.Colors.Icon,
		"stroke":  &gi.Prefs.Colors.Font,
	},
	"#label": ki.Props{
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
	},
	"#close-stretch": ki.Props{
		"width": units.NewValue(1, units.Ch),
	},
	"#close": ki.Props{
		"width":          units.NewValue(.5, units.Ex),
		"height":         units.NewValue(.5, units.Ex),
		"margin":         units.NewValue(0, units.Px),
		"padding":        units.NewValue(0, units.Px),
		"vertical-align": gi.AlignBottom,
	},
	"#shortcut": ki.Props{
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
	},
	"#sc-stretch": ki.Props{
		"min-width": units.NewValue(2, units.Em),
	},
	gi.ButtonSelectors[gi.ButtonActive]: ki.Props{
		"background-color": "linear-gradient(lighter-0, highlight-10)",
	},
	gi.ButtonSelectors[gi.ButtonInactive]: ki.Props{
		"border-color": "lighter-50",
		"color":        "lighter-50",
	},
	gi.ButtonSelectors[gi.ButtonHover]: ki.Props{
		"background-color": "linear-gradient(highlight-10, highlight-10)",
	},
	gi.ButtonSelectors[gi.ButtonFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "linear-gradient(samelight-50, highlight-10)",
	},
	gi.ButtonSelectors[gi.ButtonDown]: ki.Props{
		"color":            "lighter-90",
		"background-color": "linear-gradient(highlight-30, highlight-10)",
	},
	gi.ButtonSelectors[gi.ButtonSelected]: ki.Props{
		"background-color": "linear-gradient(pref(Select), highlight-10)",
	},
}

func (tb *TabButton) ButtonAsBase() *gi.ButtonBase {
	return &(tb.ButtonBase)
}

func (tb *TabButton) TabView() *TabView {
	tv, ok := tb.ParentByType(KiT_TabView, true)
	if !ok {
		return nil
	}
	return tv.Embed(KiT_TabView).(*TabView)
}

func (tb *TabButton) ConfigParts() {
	config, icIdx, lbIdx := tb.ConfigPartsIconLabel(string(tb.Icon), tb.Text)
	config.Add(gi.KiT_Stretch, "close-stretch")
	clsIdx := len(config)
	config.Add(gi.KiT_Action, "close")
	mods, updt := tb.Parts.ConfigChildren(config, false) // not unique names
	tb.ConfigPartsSetIconLabel(string(tb.Icon), tb.Text, icIdx, lbIdx)
	if mods {
		cls := tb.Parts.KnownChild(clsIdx).(*gi.Action)
		if tb.Indicator.IsNil() {
			tb.Indicator = "close"
		}
		tb.StylePart(gi.Node2D(cls))

		icnm := string(tb.Indicator)
		cls.SetIcon(icnm)
		cls.SetProp("no-focus", true)
		cls.ActionSig.ConnectOnly(tb.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			tbb := recv.Embed(KiT_TabButton).(*TabButton)
			tabIdx := tbb.Data.(int)
			tvv := tb.TabView()
			if tvv != nil {
				tvv.DeleteTabIndexAction(tabIdx)
			}
		})
		tb.UpdateEnd(updt)
	}
}
