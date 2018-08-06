// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  Tab Widget

// todo: this is out-of-date and non-functional

// signals that buttons can send
type TabViewSignals int64

const (
	// node was selected -- data is the tab widget
	TabSelected TabViewSignals = iota

	// tab widget unselected
	TabUnselected

	// collapsed tab widget was opened
	TabOpened

	// open tab widget was collapsed -- children not visible
	TabCollapsed

	TabViewSignalsN
)

//go:generate stringer -type=TabViewSignals

// todo: could have different positioning of the tabs?

// TabView represents children of a source node as tabs with a stacked
// layout of Frame widgets for each child in the source -- we create a
// LayoutVert with a LayoutHoriz of tab buttons and then the LayoutStacked of
// Frames
type TabView struct {
	gi.WidgetBase
	SrcNode    ki.Ptr    `desc:"Ki Node that this widget is viewing in the tree -- the source -- chilren of this node are tabs, and updates drive tab updates"`
	TabViewSig ki.Signal `json:"-" xml:"-" desc:"signal for tab widget -- see TabViewSignals for the types"`
}

var KiT_TabView = kit.Types.AddType(&TabView{}, nil)

// set the source Ki Node that generates our tabs
func (g *TabView) SetSrcNode(k ki.Ki) {
	g.SrcNode.Ptr = k
	k.NodeSignal().Connect(g.This, SrcNodeSignal) // we recv signals from source
	nm := "TabViewOf_" + k.UniqueName()
	if g.Nm == "" {
		g.SetName(nm)
	}
	g.InitTabView()
}

// todo: various other ways of selecting tabs..

// SelectTabIndex selects tab at given index, returning false if index is invalid
func (g *TabView) SelectTabIndex(idx int) bool {
	tabrow := g.TabRowLayout()
	tbk, ok := tabrow.Child(idx)
	if !ok {
		return false
	}
	tb, ok := tbk.(*gi.Button)
	if !ok {
		return false
	}
	updt := g.UpdateStart()
	g.UnselectAllTabButtons()
	tb.SetSelectedState(true)
	tabstack := g.TabStackLayout()
	tabstack.StackTop = idx
	g.UpdateEnd(updt)
	return true
}

// TabFrameAtIndex returns tab frame for given index
func (g *TabView) TabFrameAtIndex(idx int) *gi.Frame {
	tabstack := g.TabStackLayout()
	tfk, ok := tabstack.Child(idx)
	if !ok {
		return nil
	}
	tf, ok := tfk.(*gi.Frame)
	if !ok {
		return nil
	}
	return tf
}

// get the overal column layout for the tab widget
func (g *TabView) TabColLayout() *gi.Layout {
	g.InitTabView()
	return g.KnownChild(0).(*gi.Layout)
}

// get the row layout of tabs across the top of the tab widget
func (g *TabView) TabRowLayout() *gi.Layout {
	tabcol := g.TabColLayout()
	return tabcol.KnownChild(0).(*gi.Layout)
}

// get the stacked layout of tab frames
func (g *TabView) TabStackLayout() *gi.Layout {
	tabcol := g.TabColLayout()
	return tabcol.KnownChild(1).(*gi.Layout)
}

// unselect all tabs
func (g *TabView) UnselectAllTabButtons() {
	tabrow := g.TabRowLayout()
	for _, tbk := range tabrow.Kids {
		tb, ok := tbk.(*gi.Button)
		if !ok {
			continue
		}
		if tb.IsSelected() {
			updt := tb.UpdateStart()
			tb.SetSelectedState(false)
			tb.UpdateEnd(updt)
		}
	}
}

func TabButtonClicked(recv, send ki.Ki, sig int64, d interface{}) {
	g, ok := recv.(*TabView)
	if !ok {
		return
	}
	if sig == int64(gi.ButtonClicked) {
		tb, ok := send.(*gi.Button)
		if !ok {
			return
		}
		if !tb.IsSelected() {
			tabrow := g.TabRowLayout()
			butidx, ok := tabrow.Children().IndexOf(send, 0)
			// fmt.Printf("selected tab: %v\n", butidx)
			if ok {
				g.SelectTabIndex(butidx)
			}
		}
	}
}

var TabButtonProps = ki.Props{
	"border-width":        units.NewValue(1, units.Px),
	"border-radius":       units.NewValue(0, units.Px),
	"border-color":        &gi.Prefs.BorderColor,
	"border-style":        gi.BorderSolid,
	"padding":             units.NewValue(4, units.Px),
	"margin":              units.NewValue(0, units.Px),
	"background-color":    &gi.Prefs.ControlColor,
	"box-shadow.h-offset": units.NewValue(0, units.Px),
	"box-shadow.v-offset": units.NewValue(0, units.Px),
	"box-shadow.blur":     units.NewValue(0, units.Px),
	"box-shadow.color":    &gi.Prefs.ShadowColor,
	"text-align":          gi.AlignCenter,
}

// make the initial tab frames for src node
func (g *TabView) InitTabs() {
	tabrow := g.TabRowLayout()
	tabstack := g.TabStackLayout()
	if g.SrcNode.Ptr == nil {
		return
	}
	skids := *g.SrcNode.Ptr.Children()
	for _, sk := range skids {
		nm := "TabFrameOf_" + sk.UniqueName()
		tf := tabstack.AddNewChild(gi.KiT_Frame, nm).(*gi.Frame)
		tf.Lay = gi.LayoutVert
		tf.SetProp("max-width", -1.0) // stretch flex
		tf.SetProp("max-height", -1.0)
		nm = "TabOf_" + sk.UniqueName()
		tb := tabrow.AddNewChild(gi.KiT_Button, nm).(*gi.Button) // todo make tab button
		tb.Text = sk.Name()
		for key, val := range TabButtonProps {
			tb.SetProp(key, val)
		}
		tb.ButtonSig.Connect(g.This, TabButtonClicked)
	}
	g.SelectTabIndex(0)
}

// todo: update tabs from changes

// initialize the tab widget structure -- assumes it has been done if there is
// already a child node
func (g *TabView) InitTabView() {
	if len(g.Kids) == 1 {
		return
	}
	updt := g.UpdateStart()
	tabcol := g.AddNewChild(gi.KiT_Layout, "TabCol").(*gi.Layout)
	tabcol.Lay = gi.LayoutVert
	tabrow := tabcol.AddNewChild(gi.KiT_Layout, "TabRow").(*gi.Layout)
	tabrow.Lay = gi.LayoutHoriz
	tabstack := tabcol.AddNewChild(gi.KiT_Layout, "TabStack").(*gi.Layout)
	tabstack.Lay = gi.LayoutStacked
	tabstack.SetProp("max-width", -1.0) // stretch flex
	tabstack.SetProp("max-height", -1.0)
	g.InitTabs()
	g.UpdateEnd(updt)
}

////////////////////////////////////////////////////
// Node2D interface
