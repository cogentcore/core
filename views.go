// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
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

// internal indexes for accessing elements of the widget
const (
	tvBranchIdx = iota
	tvSpaceIdx
	tvLabelIdx
	tvStretchIdx
	tvMenuIdx
)

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

var KiT_TreeView = kit.Types.AddType(&TreeView{}, nil)

// todo: several functions require traversing tree -- this will require an
// interface to allow others to implement different behavior -- for now just
// explicitly checking for TreeView type

//////////////////////////////////////////////////////////////////////////////
//    End-User API

// set the source node that we are viewing
func (g *TreeView) SetSrcNode(k ki.Ki) {
	g.UpdateStart()
	if len(g.Kids) > 0 {
		g.DeleteChildren(true) // todo: later deal with destroyed
	}
	g.SrcNode.Ptr = k
	k.NodeSignal().Connect(g.This, SrcNodeSignal) // we recv signals from source
	nm := "ViewOf_" + k.UniqueName()
	if g.Nm != nm {
		g.SetName(nm)
	}
	kids := k.Children()
	// breadth first -- first make all our kids, then have them make their kids
	for _, kid := range kids {
		g.AddNewChildNamed(nil, "ViewOf_"+kid.UniqueName()) // our name is view of ki unique name
	}
	for i, kid := range kids {
		vki, _ := g.Child(i)
		vk, ok := vki.(*TreeView)
		if !ok {
			continue // shouldn't happen
		}
		vk.SetSrcNode(kid)
	}
	g.UpdateEnd()
}

// function for receiving node signals from our SrcNode
func SrcNodeSignal(tvki, send ki.Ki, sig int64, data interface{}) {
	fmt.Printf("treeview: %v got signal: %v from node: %v  data: %v\n", tvki.PathUnique(), ki.NodeSignals(sig), send.PathUnique(), data)
	if bitflag.Has(*send.Flags(), int(ki.ChildAdded)) {
		// basically we have the full sync problem here -- doesn't really help to know that there were additions -- just need to find the new nodes!
	}
}

