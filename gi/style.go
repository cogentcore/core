// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
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
// should look at https://cogentcore.org/core/docs/gi/styling.
var CustomConfigStyles func(w Widget)

////////////////////////////////////////////////////////////////////
// 	Widget Styling functions

// Style adds the given styler to the widget's Stylers.
// It is one of the main ways for both end-user and internal code
// to set the styles of a widget, in addition to StyleFirst
// and StyleFinal, which add stylers that are called before
// and after the stylers added by this function, respectively.
func (wb *WidgetBase) Style(s func(s *styles.Style)) *WidgetBase {
	wb.Stylers = append(wb.Stylers, s)
	return wb
}

// StyleFirst adds the given styler to the widget's FirstStylers.
// It is one of the main ways for both end-user and internal code
// to set the styles of a widget, in addition to Style
// and StyleFinal, which add stylers that are called after
// the stylers added by this function.
func (wb *WidgetBase) StyleFirst(s func(s *styles.Style)) *WidgetBase {
	wb.FirstStylers = append(wb.FirstStylers, s)
	return wb
}

// StyleFinal adds the given styler to the widget's FinalStylers.
// It is one of the main ways for both end-user and internal code
// to set the styles of a widget, in addition to StyleFirst
// and Style, which add stylers that are called before
// the stylers added by this function.
func (wb *WidgetBase) StyleFinal(s func(s *styles.Style)) *WidgetBase {
	wb.FinalStylers = append(wb.FinalStylers, s)
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

	pwb := wb.ParentWidget()
	if pwb != nil {
		wb.Styles.InheritFields(&pwb.Styles)
	}
	wb.RunStylers()
	wb.ApplyStylePrefs()

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

	s.Font.Family = string(AppearanceSettings.FontFamily)
}

// SetStyles sets the base, widget-universal default
// style function that applies to all widgets.
// It is added and called first in the styling order.
// Because it handles default styling in response to
// State flags such as Disabled and Selected, these state
// flags must be set prior to calling this.
// Use [StyleFirst] to add a function that is called prior to
// this, to update state flags.
func (wb *WidgetBase) SetStyles() {
	wb.Style(func(s *styles.Style) {
		fsz := AppearanceSettings.FontSize / 100
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
		s.SetAbilities(wb.This().(Widget).WidgetTooltip() != "", abilities.LongHoverable, abilities.LongPressable)

		if s.Is(states.Focused) {
			s.Border = s.MaxBorder
		}
		if s.Is(states.Selected) {
			s.Background = colors.C(colors.Scheme.Select.Container)
			s.Color = colors.Scheme.Select.OnContainer
		}
	})
}

// RunStylers runs the stylers specified in the widget's FirstStylers,
// Stylers, and FinalStylers in that order in a sequential ascending order.
func (wb *WidgetBase) RunStylers() {
	for _, s := range wb.FirstStylers {
		s(&wb.Styles)
	}
	for _, s := range wb.Stylers {
		s(&wb.Styles)
	}
	for _, s := range wb.FinalStylers {
		s(&wb.Styles)
	}
}

// ApplyStylePrefs applies [Prefs.Spacing] and [Prefs.FontSize]
// to the style values for the widget.
func (wb *WidgetBase) ApplyStylePrefs() {
	s := &wb.Styles

	spc := AppearanceSettings.Spacing / 100
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

// ApplyStyleUpdate calls ApplyStyleTree within an UpdateRender block.
// This is the main call needed to ensure that state-sensitive styling
// is updated, when state changes.
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
	rebuild := false
	var rc *RenderContext
	sz := image.Point{1920, 1280}
	if sc != nil {
		rebuild = sc.NeedsRebuild()
		rc = sc.RenderCtx()
		sz = sc.SceneGeom.Size
	}
	if rc != nil {
		st.UnContext.DPI = rc.LogicalDPI
	} else {
		st.UnContext.DPI = 160
	}
	st.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, par.X, par.Y)
	if st.Font.Face == nil || rebuild {
		st.Font = paint.OpenFont(st.FontRender(), &st.UnContext) // calls SetUnContext after updating metrics
	}
	st.ToDots()
}

// ParentActualBackground returns the actual background of
// the parent of the widget. If it has no parent, it returns nil.
func (wb *WidgetBase) ParentActualBackground() image.Image {
	pwb := wb.ParentWidget()
	if pwb == nil {
		return nil
	}
	return pwb.Styles.ActualBackground
}

// IsNthChild returns whether the node is nth-child of its parent
func (wb *WidgetBase) IsNthChild(n int) bool {
	idx := wb.IndexInParent()
	return idx == n
}

// IsFirstChild returns whether the node is the first child of its parent
func (wb *WidgetBase) IsFirstChild() bool {
	idx := wb.IndexInParent()
	return idx == 0
}

// IsLastChild returns whether the node is the last child of its parent
func (wb *WidgetBase) IsLastChild() bool {
	idx := wb.IndexInParent()
	return idx == wb.Par.NumChildren()-1
}

// IsOnlyChild returns whether the node is the only child of its parent
func (wb *WidgetBase) IsOnlyChild() bool {
	return wb.Par != nil && wb.Par.NumChildren() == 1
}
