// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

//go:generate goki generate

import (
	"fmt"
	"image"
	"log"
	"reflect"
	"sync"

	"goki.dev/gicons"
	"goki.dev/girl/gist"
	"goki.dev/goosi"
	"goki.dev/goosi/mouse"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

// Widget is the interface for all GoGi Widget Nodes
type Widget interface {
	ki.Node

	// AsWidget returns the WidgetBase embedded field for any Widget node.
	// The Widget interface defines only methods that can be overridden
	// or need to be called on other nodes.  Everything else that is common
	// to all Widgets is in the WidgetBase.
	AsWidget() *WidgetBase

	// Config configures the widget, primarily configuring its Parts.
	// it does _not_ call Config on children, just self.
	// ConfigTree handles full tree configuration.
	// This config calls UpdateStart / End, SetStyle, and SetNeedsLayout,
	// and calls ConfigWidget to do the actual configuration,
	// so it does not need to manage this housekeeping.
	Config(vp *Viewport)

	// ConfigWidget does the actual configuration of the widget,
	// primarily configuring its Parts.
	// All configuration should be robust to multiple calls
	// (i.e., use Parts.ConfigChildren with TypeAndNameList).
	// Outer Config call handles all the other infrastructure,
	// so this call just does the core configuration.
	ConfigWidget(vp *Viewport)

	// SetStyle applies style functions to the widget based on current state.
	// It is typically not overridden -- set style funcs to apply custom styling.
	SetStyle(vp *Viewport)

	// GetSize: MeLast downward pass, each node first calls
	// g.Layout.Reset(), then sets their LayoutSize according to their own
	// intrinsic size parameters, and/or those of its children if it is a
	// Layout.
	GetSize(vp *Viewport, iter int)

	// DoLayout: MeFirst downward pass (each node calls on its children at
	// appropriate point) with relevant parent BBox that the children are
	// constrained to render within -- they then intersect this BBox with
	// their own BBox (from BBox2D) -- typically just call DoLayoutBase for
	// default behavior -- and add parent position to AllocPos, and then
	// return call to DoLayoutChildren. Layout does all its sizing and
	// positioning of children in this pass, based on the GetSize data gathered
	// bottom-up and constraints applied top-down from higher levels.
	// Typically only a single iteration is required (iter = 0) but multiple
	// are supported (needed for word-wrapped text or flow layouts) -- return
	// = true indicates another iteration required (pass this up the chain).
	DoLayout(vp *Viewport, parBBox image.Rectangle, iter int) bool

	// Move2D: optional MeFirst downward pass to move all elements by given
	// delta -- used for scrolling -- the layout pass assigns canonical
	// positions, saved in AllocPosOrig and BBox, and this adds the given
	// delta to that AllocPosOrig -- each node must call ComputeBBox2D to
	// update its bounding box information given the new position.
	Move2D(delta image.Point, parBBox image.Rectangle)

	// todo: fix bbox stuff!  BBoxes is a good overall name

	// BBox2D: compute the raw bounding box of this node relative to its
	// parent viewport -- called during DoLayout to set node BBox field, which
	// is then used in setting WinBBox and VpBBox.
	BBox2D() image.Rectangle

	// Compute VpBBox and WinBBox from BBox, given parent VpBBox -- most nodes
	// call ComputeBBox2DBase but viewports require special code -- called
	// during Layout and Move.
	ComputeBBox2D(parBBox image.Rectangle, delta image.Point)

	// ChildrenBBox2D: compute the bbox available to my children (content),
	// adjusting for margins, border, padding (BoxSpace) taken up by me --
	// operates on the existing VpBBox for this node -- this is what is passed
	// down as parBBox do the children's DoLayout.
	ChildrenBBox2D() image.Rectangle

	// Render: Final rendering pass, each node is fully responsible for
	// calling Render on its own children, to provide maximum flexibility
	// (see RenderChildren for default impl) -- bracket the render calls in
	// PushBounds / PopBounds and a false from PushBounds indicates that
	// VpBBox is empty and no rendering should occur.  Typically call
	// ConnectEvents to set up connections to receive window events if
	// visible, and disconnect if not.
	Render(vp *Viewport)

	// ConnectEvents: setup connections to window events -- called in
	// Render if in bounds.  It can be useful to create modular methods for
	// different event types that can then be mix-and-matched in any more
	// specialized types.
	ConnectEvents()

	// FocusChanged is called on node for changes in focus -- see the
	// FocusChanges values.
	FocusChanged(change FocusChanges)

	// HasFocus returns true if this node has keyboard focus and should
	// receive keyboard events -- typically this just returns HasFocus based
	// on the Window-managed HasFocus flag, but some types may want to monitor
	// all keyboard activity for certain key keys..
	HasFocus() bool

	// MakeContextMenu creates the context menu items (typically Action
	// elements, but it can be anything) for a given widget, typically
	// activated by the right mouse button or equivalent.  Widget has a
	// function parameter that can be set to add context items (e.g., by Views
	// or other complex widgets) to extend functionality.
	MakeContextMenu(menu *Menu)

	// ContextMenuPos returns the default position for popup menus --
	// by default in the middle of the WinBBox, but can be adapted as
	// appropriate for different widgets.
	ContextMenuPos() image.Point

	// ContextMenu displays the context menu of various actions to perform on
	// a node -- returns immediately, and actions are all executed directly
	// (later) via the action signals.  Calls MakeContextMenu and
	// ContextMenuPos.
	ContextMenu()

	// IsVisible provides the definitive answer as to whether a given node
	// is currently visible.  It is only entirely valid after a render pass
	// for widgets in a visible window, but it checks the window and viewport
	// for their visibility status as well, which is available always.
	// This does *not* check for VpBBox level visibility, which is a further check.
	// Non-visible nodes are automatically not rendered and not connected to
	// window events.  The Invisible flag is one key element of the IsVisible
	// calculus -- it is set by e.g., TabView for invisible tabs, and is also
	// set if a widget is entirely out of render range.  But again, use
	// IsVisible as the main end-user method.
	// For robustness, it recursively calls the parent -- this is typically
	// a short path -- propagating the Invisible flag properly can be
	// very challenging without mistakenly overwriting invisibility at various
	// levels.
	IsVisible() bool

	// IsDirectWinUpload returns true if this is a node that does a direct window upload
	// e.g., for gi3d.Scene which renders directly to the window texture for maximum efficiency
	IsDirectWinUpload() bool

	// DirectWinUpload does a direct upload of contents to a window
	// Drawer compositing image, which will then be used for drawing
	// the window during a Publish() event (triggered by the window Update
	// event).  This is called by the viewport in its Update signal processing
	// routine on nodes that respond true to IsDirectWinUpload().
	// The node is also free to update itself of its own accord at any point.
	DirectWinUpload()
}

// WidgetBase is the base type for all Widget Node2D elements, which are
// managed by a containing Layout, and use all 5 rendering passes.  All
// elemental widgets must support the Inactive and Selected states in a
// reasonable way (Selected only essential when also Inactive), so they can
// function appropriately in a chooser (e.g., SliceView or TableView) -- this
// includes toggling selection on left mouse press.
type WidgetBase struct {
	ki.Node

	// todo: remove CSS stuff from here??

	// user-defined class name(s) used primarily for attaching CSS styles to different display elements -- multiple class names can be used to combine properties: use spaces to separate per css standard
	Class string `desc:"user-defined class name(s) used primarily for attaching CSS styles to different display elements -- multiple class names can be used to combine properties: use spaces to separate per css standard"`

	// cascading style sheet at this level -- these styles apply here and to everything below, until superceded -- use .class and #name Props elements to apply entire styles to given elements, and type for element type
	CSS ki.Props `xml:"css" desc:"cascading style sheet at this level -- these styles apply here and to everything below, until superceded -- use .class and #name Props elements to apply entire styles to given elements, and type for element type"`

	// [view: no-inline] aggregated css properties from all higher nodes down to me
	CSSAgg ki.Props `copy:"-" json:"-" xml:"-" view:"no-inline" desc:"aggregated css properties from all higher nodes down to me"`

	// raw original 2D bounding box for the object within its parent viewport -- used for computing VpBBox and WinBBox -- this is not updated by Move2D, whereas VpBBox etc are
	BBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"raw original 2D bounding box for the object within its parent viewport -- used for computing VpBBox and WinBBox -- this is not updated by Move2D, whereas VpBBox etc are"`

	// full object bbox -- this is BBox + Move2D delta, but NOT intersected with parent's parBBox -- used for computing color gradients or other object-specific geometry computations
	ObjBBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"full object bbox -- this is BBox + Move2D delta, but NOT intersected with parent's parBBox -- used for computing color gradients or other object-specific geometry computations"`

	// 2D bounding box for region occupied within immediate parent Viewport object that we render onto -- these are the pixels we draw into, filtered through parent bounding boxes -- used for render Bounds clipping
	VpBBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"2D bounding box for region occupied within immediate parent Viewport object that we render onto -- these are the pixels we draw into, filtered through parent bounding boxes -- used for render Bounds clipping"`

	// 2D bounding box for region occupied within parent Window object, projected all the way up to that -- these are the coordinates where we receive events, relative to the window
	WinBBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"2D bounding box for region occupied within parent Window object, projected all the way up to that -- these are the coordinates where we receive events, relative to the window"`

	// text for tooltip for this widget -- can use HTML formatting
	Tooltip string `desc:"text for tooltip for this widget -- can use HTML formatting"`

	// a slice of stylers that are called in sequential descending order (so the first added styler is called last and thus overrides all other functions) to style the element; these should be set using AddStyler, which can be called by end-user and internal code
	Stylers []Styler `json:"-" xml:"-" copy:"-" desc:"a slice of stylers that are called in sequential descending order (so the first added styler is called last and thus overrides all other functions) to style the element; these should be set using AddStyler, which can be called by end-user and internal code"`

	// override the computed styles and allow directly editing Style
	OverrideStyle bool `json:"-" xml:"-" desc:"override the computed styles and allow directly editing Style"`

	// styling settings for this widget -- set in SetSetStyle during an initialization step, and when the structure changes; they are determined by, in increasing priority order, the default values, the ki node properties, and the StyleFunc (the recommended way to set styles is through the StyleFunc -- setting this field directly outside of that will have no effect unless OverrideStyle is on)
	Style gist.Style `json:"-" xml:"-" desc:"styling settings for this widget -- set in SetSetStyle during an initialization step, and when the structure changes; they are determined by, in increasing priority order, the default values, the ki node properties, and the StyleFunc (the recommended way to set styles is through the StyleFunc -- setting this field directly outside of that will have no effect unless OverrideStyle is on)"`

	// a separate tree of sub-widgets that implement discrete parts of a widget -- positions are always relative to the parent widget -- fully managed by the widget and not saved
	Parts *Layout `json:"-" xml:"-" view-closed:"true" desc:"a separate tree of sub-widgets that implement discrete parts of a widget -- positions are always relative to the parent widget -- fully managed by the widget and not saved"`

	// all the layout state information for this widget
	LayState LayoutState `copy:"-" json:"-" xml:"-" desc:"all the layout state information for this widget"`

	// [view: -] general widget signals supported by all widgets, including select, focus, and context menu (right mouse button) events, which can be used by views and other compound widgets
	WidgetSig ki.Signal `copy:"-" json:"-" xml:"-" view:"-" desc:"general widget signals supported by all widgets, including select, focus, and context menu (right mouse button) events, which can be used by views and other compound widgets"`

	// [view: -] optional context menu function called by MakeContextMenu AFTER any native items are added -- this function can decide where to insert new elements -- typically add a separator to disambiguate
	CtxtMenuFunc CtxtMenuFunc `copy:"-" view:"-" json:"-" xml:"-" desc:"optional context menu function called by MakeContextMenu AFTER any native items are added -- this function can decide where to insert new elements -- typically add a separator to disambiguate"`

	// parent viewport.  Only for use as a last resort when arg is not available -- otherwise always use the arg.  Set during Config.
	Vp *Viewport `copy:"-" json:"-" xml:"-" desc:"parent viewport.  Only for use as a last resort when arg is not available -- otherwise always use the arg.  Set during Config."`

	// [view: -] mutex protecting the Style field
	StyMu sync.RWMutex `copy:"-" view:"-" json:"-" xml:"-" desc:"mutex protecting the Style field"`

	// [view: -] mutex protecting the BBox fields
	BBoxMu sync.RWMutex `copy:"-" view:"-" json:"-" xml:"-" desc:"mutex protecting the BBox fields"`
}

// AsWidget returns the given Ki object
// as a Widget interface and a WidgetBase.
func AsWidget(k ki.Ki) (Widget, *WidgetBase) {
	if w, ok := k.(Widget); ok {
		return w, w.AsWidget()
	}
	return nil, nil
}

func (wb *WidgetBase) CopyFieldsFrom(frm any) {
	fr, ok := frm.(*WidgetBase)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined\n", wb.Type().Name())
		return
	}
	wb.Class = fr.Class
	wb.CSS.CopyFrom(fr.CSS, true)
	wb.Tooltip = fr.Tooltip
	wb.Style.CopyFrom(&fr.Style)
}

