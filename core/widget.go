// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package core provides the core GUI functionality of Cogent Core.
package core

//go:generate core generate

import (
	"image"
	"log/slog"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// Widget is the interface that all Cogent Core satisfy.
// The core widget functionality is defined on [WidgetBase],
// and all higher-level widget types must embed it. This
// interface only contains the methods that higher-level
// widget types may need to override. You can call
// [Widget.AsWidget] to get the [WidgetBase] of a Widget
// and access the core widget functionality.
type Widget interface {
	tree.Node

	// AsWidget returns the [WidgetBase] of this Widget. Most
	// core widget functionality is implemented on [WidgetBase].
	AsWidget() *WidgetBase

	// See [WidgetBase.Style].
	Style() // TODO(config): remove

	// SizeUp (bottom-up) gathers Actual sizes from our Children & Parts,
	// based on Styles.Min / Max sizes and actual content sizing
	// (e.g., text size).  Flexible elements (e.g., Text, Flex Wrap,
	// TopAppBar) should reserve the _minimum_ size possible at this stage,
	// and then Grow based on SizeDown allocation.
	SizeUp()

	// SizeDown (top-down, multiple iterations possible) provides top-down
	// size allocations based initially on Scene available size and
	// the SizeUp Actual sizes.  If there is extra space available, it is
	// allocated according to the Grow factors.
	// Flexible elements (e.g., Flex Wrap layouts and Text with word wrap)
	// update their Actual size based on available Alloc size (re-wrap),
	// to fit the allocated shape vs. the initial bottom-up guess.
	// However, do NOT grow the Actual size to match Alloc at this stage,
	// as Actual sizes must always represent the minimums (see Position).
	// Returns true if any change in Actual size occurred.
	SizeDown(iter int) bool

	// SizeFinal: (bottom-up) similar to SizeUp but done at the end of the
	// Sizing phase: first grows widget Actual sizes based on their Grow
	// factors, up to their Alloc sizes.  Then gathers this updated final
	// actual Size information for layouts to register their actual sizes
	// prior to positioning, which requires accurate Actual vs. Alloc
	// sizes to perform correct alignment calculations.
	SizeFinal()

	// Position uses the final sizes to set relative positions within layouts
	// according to alignment settings, and Grow elements to their actual
	// Alloc size per Styles settings and widget-specific behavior.
	Position()

	// ScenePos computes scene-based absolute positions and final BBox
	// bounding boxes for rendering, based on relative positions from
	// Position step and parents accumulated position and scroll offset.
	// This is the only step needed when scrolling (very fast).
	ScenePos()

	// Render is the method that widgets should implement to define their
	// custom rendering steps. It should not typically be called outside of
	// [Widget.RenderWidget], which also does other steps applicable
	// for all widgets. The base [WidgetBase.Render] implementation
	// renders the standard box model.
	Render()

	// RenderWidget renders the widget and any parts and children that it has.
	// It does not render if the widget is invisible. It calls [Widget.Render]
	// for widget-specific rendering.
	RenderWidget()

	// WidgetTooltip returns the tooltip text that should be used for this
	// widget, and the window-relative position to use for the upper-left corner
	// of the tooltip. The current mouse position in scene-local coordinates
	// is passed to the function; if it is {-1, -1}, that indicates that
	// WidgetTooltip is being called in a Style function to determine whether
	// the widget should be [abilities.LongHoverable] and [abilities.LongPressable]
	// (if the return string is not "", then it will have those abilities
	// so that the tooltip can be displayed).
	//
	// By default, WidgetTooltip just returns [WidgetBase.Tooltip]
	// and [WidgetBase.DefaultTooltipPos], but widgets can override
	// it to do different things. For example, buttons add their
	// shortcut to the tooltip here.
	WidgetTooltip(pos image.Point) (string, image.Point)

	// ContextMenuPos returns the default position for popup menus;
	// by default in the middle its Bounding Box, but can be adapted as
	// appropriate for different widgets.
	ContextMenuPos(e events.Event) image.Point

	// ShowContextMenu displays the context menu of various actions
	// to perform on a Widget, activated by default on the ShowContextMenu
	// event, triggered by a Right mouse click.
	// Returns immediately, and actions are all executed directly
	// (later) via the action signals. Calls ContextMenu and
	// ContextMenuPos.
	ShowContextMenu(e events.Event)

	// IsVisible returns true if a node is visible for rendering according
	// to the [states.Invisible] flag on it or any of its parents.
	// This flag is also set by [styles.DisplayNone] during [ApplyStyle].
	// This does *not* check for an empty TotalBBox, indicating that the widget
	// is out of render range -- that is done by [PushBounds] prior to rendering.
	// Non-visible nodes are automatically not rendered and do not get
	// window events.
	// This call recursively calls the parent, which is typically a short path.
	IsVisible() bool

	// ChildBackground returns the background color (Image) for given child Widget.
	// By default, this is just our [Styles.Actualbackground] but it can be computed
	// specifically for the child (e.g., for zebra stripes in [ListGrid])
	ChildBackground(child Widget) image.Image

	// DirectRenderImage uploads image directly into given system.Drawer at given index
	// Typically this is a drw.SetGoImage call with an [image.RGBA], or
	// drw.SetFrameImage with a [vgpu.FrameBuffer]
	DirectRenderImage(drw system.Drawer, idx int)

	// DirectRenderDraw draws the current image at index onto the RenderWindow window,
	// typically using drw.Copy, drw.Scale, or drw.Fill.
	// flipY is the default setting for whether the Y axis needs to be flipped during drawing,
	// which is typically passed along to the Copy or Scale methods.
	DirectRenderDraw(drw system.Drawer, idx int, flipY bool)
}

// WidgetBase implements the [Widget] interface and provides the core functionality
// of a widget. You must use WidgetBase as an embedded struct in all higher-level
// widget types. It renders the standard box model, but does not layout or render
// any children; see [Frame] for that.
type WidgetBase struct {
	tree.NodeBase

	// Tooltip is the text for the tooltip for this widget,
	// which can use HTML formatting.
	Tooltip string `json:",omitempty"`

	// Parts are a separate tree of sub-widgets that can be used to store
	// orthogonal parts of a widget when necessary to separate them from children.
	// For example, trees use parts to separate their internal parts from
	// the other child tree nodes. Composite widgets like buttons should
	// NOT use parts to store their components; parts should only be used when
	// absolutely necessary.
	Parts *Frame `copier:"-" json:"-" xml:"-" set:"-"`

	// Geom has the full layout geometry for size and position of this Widget
	Geom GeomState `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// Updaters are a slice of functions called in sequential descending (reverse) order
	// in [WidgetBase.UpdateWidget] to update the widget. You can use
	// [WidgetBase.Updater] to add one. By default, this slice contains a function
	// that updates the widget's children using [WidgetBase.Make].
	Updaters []func() `copier:"-" json:"-" xml:"-" set:"-" edit:"-"`

	// Makers are a slice of functions called in sequential ascending order
	// in [WidgetBase.Make] to make the plan for how the widget's children should
	// be configured. You can use [WidgetBase.Maker] to add one.
	Makers []func(p *Plan) `copier:"-" json:"-" xml:"-" set:"-" edit:"-"`

	// If true, override the computed styles and allow directly editing Styles.
	OverrideStyle bool `copier:"-" json:"-" xml:"-" set:"-"`

	// Styles are styling settings for this widget.
	// These are set in SetApplyStyle which should be called after any Config
	// change (e.g., as done by the Update method).  See Stylers for functions
	// that set all of the styles, ordered from initial base defaults to later
	// added overrides.
	Styles styles.Style `json:"-" xml:"-" set:"-"`

	// Stylers are a slice of functions that are called in sequential
	// ascending order (so the last added styler is called last and
	// thus overrides all other functions) to style the element.
	// These should be set using Styler function. FirstStylers and
	// FinalStylers are called before and after these stylers, respectively.
	Stylers []func(s *styles.Style) `copier:"-" json:"-" xml:"-" set:"-" edit:"-"`

	// FirstStylers are a slice of functions that are called in sequential
	// ascending order (so the last added styler is called last and
	// thus overrides all other functions) to style the element.
	// These should be set using FirstStyler function. These stylers
	// are called before Stylers and FinalStylers.
	FirstStylers []func(s *styles.Style) `copier:"-" json:"-" xml:"-" set:"-"`

	// FinalStylers are a slice of functions that are called in sequential
	// ascending order (so the last added styler is called last and
	// thus overrides all other functions) to style the element.
	// These should be set using FinalStyler function. These stylers
	// are called after FirstStylers and Stylers.
	FinalStylers []func(s *styles.Style) `copier:"-" json:"-" xml:"-" set:"-"`

	// Listeners are event listener functions for processing events on this widget.
	// They are called in sequential descending order (so the last added listener
	// is called first). They should be added using the On function. FirstListeners
	// and FinalListeners are called before and after these listeners, respectively.
	Listeners events.Listeners `copier:"-" json:"-" xml:"-" set:"-"`

	// FirstListeners are event listener functions for processing events on this widget.
	// They are called in sequential descending order (so the last added listener
	// is called first). They should be added using the OnFirst function. These listeners
	// are called before Listeners and FinalListeners.
	FirstListeners events.Listeners `copier:"-" json:"-" xml:"-" set:"-"`

	// FinalListeners are event listener functions for processing events on this widget.
	// They are called in sequential descending order (so the last added listener
	// is called first). They should be added using the OnFinal function. These listeners
	// are called after FirstListeners and Listeners.
	FinalListeners events.Listeners `copier:"-" json:"-" xml:"-" set:"-"`

	// A slice of functions to call on all widgets that are added as children
	// to this widget or its children. These functions are called in sequential
	// ascending order, so the last added one is called last and thus can
	// override anything set by the other ones. These should be set using
	// OnWidgetAdded, which can be called by both end-user and internal code.
	OnWidgetAdders []func(w Widget) `copier:"-" json:"-" xml:"-" set:"-"`

	// ContextMenus is a slice of menu functions to call to construct
	// the widget's context menu on an [events.ContextMenu]. The
	// functions are called in reverse order such that the elements
	// added in the last function are the first in the menu.
	// Context menus should be added through [WidgetBase.AddContextMenu].
	// Separators will be added between each context menu function.
	ContextMenus []func(m *Scene) `copier:"-" json:"-" xml:"-" set:"-"`

	// Scene is the overall Scene to which we belong. It is automatically
	// by widgets whenever they are added to another widget parent.
	Scene *Scene `copier:"-" json:"-" xml:"-" set:"-"`

	// ValueUpdate is a function set by [Bind] that is called in
	// [WidgetBase.UpdateWidget] to update the widget's value from the bound value.
	ValueUpdate func() `copier:"-" json:"-" xml:"-" set:"-"`

	// ValueOnChange is a function set by [Bind] that is called when
	// the widget receives an [events.Change] event to update the bound value
	// from the widget's value.
	ValueOnChange func() `copier:"-" json:"-" xml:"-" set:"-"`

	// ValueTitle is the title to display for a dialog for this [Value].
	ValueTitle string

	// ValueNewWindow indicates that the dialog of a [Value] should be opened
	// as a new window, instead of a typical full window in the same current window.
	// This is set by [InitValueButton] and handled by [OpenValueDialog].
	// This is triggered by holding down the Shift key while clicking on a
	// [Value] button. Certain values such as [FileButton] may set this to true
	// in their [InitValueButton] function.
	ValueNewWindow bool `copier:"-" json:"-" xml:"-"`

	// needsRender is whether the widget needs to be rendered on the next render iteration.
	needsRender bool
}

// Init should be called by every Widget type in its custom
// Init if it has one to establish all the default styling
// and event handling that applies to all widgets.
func (wb *WidgetBase) Init() {
	wb.Styler(func(s *styles.Style) {
		s.MaxBorder.Style.Set(styles.BorderSolid)
		s.MaxBorder.Color.Set(colors.C(colors.Scheme.Primary.Base))
		s.MaxBorder.Width.Set(units.Dp(1))

		// if we are disabled, we do not react to any state changes,
		// and instead always have the same gray colors
		if s.Is(states.Disabled) {
			s.Cursor = cursors.NotAllowed
			s.Opacity = 0.38
			return
		}
		// TODO(kai): what about context menus on mobile?
		tt, _ := wb.This.(Widget).WidgetTooltip(image.Pt(-1, -1))
		s.SetAbilities(tt != "", abilities.LongHoverable, abilities.LongPressable)

		if s.Is(states.Selected) {
			s.Background = colors.C(colors.Scheme.Select.Container)
			s.Color = colors.C(colors.Scheme.Select.OnContainer)
		}
	})
	wb.FinalStyler(func(s *styles.Style) {
		if s.Is(states.Focused) {
			s.Border.Style = s.MaxBorder.Style
			s.Border.Color = s.MaxBorder.Color
			s.Border.Width = s.MaxBorder.Width
		}
		if !s.AbilityIs(abilities.Focusable) {
			// never need bigger border if not focusable
			s.MaxBorder = s.Border
		}
	})

	// TODO(kai): maybe move all of these event handling functions into one function
	wb.HandleWidgetClick()
	wb.HandleWidgetStateFromMouse()
	wb.HandleLongHoverTooltip()
	wb.HandleWidgetStateFromFocus()
	wb.HandleWidgetContextMenu()
	wb.HandleWidgetMagnify()
	wb.HandleValueOnChange()

	wb.Updater(wb.updateFromMake)
}

// OnAdd is called when widgets are added to a parent.
// It sets the scene of the widget to its widget parent.
// It should be called by all other OnAdd functions defined
// by widget types.
func (wb *WidgetBase) OnAdd() {
	if pwb := wb.ParentWidget(); pwb != nil {
		wb.Scene = pwb.Scene
	}
}

// SetScene sets the Scene pointer for this widget and all of its children.
// This can be necessary when creating widgets outside the usual "NewWidget" paradigm,
// e.g., when reading from a JSON file.
func (wb *WidgetBase) SetScene(sc *Scene) {
	wb.WidgetWalkDown(func(kwi Widget, kwb *WidgetBase) bool {
		kwb.Scene = sc
		return tree.Continue
	})
}

func (wb *WidgetBase) OnChildAdded(child tree.Node) {
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

// AsWidget returns the given [tree.Node]
// as a [Widget] interface and a [WidgetBase].
func AsWidget(n tree.Node) (Widget, *WidgetBase) {
	if w, ok := n.(Widget); ok {
		return w, w.AsWidget()
	}
	return nil, nil
}

func (wb *WidgetBase) AsWidget() *WidgetBase {
	return wb
}

func (wb *WidgetBase) CopyFieldsFrom(from tree.Node) {
	wb.NodeBase.CopyFieldsFrom(from)
	_, frm := AsWidget(from)

	n := len(wb.Stylers)
	if len(frm.Stylers) > n {
		wb.Stylers = append(wb.Stylers, frm.Stylers[n:]...)
	}
	n = len(wb.ContextMenus)
	if len(frm.ContextMenus) > n {
		wb.ContextMenus = append(wb.ContextMenus, frm.ContextMenus[n:]...)
	}
	wb.Listeners.CopyFromExtra(frm.Listeners)
	wb.FirstListeners.CopyFromExtra(frm.FirstListeners)
	wb.FinalListeners.CopyFromExtra(frm.FinalListeners)
}

func (wb *WidgetBase) Destroy() {
	wb.DeleteParts()
	wb.NodeBase.Destroy()
}

// DeleteParts deletes the widget's parts (and the children of the parts).
func (wb *WidgetBase) DeleteParts() {
	if wb.Parts != nil {
		wb.Parts.Destroy()
	}
	wb.Parts = nil
}

func (wb *WidgetBase) BaseType() *types.Type {
	return WidgetBaseType
}

// NewParts makes the Parts layout if not already there.
func (wb *WidgetBase) NewParts() *Frame {
	if wb.Parts != nil {
		return wb.Parts
	}
	wb.Parts = NewFrame()
	wb.Parts.SetName("parts")
	tree.SetParent(wb.Parts, wb) // don't add to children list
	wb.Parts.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.RenderBox = false
	})
	return wb.Parts
}

// ParentWidget returns the parent as a [WidgetBase] or nil
// if this is the root and has no parent.
func (wb *WidgetBase) ParentWidget() *WidgetBase {
	if wb.Parent == nil {
		return nil
	}
	return wb.Parent.(Widget).AsWidget()
}

// ParentWidgetIf returns the nearest widget parent
// of the widget for which the given function returns true.
// It returns nil if no such parent is found.
func (wb *WidgetBase) ParentWidgetIf(fun func(p *WidgetBase) bool) *WidgetBase {
	cur := wb
	for {
		parent := cur.Parent
		if parent == nil {
			return nil
		}
		pwi, ok := parent.(Widget)
		if !ok {
			return nil
		}
		pwb := pwi.AsWidget()
		if fun(pwb) {
			return pwb
		}
		cur = pwb
	}
}

// IsVisible returns true if a node is visible for rendering according
// to the [states.Invisible] flag on it or any of its parents.
// This flag is also set by [styles.DisplayNone] during [ApplyStyle].
// This does *not* check for an empty TotalBBox, indicating that the widget
// is out of render range -- that is done by [PushBounds] prior to rendering.
// Non-visible nodes are automatically not rendered and do not get
// window events.
// This call recursively calls the parent, which is typically a short path.
func (wb *WidgetBase) IsVisible() bool {
	if wb == nil || wb.This == nil || wb.StateIs(states.Invisible) || wb.Scene == nil {
		return false
	}
	if wb.Parent == nil {
		return true
	}
	return wb.Parent.(Widget).IsVisible()
}

// DirectRenderImage uploads image directly into given system.Drawer at given index
// Typically this is a drw.SetGoImage call with an [image.RGBA], or
// drw.SetFrameImage with a [vgpu.FrameBuffer]
func (wb *WidgetBase) DirectRenderImage(drw system.Drawer, idx int) {}

// DirectRenderDraw draws the current image at index onto the RenderWindow window,
// typically using drw.Copy, drw.Scale, or drw.Fill.
// flipY is the default setting for whether the Y axis needs to be flipped during drawing,
// which is typically passed along to the Copy or Scale methods.
func (wb *WidgetBase) DirectRenderDraw(drw system.Drawer, idx int, flipY bool) {}

// NodeWalkDown extends [tree.Node.WalkDown] to [WidgetBase.Parts],
// which is key for getting full tree traversal to work when updating,
// configuring, and styling. This implements [tree.Node.NodeWalkDown].
func (wb *WidgetBase) NodeWalkDown(fun func(tree.Node) bool) {
	if wb.Parts == nil {
		return
	}
	wb.Parts.WalkDown(fun)
}

// WidgetKidsIter iterates through the Kids, as widgets, calling the given function.
// Return [tree.Continue] (true) to continue, and [tree.Break] (false) to terminate.
func (wb *WidgetBase) WidgetKidsIter(fun func(i int, w Widget, cwb *WidgetBase) bool) {
	for i, k := range wb.Children {
		w, cwb := AsWidget(k)
		cont := fun(i, w, cwb)
		if !cont {
			break
		}
	}
}

// VisibleKidsIter iterates through the Kids, as widgets, calling the given function,
// excluding any with the *local* states.Invisible flag set (does not check parents).
// This is used e.g., for layout functions to exclude non-visible direct children.
// Return [tree.Continue] (true) to continue, and [tree.Break] (false) to terminate.
func (wb *WidgetBase) VisibleKidsIter(fun func(i int, w Widget, cwb *WidgetBase) bool) {
	for i, k := range wb.Children {
		w, cwb := AsWidget(k)
		if cwb.StateIs(states.Invisible) {
			continue
		}
		cont := fun(i, w, cwb)
		if !cont {
			break
		}
	}
}

// WidgetWalkDown is a version of [tree.Node.WalkDown] that automatically filters
// nil or deleted items and operates on [Widget] types.
// Return [tree.Continue] to continue and [tree.Break] to terminate.
func (wb *WidgetBase) WidgetWalkDown(fun func(kwi Widget, kwb *WidgetBase) bool) {
	wb.WalkDown(func(k tree.Node) bool {
		kwi, kwb := AsWidget(k)
		return fun(kwi, kwb)
	})
}

// WidgetNext returns the next widget in the tree,
// including Parts, which are considered to come after Children.
// returns nil if no more.
func WidgetNext(w Widget) Widget {
	wb := w.AsWidget()
	if !wb.HasChildren() && wb.Parts == nil {
		return WidgetNextSibling(w)
	}
	if wb.HasChildren() {
		return wb.Child(0).(Widget)
	}
	if wb.Parts != nil {
		return WidgetNext(wb.Parts.This.(Widget))
	}
	return nil
}

// WidgetNextSibling returns next sibling or nil if none,
// including Parts, which are considered to come after Children.
func WidgetNextSibling(w Widget) Widget {
	wb := w.AsWidget()
	if wb.Parent == nil {
		return nil
	}
	parent := wb.Parent.(Widget)
	myidx := wb.IndexInParent()
	if myidx >= 0 && myidx < wb.Parent.AsTree().NumChildren()-1 {
		return parent.AsTree().Child(myidx + 1).(Widget)
	}
	return WidgetNextSibling(parent)
}

// WidgetPrev returns the previous widget in the tree,
// including Parts, which are considered to come after Children.
// nil if no more.
func WidgetPrev(w Widget) Widget {
	wb := w.AsWidget()
	if wb.Parent == nil {
		return nil
	}
	parent := wb.Parent.(Widget)
	myidx := wb.IndexInParent()
	if myidx > 0 {
		nn := parent.AsTree().Child(myidx - 1).(Widget)
		return WidgetLastChildParts(nn) // go to parts
	}
	// we were children, done
	return parent
}

// WidgetLastChildParts returns the last child under given node,
// or node itself if no children.  Starts with Parts,
func WidgetLastChildParts(w Widget) Widget {
	wb := w.AsWidget()
	if wb.Parts != nil && wb.Parts.HasChildren() {
		return WidgetLastChildParts(wb.Parts.Child(wb.Parts.NumChildren() - 1).(Widget))
	}
	if wb.HasChildren() {
		return WidgetLastChildParts(wb.Child(wb.NumChildren() - 1).(Widget))
	}
	return w
}

// WidgetLastChild returns the last child under given node,
// or node itself if no children. Starts with Children, not Parts
func WidgetLastChild(w Widget) Widget {
	wb := w.AsWidget()
	if wb.HasChildren() {
		return WidgetLastChildParts(wb.Child(wb.NumChildren() - 1).(Widget))
	}
	return w
}

// WidgetNextFunc returns the next widget in the tree,
// including Parts, which are considered to come after children,
// continuing until the given function returns true.
// nil if no more.
func WidgetNextFunc(w Widget, fun func(w Widget) bool) Widget {
	for {
		nw := WidgetNext(w)
		if nw == nil {
			return nil
		}
		if fun(nw) {
			return nw
		}
		if nw == w {
			slog.Error("WidgetNextFunc", "start", w, "nw == wi", nw)
			return nil
		}
		w = nw
	}
}

// WidgetPrevFunc returns the previous widget in the tree,
// including Parts, which are considered to come after children,
// continuing until the given function returns true.
// nil if no more.
func WidgetPrevFunc(w Widget, fun func(w Widget) bool) Widget {
	for {
		pw := WidgetPrev(w)
		if pw == nil {
			return nil
		}
		if fun(pw) {
			return pw
		}
		if pw == w {
			slog.Error("WidgetPrevFunc", "start", w, "pw == wi", pw)
			return nil
		}
		w = pw
	}
}

// WidgetTooltip is the base implementation of [Widget.WidgetTooltip],
// which just returns [WidgetBase.Tooltip] and [WidgetBase.DefaultTooltipPos].
func (wb *WidgetBase) WidgetTooltip(pos image.Point) (string, image.Point) {
	return wb.Tooltip, wb.DefaultTooltipPos()
}
