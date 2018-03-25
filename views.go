// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	// "github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"image"
	"math"
	// "log"
	// "reflect"
)

////////////////////////////////////////////////////////////////////////////////////////
//  TreeView -- a widget that graphically represents / manipulates a Ki Tree

// signals that buttons can send
type TreeViewSignals int64

const (
	// node was selected -- data is the TreeView widget
	NodeSelected TreeViewSignals = iota
	// TreeView unselected
	NodeUnselected
	// collapsed TreeView was opened
	NodeOpened
	// open TreeView was collapsed -- children not visible
	NodeCollapsed
	TreeViewSignalsN
)

//go:generate stringer -type=TreeViewSignals

// these extend NodeBase NodeFlags to hold TreeView state
const (
	// node is collapsed
	NodeFlagCollapsed NodeFlags = NodeFlagsN + iota
	// node is selected
	NodeFlagSelected
	// a full re-render is required due to nature of update event -- otherwise default is local re-render
	NodeFlagFullReRender
	// the shift key was pressed, putting the selection mode into continous selection mode
	NodeFlagContinuousSelect
	// the ctrl / cmd key was pressed, putting the selection mode into continous selection mode
	NodeFlagExtendSelect
)

// mutually-exclusive button states -- determines appearance
type TreeViewStates int32

const (
	// normal state -- there but not being interacted with
	TreeViewNormalState TreeViewStates = iota
	// selected
	TreeViewSelState
	// in focus -- will respond to keyboard input
	TreeViewFocusState
	TreeViewStatesN
)

//go:generate stringer -type=TreeViewStates

// TreeView represents one node in the tree -- fully recursive -- creates
//  sub-nodes
type TreeView struct {
	WidgetBase
	SrcNode     ki.Ptr                 `desc:"Ki Node that this widget is viewing in the tree -- the source"`
	TreeViewSig ki.Signal              `json:"-" desc:"signal for TreeView -- see TreeViewSignals for the types"`
	StateStyles [TreeViewStatesN]Style `desc:"styles for different states of the widget -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	WidgetSize  Vec2D                  `desc:"just the size of our widget -- our alloc includes all of our children, but we only draw us"`
	RootWidget  *TreeView              `json:"-" desc:"cached root widget"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_TreeView = ki.Types.AddType(&TreeView{}, nil)

// todo: several functions require traversing tree -- this will require an
// interface to allow others to implement different behavior -- for now just
// explicitly checking for TreeView type

//////////////////////////////////////////////////////////////////////////////
//    End-User API

// set the source node that we are viewing
func (g *TreeView) SetSrcNode(k ki.Ki) {
	g.UpdateStart()
	if len(g.Children) > 0 {
		g.DeleteChildren(true) // todo: later deal with destroyed
	}
	g.SrcNode.Ptr = k
	k.NodeSignal().Connect(g.This, SrcNodeSignal) // we recv signals from source
	nm := "ViewOf_" + k.KiUniqueName()
	if g.Name != nm {
		g.SetName(nm)
	}
	kids := k.KiChildren()
	// breadth first -- first make all our kids, then have them make their kids
	for _, kid := range kids {
		g.AddNewChildNamed(nil, "ViewOf_"+kid.KiUniqueName()) // our name is view of ki unique name
	}
	for i, kid := range kids {
		vki, _ := g.KiChild(i)
		vk, ok := vki.(*TreeView)
		if !ok {
			continue // shouldn't happen
		}
		vk.SetSrcNode(kid)
	}
	g.UpdateEnd()
}

// function for receiving node signals from our SrcNode
func SrcNodeSignal(nwki, send ki.Ki, sig int64, data interface{}) {
	// todo: need a *node* deleted signal!  and children etc
	// track changes in source node
}

// return a list of the currently-selected source nodes
func (g *TreeView) SelectedSrcNodes() ki.Slice {
	sn := make(ki.Slice, 0)
	g.FunDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		_, gi := KiToNode2D(k)
		if gi == nil {
			return false
		}
		if k.IsType(KiT_TreeView) {
			nw := k.(*TreeView)
			sn = append(sn, nw.SrcNode.Ptr)
			return true
		} else {
			return false
		}
	})
	return sn
}

