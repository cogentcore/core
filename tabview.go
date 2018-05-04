// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log"

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
// LayoutCol with a LayoutRow of tab buttons and then the LayoutStacked of
// Frames
type TabView struct {
	WidgetBase
	SrcNode    ki.Ptr    `desc:"Ki Node that this widget is viewing in the tree -- the source -- chilren of this node are tabs, and updates drive tab updates"`
	TabViewSig ki.Signal `json:"-" xml:"-" desc:"signal for tab widget -- see TabViewSignals for the types"`
}

var KiT_TabView = kit.Types.AddType(&TabView{}, nil)

func (n *TabView) New() ki.Ki { return &TabView{} }

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

// select tab at given index
func (g *TabView) SelectTabIndex(idx int) error {
	tabrow := g.TabRowLayout()
	idx, err := tabrow.Children().ValidIndex(idx)
	if err != nil {
		return err
	}
	tbk := tabrow.Child(idx)
	tb, ok := tbk.(*Button)
	if !ok {
		return nil
	}
	updt := g.UpdateStart()
	g.UnselectAllTabButtons()
	tb.SetSelected(true)
	tabstack := g.TabStackLayout()
	tabstack.ShowChildAtIndex(idx)
	g.UpdateEnd(updt)
	return nil
}

// get tab frame for given index
func (g *TabView) TabFrameAtIndex(idx int) *Frame {
	tabstack := g.TabStackLayout()
	idx, err := tabstack.Children().ValidIndex(idx)
	if err != nil {
		log.Printf("%v", err)
		return nil
	}
	tfk := tabstack.Child(idx)
	tf, ok := tfk.(*Frame)
	if !ok {
		return nil
	}
	return tf
}

// get the overal column layout for the tab widget
func (g *TabView) TabColLayout() *Layout {
	g.InitTabView()
	return g.Child(0).(*Layout)
}

// get the row layout of tabs across the top of the tab widget
func (g *TabView) TabRowLayout() *Layout {
	tabcol := g.TabColLayout()
	return tabcol.Child(0).(*Layout)
}

// get the stacked layout of tab frames
func (g *TabView) TabStackLayout() *Layout {
	tabcol := g.TabColLayout()
	return tabcol.Child(1).(*Layout)
}

// unselect all tabs
func (g *TabView) UnselectAllTabButtons() {
	tabrow := g.TabRowLayout()
	for _, tbk := range tabrow.Kids {
		tb, ok := tbk.(*Button)
		if !ok {
			continue
		}
		if tb.IsSelected() {
			updt := tb.UpdateStart()
			tb.SetSelected(false)
			tb.UpdateEnd(updt)
		}
	}
}

func TabButtonClicked(recv, send ki.Ki, sig int64, d interface{}) {
	g, ok := recv.(*TabView)
	if !ok {
		return
	}
	if sig == int64(ButtonClicked) {
		tb, ok := send.(*Button)
		if !ok {
			return
		}
		if !tb.IsSelected() {
			tabrow := g.TabRowLayout()
			butidx := tabrow.ChildIndex(send, 0)
			// fmt.Printf("selected tab: %v\n", butidx)
			if butidx >= 0 {
				g.SelectTabIndex(butidx)
			}
		}
	}
}

var TabButtonProps = ki.Props{
	"border-width":        units.NewValue(1, units.Px),
	"border-radius":       units.NewValue(0, units.Px),
	"border-color":        &Prefs.BorderColor,
	"border-style":        BorderSolid,
	"padding":             units.NewValue(4, units.Px),
	"margin":              units.NewValue(0, units.Px),
	"background-color":    &Prefs.ControlColor,
	"box-shadow.h-offset": units.NewValue(0, units.Px),
	"box-shadow.v-offset": units.NewValue(0, units.Px),
	"box-shadow.blur":     units.NewValue(0, units.Px),
	"box-shadow.color":    &Prefs.ShadowColor,
	"text-align":          AlignCenter,
}

// make the initial tab frames for src node
func (g *TabView) InitTabs() {
	tabrow := g.TabRowLayout()
	tabstack := g.TabStackLayout()
	if g.SrcNode.Ptr == nil {
		return
	}
	skids := g.SrcNode.Ptr.Children()
	for _, sk := range skids {
		nm := "TabFrameOf_" + sk.UniqueName()
		tf := tabstack.AddNewChild(KiT_Frame, nm).(*Frame)
		tf.Lay = LayoutCol
		tf.SetProp("max-width", -1.0) // stretch flex
		tf.SetProp("max-height", -1.0)
		nm = "TabOf_" + sk.UniqueName()
		tb := tabrow.AddNewChild(KiT_Button, nm).(*Button) // todo make tab button
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
	tabcol := g.AddNewChild(KiT_Layout, "TabCol").(*Layout)
	tabcol.Lay = LayoutCol
	tabrow := tabcol.AddNewChild(KiT_Layout, "TabRow").(*Layout)
	tabrow.Lay = LayoutRow
	tabstack := tabcol.AddNewChild(KiT_Layout, "TabStack").(*Layout)
	tabstack.Lay = LayoutStacked
	tabstack.SetProp("max-width", -1.0) // stretch flex
	tabstack.SetProp("max-height", -1.0)
	g.InitTabs()
	g.UpdateEnd(updt)
}

////////////////////////////////////////////////////
// Node2D interface

// check for interface implementation
var _ Node2D = &TabView{}
