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
	"goki.dev/prof/v2"
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
func (wb *WidgetBase) ApplyStyleParts(sc *Scene) {
	if wb.Parts == nil {
		return
	}
	wb.Parts.ApplyStyleTree(sc)
}

// ApplyStyleWidget is the primary styling function for all Widgets.
// Handles inheritance and runs the Styler functions.
// Must be called under a StyMu Lock
func (wb *WidgetBase) ApplyStyleWidget(sc *Scene) {
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

	// note: this does not un-set the Invisible if not None, because all kinds of things
	// can turn invisible to off.
	if wb.Styles.Display == styles.DisplayNone {
		wb.SetState(true, states.Invisible)
	}
	SetUnitContext(&wb.Styles, sc, mat32.Vec2{}, mat32.Vec2{})
	sc.SetCurrentColor(wb.Styles.Color) // todo: do we need this anymore?
	wb.ApplyStyleParts(sc)
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
}

// DefaultStyleWidget applies the base, widget-universal default
// styles to the widget. It is called automatically in [ApplyStyleWidget]
// and should not need to be called by end-user code.
func (wb *WidgetBase) DefaultStyleWidget() {
	s := &wb.Styles

	s.Font.Family = string(Prefs.FontFamily)
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
		// this will get the state layer for the disabled state
		s.StateLayer = s.State.StateLayer()
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

func (wb *WidgetBase) ApplyStyleUpdate(sc *Scene) {
	wi := wb.This().(Widget)
	updt := wb.UpdateStart()
	wi.ApplyStyle(sc)
	wb.UpdateEnd(updt)
	wb.SetNeedsRenderUpdate(sc, updt)
}

func (wb *WidgetBase) ApplyStyle(sc *Scene) {
	wb.StyMu.Lock() // todo: needed??  maybe not.
	defer wb.StyMu.Unlock()

	wb.ApplyStyleWidget(sc)
}

// SetUnitContext sets the unit context based on size of scene, element, and parent
// element (from bbox) and then caches everything out in terms of raw pixel
// dots for rendering.
// Zero values for element and parent size are ignored.
func SetUnitContext(st *styles.Style, sc *Scene, el, par mat32.Vec2) {
	if sc != nil {
		rc := sc.RenderCtx()
		if rc != nil {
			st.UnContext.DPI = rc.LogicalDPI
			// fmt.Println("dpi:", rc.LogicalDPI)
		} else {
			st.UnContext.DPI = 96
		}
		sz := sc.SceneGeom.Size
		st.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, par.X, par.Y)
	}
	if st.Font.Face == nil || sc.NeedsRebuild() {
		pr := prof.Start("SetUnitContext-OpenFont")
		st.Font = paint.OpenFont(st.FontRender(), &st.UnContext) // calls SetUnContext after updating metrics
		pr.End()
	}
	ptd := prof.Start("SetUnitContext-ToDots")
	st.ToDots()
	// fmt.Println("uc:", st.UnContext.String())
	ptd.End()
}

// ParentBackgroundColor returns the background color and state layer
// of the nearest widget parent of the widget that has a defined
// background color or state layer, using a recursive approach to
// get further parent background colors for widgets with a state layer
// but not a background color. If no such parent is found,
// it returns a transparent background color and a 0 state layer.
func (wb *WidgetBase) ParentBackgroundColor() (colors.Full, float32) {
	// todo: this style reading requires a mutex!
	_, pwb := wb.ParentWidgetIf(func(p *WidgetBase) bool {
		// if we have a color or a state layer, we are a relevant breakpoint
		return !p.Styles.BackgroundColor.IsNil() || p.Styles.StateLayer > 0
	})
	if pwb == nil {
		return colors.Full{}, 0
	}
	// If we don't have a background color ourselves (but we have a state layer),
	// we recursively get our parent's background color and apply our state layer
	// to it. This makes state layers work on transparent elements.
	if pwb.Styles.BackgroundColor.IsNil() {
		pbc, _ := pwb.ParentBackgroundColor()
		return pbc, pwb.Styles.StateLayer
	}
	// Otherwise, we can directly apply the state layer to our background color
	return pwb.Styles.BackgroundColor, pwb.Styles.StateLayer
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