// return a list of the currently-selected TreeViews
func (g *TreeView) SelectedTreeViews() []*TreeView {
	sn := make([]*TreeView, 0)
	g.FunDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		_, gi := KiToNode2D(k)
		if gi == nil {
			return false
		}
		if k.IsType(KiT_TreeView) {
			nw := k.(*TreeView)
			sn = append(sn, nw)
			return true
		} else {
			return false
		}
	})
	return sn
}

//////////////////////////////////////////////////////////////////////////////
//    Implementation

// root node of TreeView tree -- several properties stored there
func (g *TreeView) RootTreeView() *TreeView {
	rn := g
	g.FunUp(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		_, pg := KiToNode2D(k)
		if pg == nil {
			return false
		}
		if k.IsType(KiT_TreeView) {
			rn = k.(*TreeView)
			return true
		} else {
			return false
		}
	})
	return rn
}

// is this node itself collapsed?
func (g *TreeView) IsCollapsed() bool {
	return ki.HasBitFlag(g.NodeFlags, int(NodeFlagCollapsed))
}

// does this node have a collapsed parent? if so, don't render!
func (g *TreeView) HasCollapsedParent() bool {
	pcol := false
	g.FunUpParent(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		_, pg := KiToNode2D(k)
		if pg == nil {
			return false
		}
		if k.IsType(KiT_TreeView) {
			if ki.HasBitFlag(pg.NodeFlags, int(NodeFlagCollapsed)) {
				pcol = true
				return false
			}
		}
		return true
	})
	return pcol
}

// is this node selected?
func (g *TreeView) IsSelected() bool {
	return ki.HasBitFlag(g.NodeFlags, int(NodeFlagSelected))
}

func (g *TreeView) GetLabel() string {
	label := ""
	if g.IsCollapsed() { // todo: temp hack
		label = "> "
	} else {
		label = "v "
	}
	label += g.SrcNode.Ptr.KiName()
	return label
}

// a select action has been received (e.g., a mouse click) -- translate into
// selection updates
func (g *TreeView) SelectNodeAction() {
	rn := g.RootWidget
	if ki.HasBitFlag(rn.NodeFlags, int(NodeFlagExtendSelect)) {
		if g.IsSelected() {
			g.UnselectNode()
		} else {
			g.SelectNode()
		}
	} else { // todo: continuous a bit trickier
		if g.IsSelected() {
			// nothing..
		} else {
			rn.UnselectAll()
			g.SelectNode()
		}
	}
}

func (g *TreeView) SelectNode() {
	if !g.IsSelected() {
		g.UpdateStart()
		ki.SetBitFlag(&g.NodeFlags, int(NodeFlagSelected))
		g.GrabFocus() // focus always follows select  todo: option
		g.TreeViewSig.Emit(g.This, int64(NodeSelected), nil)
		// fmt.Printf("selected node: %v\n", g.Name)
		g.UpdateEndAll() // grab focus means allow kids to update too
	}
}

func (g *TreeView) UnselectNode() {
	if g.IsSelected() {
		g.UpdateStart()
		ki.ClearBitFlag(&g.NodeFlags, int(NodeFlagSelected))
		g.TreeViewSig.Emit(g.This, int64(NodeUnselected), nil)
		// fmt.Printf("unselectednode: %v\n", g.Name)
		g.UpdateEnd()
	}
}

// unselect everything below me -- call on Root to clear all
func (g *TreeView) UnselectAll() {
	g.UpdateStart()
	g.FunDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		_, gi := KiToNode2D(k)
		if gi == nil {
			return false
		}
		if k.IsType(KiT_TreeView) {
			nw := k.(*TreeView)
			nw.UnselectNode()
			return true
		} else {
			return false
		}
	})
	g.UpdateEndAll()
}

// unselect everything below me -- call on Root to clear all
func (g *TreeView) RootUnselectAll() {
	g.RootWidget.UnselectAll()
}

func (g *TreeView) CollapseNode() {
	if !g.IsCollapsed() {
		g.UpdateStart()
		ki.SetBitFlag(&g.NodeFlags, int(NodeFlagFullReRender))
		ki.SetBitFlag(&g.NodeFlags, int(NodeFlagCollapsed))
		g.TreeViewSig.Emit(g.This, int64(NodeCollapsed), nil)
		// fmt.Printf("collapsed node: %v\n", g.Name)
		g.UpdateEnd()
	}
}

