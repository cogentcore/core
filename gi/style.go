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
func (wb *WidgetBase) Style(s func(s *styles.Style)) Widget {
	wb.Stylers = append(wb.Stylers, s)
	return wb.This().(Widget)
}

// STYTODO: figure out what to do with this
// // AddChildStyler is a helper function that adds the
// // given styler to the child of the given name
// // if it exists, starting searching at the given start index.
// func (wb *WidgetBase) AddChildStyler(childName string, startIdx int, s Styler) {
// 	child := wb.ChildByName(childName, startIdx)
// 	if child != nil {
// 		wb, ok := child.Embed(TypeWidgetBase).(*WidgetBase)
// 		if ok {
// 			wb.Style(func(s *styles.Style) {
// 				f(wb)
// 			})
// 		}
// 	}
// }

// TODO: get rid of this!?

// ActiveStyle satisfies the ActiveStyler interface
// and returns the active style of the widget
func (wb *WidgetBase) ActiveStyle() *styles.Style {
	return &wb.Styles
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

// ParentActiveStyle returns parent's active style or nil if not avail.
// Calls StyleRLock so must call ParentStyleRUnlock when done.
func (wb *WidgetBase) ParentActiveStyle() *styles.Style {
	if wb.Par == nil {
		return nil
	}
	if ps, ok := wb.Par.(styles.ActiveStyler); ok {
		st := ps.ActiveStyle()
		return st
	}
	return nil
}

// ParentStyleRUnlock unlocks the parent's style
func (wb *WidgetBase) ParentStyleRUnlock() {
	if wb.Par == nil {
		return
	}
	if ps, ok := wb.Par.(styles.ActiveStyler); ok {
		ps.StyleRUnlock()
	}
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
	pr := prof.Start("ApplyStyleWidget")
	defer pr.End()

	if wb.Sc == nil && sc != nil {
		wb.Sc = sc
	}
	if wb.Sc == nil {
		return
		// slog.Error("Scene is nil", "widget", wb)
	}

	if wb.OverrideStyle {
		return
	}

	wb.ResetStyleWidget()

	// todo: remove all these prof steps -- should be much less now..
	pin := prof.Start("ApplyStyleWidget-Inherit")

	if parSty := wb.ParentActiveStyle(); parSty != nil {
		wb.Styles.InheritFields(parSty)
		// wb.ParentStyleRUnlock()
	}
	pin.End()

	wb.DefaultStyleWidget()

	prun := prof.Start("ApplyStyleWidget-RunStylers")
	wb.RunStylers()
	prun.End()

	// we automatically apply Prefs.DensityMul after we run all of the stylers
	wb.ApplyPrefsDensityMul()

	// note: it is critical to do this styling here so that layout getsizes
	// has the proper info for laying out items
	puc := prof.Start("ApplyStyleWidget-SetUnitContext")
	SetUnitContext(&wb.Styles, wb.Sc, mat32.Vec2{}, mat32.Vec2{})
	puc.End()

	psc := prof.Start("ApplyStyleWidget-SetCurrentColor")
	sc.SetCurrentColor(wb.Styles.Color)

	wb.ApplyStyleParts(sc)

	psc.End()
}

// InitStyleWidget resets the widget styles and applies the basic
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
	s.SetAbilities(wb.Tooltip != "", abilities.LongHoverable)

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

// ApplyPrefsDensityMul multiplies all of the margin and padding
// values for the widget by the result of [Prefs.DensityMul]
func (wb *WidgetBase) ApplyPrefsDensityMul() {
	wb.Styles.Margin.Top.Val *= Prefs.DensityMul()
	wb.Styles.Margin.Right.Val *= Prefs.DensityMul()
	wb.Styles.Margin.Bottom.Val *= Prefs.DensityMul()
	wb.Styles.Margin.Left.Val *= Prefs.DensityMul()

	wb.Styles.Padding.Top.Val *= Prefs.DensityMul()
	wb.Styles.Padding.Right.Val *= Prefs.DensityMul()
	wb.Styles.Padding.Bottom.Val *= Prefs.DensityMul()
	wb.Styles.Padding.Left.Val *= Prefs.DensityMul()
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
// dots for rendering -- call at start of render. Zero values for element and parent size are ignored.
func SetUnitContext(st *styles.Style, sc *Scene, el, par mat32.Vec2) {
	if sc != nil {
		rc := sc.RenderCtx()
		if rc != nil {
			st.UnContext.DPI = rc.LogicalDPI
			// } else {
			// 	slog.Error("SetUnitContext RenderCtx is nil", "scene", sc.Name())
		}
		if sc.RenderState.Image != nil {
			sz := sc.Geom.Size // Render.Image.Bounds().Size()
			st.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, par.X, par.Y)
		}
		// } else {
		// 	slog.Error("SetUnitContext Scene nil!")
	}
	pr := prof.Start("SetUnitContext-OpenFont")
	st.Font = paint.OpenFont(st.FontRender(), &st.UnContext) // calls SetUnContext after updating metrics
	pr.End()
	ptd := prof.Start("SetUnitContext-ToDots")
	st.ToDots()
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

/////////////////////////////////////////////////////////////////
// Style helper methods

// SetMinPrefWidth sets minimum and preferred width;
// will get at least this amount; max unspecified.
// This adds a styler that calls [styles.Style.SetMinPrefWidth].
func (wb *WidgetBase) SetMinPrefWidth(val units.Value) Widget {
	wb.Style(func(s *styles.Style) {
		s.SetMinPrefWidth(val)
	})
	return wb.This().(Widget)
}

// SetMinPrefHeight sets minimum and preferred height;
// will get at least this amount; max unspecified.
// This adds a styler that calls [styles.Style.SetMinPrefHeight].
func (wb *WidgetBase) SetMinPrefHeight(val units.Value) Widget {
	wb.Style(func(s *styles.Style) {
		s.SetMinPrefHeight(val)
	})
	return wb.This().(Widget)
}

// SetStretchMaxWidth sets stretchy max width (-1);
// can grow to take up avail room.
// This adds a styler that calls [styles.Style.SetStretchMaxWidth].
func (wb *WidgetBase) SetStretchMaxWidth() Widget {
	wb.Style(func(s *styles.Style) {
		s.SetStretchMaxWidth()
	})
	return wb.This().(Widget)
}

// SetStretchMaxHeight sets stretchy max height (-1);
// can grow to take up avail room.
// This adds a styler that calls [styles.Style.SetStretchMaxHeight].
func (wb *WidgetBase) SetStretchMaxHeight() Widget {
	wb.Style(func(s *styles.Style) {
		s.SetStretchMaxHeight()
	})
	return wb.This().(Widget)
}

// SetStretchMax sets stretchy max width and height (-1);
// can grow to take up avail room.
// This adds a styler that calls [styles.Style.SetStretchMax].
func (wb *WidgetBase) SetStretchMax() Widget {
	wb.Style(func(s *styles.Style) {
		s.SetStretchMaxWidth()
		s.SetStretchMaxHeight()
	})
	return wb.This().(Widget)
}

// SetFixedWidth sets all width style options
// (Width, MinWidth, and MaxWidth) to
// the given fixed width unit value.
// This adds a styler that calls [styles.Style.SetFixedWidth].
func (wb *WidgetBase) SetFixedWidth(val units.Value) Widget {
	wb.Style(func(s *styles.Style) {
		s.SetFixedWidth(val)
	})
	return wb.This().(Widget)
}

// SetFixedHeight sets all height style options
// (Height, MinHeight, and MaxHeight) to
// the given fixed height unit value.
// This adds a styler that calls [styles.Style.SetFixedHeight].
func (wb *WidgetBase) SetFixedHeight(val units.Value) Widget {
	wb.Style(func(s *styles.Style) {
		s.SetFixedHeight(val)
	})
	return wb.This().(Widget)
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
