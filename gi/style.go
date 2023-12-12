// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/girl/abilities"
	"goki.dev/girl/paint"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/mat32/v2"
)

// Styling logic:
//
// see render.go for rendering logic
//
// Style funcs require pervasive access to (distant) parent styles
// while are themselves modifying their own styles.
// Styles are used in rendering and layout.
//
// Therefore, there is significant risk of read / write race errors.
//
// However, the render update logic should mitigate most of these:
//
// During normal event or anim-triggered updates, there is an
// UpdateStart at the start of any changes that

// CustomConfigStyles is the custom, global style configuration function
// that is called on all widgets to configure their style functions.
// By default, it is nil. If you set it, you should mostly call
// AddStyleFunc within it. For reference on
// how you should structure your CustomStyleFunc, you
// should look at https://goki.dev/docs/gi/styling.
var CustomConfigStyles func(w Widget)

////////////////////////////////////////////////////////////////////
// 	Widget Styling functions

// Style adds the given styler to the widget's stylers.
// It is the main way for both end-user and internal code
// to set the styles of a widget.
// It should only be done before showing the scene
// during initial configuration -- otherwise requries
// a StyMu mutex lock.
func (wb *WidgetBase) Style(s func(s *styles.Style)) *WidgetBase {
	wb.Stylers = append(wb.Stylers, s)
	return wb
}

// StyleRLock does a read-lock for reading the style
func (wb *WidgetBase) StyleRLock() {
	wb.StyMu.RLock()
}

// StyleRUnlock unlocks the read-lock
func (wb *WidgetBase) StyleRUnlock() {
	wb.StyMu.RUnlock()
}

// BoxSpace returns the style BoxSpace value under read lock
func (wb *WidgetBase) BoxSpace() styles.SideFloats {
	wb.StyMu.RLock()
	bs := wb.Styles.BoxSpace()
	wb.StyMu.RUnlock()
	return bs
}

// ApplyStyleParts styles the parts.
// Automatically called by the default ApplyStyleWidget function.
func (wb *WidgetBase) ApplyStyleParts() {
	if wb.Parts == nil {
		return
	}
	wb.Parts.ApplyStyleTree()
}

// ApplyStyleWidget is the primary styling function for all Widgets.
// Handles inheritance and runs the Styler functions.
// Must be called under a StyMu Lock
func (wb *WidgetBase) ApplyStyleWidget() {
	if wb.OverrideStyle {
		return
	}
	wb.ResetStyleWidget()

	_, pwb := wb.ParentWidget()
	if pwb != nil {
		wb.Styles.InheritFields(&pwb.Styles)
	}
	wb.DefaultStyleWidget()
	wb.RunStylers()
	wb.ApplyStylePrefs()
	wb.Styles.ComputeActualBackgroundColor(wb.ParentBackgroundColor())

	// note: this does not un-set the Invisible if not None, because all kinds of things
	// can turn invisible to off.
	if wb.Styles.Display == styles.DisplayNone {
		wb.SetState(true, states.Invisible)
	}
	SetUnitContext(&wb.Styles, wb.Sc, mat32.Vec2{}, mat32.Vec2{})
	wb.ApplyStyleParts()
}

// ResetStyleWidget resets the widget styles and applies the basic
// default styles specified in [styles.Style.Defaults]. It is called
// automatically in [ApplyStyleWidget]
// and should not need to be called by end-user code.
func (wb *WidgetBase) ResetStyleWidget() {
	s := &wb.Styles

	// need to persist state
	state := s.State
	*s = styles.Style{}
	s.Defaults()
	s.State = state

	// default to state layer associated with the state,
	// which the developer can override in their stylers
	// wb.Transition(&s.StateLayer, s.State.StateLayer(), 200*time.Millisecond, LinearTransition)
	s.StateLayer = s.State.StateLayer()

	s.Font.Family = string(Prefs.FontFamily)
}