func (g *TreeView) OpenNode() {
	if g.IsCollapsed() {
		g.UpdateStart()
		ki.SetBitFlag(&g.NodeFlags, int(NodeFlagFullReRender))
		ki.ClearBitFlag(&g.NodeFlags, int(NodeFlagCollapsed))
		g.TreeViewSig.Emit(g.This, int64(NodeOpened), nil)
		// fmt.Printf("opened node: %v\n", g.Name)
		g.UpdateEnd()
	}
}

func (g *TreeView) SetContinuousSelect() {
	rn := g.RootWidget
	ki.SetBitFlag(&rn.NodeFlags, int(NodeFlagContinuousSelect))
}

func (g *TreeView) SetExtendSelect() {
	rn := g.RootWidget
	ki.SetBitFlag(&rn.NodeFlags, int(NodeFlagExtendSelect))
}

func (g *TreeView) ClearSelectMods() {
	rn := g.RootWidget
	ki.ClearBitFlag(&rn.NodeFlags, int(NodeFlagContinuousSelect))
	ki.ClearBitFlag(&rn.NodeFlags, int(NodeFlagExtendSelect))
}

////////////////////////////////////////////////////
// Node2D interface

func (g *TreeView) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *TreeView) AsViewport2D() *Viewport2D {
	return nil
}

func (g *TreeView) AsLayout2D() *Layout {
	return nil
}

func (g *TreeView) InitNode2D() {
	g.InitNode2DBase()
	g.ReceiveEventType(MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		_, ok := recv.(*TreeView)
		if !ok {
			return
		}
		// todo: specifically on down?  needed this for emergent
	})
	g.ReceiveEventType(MouseUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		fmt.Printf("button %v pressed!\n", recv.PathUnique())
		ab, ok := recv.(*TreeView)
		if ok {
			ab.SelectNodeAction()
		}
	})
	g.ReceiveEventType(KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*TreeView)
		if ok {
			kt, ok := d.(KeyTypedEvent)
			if ok {
				// fmt.Printf("TreeView key: %v\n", kt.Chord)
				kf := KeyFun(kt.Key, kt.Chord)
				switch kf {
				case KeyFunSelectItem:
					ab.SelectNodeAction()
				case KeyFunCancelSelect:
					ab.RootUnselectAll()
				case KeyFunMoveRight:
					ab.OpenNode()
				case KeyFunMoveLeft:
					ab.CollapseNode()
					// todo; move down / up -- selectnext / prev
				}
			}
		}
	})
	g.ReceiveEventType(KeyDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*TreeView)
		if ok {
			kt, ok := d.(KeyDownEvent)
			if ok {
				kf := KeyFun(kt.Key, "")
				// fmt.Printf("TreeView key down: %v\n", kt.Key)
				switch kf {
				case KeyFunShift:
					ab.SetContinuousSelect()
				case KeyFunCtrl:
					ab.SetExtendSelect()
				}
			}
		}
	})
	g.ReceiveEventType(KeyUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*TreeView)
		if ok {
			ab.ClearSelectMods()
		}
	})
}

var TreeViewProps = []map[string]interface{}{
	{
		"border-width":  "0px",
		"border-radius": "0px",
		"padding":       "1px",
		"margin":        "1px",
		// "font-family":         "Arial", // this is crashing
		"font-size":        "24pt", // todo: need to get dpi!
		"text-align":       "left",
		"color":            "black",
		"background-color": "#FFF", // todo: get also from user, type on viewed node
	}, { // selected
		"background-color": "#CFC", // todo: also
	}, { // focused
		"background-color": "#CCF", // todo: also
	},
}

