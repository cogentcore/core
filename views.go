// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"reflect"

	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  TreeView -- a widget that graphically represents / manipulates a Ki Tree

// signals that treeview can send
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

// todo: could create an interface for TreeView types -- right now just using
// EmbeddedStruct to make everything general for anything that might embed TreeView

//////////////////////////////////////////////////////////////////////////////
//    End-User API

// set the source node that we are viewing
func (g *TreeView) SetSrcNode(sk ki.Ki) {
	g.SrcNode.Ptr = sk
	sk.NodeSignal().Connect(g.This, SrcNodeSignal) // we recv signals from source
	g.SyncToSrc()
}

// sync with the source tree
func (g *TreeView) SyncToSrc() {
	g.UpdateStart()
	sk := g.SrcNode.Ptr
	nm := "ViewOf_" + sk.UniqueName()
	if g.Nm != nm {
		g.SetName(nm)
	}
	skids := sk.Children()
	tnl := make(kit.TypeAndNameList, 0, len(skids))
	typ := g.This.Type() // always make our type
	for _, skid := range skids {
		tnl.Add(typ, "ViewOf_"+skid.UniqueName())
	}
	updt := g.ConfigChildren(tnl, false) // preserves existing to greatest extent possible
	if updt {
		bitflag.Set(&g.NodeFlags, int(NodeFlagFullReRender))
		win := g.ParentWindow()
		if win != nil {
			for _, vki := range g.Deleted {
				vk := vki.EmbeddedStruct(KiT_TreeView).(*TreeView)
				vk.DisconnectAllEventsTree(win)
			}
		}
	}
	for i, vki := range g.Kids {
		vk := vki.EmbeddedStruct(KiT_TreeView).(*TreeView)
		skid := sk.Children()[i]
		if vk.SrcNode.Ptr != skid {
			vk.SetSrcNode(skid)
		}
	}
	g.UpdateEnd()
}

// function for receiving node signals from our SrcNode
func SrcNodeSignal(tvki, send ki.Ki, sig int64, data interface{}) {
	tv := tvki.EmbeddedStruct(KiT_TreeView).(*TreeView)
	// fmt.Printf("treeview: %v got signal: %v from node: %v  data: %v\n", tv.PathUnique(), ki.NodeSignals(sig), send.PathUnique(), data)
	if bitflag.HasMask(*send.Flags(), int64(ki.ChildUpdateFlagsMask)) {
		tv.SyncToSrc()
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
func (g *TreeView) SelectAction() {
	win := g.Viewport.ParentWindow()
	if win != nil {
		win.UpdateStart()
	}
	rn := g.RootWidget
	if bitflag.Has(rn.NodeFlags, int(NodeFlagExtendSelect)) {
		if g.IsSelected() {
			g.Unselect()
		} else {
			g.Select()
		}
	} else { // todo: continuous a bit trickier
		if g.IsSelected() {
			// nothing..
		} else {
			rn.UnselectAll()
			g.Select()
		}
	}
	if win != nil {
		win.UpdateEnd()
	}
}

func (g *TreeView) Select() {
	if !g.IsSelected() {
		g.UpdateStart()
		bitflag.Set(&g.NodeFlags, int(NodeFlagSelected))
		g.GrabFocus() // focus always follows select  todo: option
		g.TreeViewSig.Emit(g.This, int64(NodeSelected), nil)
		g.UpdateEnd()
	}
}

func (g *TreeView) Unselect() {
	if g.IsSelected() {
		g.UpdateStart()
		bitflag.Clear(&g.NodeFlags, int(NodeFlagSelected))
		g.TreeViewSig.Emit(g.This, int64(NodeUnselected), nil)
		g.UpdateEnd()
	}
}

// unselect everything below me -- call on Root to clear all
func (g *TreeView) UnselectAll() {
	win := g.Viewport.ParentWindow()
	if win != nil {
		win.UpdateStart()
	}
	g.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		_, gi := KiToNode2D(k)
		if gi == nil {
			return false
		}
		if k.TypeEmbeds(KiT_TreeView) {
			nw := k.EmbeddedStruct(KiT_TreeView).(*TreeView)
			nw.Unselect()
			return true
		} else {
			return false
		}
	})
	if win != nil {
		win.UpdateEnd()
	}
}

// unselect everything below me -- call on Root to clear all
func (g *TreeView) RootUnselectAll() {
	g.RootWidget.UnselectAll()
}