func (wb *WidgetBase) Disconnect() {
	wb.Node.Disconnect()
	wb.WidgetSig.DisconnectAll()
	if wb.Parts != nil {
		wb.Parts.DisconnectAll()
	}
}

func (wb *WidgetBase) AsWidget() *WidgetBase {
	return wb
}

func (wb *WidgetBase) BaseIface() reflect.Type {
	return laser.TypeFor[Widget]()
}

// ParentWidget returns the parent as a (Widget, *WidgetBase)
// or nil if this is the root and has no parent.
func (wb *WidgetBase) ParentWidget() (Widget, *WidgetBase) {
	if wb.Par == nil {
		return nil, nil
	}
	wi := wb.Par.(Widget)
	return wi, wi.AsWidget()
}

// ParentWidgetIf returns the nearest widget parent
// of the widget for which the given function returns true.
// It returns nil if no such parent is found;
// see [ParentWidgetIfTry] for a version with an error.
func (wb *WidgetBase) ParentWidgetIf(fun func(p *WidgetBase) bool) (Widget, *WidgetBase) {
	pwi, pwb, _ := wb.ParentWidgetIfTry(fun)
	return par
}

// ParentWidgetIfTry returns the nearest widget parent
// of the widget for which the given function returns true.
// It returns an error if no such parent is found; see
// [ParentWidgetIf] for a version without an error.
func (wb *WidgetBase) ParentWidgetIfTry(fun func(p *WidgetBase) bool) (Widget, *WidgetBase, error) {
	cur := wb
	for {
		par := cur.Par
		if par == nil {
			return nil, fmt.Errorf("(gi.WidgetBase).ParentWidgetIfTry: widget %v is the root", wb)
		}
		pwi := wb.Par.(Widget)
		pwb := pwi.AsWidget()
		if fun(pwb) {
			return pwi, pwb, nil
		}
		cur = pwb
	}
}