func (g *TreeView) Style2D() {
	// we can focus by default
	ki.SetBitFlag(&g.NodeFlags, int(CanFocus))
	// first do our normal default styles
	g.Style.SetStyle(nil, &StyleDefault, TreeViewProps[0])
	// then style with user props
	g.Style2DWidget()
	// now get styles for the different states
	for i := 0; i < int(TreeViewStatesN); i++ {
		g.StateStyles[i] = g.Style
		g.StateStyles[i].SetStyle(nil, &StyleDefault, TreeViewProps[i])
		g.StateStyles[i].SetUnitContext(&g.Viewport.Render, 0)
	}
	// todo: how to get state-specific user prefs?  need an extra prefix..
}

func (g *TreeView) Layout2D(iter int) {
	if iter == 0 {
		g.RootWidget = g.RootTreeView() // cache

		g.InitLayout2D()
		st := &g.Style
		pc := &g.Paint
		var w, h float64

		if g.HasCollapsedParent() {
			ki.ClearBitFlag(&g.NodeFlags, int(CanFocus))
			return // nothing
		}
		ki.SetBitFlag(&g.NodeFlags, int(CanFocus))

		label := g.GetLabel()

		w, h = pc.MeasureString(label)
		if st.Layout.Width.Dots > 0 {
			w = math.Max(st.Layout.Width.Dots, w)
		}
		if st.Layout.Height.Dots > 0 {
			h = math.Max(st.Layout.Height.Dots, h)
		}
		w += 2.0*st.Padding.Dots + 2.0*st.Layout.Margin.Dots
		h += 2.0*st.Padding.Dots + 2.0*st.Layout.Margin.Dots

		g.WidgetSize = Vec2D{w, h}

		if !g.IsCollapsed() {
			// we layout children under us
			for _, kid := range g.Children {
				_, gi := KiToNode2D(kid)
				if gi != nil {
					gi.LayData.AllocPos.Y = h
					gi.LayData.AllocPos.X = 20 // indent children -- todo: make a property
					h += gi.LayData.AllocSize.Y
					w = math.Max(w, gi.LayData.AllocPos.X+gi.LayData.AllocSize.X) // use max
				}
			}
		}
		g.LayData.AllocSize = Vec2D{w, h}
		g.WidgetSize.X = w // stretch
	} else {
		g.AddParentPos()
		rn := g.RootWidget
		g.LayData.AllocSize.X = rn.LayData.AllocSize.X - (g.LayData.AllocPos.X - rn.LayData.AllocPos.X)
		g.WidgetSize.X = g.LayData.AllocSize.X
		gii, _ := KiToNode2D(g.This)
		g.SetWinBBox(gii.Node2DBBox())
	}

	// todo: test for use of parent-el relative units -- indicates whether multiple loops
	// are required
	g.Style.SetUnitContext(&g.Viewport.Render, 0)
	// now get styles for the different states
	for i := 0; i < int(TreeViewStatesN); i++ {
		g.StateStyles[i].SetUnitContext(&g.Viewport.Render, 0)
	}

}

func (g *TreeView) Node2DBBox() image.Rectangle {
	// we have unusual situation of bbox != alloc
	tp := g.Paint.TransformPoint(g.LayData.AllocPos.X, g.LayData.AllocPos.Y)
	ts := g.Paint.TransformPoint(g.WidgetSize.X, g.WidgetSize.Y)
	return image.Rect(int(tp.X), int(tp.Y), int(tp.X+ts.X), int(tp.Y+ts.Y))
}

func (g *TreeView) Render2D() {
	if g.PushBounds() {
		// reset for next update
		ki.ClearBitFlag(&g.NodeFlags, int(NodeFlagFullReRender))

		if g.HasCollapsedParent() {
			return // nothing
		}

		if g.IsSelected() {
			g.Style = g.StateStyles[TreeViewSelState]
		} else if g.HasFocus() {
			g.Style = g.StateStyles[TreeViewFocusState]
		} else {
			g.Style = g.StateStyles[TreeViewNormalState]
		}

		pc := &g.Paint
		rs := &g.Viewport.Render
		st := &g.Style
		pc.FontStyle = st.Font
		pc.TextStyle = st.Text
		pc.StrokeStyle.SetColor(&st.Border.Color)
		pc.StrokeStyle.Width = st.Border.Width
		pc.FillStyle.SetColor(&st.Background.Color)
		// g.DrawStdBox()
		pos := g.LayData.AllocPos.AddVal(st.Layout.Margin.Dots)
		sz := g.WidgetSize.AddVal(-2.0 * st.Layout.Margin.Dots)
		g.DrawBoxImpl(pos, sz, st.Border.Radius.Dots)

		pc.StrokeStyle.SetColor(&st.Color) // ink color

		pos = g.LayData.AllocPos.AddVal(st.Layout.Margin.Dots + st.Padding.Dots)
		// sz = g.LayData.AllocSize.AddVal(-2.0 * (st.Layout.Margin.Dots + st.Padding.Dots))

		label := g.GetLabel()
		// fmt.Printf("rendering: %v\n", label)

		pc.DrawStringAnchored(rs, label, pos.X, pos.Y, 0.0, 0.9, sz.X)

		g.Render2DChildren()
	}
	g.PopBounds()
}