func (g *TreeView) MoveDown() {
	if g.Par == nil {
		return
	}
	if g.IsCollapsed() || !g.HasChildren() { // next sibling
		g.MoveDownSibling()
	} else {
		if g.HasChildren() {
			nn := g.Child(0).EmbeddedStruct(KiT_TreeView).(*TreeView)
			if nn != nil {
				nn.SelectAction()
			}
		}
	}
}

// move down only to siblings, not down into children
func (g *TreeView) MoveDownSibling() {
	if g.Par == nil {
		return
	}
	if g == g.RootWidget {
		return
	}
	myidx := g.Index()
	if myidx < len(g.Par.Children())-1 {
		nn := g.Par.Child(myidx + 1).EmbeddedStruct(KiT_TreeView).(*TreeView)
		if nn != nil {
			nn.SelectAction()
		}
	} else {
		g.Par.EmbeddedStruct(KiT_TreeView).(*TreeView).MoveDownSibling() // try up
	}
}

func (g *TreeView) MoveUp() {
	if g.Par == nil || g == g.RootWidget {
		return
	}
	myidx := g.Index()
	if myidx > 0 {
		nn := g.Par.Child(myidx - 1).EmbeddedStruct(KiT_TreeView).(*TreeView)
		if nn != nil {
			nn.MoveToLastChild()
		}
	} else {
		if g.Par != nil {
			nn := g.Par.EmbeddedStruct(KiT_TreeView).(*TreeView)
			if nn != nil {
				nn.SelectAction()
			}
		}
	}
}

// move up to the last child under me
func (g *TreeView) MoveToLastChild() {
	if g.Par == nil || g == g.RootWidget {
		return
	}
	if !g.IsCollapsed() && g.HasChildren() {
		nn := g.Child(-1).EmbeddedStruct(KiT_TreeView).(*TreeView)
		if nn != nil {
			nn.MoveToLastChild()
		}
	} else {
		g.SelectAction()
	}
}

func (g *TreeView) Collapse() {
	if !g.IsCollapsed() {
		g.UpdateStart()
		if g.HasChildren() {
			bitflag.Set(&g.NodeFlags, int(NodeFlagFullReRender))
		}
		bitflag.Set(&g.NodeFlags, int(NodeFlagCollapsed))
		g.TreeViewSig.Emit(g.This, int64(NodeCollapsed), nil)
		g.UpdateEnd()
	}
}

func (g *TreeView) Expand() {
	if g.IsCollapsed() {
		g.UpdateStart()
		if g.HasChildren() {
			bitflag.Set(&g.NodeFlags, int(NodeFlagFullReRender))
		}
		bitflag.Clear(&g.NodeFlags, int(NodeFlagCollapsed))
		g.TreeViewSig.Emit(g.This, int64(NodeOpened), nil)
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

// insert a new node in the source tree
func (g *TreeView) SrcInsertAfter() {
	ttl := "TreeView Insert After"
	if g.IsField() {
		// todo: disable menu!
		PromptDialog(g.Viewport, ttl, "Cannot insert after fields", true, false, nil, nil)
		return
	}
	sk := g.SrcNode.Ptr
	par := sk.Parent()
	if par == nil {
		PromptDialog(g.Viewport, ttl, "Cannot insert after the root of the tree", true, false, nil, nil)
		return
	}
	myidx := sk.Index()
	NewKiDialog(g.Viewport, reflect.TypeOf((*Node2D)(nil)).Elem(), ttl, "Number and Type of Items to Insert:", g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(DialogAccepted) {
			tv, _ := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			sk := tv.SrcNode.Ptr
			par := sk.Parent()
			dlg, _ := send.(*Dialog)
			n, typ := NewKiDialogValues(dlg)
			par.UpdateStart()
			for i := 0; i < n; i++ {
				nm := fmt.Sprintf("New%v%v", typ.Name(), myidx+1+i)
				par.InsertNewChildNamed(typ, myidx+1+i, nm)
			}
			par.UpdateEnd()
		}
	})
}

// insert a new node in the source tree
func (g *TreeView) SrcInsertBefore() {
	ttl := "TreeView Insert Before"
	if g.IsField() {
		// todo: disable menu!
		PromptDialog(g.Viewport, ttl, "Cannot insert before fields", true, false, nil, nil)
		return
	}
	sk := g.SrcNode.Ptr
	par := sk.Parent()
	if par == nil {
		PromptDialog(g.Viewport, ttl, "Cannot insert before the root of the tree", true, false, nil, nil)
		return
	}
	myidx := sk.Index()

	NewKiDialog(g.Viewport, reflect.TypeOf((*Node2D)(nil)).Elem(), ttl, "Number and Type of Items to Insert:", g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(DialogAccepted) {
			tv, _ := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			sk := tv.SrcNode.Ptr
			par := sk.Parent()
			dlg, _ := send.(*Dialog)
			n, typ := NewKiDialogValues(dlg)
			par.UpdateStart()
			for i := 0; i < n; i++ {
				nm := fmt.Sprintf("New%v%v", typ.Name(), myidx+i)
				par.InsertNewChildNamed(typ, myidx+i, nm)
			}
			par.UpdateEnd()
		}
	})
}