func (wb *WidgetBase) Config(vp *Viewport) {
	if wb.This() == nil {
		return
	}
	wi := wb.This().(Widget)
	updt := wi.UpdateStart()
	wb.Style.Defaults()    // reset
	wb.LayState.Defaults() // doesn't overwrite
	wi.ConfigWidget(vp)
	wi.SetStyle(vp)
	wi.UpdateEnd(updt)
	wb.SetNeedsLayout(vp, updt)
}

// ConnectEvents is the default event connection function
// for Widget objects. It calls [WidgetEvents], so any Widget
// implementing a custom ConnectEvents function should
// first call [WidgetEvents].
func (wb *WidgetBase) ConnectEvents() {
	wb.WidgetEvents()
}

// WidgetEvents connects the default events for Widget objects.
// Any Widget implementing a custom ConnectEvents function
// should first call this function.
func (wb *WidgetBase) WidgetEvents() {
	// TODO: figure out connect events situation not working
	// nb.WidgetMouseEvent()
	wb.WidgetMouseFocusEvent()
}

// WidgetMouseFocusEvent does the default handling for mouse click events for the Widget
func (wb *WidgetBase) WidgetMouseEvent() {
	wb.ConnectEvent(goosi.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, data any) {
		if wb.IsDisabled() {
			return
		}

		me := data.(*mouse.Event)
		me.SetProcessed()

		wb.WidgetOnMouseEvent(me)
	})
}