// DefaultStyleWidget applies the base, widget-universal default
// styles to the widget. It is called automatically in [ApplyStyleWidget]
// and should not need to be called by end-user code.
func (wb *WidgetBase) DefaultStyleWidget() {
	s := &wb.Styles

	fsz := Prefs.FontSize / 100
	s.Font.Size.Val *= fsz
	s.Text.LineHeight.Val *= fsz

	s.MaxBorder.Style.Set(styles.BorderSolid)
	s.MaxBorder.Color.Set(colors.Scheme.Primary.Base)
	s.MaxBorder.Width.Set(units.Dp(1))

	// if we are disabled, we do not react to any state changes,
	// and instead always have the same gray colors
	if s.Is(states.Disabled) {
		s.Cursor = cursors.NotAllowed
		s.Opacity = 0.38
		return
	}
	// TODO(kai): what about context menus on mobile?
	s.SetAbilities(wb.Tooltip != "", abilities.LongHoverable, abilities.LongPressable)

	if s.Is(states.Focused) {
		s.Border = s.MaxBorder
	}
	if s.Is(states.Selected) {
		s.BackgroundColor.SetSolid(colors.Scheme.Select.Container)
		s.Color = colors.Scheme.Select.OnContainer
	}
}

// RunStylers runs the style functions specified in
// the StyleFuncs field in sequential ascending order.
func (wb *WidgetBase) RunStylers() {
	for _, s := range wb.Stylers {
		s(&wb.Styles)
	}
}

// ApplyStylePrefs applies [Prefs.Spacing] and [Prefs.FontSize]
// to the style values for the widget.
func (wb *WidgetBase) ApplyStylePrefs() {
	s := &wb.Styles

	spc := Prefs.Spacing / 100
	s.Margin.Top.Val *= spc
	s.Margin.Right.Val *= spc
	s.Margin.Bottom.Val *= spc
	s.Margin.Left.Val *= spc
	s.Padding.Top.Val *= spc
	s.Padding.Right.Val *= spc
	s.Padding.Bottom.Val *= spc
	s.Padding.Left.Val *= spc
	s.Gap.X.Val *= spc
	s.Gap.Y.Val *= spc
}

func (wb *WidgetBase) ApplyStyleUpdate() {
	updt := wb.UpdateStart()
	wb.ApplyStyleTree()
	wb.UpdateEndRender(updt)
}

func (wb *WidgetBase) ApplyStyle() {
	wb.StyMu.Lock() // todo: needed??  maybe not.
	defer wb.StyMu.Unlock()

	wb.ApplyStyleWidget()
}

// SetUnitContext sets the unit context based on size of scene, element, and parent
// element (from bbox) and then caches everything out in terms of raw pixel
// dots for rendering.
// Zero values for element and parent size are ignored.
func SetUnitContext(st *styles.Style, sc *Scene, el, par mat32.Vec2) {
	rebuild := sc.NeedsRebuild()
	rc := sc.RenderCtx()
	if rc != nil {
		st.UnContext.DPI = rc.LogicalDPI
	} else {
		st.UnContext.DPI = 96
	}
	sz := sc.SceneGeom.Size
	st.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, par.X, par.Y)
	if st.Font.Face == nil || rebuild {
		st.Font = paint.OpenFont(st.FontRender(), &st.UnContext) // calls SetUnContext after updating metrics
	}
	st.ToDots()
}

// ParentBackgroundColor returns the background color, state layer, and opacity
// of the nearest widget parent of the widget that has a defined background color,
// non-0 state later, or non-1 opacity, using a recursive approach to get further
// parent background colors for widgets with a non-0 state layer or non-1 opacity but
// not a defined background color. If no such parent is found, it returns a
// transparent background color, a 0 state layer, and a 1 opacity.
func (wb *WidgetBase) ParentBackgroundColor() *colors.Full {
	_, pw := wb.ParentWidget()
	if pw == nil {
		return &colors.Full{}
	}
	return &pw.Styles.ActualBackgroundColor
}

// IsNthChild returns whether the node is nth-child of its parent
func (wb *WidgetBase) IsNthChild(n int) bool {
	idx, ok := wb.IndexInParent()
	return ok && idx == n
}

// IsFirstChild returns whether the node is the first child of its parent
func (wb *WidgetBase) IsFirstChild() bool {
	idx, ok := wb.IndexInParent()
	return ok && idx == 0
}

// IsLastChild returns whether the node is the last child of its parent
func (wb *WidgetBase) IsLastChild() bool {
	idx, ok := wb.IndexInParent()
	return ok && idx == wb.Par.NumChildren()-1
}

// IsOnlyChild returns whether the node is the only child of its parent
func (wb *WidgetBase) IsOnlyChild() bool {
	return wb.Par != nil && wb.Par.NumChildren() == 1
}