// add a new child node in the source tree
func (g *TreeView) SrcAddChild() {
	ttl := "TreeView Add Child"
	NewKiDialog(g.Viewport, reflect.TypeOf((*Node2D)(nil)).Elem(), ttl, "Number and Type of Items to Add:", g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(DialogAccepted) {
			tv, _ := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			sk := tv.SrcNode.Ptr
			dlg, _ := send.(*Dialog)
			n, typ := NewKiDialogValues(dlg)
			sk.UpdateStart()
			for i := 0; i < n; i++ {
				nm := fmt.Sprintf("New%v%v", typ.Name(), i)
				sk.AddNewChildNamed(typ, nm)
			}
			sk.UpdateEnd()
		}
	})
}

// delete me in source tree
func (g *TreeView) SrcDelete() {
	if g.IsField() {
		PromptDialog(g.Viewport, "TreeView Delete", "Cannot delete fields", true, false, nil, nil)
		return
	}
	if g.RootWidget.This == g.This {
		PromptDialog(g.Viewport, "TreeView Delete", "Cannot delete the root of the tree", true, false, nil, nil)
		return
	}
	g.MoveDown()
	sk := g.SrcNode.Ptr
	sk.Delete(true)
}

// duplicate item in source tree, add after
func (g *TreeView) SrcDuplicate() {
	if g.IsField() {
		PromptDialog(g.Viewport, "TreeView Duplicate", "Cannot delete fields", true, false, nil, nil)
		return
	}
	sk := g.SrcNode.Ptr
	par := sk.Parent()
	if par == nil {
		PromptDialog(g.Viewport, "TreeView Duplicate", "Cannot duplicate the root of the tree", true, false, nil, nil)
		return
	}
	myidx := sk.Index()
	nm := fmt.Sprintf("%vCopy", sk.Name())
	par.InsertChildNamed(sk.Clone(), myidx+1, nm)
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
	g.Parts.Lay = LayoutRow
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	config.Add(KiT_Action, "Branch")
	config.Add(KiT_Space, "Space")
	config.Add(KiT_Label, "Label")
	config.Add(KiT_Stretch, "Stretch")
	config.Add(KiT_MenuButton, "Menu")
	updt := g.Parts.ConfigChildren(config, false) // not unique names

	// todo: create a toggle button widget that has 2 different states with icons pre-loaded
	wb := g.Parts.Child(tvBranchIdx).(*Action)
	if g.IsCollapsed() {
		wb.Icon = IconByName("widget-right-wedge")
	} else {
		wb.Icon = IconByName("widget-down-wedge")
	}
	if updt {
		g.PartStyleProps(wb.This, TreeViewProps[0])
		wb.ActionSig.Connect(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			tv, _ := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tv.ToggleCollapse()
		})
	}

	lbl := g.Parts.Child(tvLabelIdx).(*Label)
	lbl.Text = g.Label()
	if updt {
		g.PartStyleProps(lbl.This, TreeViewProps[0])
		// lbl.ReceiveEventType(oswin.MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		// 	_, ok := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
		// 	if !ok {
		// 		return
		// 	}
		// 	// todo: specifically on down?  needed this for emergent
		// })
		lbl.ReceiveEventType(oswin.MouseUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
			lb, _ := recv.(*Label)
			tv := lb.Parent().Parent().EmbeddedStruct(KiT_TreeView).(*TreeView)
			tv.SelectAction()
		})
	}

	mb := g.Parts.Child(tvMenuIdx).(*MenuButton)
	if updt {
		mb.Text = "..."
		g.PartStyleProps(mb.This, TreeViewProps[0])

		// todo: shortcuts!
		mb.AddMenuText("Add Child", g.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tv.SrcAddChild()
		})
		mb.AddMenuText("Insert Before", g.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tv.SrcInsertBefore()
		})
		mb.AddMenuText("Insert After", g.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tv.SrcInsertAfter()
		})
		mb.AddMenuText("Duplicate", g.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tv.SrcDuplicate()
		})
		mb.AddMenuText("Delete", g.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tv.SrcDelete()
		})
	}
}

func (g *TreeView) ConfigPartsIfNeeded() {
	lbl := g.Parts.Child(tvLabelIdx).(*Label)
	lbl.Text = g.Label()
}