// WidgetOnMouseEvent is the function called on Widget objects
// when they get a mouse click event. If you are declaring a custom
// mouse event function, you should call this function first.
func (wb *WidgetBase) WidgetOnMouseEvent(me *mouse.Event) {
	wb.SetActiveState(me.Action == mouse.Press)
	wb.SetNeedsStyle()
	wb.UpdateSig()
}

// WidgetMouseFocusEvent does the default handling for mouse focus events for the Widget
func (wb *WidgetBase) WidgetMouseFocusEvent() {
	wb.ConnectEvent(goosi.MouseFocusEvent, RegPri, func(recv, send ki.Ki, sig int64, data any) {
		if wb.IsDisabled() {
			return
		}

		me := data.(*mouse.FocusEvent)
		me.SetProcessed()

		wb.WidgetOnMouseFocusEvent(me)
	})
}

// WidgetOnMouseFocusEvent is the function called on Widget objects
// when they get a mouse foucs event. If you are declaring a custom
// mouse foucs event function, you should call this function first.
func (wb *WidgetBase) WidgetOnMouseFocusEvent(me *mouse.FocusEvent) {
	enter := me.Action == mouse.Enter
	wb.SetHoveredState(enter)
	wb.SetNeedsStyle()
	wb.UpdateSig()
	// TODO: trigger mouse focus exit after clicking down
	// while leaving; then clear active here
	// // if !enter {
	// // 	nb.ClearActive()
	// }
}