func (g *TreeView) CanReRender2D() bool {
	if ki.HasBitFlag(g.NodeFlags, int(NodeFlagFullReRender)) {
		return false
	} else {
		return true
	}
}

func (g *TreeView) FocusChanged2D(gotFocus bool) {
	// todo: good to somehow indicate focus
	// Qt does it by changing the color of the little toggle widget!  sheesh!
	g.UpdateStart()
	g.UpdateEnd()
}

// check for interface implementation
var _ Node2D = &TreeView{}

////////////////////////////////////////////////////////////////////////////////////////
//  Tab Widget

// signals that buttons can send
type TabWidgetSignals int64

const (
	// node was selected -- data is the tab widget
	TabSelected TabWidgetSignals = iota
	// tab widget unselected
	TabUnselected
	// collapsed tab widget was opened
	TabOpened
	// open tab widget was collapsed -- children not visible
	TabCollapsed
	TabWidgetSignalsN
)

//go:generate stringer -type=TabWidgetSignals

// todo: could have different positioning of the tabs?

// TabWidget represents children of a source node as tabs with a stacked
// layout of Frame widgets for each child in the source -- we create a
// LayoutCol with a LayoutRow of tab buttons and then the LayoutStacked of
// Frames
type TabWidget struct {
	WidgetBase
	SrcNode      ki.Ptr    `desc:"Ki Node that this widget is viewing in the tree -- the source -- chilren of this node are tabs, and updates drive tab updates"`
	TabWidgetSig ki.Signal `json:"-" desc:"signal for tab widget -- see TabWidgetSignals for the types"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_TabWidget = ki.Types.AddType(&TabWidget{}, nil)

// set the source Ki Node that generates our tabs
func (g *TabWidget) SetSrcNode(k ki.Ki) {
	g.SrcNode.Ptr = k
	k.NodeSignal().Connect(g.This, SrcNodeSignal) // we recv signals from source
	nm := "TabViewOf_" + k.KiUniqueName()
	if g.Name == "" {
		g.SetName(nm)
	}
	g.InitTabWidget()
}

// todo: various other ways of selecting tabs..

// select tab at given index
func (g *TabWidget) SelectTabIndex(idx int) error {
	tabrow := g.TabRowLayout()
	tbk, err := tabrow.KiChild(idx)
	if err != nil {
		return err
	}
	tb, ok := tbk.(*Button)
	if !ok {
		return nil
	}
	g.UpdateStart()
	g.UnselectAllTabButtons()
	tb.SetSelected(true)
	tabstack := g.TabStackLayout()
	tabstack.ShowChildAtIndex(idx)
	g.UpdateEnd()
	return nil
}

// get tab frame for given index
func (g *TabWidget) TabFrameAtIndex(idx int) *Frame {
	tabstack := g.TabStackLayout()
	tfk, err := tabstack.KiChild(idx)
	if err != nil {
		return nil
	}
	tf, ok := tfk.(*Frame)
	if !ok {
		return nil
	}
	return tf
}

// get the overal column layout for the tab widget
func (g *TabWidget) TabColLayout() *Layout {
	g.InitTabWidget()
	ch, _ := g.KiChild(0)
	return ch.(*Layout)
}

// get the row layout of tabs across the top of the tab widget
func (g *TabWidget) TabRowLayout() *Layout {
	tabcol := g.TabColLayout()
	ch, _ := tabcol.KiChild(0)
	return ch.(*Layout)
}

// get the stacked layout of tab frames
func (g *TabWidget) TabStackLayout() *Layout {
	tabcol := g.TabColLayout()
	ch, _ := tabcol.KiChild(1)
	return ch.(*Layout)
}

// unselect all tabs
func (g *TabWidget) UnselectAllTabButtons() {
	tabrow := g.TabRowLayout()
	for _, tbk := range tabrow.Children {
		tb, ok := tbk.(*Button)
		if !ok {
			continue
		}
		if tb.IsSelected() {
			tb.UpdateStart()
			tb.SetSelected(false)
			tb.UpdateEnd()
		}
	}
}

func TabButtonClicked(recv, send ki.Ki, sig int64, d interface{}) {
	g, ok := recv.(*TabWidget)
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
			butidx := tabrow.FindChildIndex(send, 0)
			// fmt.Printf("selected tab: %v\n", butidx)
			if butidx >= 0 {
				g.SelectTabIndex(butidx)
			}
		}
	}
}

var TabButtonProps = map[string]interface{}{
	"border-width":        "1px",
	"border-radius":       "0px",
	"border-color":        "black",
	"border-style":        "solid",
	"padding":             "4px",
	"margin":              "0px",
	"box-shadow.h-offset": "0px",
	"box-shadow.v-offset": "0px",
	"box-shadow.blur":     "0px",
	// "font-family":         "Arial", // this is crashing
	"font-size":        "24pt",
	"text-align":       "center",
	"color":            "black",
	"background-color": "#EEF",
}

// make the initial tab frames for src node
func (g *TabWidget) InitTabs() {
	tabrow := g.TabRowLayout()
	tabstack := g.TabStackLayout()
	if g.SrcNode.Ptr == nil {
		return
	}
	skids := g.SrcNode.Ptr.KiChildren()
	for _, sk := range skids {
		nm := "TabFrameOf_" + sk.KiUniqueName()
		tf := tabstack.AddNewChildNamed(KiT_Frame, nm).(*Frame)
		tf.Lay = LayoutCol
		tf.SetProp("max-width", -1.0) // stretch flex
		tf.SetProp("max-height", -1.0)
		nm = "TabOf_" + sk.KiUniqueName()
		tb := tabrow.AddNewChildNamed(KiT_Button, nm).(*Button) // todo make tab button
		tb.Text = sk.KiName()
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
func (g *TabWidget) InitTabWidget() {
	if len(g.Children) == 1 {
		return
	}
	g.UpdateStart()
	tabcol := g.AddNewChildNamed(KiT_Layout, "TabCol").(*Layout)
	tabcol.Lay = LayoutCol
	tabrow := tabcol.AddNewChildNamed(KiT_Layout, "TabRow").(*Layout)
	tabrow.Lay = LayoutRow
	tabstack := tabcol.AddNewChildNamed(KiT_Layout, "TabStack").(*Layout)
	tabstack.Lay = LayoutStacked
	tabstack.SetProp("max-width", -1.0) // stretch flex
	tabstack.SetProp("max-height", -1.0)
	g.InitTabs()
	g.UpdateEnd()
}

////////////////////////////////////////////////////
// Node2D interface

func (g *TabWidget) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *TabWidget) AsViewport2D() *Viewport2D {
	return nil
}

func (g *TabWidget) AsLayout2D() *Layout {
	return nil
}

func (g *TabWidget) InitNode2D() {
	g.InitNode2DBase()
}

func (g *TabWidget) Style2D() {
	// // first do our normal default styles
	// g.Style.SetStyle(nil, &StyleDefault, TabWidgetProps[0])
	// then style with user props
	g.Style2DWidget()
}

func (g *TabWidget) Layout2D(iter int) {
	g.BaseLayout2D(iter)
}

func (g *TabWidget) Node2DBBox() image.Rectangle {
	return g.WinBBoxFromAlloc()
}

func (g *TabWidget) Render2D() {
	if g.PushBounds() {
		g.Render2DChildren()
	}
	g.PopBounds()
}

func (g *TabWidget) CanReRender2D() bool {
	return true
}

func (g *TabWidget) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &TabWidget{}