func (g *TreeView) Init2D() {
	g.Init2DWidget()
	g.ConfigParts()
	g.ReceiveEventType(oswin.KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		tv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
		kt := d.(*oswin.KeyTypedEvent)
		// fmt.Printf("TreeView key: %v\n", kt.Chord)
		kf := KeyFun(kt.Key, kt.Chord)
		switch kf {
		case KeyFunSelectItem:
			tv.SelectAction()
			kt.SetProcessed()
		case KeyFunCancelSelect:
			tv.RootUnselectAll()
			kt.SetProcessed()
		case KeyFunMoveRight:
			tv.Expand()
			kt.SetProcessed()
		case KeyFunMoveLeft:
			tv.Collapse()
			kt.SetProcessed()
		case KeyFunMoveDown:
			tv.MoveDown()
			kt.SetProcessed()
		case KeyFunMoveUp:
			tv.MoveUp()
			kt.SetProcessed()
		case KeyFunDelete:
			tv.SrcDelete()
			kt.SetProcessed()
		case KeyFunDuplicate:
			tv.SrcDuplicate()
			kt.SetProcessed()
		case KeyFunInsert:
			tv.SrcInsertBefore()
			kt.SetProcessed()
		case KeyFunInsertAfter:
			tv.SrcInsertAfter()
			kt.SetProcessed()
		}
	})
	g.ReceiveEventType(oswin.KeyDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
		kt := d.(*oswin.KeyDownEvent)
		kf := KeyFun(kt.Key, "")
		// fmt.Printf("TreeView key down: %v\n", kt.Key)
		switch kf {
		case KeyFunShift:
			ab.SetContinuousSelect()
			kt.SetProcessed()
		case KeyFunCtrl:
			ab.SetExtendSelect()
			kt.SetProcessed()
		}
	})
	g.ReceiveEventType(oswin.KeyUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
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
			"vertical-align": AlignMiddle,
			"margin":         units.NewValue(0, units.Px),
			"padding":        units.NewValue(0, units.Px),
			"#icon": map[string]interface{}{
				"width":   units.NewValue(.8, units.Em), // todo: this has to be .8 else text label doesn't render sometimes
				"height":  units.NewValue(.8, units.Em),
				"margin":  units.NewValue(0, units.Px),
				"padding": units.NewValue(0, units.Px),
			},
		},
		"#space": map[string]interface{}{
			"width": units.NewValue(.5, units.Em),
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
		return // nothing
	}
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
		g.LayData.AllocPos.X = -1000000 // put it very far off screen..
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
	}
	g.Layout2DChildren()
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
		g.ConfigPartsIfNeeded()
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
		// todo: this can be fixed by adding a previous size to layout and it can clear that
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
//  StructView -- a widget that graphically represents / manipulates a struct
//  as a property editor

// signals that structview can send
type StructViewSignals int64

const (
	// struct field was edited -- data is the field name
	StructFieldEdited StructViewSignals = iota
	StructViewSignalsN
)

//go:generate stringer -type=TreeViewSignals

// todo: sub-editor panels with shared menubutton panel at bottom.. not clear that that is necc -- probably better to have each sub-panel fully separate

// StructView represents a struct, creating a property editor of the fields -- constructs Children widgets to show the field names and editor fields for each field, within an overall frame with an optional title, and a button box at the bottom where methods can be invoked
type StructView struct {
	Frame
	Struct        interface{}           `desc:"the struct that we are a view onto"`
	StructViewSig ki.Signal             `json:"-" desc:"signal for StructView -- see StructViewSignals for the types"`
	Title         string                `desc:"title / prompt to show above the editor fields"`
	Fields        []reflect.StructField `desc:"overall field information"`
	FieldVals     []reflect.Value       `desc:"reflect.Value for the fields"`
}

var KiT_StructView = kit.Types.AddType(&StructView{}, nil)

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// SetStruct sets the source struct that we are viewing -- rebuilds the children to represent this struct
func (sv *StructView) SetStruct(st interface{}) {
	sv.Struct = st
	sv.UpdateFromStruct()
}

var StructViewProps = map[string]interface{}{
	"#frame": map[string]interface{}{
		"border-width":        units.NewValue(2, units.Px),
		"margin":              units.NewValue(8, units.Px),
		"padding":             units.NewValue(4, units.Px),
		"box-shadow.h-offset": units.NewValue(4, units.Px),
		"box-shadow.v-offset": units.NewValue(4, units.Px),
		"box-shadow.blur":     units.NewValue(4, units.Px),
		"box-shadow.color":    "#CCC",
	},
	"#title": map[string]interface{}{
		// todo: add "bigger" font
		"max-width":        units.NewValue(-1, units.Px),
		"text-align":       AlignCenter,
		"vertical-align":   AlignTop,
		"background-color": "none",
	},
	"#prompt": map[string]interface{}{
		"max-width":        units.NewValue(-1, units.Px),
		"text-align":       AlignLeft,
		"vertical-align":   AlignTop,
		"background-color": "none",
	},
}