// return a list of the currently-selected source nodes
func (g *TreeView) SelectedSrcNodes() ki.Slice {
	sn := make(ki.Slice, 0)
	g.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		_, gi := KiToNode2D(k)
		if gi == nil {
			return false
		}
		if k.TypeEmbeds(KiT_TreeView) {
			nw := k.EmbeddedStruct(KiT_TreeView).(*TreeView)
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
	g.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		_, gi := KiToNode2D(k)
		if gi == nil {
			return false
		}
		if k.TypeEmbeds(KiT_TreeView) {
			nw := k.EmbeddedStruct(KiT_TreeView).(*TreeView)
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
	g.FuncUp(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		_, pg := KiToNode2D(k)
		if pg == nil {
			return false
		}
		if k.TypeEmbeds(KiT_TreeView) {
			rn = k.EmbeddedStruct(KiT_TreeView).(*TreeView)
			return true
		} else {
			return false
		}
	})
	return rn
}

// is this node itself collapsed?
func (g *TreeView) IsCollapsed() bool {
	return bitflag.Has(g.NodeFlags, int(NodeFlagCollapsed))
}

// does this node have a collapsed parent? if so, don't render!
func (g *TreeView) HasCollapsedParent() bool {
	pcol := false
	g.FuncUpParent(0, g.This, func(k ki.Ki, level int, d interface{}) bool {
		_, pg := KiToNode2D(k)
		if pg == nil {
			return false
		}
		if pg.TypeEmbeds(KiT_TreeView) {
			// nw := pg.EmbeddedStruct(KiT_TreeView).(*TreeView)
			if bitflag.Has(pg.NodeFlags, int(NodeFlagCollapsed)) {
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
	return bitflag.Has(g.NodeFlags, int(NodeFlagSelected))
}

func (g *TreeView) Label() string {
	return g.SrcNode.Ptr.Name()
}

// a select action has been received (e.g., a mouse click) -- translate into
// selection updates
func (g *TreeView) SelectNodeAction() {
	rn := g.RootWidget
	if bitflag.Has(rn.NodeFlags, int(NodeFlagExtendSelect)) {
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
		bitflag.Set(&g.NodeFlags, int(NodeFlagSelected))
		g.GrabFocus() // focus always follows select  todo: option
		g.TreeViewSig.Emit(g.This, int64(NodeSelected), nil)
		// fmt.Printf("selected node: %v\n", g.Nm)
		g.UpdateEnd()
	}
}

func (g *TreeView) UnselectNode() {
	if g.IsSelected() {
		g.UpdateStart()
		bitflag.Clear(&g.NodeFlags, int(NodeFlagSelected))
		g.TreeViewSig.Emit(g.This, int64(NodeUnselected), nil)
		// fmt.Printf("unselectednode: %v\n", g.Nm)
		g.UpdateEnd()
	}
}

// unselect everything below me -- call on Root to clear all
func (g *TreeView) UnselectAll() {
	g.UpdateStart()
	g.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		_, gi := KiToNode2D(k)
		if gi == nil {
			return false
		}
		if k.TypeEmbeds(KiT_TreeView) {
			nw := k.EmbeddedStruct(KiT_TreeView).(*TreeView)
			nw.UnselectNode()
			return true
		} else {
			return false
		}
	})
	g.UpdateEnd()
}

// unselect everything below me -- call on Root to clear all
func (g *TreeView) RootUnselectAll() {
	g.RootWidget.UnselectAll()
}

func (g *TreeView) Collapse() {
	if !g.IsCollapsed() {
		g.UpdateStart()
		bitflag.Set(&g.NodeFlags, int(NodeFlagFullReRender))
		bitflag.Set(&g.NodeFlags, int(NodeFlagCollapsed))
		g.TreeViewSig.Emit(g.This, int64(NodeCollapsed), nil)
		// fmt.Printf("collapsed node: %v\n", g.Nm)
		g.UpdateEnd()
	}
}

func (g *TreeView) Expand() {
	if g.IsCollapsed() {
		g.UpdateStart()
		bitflag.Set(&g.NodeFlags, int(NodeFlagFullReRender))
		bitflag.Clear(&g.NodeFlags, int(NodeFlagCollapsed))
		g.TreeViewSig.Emit(g.This, int64(NodeOpened), nil)
		// fmt.Printf("expanded node: %v\n", g.Nm)
		g.UpdateEnd()
	}
}

func (g *TreeView) ToggleCollapse() {
	if g.IsCollapsed() {
		g.Expand()
	} else {
		g.Collapse()
	}
}

func (g *TreeView) InsertAfter() {
	if g.IsField() {
		fmt.Printf("cannot insert after fields\n") // todo: dialog, disable menu
	}
	par := g.SrcNode.Ptr.Parent()
	if par == nil {
		fmt.Printf("no parent to insert in\n") // todo: dialog
	}
	myidx := par.ChildIndex(g.This, 0)
	par.InsertNewChild(nil, myidx+1)
}

func (g *TreeView) SetContinuousSelect() {
	rn := g.RootWidget
	bitflag.Set(&rn.NodeFlags, int(NodeFlagContinuousSelect))
}

func (g *TreeView) SetExtendSelect() {
	rn := g.RootWidget
	bitflag.Set(&rn.NodeFlags, int(NodeFlagExtendSelect))
}

func (g *TreeView) ClearSelectMods() {
	rn := g.RootWidget
	bitflag.Clear(&rn.NodeFlags, int(NodeFlagContinuousSelect))
	bitflag.Clear(&rn.NodeFlags, int(NodeFlagExtendSelect))
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

// qt calls the open / close thing a "branch"
// http://doc.qt.io/qt-5/stylesheet-examples.html#customizing-qtreeview

func (g *TreeView) ConfigParts() {
	// todo: add some styles for button layout
	g.Parts.Lay = LayoutRow
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	config.Add(KiT_Action, "Branch")
	config.Add(KiT_Space, "Space")
	config.Add(KiT_Label, "Label")
	config.Add(KiT_Stretch, "Stretch")
	config.Add(KiT_MenuButton, "Menu")
	updt := g.Parts.ConfigChildren(config, false) // not unique names

	// todo: create a toggle button widget that has 2 different states with icons pre-loaded
	wbk, _ := g.Parts.Child(tvBranchIdx)
	wb := wbk.(*Action)
	if g.IsCollapsed() {
		wb.Icon = IconByName("widget-right-wedge")
	} else {
		wb.Icon = IconByName("widget-down-wedge")
	}
	if updt {
		g.PartStyleProps(wb.This, TreeViewProps[0])
		wb.ActionSig.Connect(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			tv, ok := recv.(*TreeView)
			if ok {
				tv.ToggleCollapse()
			}
		})
	}

	lbk, _ := g.Parts.Child(tvLabelIdx)
	lbl := lbk.(*Label)
	lbl.Text = g.Label()
	if updt {
		g.PartStyleProps(lbl.This, TreeViewProps[0])
		// lbl.ReceiveEventType(oswin.MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		// 	_, ok := recv.(*TreeView)
		// 	if !ok {
		// 		return
		// 	}
		// 	// todo: specifically on down?  needed this for emergent
		// })
		lbl.ReceiveEventType(oswin.MouseUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
			lb, _ := recv.(*Label)
			tv := lb.Parent().Parent().(*TreeView)
			tv.SelectNodeAction()
		})
	}

	mbk, _ := g.Parts.Child(tvMenuIdx)
	mb := mbk.(*MenuButton)
	if updt {
		mb.Text = "..."
		g.PartStyleProps(mb.This, TreeViewProps[0])

		mb.AddMenuText("Insert After", g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			tv := recv.(*TreeView)
			tv.InsertAfter()
		})
	}
}

func (g *TreeView) Init2D() {
	g.Init2DWidget()
	g.ConfigParts()
	g.ReceiveEventType(oswin.KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab := recv.(*TreeView)
		kt := d.(oswin.KeyTypedEvent)
		// fmt.Printf("TreeView key: %v\n", kt.Chord)
		kf := KeyFun(kt.Key, kt.Chord)
		switch kf {
		case KeyFunSelectItem:
			ab.SelectNodeAction()
		case KeyFunCancelSelect:
			ab.RootUnselectAll()
		case KeyFunMoveRight:
			ab.Expand()
		case KeyFunMoveLeft:
			ab.Collapse()
			// todo; move down / up -- selectnext / prev
		}
	})
	g.ReceiveEventType(oswin.KeyDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab := recv.(*TreeView)
		kt := d.(oswin.KeyDownEvent)
		kf := KeyFun(kt.Key, "")
		// fmt.Printf("TreeView key down: %v\n", kt.Key)
		switch kf {
		case KeyFunShift:
			ab.SetContinuousSelect()
		case KeyFunCtrl:
			ab.SetExtendSelect()
		}
	})
	g.ReceiveEventType(oswin.KeyUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab := recv.(*TreeView)
		ab.ClearSelectMods()
	})
}

