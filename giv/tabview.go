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
	fr.InsertChild(widg, idx)
	tab := tb.InsertNewChild(gi.KiT_Action, idx, label).(*gi.Action)
	tab.Data = idx
	tab.SetText(label)
	tab.ActionSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TabView).(*TabView)
		act := send.(*gi.Action)
		tabIdx := act.Data.(int)
		tvv.SelectTabIndex(tabIdx)
	})
}

// AddNewTab adds a new widget as a new tab of given widget type, with given
// tab label, and returns the new widget
func (tv *TabView) AddNewTab(typ reflect.Type, label string) gi.Node2D {
	fr := tv.Frame()
	idx := len(*fr.Children())
	widg := tv.InsertNewTab(typ, label, idx)
	return widg
}

// InsertNewTab inserts a new widget of given type into given index position
// within list of tabs, and returns that new widget
func (tv *TabView) InsertNewTab(typ reflect.Type, label string, idx int) gi.Node2D {
	fr := tv.Frame()
	tb := tv.Tabs()
	widg := fr.InsertNewChild(typ, idx, label).(gi.Node2D)
	tab := tb.InsertNewChild(gi.KiT_Action, idx, label).(*gi.Action)
	tab.Data = idx
	tab.SetText(label)
	tab.ActionSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvv := recv.Embed(KiT_TabView).(*TabView)
		act := send.(*gi.Action)
		tabIdx := act.Data.(int)
		tvv.SelectTabIndex(tabIdx)
	})
	return widg
}

// SelectTabIndex selects tab at given index, returning it -- returns false if
// index is invalid (emits log message)
func (tv *TabView) SelectTabIndex(idx int) (gi.Node2D, bool) {
	fr := tv.Frame()
	tb := tv.Tabs()
	sz := len(*fr.Children())
	if idx < 0 || idx >= sz {
		log.Printf("giv.TabView: index %v out of range for number of tabs: %v\n", idx, sz)
		return nil, false
	}
	tab := tb.KnownChild(idx).(*gi.Action)
	widg := fr.KnownChild(idx).(gi.Node2D)
	updt := tv.UpdateStart()
	tv.UnselectAllTabs()
	tab.SetSelectedState(true)
	fr.StackTop = idx
	tv.UpdateEnd(updt)
	return widg, true
}

// TabViewSignals are signals that the TabView can send
type TabViewSignals int64

const (
	// TabSelected indicates node was selected -- data is the tab widget
	TabSelected TabViewSignals = iota

	TabViewSignalsN
)

//go:generate stringer -type=TabViewSignals

// InitTabView initializes the tab widget children if it hasn't been done yet
func (tv *TabView) InitTabView() {
	if len(tv.Kids) == 2 {
		return
	}
	updt := tv.UpdateStart()
	tv.Lay = gi.LayoutVert

	tabs := tv.AddNewChild(gi.KiT_Layout, "tabs").(*gi.Layout)
	tabs.Lay = gi.LayoutHoriz
	tabs.SetStretchMaxWidth()
	// tabs.SetStretchMaxHeight()
	tabs.SetMinPrefWidth(units.NewValue(10, units.Em))
	tabs.SetProp("overflow", "hidden") // no scrollbars!

	frame := tv.AddNewChild(gi.KiT_Frame, "frame").(*gi.Frame)
	frame.Lay = gi.LayoutStacked
	frame.SetMinPrefWidth(units.NewValue(10, units.Em))
	frame.SetMinPrefHeight(units.NewValue(7, units.Em))
	frame.SetStretchMaxWidth()
	frame.SetStretchMaxHeight()

	tv.UpdateEnd(updt)
}

// Tabs returns the layout containing the tabs -- the first element within us
func (tv *TabView) Tabs() *gi.Layout {
	tv.InitTabView()
	return tv.KnownChild(0).(*gi.Layout)
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
		tb, ok := tbk.(*gi.Action)
		if !ok {
			continue
		}
		if tb.IsSelected() {
			tb.SetSelectedState(false)
		}
	}
}

// var TabButtonProps = ki.Props{
// 	"border-width":        units.NewValue(1, units.Px),
// 	"border-radius":       units.NewValue(0, units.Px),
// 	"border-color":        &gi.Prefs.Colors.Border,
// 	"border-style":        gi.BorderSolid,
// 	"padding":             units.NewValue(4, units.Px),
// 	"margin":              units.NewValue(0, units.Px),
// 	"background-color":    &gi.Prefs.Colors.Control,
// 	"box-shadow.h-offset": units.NewValue(0, units.Px),
// 	"box-shadow.v-offset": units.NewValue(0, units.Px),
// 	"box-shadow.blur":     units.NewValue(0, units.Px),
// 	"box-shadow.color":    &gi.Prefs.Colors.Shadow,
// 	"text-align":          gi.AlignCenter,
// }