// SetFrame configures view as a frame
func (sv *StructView) SetFrame() {
	sv.Lay = LayoutCol
	sv.PartStyleProps(sv, StructViewProps)
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (g *StructView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	config.Add(KiT_Label, "Title")
	config.Add(KiT_Space, "TitleSpace")
	config.Add(KiT_Layout, "StructGrid")
	config.Add(KiT_Space, "GridSpace")
	config.Add(KiT_Layout, "ButtonBox")
	return config
}

// StdConfig configures a standard setup of the overall Frame
func (sv *StructView) StdConfig() {
	sv.SetFrame()
	config := sv.StdFrameConfig()
	sv.ConfigChildren(config, false)
}

// SetTitle sets the title and updates the Title label
func (sv *StructView) SetTitle(title string) {
	sv.Title = title
	lab, _ := sv.TitleWidget()
	if lab != nil {
		lab.Text = title
	}
}

// Title returns the title label widget, and its index, within frame -- nil, -1 if not found
func (sv *StructView) TitleWidget() (*Label, int) {
	idx := sv.ChildIndexByName("Title", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Label), idx
}

// StructGrid returns the StructGrid grid layout widget, which contains all the fields and values, and its index, within frame -- nil, -1 if not found
func (sv *StructView) StructGrid() (*Layout, int) {
	idx := sv.ChildIndexByName("StructGrid", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (sv *StructView) ButtonBox() (*Layout, int) {
	idx := sv.ChildIndexByName("ButtonBox", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ConfigStructGrid configures the StructGrid for the current struct
func (sv *StructView) ConfigStructGrid() {
	if sv.Struct == nil {
		return
	}
	sg, _ := sv.StructGrid()
	if sg == nil {
		return
	}
	sg.Lay = LayoutGrid
	sg.SetProp("columns", 2)
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	// always start fresh!
	sv.Fields = make([]reflect.StructField, 0)
	sv.FieldVals = make([]reflect.Value, 0)
	kit.FlatFieldsValueFun(sv.Struct, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		// todo: check tags, skip various etc
		labnm := fmt.Sprintf("Lbl%v", field.Name)
		valnm := fmt.Sprintf("Val%v", field.Name)
		config.Add(KiT_Label, labnm)
		config.Add(KiT_TextField, valnm) // todo: extend to diff types using interface..
		sv.Fields = append(sv.Fields, field)
		sv.FieldVals = append(sv.FieldVals, fieldVal)
		return true
	})
	sg.ConfigChildren(config, false)
	for i, fv := range sv.FieldVals {
		lbl := sg.Child(i * 2).(*Label)
		lbl.Text = sv.Fields[i].Name
		tf := sg.Child(i*2 + 1).(*TextField)
		tf.SetProp("max-width", -1)
		tf.Text = kit.ToString(fv.Interface())
		tf.TextFieldSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			// ssv, _ := recv.EmbeddedStruct(KiT_StructView).(*StructView)
			// tf := send.(*TextField)
			// // todo: set value from text field!
		})
	}
}

func (sv *StructView) UpdateFromStruct() {
	sv.StdConfig()
	if sv.Title == "" {
		typ := kit.NonPtrType(reflect.TypeOf(sv.Struct))
		sv.SetTitle(fmt.Sprintf("Properties of %v", typ.Name()))
	}
	sv.ConfigStructGrid()
}

////////////////////////////////////////////////////
// Node2D interface

// check for interface implementation
var _ Node2D = &StructView{}

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
	idx, err := tabrow.Children().ValidIndex(idx)
	if err != nil {
		return err
	}
	tbk := tabrow.Child(idx)
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
func (g *TabWidget) TabColLayout() *Layout {
	g.InitTabWidget()
	return g.Child(0).(*Layout)
}

// get the row layout of tabs across the top of the tab widget
func (g *TabWidget) TabRowLayout() *Layout {
	tabcol := g.TabColLayout()
	return tabcol.Child(0).(*Layout)
}

// get the stacked layout of tab frames
func (g *TabWidget) TabStackLayout() *Layout {
	tabcol := g.TabColLayout()
	return tabcol.Child(1).(*Layout)
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
