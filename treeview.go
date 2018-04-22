// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"reflect"

	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/oswin/key"
	"github.com/rcoreilly/goki/gi/oswin/mouse"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  TreeView -- a widget that graphically represents / manipulates a Ki Tree

// signals that treeview can send -- these are all sent from the root tree
// view widget node, with data being the relevant node widget
type TreeViewSignals int64

const (
	// node was selected
	TreeViewSelected TreeViewSignals = iota
	// TreeView unselected
	TreeViewUnselected
	// closed TreeView was opened
	TreeViewOpened
	// open TreeView was closed -- children not visible
	TreeViewClosed
	TreeViewSignalsN
)

//go:generate stringer -type=TreeViewSignals

// todo: continuous select, extend select can now be read directly from mouse event

// these extend NodeBase NodeFlags to hold TreeView state
const (
	// node is closed
	TreeViewFlagClosed NodeFlags = NodeFlagsN + iota
	// node is selected
	TreeViewFlagSelected
	// the shift key was pressed, putting the selection mode into continous selection mode
	TreeViewFlagContinuousSelect
	// the ctrl / cmd key was pressed, putting the selection mode into continous selection mode
	TreeViewFlagExtendSelect
)

// mutually-exclusive tree view states -- determines appearance
type TreeViewStates int32