// ConnectEvents is the default event connection function
// for widgets. It calls [WidgetEvents], so any widget
// implementing a custom ConnectEvents function should
// first call [WidgetEvents].
func (wb *WidgetBase) ConnectEvents() {
	wb.WidgetEvents()
}

// WidgetEvents connects the default events for widgets.
// Any widget implementing a custom ConnectEvents function
// should first call this function.
func (wb *WidgetBase) WidgetEvents() {
	wb.WidgetEvents()
	wb.HoverTooltipEvent()
}

// WidgetSignals are general signals that all widgets can send, via WidgetSig
// signal
type WidgetSignals int64

const (
	// WidgetSelected is triggered when a widget is selected, typically via
	// left mouse button click (see EmitSelectedSignal) -- is NOT contingent
	// on actual IsSelected status -- just reports the click event.
	// The data is the index of the selected item for multi-item widgets
	// (-1 = none / unselected)
	WidgetSelected WidgetSignals = iota

	// WidgetFocused is triggered when a widget receives keyboard focus (see
	// EmitFocusedSignal -- call in FocusChanged for gotFocus
	WidgetFocused

	// WidgetContextMenu is triggered when a widget receives a
	// right-mouse-button press, BEFORE generating and displaying the context
	// menu, so that relevant state can be updated etc (see
	// EmitContextMenuSignal)
	WidgetContextMenu

	WidgetSignalsN
)

// EmitSelectedSignal emits the WidgetSelected signal for this widget
func (wb *WidgetBase) EmitSelectedSignal() {
	wb.WidgetSig.Emit(wb.This(), int64(WidgetSelected), nil)
}

// EmitFocusedSignal emits the WidgetFocused signal for this widget
func (wb *WidgetBase) EmitFocusedSignal() {
	wb.WidgetSig.Emit(wb.This(), int64(WidgetFocused), nil)
}

// EmitContextMenuSignal emits the WidgetContextMenu signal for this widget
func (wb *WidgetBase) EmitContextMenuSignal() {
	wb.WidgetSig.Emit(wb.This(), int64(WidgetContextMenu), nil)
}