var TreeViewProps = []map[string]interface{}{
	{
		"border-width":     units.NewValue(0, units.Px),
		"border-radius":    units.NewValue(0, units.Px),
		"padding":          units.NewValue(1, units.Px),
		"margin":           units.NewValue(1, units.Px),
		"text-align":       AlignLeft,
		"vertical-align":   AlignTop,
		"color":            color.Black,
		"background-color": "#FFF", // todo: get also from user, type on viewed node
		"#branch": map[string]interface{}{
			// "width":   units.NewValue(1, units.Em),
			// "height":  units.NewValue(1, units.Em),
			"vertical-align": AlignBottom,
			"margin":         units.NewValue(2, units.Px),
			"padding":        units.NewValue(2, units.Px),
			"#icon": map[string]interface{}{
				"width":   units.NewValue(1.5, units.Ex),
				"height":  units.NewValue(1.5, units.Ex),
				"margin":  units.NewValue(2, units.Px),
				"padding": units.NewValue(2, units.Px),
			},
		},
		"#label": map[string]interface{}{
			"margin":           units.NewValue(0, units.Px),
			"padding":          units.NewValue(0, units.Px),
			"background-color": "none",
		},
		"#menu": map[string]interface{}{
			"border-width":        units.NewValue(0, units.Px),
			"border-radius":       units.NewValue(0, units.Px),
			"border-color":        "none",
			"padding":             units.NewValue(2, units.Px),
			"margin":              units.NewValue(0, units.Px),
			"box-shadow.h-offset": units.NewValue(0, units.Px),
			"box-shadow.v-offset": units.NewValue(0, units.Px),
			"box-shadow.blur":     units.NewValue(0, units.Px),
			"indicator":           "none",
		},
	}, { // selected
		"background-color": "#CFC", // todo: also
	}, { // focused
		"background-color": "#CCF", // todo: also
	},
}