const (
	// normal state -- there but not being interacted with
	TreeViewActive TreeViewStates = iota
	// selected
	TreeViewSel
	// in focus -- will respond to keyboard input
	TreeViewFocus
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
	TreeViewSig ki.Signal              `json:"-" xml:"-" desc:"signal for TreeView -- all are emitted from the root tree view widget, with data = affected node -- see TreeViewSignals for the types"`
	StateStyles [TreeViewStatesN]Style `json:"-" xml:"-" desc:"styles for different states of the widget -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	WidgetSize  Vec2D                  `desc:"just the size of our widget -- our alloc includes all of our children, but we only draw us"`
	RootWidget  *TreeView              `json:"-" desc:"cached root widget"`
}

var KiT_TreeView = kit.Types.AddType(&TreeView{}, nil)

// todo: could create an interface for TreeView types -- right now just using
// EmbeddedStruct to make everything general for anything that might embed TreeView

//////////////////////////////////////////////////////////////////////////////
//    End-User API

// set the source node that we are viewing
func (tv *TreeView) SetSrcNode(sk ki.Ki) {
	updt := false
	if tv.SrcNode.Ptr != sk {
		updt = tv.UpdateStart()
		tv.SrcNode.Ptr = sk
	}
	sk.NodeSignal().Connect(tv.This, SrcNodeSignal) // we recv signals from source
	tv.SyncToSrc()
	tv.UpdateEnd(updt)
}

// sync with the source tree
func (tv *TreeView) SyncToSrc() {
	sk := tv.SrcNode.Ptr
	nm := "ViewOf_" + sk.UniqueName()
	tv.SetName(nm)
	vcprop := "view-closed"
	skids := sk.Children()
	tnl := make(kit.TypeAndNameList, 0, len(skids))
	typ := tv.This.Type() // always make our type
	flds := make([]ki.Ki, 0)
	fldClosed := make([]bool, 0)
	sk.FuncFields(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		flds = append(flds, k)
		tnl.Add(typ, "ViewOf_"+k.Name())
		ft := sk.FieldTag(k.Name(), vcprop)
		cls := false
		if vc, ok := kit.ToBool(ft); ok && vc {
			cls = true
		} else {
			if vc, ok := kit.ToBool(k.Prop(vcprop, false, true)); vc && ok {
				cls = true
			}
		}
		fldClosed = append(fldClosed, cls)
		return true
	})
	for _, skid := range skids {
		tnl.Add(typ, "ViewOf_"+skid.UniqueName())
	}
	mods, updt := tv.ConfigChildren(tnl, false)
	if mods {
		tv.SetFullReRender()
		win := tv.ParentWindow()
		if win != nil {
			for _, vki := range tv.Deleted {
				vk := vki.EmbeddedStruct(KiT_TreeView).(*TreeView)
				vk.DisconnectAllEventsTree(win)
			}
		}
	}
	idx := 0
	for i, fld := range flds {
		vk := tv.Kids[idx].EmbeddedStruct(KiT_TreeView).(*TreeView)
		if vk.SrcNode.Ptr != fld {
			vk.SetSrcNode(fld)
			vk.SetClosedState(fldClosed[i])
		}
		idx++
	}
	for _, skid := range sk.Children() {
		vk := tv.Kids[idx].EmbeddedStruct(KiT_TreeView).(*TreeView)
		if vk.SrcNode.Ptr != skid {
			vk.SetSrcNode(skid)
			if vc, ok := kit.ToBool(skid.Prop(vcprop, false, true)); vc && ok {
				vk.SetClosed()
			}
		}
		idx++
	}
	tv.UpdateEnd(updt)
}

// function for receiving node signals from our SrcNode
func SrcNodeSignal(tvki, send ki.Ki, sig int64, data interface{}) {
	tv := tvki.EmbeddedStruct(KiT_TreeView).(*TreeView)
	// fmt.Printf("treeview: %v got signal: %v from node: %v  data: %v  flags %v\n", tv.PathUnique(), ki.NodeSignals(sig), send.PathUnique(), data, *send.Flags())
	if data != nil {
		dflags := data.(int64)
		if bitflag.HasMask(dflags, int64(ki.ChildUpdateFlagsMask)) {
			tv.SyncToSrc()
		} else if bitflag.HasMask(dflags, int64(ki.ValUpdateFlagsMask)) {
			tv.UpdateSig()
		}
	}
}

// return a list of the currently-selected source nodes
func (tv *TreeView) SelectedSrcNodes() ki.Slice {
	sn := make(ki.Slice, 0)
	tv.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
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
func (tv *TreeView) SelectedTreeViews() []*TreeView {
	sn := make([]*TreeView, 0)
	tv.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
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
func (tv *TreeView) RootTreeView() *TreeView {
	rn := tv
	tv.FuncUp(0, tv.This, func(k ki.Ki, level int, d interface{}) bool {
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

// is this node itself closed?
func (tv *TreeView) IsClosed() bool {
	return bitflag.Has(tv.Flag, int(TreeViewFlagClosed))
}

func (tv *TreeView) SetClosed() {
	bitflag.Set(&tv.Flag, int(TreeViewFlagClosed))
}

func (tv *TreeView) SetClosedState(closed bool) {
	bitflag.SetState(&tv.Flag, closed, int(TreeViewFlagClosed))
}

// does this node have a closed parent? if so, don't render!
func (tv *TreeView) HasClosedParent() bool {
	pcol := false
	tv.FuncUpParent(0, tv.This, func(k ki.Ki, level int, d interface{}) bool {
		_, pg := KiToNode2D(k)
		if pg == nil {
			return false
		}
		if pg.TypeEmbeds(KiT_TreeView) {
			// nw := pg.EmbeddedStruct(KiT_TreeView).(*TreeView)
			if bitflag.Has(pg.Flag, int(TreeViewFlagClosed)) {
				pcol = true
				return false
			}
		}
		return true
	})
	return pcol
}

// is this node selected?
func (tv *TreeView) IsSelected() bool {
	return bitflag.Has(tv.Flag, int(TreeViewFlagSelected))
}

func (tv *TreeView) Label() string {
	return tv.SrcNode.Ptr.Name()
}

// a select action has been received (e.g., a mouse click) -- translate into
// selection updates
func (tv *TreeView) SelectAction() {
	win := tv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	rn := tv.RootWidget
	if bitflag.Has(rn.Flag, int(TreeViewFlagExtendSelect)) {
		if tv.IsSelected() {
			tv.Unselect()
		} else {
			tv.Select()
		}
	} else { // todo: continuous a bit trickier
		if tv.IsSelected() {
			// nothing..
		} else {
			rn.UnselectAll()
			tv.Select()
		}
	}
	if win != nil {
		win.UpdateEnd(updt)
	}
}

func (tv *TreeView) Select() {
	if !tv.IsSelected() {
		bitflag.Set(&tv.Flag, int(TreeViewFlagSelected))
		tv.GrabFocus() // focus always follows select  todo: option
		tv.RootWidget.TreeViewSig.Emit(tv.RootWidget.This, int64(TreeViewSelected), tv.This)
		tv.UpdateSig()
	}
}

func (tv *TreeView) Unselect() {
	if tv.IsSelected() {
		bitflag.Clear(&tv.Flag, int(TreeViewFlagSelected))
		tv.RootWidget.TreeViewSig.Emit(tv.RootWidget.This, int64(TreeViewUnselected), tv.This)
		tv.UpdateSig()
	}
}

// unselect everything below me -- call on Root to clear all
func (tv *TreeView) UnselectAll() {
	win := tv.Viewport.Win
	updt := false
	if win != nil {
		updt = win.UpdateStart()
	}
	tv.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
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
		win.UpdateEnd(updt)
	}
}

// unselect everything below me -- call on Root to clear all
func (tv *TreeView) RootUnselectAll() {
	tv.RootWidget.UnselectAll()
}

func (tv *TreeView) MoveDown() {
	if tv.Par == nil {
		return
	}
	if tv.IsClosed() || !tv.HasChildren() { // next sibling
		tv.MoveDownSibling()
	} else {
		if tv.HasChildren() {
			nn := tv.Child(0).EmbeddedStruct(KiT_TreeView).(*TreeView)
			if nn != nil {
				nn.SelectAction()
			}
		}
	}
}

// move down only to siblings, not down into children
func (tv *TreeView) MoveDownSibling() {
	if tv.Par == nil {
		return
	}
	if tv == tv.RootWidget {
		return
	}
	myidx := tv.Index()
	if myidx < len(tv.Par.Children())-1 {
		nn := tv.Par.Child(myidx + 1).EmbeddedStruct(KiT_TreeView).(*TreeView)
		if nn != nil {
			nn.SelectAction()
		}
	} else {
		tv.Par.EmbeddedStruct(KiT_TreeView).(*TreeView).MoveDownSibling() // try up
	}
}

func (tv *TreeView) MoveUp() {
	if tv.Par == nil || tv == tv.RootWidget {
		return
	}
	myidx := tv.Index()
	if myidx > 0 {
		nn := tv.Par.Child(myidx - 1).EmbeddedStruct(KiT_TreeView).(*TreeView)
		if nn != nil {
			nn.MoveToLastChild()
		}
	} else {
		if tv.Par != nil {
			nn := tv.Par.EmbeddedStruct(KiT_TreeView).(*TreeView)
			if nn != nil {
				nn.SelectAction()
			}
		}
	}
}

// move up to the last child under me
func (tv *TreeView) MoveToLastChild() {
	if tv.Par == nil || tv == tv.RootWidget {
		return
	}
	if !tv.IsClosed() && tv.HasChildren() {
		nn := tv.Child(-1).EmbeddedStruct(KiT_TreeView).(*TreeView)
		if nn != nil {
			nn.MoveToLastChild()
		}
	} else {
		tv.SelectAction()
	}
}

func (tv *TreeView) Close() {
	if !tv.IsClosed() {
		updt := tv.UpdateStart()
		if tv.HasChildren() {
			tv.SetFullReRender()
		}
		bitflag.Set(&tv.Flag, int(TreeViewFlagClosed))
		tv.RootWidget.TreeViewSig.Emit(tv.RootWidget.This, int64(TreeViewClosed), tv.This)
		tv.UpdateEnd(updt)
	}
}

func (tv *TreeView) Open() {
	if tv.IsClosed() {
		updt := tv.UpdateStart()
		if tv.HasChildren() {
			tv.SetFullReRender()
		}
		bitflag.Clear(&tv.Flag, int(TreeViewFlagClosed))
		tv.RootWidget.TreeViewSig.Emit(tv.RootWidget.This, int64(TreeViewOpened), tv.This)
		tv.UpdateEnd(updt)
	}
}

func (tv *TreeView) ToggleClose() {
	if tv.IsClosed() {
		tv.Open()
	} else {
		tv.Close()
	}
}

// insert a new node in the source tree
func (tv *TreeView) SrcInsertAfter() {
	ttl := "TreeView Insert After"
	if tv.IsField() {
		// todo: disable menu!
		PromptDialog(tv.Viewport, ttl, "Cannot insert after fields", true, false, nil, nil)
		return
	}
	sk := tv.SrcNode.Ptr
	par := sk.Parent()
	if par == nil {
		PromptDialog(tv.Viewport, ttl, "Cannot insert after the root of the tree", true, false, nil, nil)
		return
	}
	myidx := sk.Index()
	NewKiDialog(tv.Viewport, reflect.TypeOf((*Node2D)(nil)).Elem(), ttl, "Number and Type of Items to Insert:", tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(DialogAccepted) {
			tv, _ := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			sk := tv.SrcNode.Ptr
			par := sk.Parent()
			dlg, _ := send.(*Dialog)
			n, typ := NewKiDialogValues(dlg)
			updt := par.UpdateStart()
			for i := 0; i < n; i++ {
				nm := fmt.Sprintf("New%v%v", typ.Name(), myidx+1+i)
				par.InsertNewChild(typ, myidx+1+i, nm)
			}
			par.UpdateEnd(updt)
		}
	})
}

// insert a new node in the source tree
func (tv *TreeView) SrcInsertBefore() {
	ttl := "TreeView Insert Before"
	if tv.IsField() {
		// todo: disable menu!
		PromptDialog(tv.Viewport, ttl, "Cannot insert before fields", true, false, nil, nil)
		return
	}
	sk := tv.SrcNode.Ptr
	par := sk.Parent()
	if par == nil {
		PromptDialog(tv.Viewport, ttl, "Cannot insert before the root of the tree", true, false, nil, nil)
		return
	}
	myidx := sk.Index()

	NewKiDialog(tv.Viewport, reflect.TypeOf((*Node2D)(nil)).Elem(), ttl, "Number and Type of Items to Insert:", tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(DialogAccepted) {
			tv, _ := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			sk := tv.SrcNode.Ptr
			par := sk.Parent()
			dlg, _ := send.(*Dialog)
			n, typ := NewKiDialogValues(dlg)
			updt := par.UpdateStart()
			for i := 0; i < n; i++ {
				nm := fmt.Sprintf("New%v%v", typ.Name(), myidx+i)
				par.InsertNewChild(typ, myidx+i, nm)
			}
			par.UpdateEnd(updt)
		}
	})
}

// add a new child node in the source tree
func (tv *TreeView) SrcAddChild() {
	ttl := "TreeView Add Child"
	NewKiDialog(tv.Viewport, reflect.TypeOf((*Node2D)(nil)).Elem(), ttl, "Number and Type of Items to Add:", tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(DialogAccepted) {
			tv, _ := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			sk := tv.SrcNode.Ptr
			dlg, _ := send.(*Dialog)
			n, typ := NewKiDialogValues(dlg)
			updt := sk.UpdateStart()
			for i := 0; i < n; i++ {
				nm := fmt.Sprintf("New%v%v", typ.Name(), i)
				sk.AddNewChild(typ, nm)
			}
			sk.UpdateEnd(updt)
		}
	})
}

// delete me in source tree
func (tv *TreeView) SrcDelete() {
	if tv.IsField() {
		PromptDialog(tv.Viewport, "TreeView Delete", "Cannot delete fields", true, false, nil, nil)
		return
	}
	if tv.RootWidget.This == tv.This {
		PromptDialog(tv.Viewport, "TreeView Delete", "Cannot delete the root of the tree", true, false, nil, nil)
		return
	}
	tv.MoveDown()
	sk := tv.SrcNode.Ptr
	sk.Delete(true)
}

// duplicate item in source tree, add after
func (tv *TreeView) SrcDuplicate() {
	if tv.IsField() {
		PromptDialog(tv.Viewport, "TreeView Duplicate", "Cannot delete fields", true, false, nil, nil)
		return
	}
	sk := tv.SrcNode.Ptr
	par := sk.Parent()
	if par == nil {
		PromptDialog(tv.Viewport, "TreeView Duplicate", "Cannot duplicate the root of the tree", true, false, nil, nil)
		return
	}
	myidx := sk.Index()
	nm := fmt.Sprintf("%vCopy", sk.Name())
	nwkid := sk.Clone()
	nwkid.SetName(nm)
	par.InsertChild(nwkid, myidx+1)
}

func (tv *TreeView) SetContinuousSelect() {
	rn := tv.RootWidget
	bitflag.Set(&rn.Flag, int(TreeViewFlagContinuousSelect))
}

func (tv *TreeView) SetExtendSelect() {
	rn := tv.RootWidget
	bitflag.Set(&rn.Flag, int(TreeViewFlagExtendSelect))
}

func (tv *TreeView) ClearSelectMods() {
	rn := tv.RootWidget
	bitflag.Clear(&rn.Flag, int(TreeViewFlagContinuousSelect))
	bitflag.Clear(&rn.Flag, int(TreeViewFlagExtendSelect))
}

////////////////////////////////////////////////////
// Node2D interface

// qt calls the open / close thing a "branch"
// http://doc.qt.io/qt-5/stylesheet-examples.html#customizing-qtreeview

func (tv *TreeView) ConfigParts() {
	tv.Parts.Lay = LayoutRow
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	config.Add(KiT_CheckBox, "branch")
	config.Add(KiT_Space, "space")
	config.Add(KiT_Label, "label")
	config.Add(KiT_Stretch, "stretch")
	config.Add(KiT_MenuButton, "menu")
	mods, updt := tv.Parts.ConfigChildren(config, false) // not unique names

	wb := tv.Parts.Child(tvBranchIdx).(*CheckBox)
	wb.Icon = IconByName("widget-wedge-down") // todo: style
	wb.IconOff = IconByName("widget-wedge-right")
	if mods {
		tv.PartStyleProps(wb.This, TreeViewProps[0])
		wb.ButtonSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(ButtonToggled) {
				tvr, _ := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
				tvr.ToggleClose()
			}
		})
	}

	lbl := tv.Parts.Child(tvLabelIdx).(*Label)
	lbl.Text = tv.Label()
	if mods {
		tv.PartStyleProps(lbl.This, TreeViewProps[0])
		lbl.ReceiveEventType(oswin.MouseEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
			lb, _ := recv.(*Label)
			me := d.(*mouse.Event)
			if me.Action == mouse.Release {
				tv := lb.Parent().Parent().EmbeddedStruct(KiT_TreeView).(*TreeView)
				tv.SelectAction()
			}
		})
	}

	mb := tv.Parts.Child(tvMenuIdx).(*MenuButton)
	if mods {
		mb.Text = "..."
		tv.PartStyleProps(mb.This, TreeViewProps[0])

		// todo: shortcuts!
		mb.AddMenuText("Add Child", tv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tv.SrcAddChild()
		})
		mb.AddMenuText("Insert Before", tv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tv.SrcInsertBefore()
		})
		mb.AddMenuText("Insert After", tv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tv.SrcInsertAfter()
		})
		mb.AddMenuText("Duplicate", tv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tv.SrcDuplicate()
		})
		mb.AddMenuText("Delete", tv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
			tv.SrcDelete()
		})
	}
	tv.Parts.UpdateEnd(updt)
}

func (tv *TreeView) ConfigPartsIfNeeded() {
	if !tv.Parts.HasChildren() {
		tv.ConfigParts()
	}
	lbl := tv.Parts.Child(tvLabelIdx).(*Label)
	lbl.Text = tv.Label()
	wb := tv.Parts.Child(tvBranchIdx).(*CheckBox)
	wb.SetChecked(!tv.IsClosed())
}

func (tv *TreeView) Init2D() {
	tv.Init2DWidget()
	tv.ConfigParts()
	tv.ReceiveEventType(oswin.KeyChordEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		tv := recv.EmbeddedStruct(KiT_TreeView).(*TreeView)
		kt := d.(*key.ChordEvent)
		// fmt.Printf("TreeView key: %v\n", kt.Chord)
		kf := KeyFun(kt.ChordString())
		switch kf {
		case KeyFunSelectItem:
			tv.SelectAction()
			kt.SetProcessed()
		case KeyFunCancelSelect:
			tv.RootUnselectAll()
			kt.SetProcessed()
		case KeyFunMoveRight:
			tv.Open()
			kt.SetProcessed()
		case KeyFunMoveLeft:
			tv.Close()
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
}

var TreeViewProps = []ki.Props{
	{
		"border-width":     units.NewValue(0, units.Px),
		"border-radius":    units.NewValue(0, units.Px),
		"padding":          units.NewValue(1, units.Px),
		"margin":           units.NewValue(1, units.Px),
		"text-align":       AlignLeft,
		"vertical-align":   AlignTop,
		"color":            color.Black,
		"background-color": "#FFF", // todo: get also from user, type on viewed node
		"#branch": ki.Props{
			"vertical-align": AlignMiddle,
			"margin":         units.NewValue(0, units.Px),
			"padding":        units.NewValue(0, units.Px),
			"#icon0": ki.Props{
				"width":   units.NewValue(.8, units.Em), // todo: this has to be .8 else text label doesn't render sometimes
				"height":  units.NewValue(.8, units.Em),
				"margin":  units.NewValue(0, units.Px),
				"padding": units.NewValue(0, units.Px),
			},
			"#icon1": ki.Props{
				"width":   units.NewValue(.8, units.Em), // todo: this has to be .8 else text label doesn't render sometimes
				"height":  units.NewValue(.8, units.Em),
				"margin":  units.NewValue(0, units.Px),
				"padding": units.NewValue(0, units.Px),
			},
		},
		"#space": ki.Props{
			"width": units.NewValue(.5, units.Em),
		},
		"#label": ki.Props{
			"margin":           units.NewValue(0, units.Px),
			"padding":          units.NewValue(0, units.Px),
			"min-width":        units.NewValue(16, units.Ex),
			"background-color": "none",
		},
		"#menu": ki.Props{
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

func (tv *TreeView) Style2D() {
	if tv.HasClosedParent() {
		return
	}
	tv.ConfigParts()
	bitflag.Set(&tv.Flag, int(CanFocus))
	tv.Style2DWidget(TreeViewProps[0])
	for i := 0; i < int(TreeViewStatesN); i++ {
		tv.StateStyles[i] = tv.Style
		tv.StateStyles[i].SetStyle(nil, &StyleDefault, TreeViewProps[i])
		tv.StateStyles[i].SetUnitContext(tv.Viewport, Vec2DZero)
	}
}

// TreeView is tricky for alloc because it is both a layout of its children but has to
// maintain its own bbox for its own widget.

func (tv *TreeView) Size2D() {
	tv.RootWidget = tv.RootTreeView() // cache

	tv.InitLayout2D()
	if tv.HasClosedParent() {
		return // nothing
	}
	tv.SizeFromParts() // get our size from parts
	tv.WidgetSize = tv.LayData.AllocSize
	h := tv.WidgetSize.Y
	w := tv.WidgetSize.X

	if !tv.IsClosed() {
		// we layout children under us
		for _, kid := range tv.Kids {
			_, gi := KiToNode2D(kid)
			if gi != nil {
				h += gi.LayData.AllocSize.Y
				w = math.Max(w, 10+gi.LayData.AllocSize.X) // 10 = indent, use max
			}
		}
	}
	tv.LayData.AllocSize = Vec2D{w, h}
	tv.WidgetSize.X = w // stretch
}

func (tv *TreeView) Layout2DParts(parBBox image.Rectangle) {
	spc := tv.Style.BoxSpace()
	tv.Parts.LayData.AllocPos = tv.LayData.AllocPos.AddVal(spc)
	tv.Parts.LayData.AllocSize = tv.WidgetSize.AddVal(-2.0 * spc)
	tv.Parts.Layout2D(parBBox)
}

func (tv *TreeView) Layout2D(parBBox image.Rectangle) {
	if tv.HasClosedParent() {
		tv.LayData.AllocPosRel.X = -1000000 // put it very far off screen..
	}
	tv.ConfigParts()

	psize := tv.AddParentPos() // have to add our pos first before computing below:

	rn := tv.RootWidget
	// our alloc size is root's size minus our total indentation
	tv.LayData.AllocSize.X = rn.LayData.AllocSize.X - (tv.LayData.AllocPos.X - rn.LayData.AllocPos.X)
	tv.WidgetSize.X = tv.LayData.AllocSize.X

	tv.LayData.AllocPosOrig = tv.LayData.AllocPos
	tv.Style.SetUnitContext(tv.Viewport, psize) // update units with final layout
	tv.Paint.SetUnitContext(tv.Viewport, psize) // always update paint
	tv.BBox = tv.This.(Node2D).BBox2D()         // only compute once, at this point
	tv.This.(Node2D).ComputeBBox2D(parBBox, image.ZP)

	if Layout2DTrace {
		fmt.Printf("Layout: %v reduced X allocsize: %v rn: %v  pos: %v rn pos: %v\n", tv.PathUnique(), tv.WidgetSize.X, rn.LayData.AllocSize.X, tv.LayData.AllocPos.X, rn.LayData.AllocPos.X)
		fmt.Printf("Layout: %v alloc pos: %v size: %v vpbb: %v winbb: %v\n", tv.PathUnique(), tv.LayData.AllocPos, tv.LayData.AllocSize, tv.VpBBox, tv.WinBBox)
	}

	tv.Layout2DParts(parBBox) // use OUR version
	for i := 0; i < int(TreeViewStatesN); i++ {
		tv.StateStyles[i].CopyUnitContext(&tv.Style.UnContext)
	}
	h := tv.WidgetSize.Y
	if !tv.IsClosed() {
		for _, kid := range tv.Kids {
			_, gi := KiToNode2D(kid)
			if gi != nil {
				gi.LayData.AllocPosRel.Y = h
				gi.LayData.AllocPosRel.X = 10 // indent
				h += gi.LayData.AllocSize.Y
			}
		}
	}
	tv.Layout2DChildren()
}

func (tv *TreeView) BBox2D() image.Rectangle {
	// we have unusual situation of bbox != alloc
	tp := tv.LayData.AllocPos.ToPointFloor()
	ts := tv.WidgetSize.ToPointCeil()
	return image.Rect(tp.X, tp.Y, tp.X+ts.X, tp.Y+ts.Y)
}

func (tv *TreeView) ChildrenBBox2D() image.Rectangle {
	ar := tv.BBoxFromAlloc() // need to use allocated size which includes children
	if tv.Par != nil {       // use parents children bbox to determine where we can draw
		pgi, _ := KiToNode2D(tv.Par)
		ar = ar.Intersect(pgi.ChildrenBBox2D())
	}
	return ar
}

func (tv *TreeView) Render2D() {
	if tv.HasClosedParent() {
		return // nothing
	}
	if tv.PushBounds() {
		tv.ConfigPartsIfNeeded()
		// reset for next update
		tv.ClearFullReRender()

		if tv.IsSelected() {
			tv.Style = tv.StateStyles[TreeViewSel]
		} else if tv.HasFocus() {
			tv.Style = tv.StateStyles[TreeViewFocus]
		} else {
			tv.Style = tv.StateStyles[TreeViewActive]
		}

		// note: this is std except using WidgetSize instead of AllocSize
		pc := &tv.Paint
		st := &tv.Style
		pc.FontStyle = st.Font
		pc.TextStyle = st.Text
		pc.StrokeStyle.SetColor(&st.Border.Color)
		pc.StrokeStyle.Width = st.Border.Width
		pc.FillStyle.SetColor(&st.Background.Color)
		// tv.RenderStdBox()
		pos := tv.LayData.AllocPos.AddVal(st.Layout.Margin.Dots)
		sz := tv.WidgetSize.AddVal(-2.0 * st.Layout.Margin.Dots)
		tv.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
		tv.Render2DParts()
		tv.PopBounds()
	}
	// we always have to render our kids b/c we could be out of scope but they could be in!
	tv.Render2DChildren()
}

func (tv *TreeView) ReRender2D() (node Node2D, layout bool) {
	if tv.NeedsFullReRender() {
		node = nil
		layout = false
	} else {
		node = tv.This.(Node2D)
		layout = false
	}
	return
}

func (tv *TreeView) FocusChanged2D(gotFocus bool) {
	tv.UpdateSig()
}

// check for interface implementation
var _ Node2D = &TreeView{}