// WidgetMouseEvents connects to either or both mouse events -- IMPORTANT: if
// you need to also connect to other mouse events, you must copy this code --
// all processing of a mouse event must happen within one function b/c there
// can only be one registered per receiver and event type.  sel = Left button
// mouse.Press event, toggles the selected state, and emits a SelectedEvent.
// ctxtMenu = connects to Right button mouse.Press event, and sends a
// WidgetSig WidgetContextMenu signal, followed by calling ContextMenu method
// -- signal can be used to change state prior to generating context menu,
// including setting a CtxtMenuFunc that removes all items and thus negates
// the presentation of any menu
func (wb *WidgetBase) WidgetMouseEvents(sel, ctxtMenu bool) {
	if !sel && !ctxtMenu {
		return
	}
	wb.ConnectEvent(goosi.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		if sel {
			if me.Action == mouse.Press && me.Button == mouse.Left {
				me.SetProcessed()
				wbb := recv.Embed(TypeWidgetBase).(*WidgetBase)
				wbb.SetSelectedState(!wbb.IsSelected())
				wbb.EmitSelectedSignal()
				wbb.UpdateSig()
			}
		}
		if ctxtMenu {
			if me.Action == mouse.Release && me.Button == mouse.Right {
				me.SetProcessed()
				wbb := recv.Embed(TypeWidgetBase).(*WidgetBase)
				wbb.EmitContextMenuSignal()
				wbb.This().(Widget).ContextMenu()
			}
		}
	})
}

///////////////////////////////////////////////////////////////////
// ConfigParts building-blocks

// ConfigPartsIconLabel adds to config to create parts, of icon
// and label left-to right in a row, based on whether items are nil or empty
func (wb *WidgetBase) ConfigPartsIconLabel(config *ki.TypeAndNameList, icnm gicons.Icon, txt string) (icIdx, lbIdx int) {
	if wb.Style.Template != "" {
		wb.Parts.Style.Template = wb.Style.Template + ".Parts"
	}
	icIdx = -1
	lbIdx = -1
	if TheIconMgr.IsValid(icnm) {
		icIdx = len(*config)
		config.Add(TypeIcon, "icon")
		if txt != "" {
			config.Add(TypeSpace, "space")
		}
	}
	if txt != "" {
		lbIdx = len(*config)
		config.Add(TypeLabel, "label")
	}
	return
}

// ConfigPartsSetIconLabel sets the icon and text values in parts, and get
// part style props, using given props if not set in object props
func (wb *WidgetBase) ConfigPartsSetIconLabel(icnm gicons.Icon, txt string, icIdx, lbIdx int) {
	if icIdx >= 0 {
		ic := wb.Parts.Child(icIdx).(*Icon)
		if wb.Style.Template != "" {
			ic.Style.Template = wb.Style.Template + ".icon"
		}
		ic.SetIcon(icnm)
	}
	if lbIdx >= 0 {
		lbl := wb.Parts.Child(lbIdx).(*Label)
		if wb.Style.Template != "" {
			lbl.Style.Template = wb.Style.Template + ".icon"
		}
		if lbl.Text != txt {
			// avoiding SetText here makes it so label default
			// styles don't end up first, which is needed for
			// parent styles to override. However, there might have
			// been a reason for calling SetText, so we will see if
			// any bugs show up. TODO: figure out a good long-term solution for this.
			lbl.Text = txt
			// lbl.SetText(txt)
		}
	}
}

// PartsNeedUpdateIconLabel check if parts need to be updated -- for ConfigPartsIfNeeded
func (wb *WidgetBase) PartsNeedUpdateIconLabel(icnm gicons.Icon, txt string) bool {
	if TheIconMgr.IsValid(icnm) {
		ick := wb.Parts.ChildByName("icon", 0)
		if ick == nil {
			return true
		}
		ic := ick.(*Icon)
		if !ic.HasChildren() || ic.IconNm != icnm || wb.NeedsFullReRender() {
			return true
		}
	} else {
		cn := wb.Parts.ChildByName("icon", 0)
		if cn != nil { // need to remove it
			return true
		}
	}
	if txt != "" {
		lblk := wb.Parts.ChildByName("label", 2)
		if lblk == nil {
			return true
		}
		lbl := lblk.(*Label)
		lbl.Style.Color = wb.Style.Color
		if lbl.Text != txt {
			return true
		}
	} else {
		cn := wb.Parts.ChildByName("label", 2)
		if cn != nil {
			return true
		}
	}
	return false
}

