// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

//go:generate goki generate

import (
	"fmt"
	"image"
	"log"
	"sync"

	"goki.dev/enums"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// Widget is the interface for all GoGi Widget Nodes
type Widget interface {
	ki.Ki

	// OnWidgetAdded adds a function to call when a widget is added
	// as a child to the widget or any of its children.
	OnWidgetAdded(f func(w Widget)) *WidgetBase

	// Style sets the styling of the widget by adding a Styler function
	Style(s func(s *styles.Style)) *WidgetBase

	// AsWidget returns the WidgetBase embedded field for any Widget node.
	// The Widget interface defines only methods that can be overridden
	// or need to be called on other nodes.  Everything else that is common
	// to all Widgets is in the WidgetBase.
	AsWidget() *WidgetBase

	// Config configures the widget, primarily configuring its Parts.
	// it does _not_ call Config on children, just self.
	// ApplyStyle must generally be called after Config - it is called
	// automatically when Scene is first shown, but must be called
	// manually thereafter as needed after configuration changes.
	// See ReConfig for a convenience function that does both.
	// ConfigScene on Scene handles full tree configuration.
	// This config calls UpdateStart / End, and SetNeedsLayout,
	// and calls ConfigWidget to do the actual configuration,
	// so it does not need to manage this housekeeping.
	// Thus, this Config call is typically never changed, and
	// all custom configuration should happen in ConfigWidget.
	Config(sc *Scene)

	// ConfigWidget does the actual configuration of the widget,
	// primarily configuring its Parts.
	// All configuration should be robust to multiple calls
	// (i.e., use Parts.ConfigChildren with Config).
	// Outer Config call handles all the other infrastructure,
	// so this call just does the core configuration.
	ConfigWidget(sc *Scene)

	// Update calls Config and ApplyStyle on this widget.
	// This should be called if any config options are changed
	// while the Scene is being viewed.
	Update()

	// StateIs returns true if given Style.State flag is set
	StateIs(flag enums.BitFlag) bool

	// AbilityIs returns true if given Style.Abilities flag is set
	AbilityIs(flag enums.BitFlag) bool

	// SetState sets the given [styles.Style.State] flags
	SetState(on bool, state ...enums.BitFlag) *WidgetBase

	// SetAbilities sets the given [styles.Style.Abilities] flags
	SetAbilities(on bool, able ...enums.BitFlag) *WidgetBase

	// ApplyStyle applies style functions to the widget based on current state.
	// It is typically not overridden -- set style funcs to apply custom styling.
	ApplyStyle(sc *Scene)

	// SizeUp (bottom-up): gathers sizes from our Children & Parts,
	// based only on Min style sizes and actual content sizing.
	// Flexible elements (e.g., Text, Flex Wrap, TopAppBar) allocate
	// optimistically along their main axis, up to any optional Max size.
	SizeUp(sc *Scene)

	//	SizeDown (top-down, multiple iterations possible): assigns sizes based
	// on allocated parent avail size, giving extra space based on Grow factors,
	// and flexible elements wrap / config to fit top-down constraint along main
	// axis, producing a (new) top-down size expanding in cross axis as needed
	// (or removing items that don't fit, etc).  Wrap & Grid layouts assign
	// X,Y index coordinates to items during this pass.
	SizeDown(sc *Scene, iter int, allocTotal mat32.Vec2) bool

	// Position: uses the final sizes to position everything within layouts
	// according to alignment settings.
	Position(sc *Scene)

	// ScenePos: scene-based position and final BBox is computed based on
	// parents accumulated position and scrollbar position.
	// This step can be performed when scrolling after updating Scroll.
	ScenePos(sc *Scene)

	// Render: Actual rendering pass, each node is fully responsible for
	// calling Render on its own children, to provide maximum flexibility
	// (see RenderChildren for default impl) -- bracket the render calls in
	// PushBounds / PopBounds and a false from PushBounds indicates that
	// ScBBox is empty and no rendering should occur.
	Render(sc *Scene)

	// On adds an event listener function for the given event type
	On(etype events.Types, fun func(e events.Event)) *WidgetBase

	// Helper functions for common event types
	// TODO(kai/menu): should we have these in the Widget interface?
	// we need them for OnWidgetAdded functions

	// OnClick adds an event listener function for [events.Click] events
	OnClick(fun func(e events.Event)) *WidgetBase

	// HandleEvent sends the given event to all Listeners for that event type.
	// It also checks if the State has changed and calls ApplyStyle if so.
	// If more significant Config level changes are needed due to an event,
	// the event handler must do this itself.
	HandleEvent(e events.Event)

	// Send sends an NEW event of given type to this widget,
	// optionally starting from values in the given original event
	// (recommended to include where possible).
	// Do NOT send an existing event using this method if you
	// want the Handled state to persist throughout the call chain;
	// call HandleEvent directly for any existing events.
	Send(e events.Types, orig ...events.Event)

	// ContextMenu adds the context menu items (typically [Button]s)
	// for the widget to the given menu scene. No context menu is defined
	// by default, but widget types can implement this function if they
	// have a context menu. ContextMenu also calls
	// [WidgetBase.CustomContextMenu] if it is not nil.
	ContextMenu(m *Scene)

	// ContextMenuPos returns the default position for popup menus --
	// by default in the middle its Bounding Box, but can be adapted as
	// appropriate for different widgets.
	ContextMenuPos(e events.Event) image.Point

	// ShowContextMenu displays the context menu of various actions
	// to perform on a Widget, activated by default on the ShowContextMenu
	// event, triggered by a Right mouse click ()
	// -- returns immediately, and actions are all executed directly
	// (later) via the action signals.  Calls MakeContextMenu and
	// ContextMenuPos.
	ShowContextMenu(e events.Event)

	// IsVisible provides the definitive answer as to whether a given node
	// is currently visible.  It is only entirely valid after a render pass
	// for widgets in a visible window, but it checks the window and scene
	// for their visibility status as well, which is available always.
	// This does *not* check for ScBBox level visibility, which is a further check.
	// Non-visible nodes are automatically not rendered and do not get
	// window events.  The Invisible states flag is a key element of the IsVisible
	// calculus -- it is set by e.g., Tabs for invisible tabs, and is also
	// set if a widget is entirely out of render range.
	// For robustness, it recursively calls the parent -- this is typically
	// a short path -- propagating the Invisible flag properly can be
	// very challenging without mistakenly overwriting invisibility at various
	// levels.
	IsVisible() bool

	// todo: revisit this -- in general anything with a largish image (including svg,
	// SubScene, but not Icon) should get put on a list so the RenderWin Drawer just
	// directly uploads its image.

	// IsDirectWinUpload returns true if this is a node that does a direct window upload
	// e.g., for gi3d.Scene which renders directly to the window texture for maximum efficiency
	IsDirectWinUpload() bool

	// DirectWinUpload does a direct upload of contents to a window
	// Drawer compositing image, which will then be used for drawing
	// the window during a Publish() event (triggered by the window Update
	// event).  This is called by the scene in its Update signal processing
	// routine on nodes that respond true to IsDirectWinUpload().
	// The node is also free to update itself of its own accord at any point.
	DirectWinUpload()
}

// WidgetBase is the base type for all Widget Widget elements, which are
// managed by a containing Layout, and use all 5 rendering passes.  All
// elemental widgets must support the ReadOnly and Selected states in a
// reasonable way (Selected only essential when also ReadOnly), so they can
// function appropriately in a chooser (e.g., SliceView or TableView) -- this
// includes toggling selection on left mouse press.
type WidgetBase struct {
	ki.Node

	// text for the tooltip for this widget, which can use HTML formatting
	Tooltip string

	// todo: remove CSS stuff from here??

	// user-defined class name(s) used primarily for attaching CSS styles to different display elements -- multiple class names can be used to combine properties: use spaces to separate per css standard
	Class string

	// cascading style sheet at this level -- these styles apply here and to everything below, until superceded -- use .class and #name Props elements to apply entire styles to given elements, and type for element type
	CSS ki.Props `set:"-"`

	// aggregated css properties from all higher nodes down to me
	CSSAgg ki.Props `copy:"-" json:"-" xml:"-" view:"no-inline" set:"-"`

	// Alloc is layout allocation state: contains full size and position info
	Alloc LayoutState `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// A slice of functions to call on all widgets that are added as children to this widget or its children.
	// These functions are called in sequential ascending order, so the last added one is called
	// last and thus can override anything set by the other ones. These should be set using
	// OnWidgetAdded, which can be called by both end-user and internal code.
	OnWidgetAdders []func(w Widget) `copy:"-" json:"-" xml:"-" set:"-"`

	// a slice of stylers that are called in sequential ascending order (so the last added styler is called last and thus overrides all other functions) to style the element; these should be set using Style, which can be called by end-user and internal code
	Stylers []func(s *styles.Style) `copy:"-" json:"-" xml:"-" set:"-"`

	// override the computed styles and allow directly editing Style
	OverrideStyle bool `copy:"-" json:"-" xml:"-" set:"-"`

	// styling settings for this widget -- set in SetApplyStyle during an initialization step, and when the structure changes; they are determined by, in increasing priority order, the default values, the ki node properties, and the StyleFunc (the recommended way to set styles is through the StyleFunc -- setting this field directly outside of that will have no effect unless OverrideStyle is on)
	Styles styles.Style `copy:"-" json:"-" xml:"-" set:"-"`

	// Listeners are event listener functions for processing events on this widget.
	// type specific Listeners are added in OnInit when the widget is initialized.
	Listeners events.Listeners `copy:"-" json:"-" xml:"-" set:"-"`

	// a separate tree of sub-widgets that implement discrete parts of a widget -- positions are always relative to the parent widget -- fully managed by the widget and not saved
	Parts *Layout `copy:"-" json:"-" xml:"-" view-closed:"true" set:"-"`

	// all the layout state information for this widget
	LayState LayoutState `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// an optional context menu constructor function called by [Widget.MakeContextMenu] after any type-specified items are added.
	// This function can decide where to insert new elements, and it should typically add a separator to disambiguate.
	CustomContextMenu func(m *Scene) `copy:"-" json:"-" xml:"-"`

	// parent scene.  Only for use as a last resort when arg is not available -- otherwise always use the arg.  Set during Config.
	Sc *Scene `copy:"-" json:"-" xml:"-" set:"-"`

	// mutex protecting the Style field
	StyMu sync.RWMutex `copy:"-" view:"-" json:"-" xml:"-" set:"-"`

	// mutex protecting the BBox fields
	BBoxMu sync.RWMutex `copy:"-" view:"-" json:"-" xml:"-" set:"-"`
}

func (wb *WidgetBase) FlagType() enums.BitFlag {
	return WidgetFlags(wb.Flags)
}

func (wb *WidgetBase) OnChildAdded(child ki.Ki) {
	w, _ := AsWidget(child)
	if w == nil {
		return
	}
	for _, f := range wb.OnWidgetAdders {
		f(w)
	}
}

// OnWidgetAdded adds a function to call when a widget is added
// as a child to the widget or any of its children.
func (wb *WidgetBase) OnWidgetAdded(fun func(w Widget)) *WidgetBase {
	wb.OnWidgetAdders = append(wb.OnWidgetAdders, fun)
	return wb
}

// AsWidget returns the given Ki object
// as a Widget interface and a WidgetBase.
func AsWidget(k ki.Ki) (Widget, *WidgetBase) {
	if k == nil || k.This() == nil {
		return nil, nil
	}
	if w, ok := k.This().(Widget); ok {
		return w, w.AsWidget()
	}
	return nil, nil
}

func (wb *WidgetBase) AsWidget() *WidgetBase {
	return wb
}

// AsWidgetBase returns the given Ki object as a WidgetBase, or nil.
// for direct use of the return value in cases where that is needed.
func AsWidgetBase(k ki.Ki) *WidgetBase {
	_, wb := AsWidget(k)
	return wb
}

func (wb *WidgetBase) CopyFieldsFrom(frm any) {
	fr, ok := frm.(*WidgetBase)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined\n", wb.KiType().Name)
		return
	}
	wb.Class = fr.Class
	wb.CSS.CopyFrom(fr.CSS, true)
	wb.Tooltip = fr.Tooltip
	wb.Styles.CopyFrom(&fr.Styles)
	wb.Stylers = fr.Stylers
	wb.Listeners = fr.Listeners // direct copy -- functions..
	wb.CustomContextMenu = fr.CustomContextMenu
}