func (g *TreeView) Style2D() {
	if g.HasCollapsedParent() {
		return
	}
	g.ConfigParts()
	bitflag.Set(&g.NodeFlags, int(CanFocus))
	g.Style2DWidget(TreeViewProps[0])
	for i := 0; i < int(TreeViewStatesN); i++ {
		g.StateStyles[i] = g.Style
		g.StateStyles[i].SetStyle(nil, &StyleDefault, TreeViewProps[i])
		g.StateStyles[i].SetUnitContext(g.Viewport, Vec2DZero)
	}
}

// TreeView is tricky for alloc because it is both a layout of its children but has to
// maintain its own bbox for its own widget.

func (g *TreeView) Size2D() {
	g.RootWidget = g.RootTreeView() // cache

	g.InitLayout2D()
	if g.HasCollapsedParent() {
		bitflag.Clear(&g.NodeFlags, int(CanFocus))
		return // nothing
	}
	bitflag.Set(&g.NodeFlags, int(CanFocus))
	g.SizeFromParts() // get our size from parts
	g.WidgetSize = g.LayData.AllocSize
	h := g.WidgetSize.Y
	w := g.WidgetSize.X

	if !g.IsCollapsed() {
		// we layout children under us
		for _, kid := range g.Kids {
			_, gi := KiToNode2D(kid)
			if gi != nil {
				h += gi.LayData.AllocSize.Y
				w = math.Max(w, 20+gi.LayData.AllocSize.X) // 20 = indent, use max
			}
		}
	}
	g.LayData.AllocSize = Vec2D{w, h}
	g.WidgetSize.X = w // stretch
}

func (g *TreeView) Layout2DParts(parBBox image.Rectangle) {
	spc := g.Style.BoxSpace()
	g.Parts.LayData.AllocPos = g.LayData.AllocPos.AddVal(spc)
	g.Parts.LayData.AllocSize = g.WidgetSize.AddVal(-2.0 * spc)
	g.Parts.Layout2D(parBBox)
}