// SetFullReRenderIconLabel sets the icon and label to be re-rendered, needed
// when styles change
func (wb *WidgetBase) SetFullReRenderIconLabel() {
	if ick := wb.Parts.ChildByName("icon", 0); ick != nil {
		ic := ick.(*Icon)
		ic.SetFullReRender()
	}
	if lblk := wb.Parts.ChildByName("label", 2); lblk != nil {
		lbl := lblk.(*Label)
		lbl.SetFullReRender()
	}
	wb.Parts.StyMu.Lock()
	wb.Parts.SetStyleWidget() // restyle parent so parts inherit
	wb.Parts.StyMu.Unlock()
}

func (wb *WidgetBase) IsVisible() bool {
	if wb == nil || wb.This() == nil || wb.IsInvisible() {
		return false
	}
	if wb.Par == nil || wb.Par.This() == nil {
		return false
	}
	return wb.Par.This().(Node2D).IsVisible()
}

func (wb *WidgetBase) IsDirectWinUpload() bool {
	return false
}

func (wb *WidgetBase) DirectWinUpload() {
}

// ConnectEvent connects this node to receive a given type of GUI event
// signal from the parent window -- typically connect only visible nodes, and
// disconnect when not visible
func (wb *WidgetBase) ConnectEvent(et goosi.EventType, pri EventPris, fun ki.RecvFunc) {
	em := wb.EventMgr()
	if em != nil {
		em.ConnectEvent(wb.This(), et, pri, fun)
	}
}

// DisconnectEvent disconnects this receiver from receiving given event
// type -- pri is priority -- pass AllPris for all priorities -- see also
// DisconnectAllEvents
func (wb *WidgetBase) DisconnectEvent(et goosi.EventType, pri EventPris) {
	em := wb.EventMgr()
	if em != nil {
		em.DisconnectEvent(wb.This(), et, pri)
	}
}

// DisconnectAllEvents disconnects node from all window events -- typically
// disconnect when not visible -- pri is priority -- pass AllPris for all priorities.
// This goes down the entire tree from this node on down, as typically everything under
// will not get an explicit disconnect call because no further updating will happen
func (wb *WidgetBase) DisconnectAllEvents(pri EventPris) {
	em := wb.EventMgr()
	if em == nil {
		return
	}
	wb.FuncDownMeFirst(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		_, ni := KiToNode2D(k)
		if ni == nil || ni.IsDeleted() || ni.IsDestroyed() {
			return ki.Break // going into a different type of thing, bail
		}
		em.DisconnectAllEvents(ni.This(), pri)
		return ki.Continue
	})
}

// ParentLayout returns the parent layout
func (wb *WidgetBase) ParentLayout() *Layout {
	ly := wb.ParentByType(TypeLayout, ki.Embeds)
	if ly == nil {
		return nil
	}
	return ly.Embed(TypeLayout).(*Layout)
}

// ParentScrollLayout returns the parent layout that has active scrollbars
func (wb *WidgetBase) ParentScrollLayout() *Layout {
	lyk := wb.ParentByType(TypeLayout, ki.Embeds)
	if lyk == nil {
		return nil
	}
	ly := lyk.Embed(TypeLayout).(*Layout)
	if ly.HasAnyScroll() {
		return ly
	}
	return ly.ParentScrollLayout()
}

// ScrollToMe tells my parent layout (that has scroll bars) to scroll to keep
// this widget in view -- returns true if scrolled
func (wb *WidgetBase) ScrollToMe() bool {
	ly := wb.ParentScrollLayout()
	if ly == nil {
		return false
	}
	return ly.ScrollToItem(wb.This().(Node2D))
}