func (wb *WidgetBase) Destroy() {
	if wb.Parts != nil {
		wb.Parts.DeleteChildren(true) // first delete all my children
	}
	wb.Parts = nil
	wb.Node.Destroy()
}

func (wb *WidgetBase) BaseType() *gti.Type {
	return WidgetBaseType
}

// NewParts makes the Parts layout if not already there,
// with given layout orientation
func (wb *WidgetBase) NewParts() *Layout {
	if wb.Parts != nil {
		return wb.Parts
	}
	parts := &Layout{}
	parts.InitName(parts, "parts")
	ki.SetParent(parts, wb.This())
	parts.SetFlag(true, ki.Field)
	parts.SetFlag(false, ki.Updating) // we inherit this from parent, but parent doesn't auto-clear us
	wb.Parts = parts
	return parts
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
	return pwi, pwb
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
			return nil, nil, fmt.Errorf("(gi.WidgetBase).ParentWidgetIfTry: got to root: %v without finding", cur)
		}
		pwi, ok := par.(Widget)
		if !ok {
			return nil, nil, fmt.Errorf("(gi.WidgetBase).ParentWidgetIfTry: parent is not a widget: %v", par)
		}
		pwb := pwi.AsWidget()
		if fun(pwb) {
			return pwi, pwb, nil
		}
		cur = pwb
	}
}