func (g *TreeView) Layout2D(parBBox image.Rectangle) {
	if g.HasCollapsedParent() {
		return
	}
	g.ConfigParts()

	psize := g.AddParentPos() // have to add our pos first before computing below:

	rn := g.RootWidget
	// our alloc size is root's size minus our total indentation
	g.LayData.AllocSize.X = rn.LayData.AllocSize.X - (g.LayData.AllocPos.X - rn.LayData.AllocPos.X)
	g.WidgetSize.X = g.LayData.AllocSize.X

	g.LayData.AllocPosOrig = g.LayData.AllocPos
	g.Style.SetUnitContext(g.Viewport, psize) // update units with final layout
	g.Paint.SetUnitContext(g.Viewport, psize) // always update paint
	g.This.(Node2D).ComputeBBox2D(parBBox)

	if Layout2DTrace {
		fmt.Printf("Layout: %v reduced X allocsize: %v rn: %v  pos: %v rn pos: %v\n", g.PathUnique(), g.WidgetSize.X, rn.LayData.AllocSize.X, g.LayData.AllocPos.X, rn.LayData.AllocPos.X)
		fmt.Printf("Layout: %v alloc pos: %v size: %v vpbb: %v winbb: %v\n", g.PathUnique(), g.LayData.AllocPos, g.LayData.AllocSize, g.VpBBox, g.WinBBox)
	}

	g.Layout2DParts(parBBox) // use OUR version
	for i := 0; i < int(TreeViewStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	h := g.WidgetSize.Y
	if !g.IsCollapsed() {
		for _, kid := range g.Kids {
			_, gi := KiToNode2D(kid)
			if gi != nil {
				gi.LayData.AllocPos.Y = h
				gi.LayData.AllocPos.X = 20 // indent
				h += gi.LayData.AllocSize.Y
			}
		}
		g.Layout2DChildren()
	}
}

func (g *TreeView) BBox2D() image.Rectangle {
	// we have unusual situation of bbox != alloc
	tp := g.LayData.AllocPos.ToPointFloor()
	ts := g.WidgetSize.ToPointCeil()
	return image.Rect(tp.X, tp.Y, tp.X+ts.X, tp.Y+ts.Y)
}

func (g *TreeView) ComputeBBox2D(parBBox image.Rectangle) {
	g.ComputeBBox2DWidget(parBBox)
}

func (g *TreeView) ChildrenBBox2D() image.Rectangle {
	ar := g.BBoxFromAlloc() // need to use allocated size which includes children
	if g.Par != nil {       // use parents children bbox to determine where we can draw
		pgi, _ := KiToNode2D(g.Par)
		ar = ar.Intersect(pgi.ChildrenBBox2D())
	}
	return ar
}

func (g *TreeView) Move2D(delta Vec2D, parBBox image.Rectangle) {
	g.Move2DWidget(delta, parBBox)
	g.Move2DChildren(delta)
}

func (g *TreeView) Render2D() {
	if g.HasCollapsedParent() {
		return // nothing
	}
	if g.PushBounds() {
		// reset for next update
		bitflag.Clear(&g.NodeFlags, int(NodeFlagFullReRender))

		if g.IsSelected() {
			g.Style = g.StateStyles[TreeViewSelState]
		} else if g.HasFocus() {
			g.Style = g.StateStyles[TreeViewFocusState]
		} else {
			g.Style = g.StateStyles[TreeViewNormalState]
		}

		// note: this is std except using WidgetSize instead of AllocSize
		pc := &g.Paint
		st := &g.Style
		pc.FontStyle = st.Font
		pc.TextStyle = st.Text
		pc.StrokeStyle.SetColor(&st.Border.Color)
		pc.StrokeStyle.Width = st.Border.Width
		pc.FillStyle.SetColor(&st.Background.Color)
		// g.RenderStdBox()
		pos := g.LayData.AllocPos.AddVal(st.Layout.Margin.Dots)
		sz := g.WidgetSize.AddVal(-2.0 * st.Layout.Margin.Dots)
		g.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
		g.Render2DParts()
		g.PopBounds()
	}
	// we always have to render our kids b/c we could be out of scope but they could be in!
	g.Render2DChildren()
}

func (g *TreeView) ReRender2D() (node Node2D, layout bool) {
	if bitflag.Has(g.NodeFlags, int(NodeFlagFullReRender)) {
		// dynamic re-rendering doesn't work b/c we don't clear out the full space we
		// used to take!
		// rwly := g.RootWidget.ParentLayout()
		// if rwly != nil {
		// 	node = rwly.This.(Node2D)
		// 	layout = true
		// } else {
		// node = g.RootWidget.This.(Node2D)
		// layout = false
		// }
		node = nil
		layout = false
	} else {
		node = g.This.(Node2D)
		layout = false
	}
	return
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

var KiT_TabWidget = kit.Types.AddType(&TabWidget{}, nil)

// set the source Ki Node that generates our tabs
func (g *TabWidget) SetSrcNode(k ki.Ki) {
	g.SrcNode.Ptr = k
	k.NodeSignal().Connect(g.This, SrcNodeSignal) // we recv signals from source
	nm := "TabViewOf_" + k.UniqueName()
	if g.Nm == "" {
		g.SetName(nm)
	}
	g.InitTabWidget()
}

// todo: various other ways of selecting tabs..

// select tab at given index
func (g *TabWidget) SelectTabIndex(idx int) error {
	tabrow := g.TabRowLayout()
	tbk, err := tabrow.Child(idx)
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
	tfk, err := tabstack.Child(idx)
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
	ch, _ := g.Child(0)
	return ch.(*Layout)
}

// get the row layout of tabs across the top of the tab widget
func (g *TabWidget) TabRowLayout() *Layout {
	tabcol := g.TabColLayout()
	ch, _ := tabcol.Child(0)
	return ch.(*Layout)
}

// get the stacked layout of tab frames
func (g *TabWidget) TabStackLayout() *Layout {
	tabcol := g.TabColLayout()
	ch, _ := tabcol.Child(1)
	return ch.(*Layout)
}

// unselect all tabs
func (g *TabWidget) UnselectAllTabButtons() {
	tabrow := g.TabRowLayout()
	for _, tbk := range tabrow.Kids {
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
			butidx := tabrow.ChildIndex(send, 0)
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
	"text-align":          "center",
	"color":               "black",
	"background-color":    "#EEF",
}

// make the initial tab frames for src node
func (g *TabWidget) InitTabs() {
	tabrow := g.TabRowLayout()
	tabstack := g.TabStackLayout()
	if g.SrcNode.Ptr == nil {
		return
	}
	skids := g.SrcNode.Ptr.Children()
	for _, sk := range skids {
		nm := "TabFrameOf_" + sk.UniqueName()
		tf := tabstack.AddNewChildNamed(KiT_Frame, nm).(*Frame)
		tf.Lay = LayoutCol
		tf.SetProp("max-width", -1.0) // stretch flex
		tf.SetProp("max-height", -1.0)
		nm = "TabOf_" + sk.UniqueName()
		tb := tabrow.AddNewChildNamed(KiT_Button, nm).(*Button) // todo make tab button
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
func (g *TabWidget) InitTabWidget() {
	if len(g.Kids) == 1 {
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

func (g *TabWidget) Init2D() {
	g.Init2DWidget()
}

func (g *TabWidget) Style2D() {
	g.Style2DWidget(nil)
}

func (g *TabWidget) Size2D() {
	g.InitLayout2D()
}

func (g *TabWidget) Layout2D(parBBox image.Rectangle) {
	g.Layout2DWidget(parBBox)
	g.Layout2DChildren()
}

func (g *TabWidget) BBox2D() image.Rectangle {
	return g.BBoxFromAlloc()
}

func (g *TabWidget) ComputeBBox2D(parBBox image.Rectangle) {
	g.ComputeBBox2DWidget(parBBox)
}

func (g *TabWidget) Move2D(delta Vec2D, parBBox image.Rectangle) {
	g.Move2DWidget(delta, parBBox)
	g.Move2DChildren(delta)
}

func (g *TabWidget) ChildrenBBox2D() image.Rectangle {
	return g.ChildrenBBox2DWidget()
}

func (g *TabWidget) Render2D() {
	if g.PushBounds() {
		g.Render2DChildren()
		g.PopBounds()
	}
}

func (g *TabWidget) ReRender2D() (node Node2D, layout bool) {
	node = g.This.(Node2D)
	layout = false
	return
}

func (g *TabWidget) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &TabWidget{}