func (wb *WidgetBase) IsVisible() bool {
	if wb == nil || wb.This() == nil || wb.Is(ki.Deleted) || wb.StateIs(states.Invisible) || wb.Sc == nil {
		return false
	}
	if wb.Par == nil || wb.Par.This() == nil {
		return true
	}
	return wb.Par.This().(Widget).IsVisible()
}

func (wb *WidgetBase) IsDirectWinUpload() bool {
	return false
}

func (wb *WidgetBase) DirectWinUpload() {
}

// WidgetKidsIter iterates through the Kids, as widgets, calling the given function.
// return false to terminate.
func (wb *WidgetBase) WidgetKidsIter(fun func(i int, kwi Widget, kwb *WidgetBase) bool) {
	for i, k := range wb.Kids {
		i := i
		kwi, kwb := AsWidget(k)
		if kwi == nil || kwi.This() == nil || kwi.Is(ki.Deleted) {
			break
		}
		cont := fun(i, kwi, kwb)
		if !cont {
			break
		}
	}
}

// WidgetWalkPre is a version of the ki WalkPre iterator that automatically filters
// nil or deleted items and operates on Widget types.
func (wb *WidgetBase) WidgetWalkPre(fun func(kwi Widget, kwb *WidgetBase) bool) {
	wb.WalkPre(func(k ki.Ki) bool {
		kwi, kwb := AsWidget(k)
		if kwi == nil || kwi.This() == nil || kwi.Is(ki.Deleted) {
			return ki.Break
		}
		return fun(kwi, kwb)
	})
}

// WidgetWalkPost is a version of the ki WalkPost iterator that automatically filters
// nil or deleted items and operates on Widget types.
func (wb *WidgetBase) WidgetWalkPost(fun func(kwi Widget, kwb *WidgetBase) bool) {
	wb.WalkPost(func(k ki.Ki) bool {
		kwi, kwb := AsWidget(k)
		if kwi == nil || kwi.This() == nil || kwi.Is(ki.Deleted) {
			return ki.Break
		}
		return fun(kwi, kwb)
	})
}
