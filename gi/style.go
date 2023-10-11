// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log/slog"

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

// Styler is a fuction that can be used to style an element.
// They are the building blocks of the GoGi styling system.
// They can be used as a closure and capture surrounding context,
// but they are passed the style for convenience and so that they
// can be used for multiple elements if desired; you can get most
// of the information you need from the function. A Styler should be
// added to a widget through the [WidgetBase.AddStyles] method.
// We use stylers for styling because they give you complete
// control and full programming logic without any CSS-selector magic.
type Styler func(s *styles.Style)

func (sc *Scene) SetDefaultStyle() {
	sc.AddStyles(func(s *styles.Style) {
		s.Cursor = cursors.Arrow
		s.BackgroundColor.SetSolid(colors.Scheme.Background)
		s.Color = colors.Scheme.OnBackground
	})
}

////////////////////////////////////////////////////////////////////
// 	Widget Styling functions

// AddStyles adds the given styler to the widget's stylers.
// It is the main way for both end-user and internal code
// to set the styles of a widget.
// It should only be done before showing the scene
// during initial configuration -- otherwise requries
// a StyMu mutex lock.
func (wb *WidgetBase) AddStyles(s Styler) Widget {
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
// 			wb.AddStyles(func(s *styles.Style) {
// 				f(wb)
// 			})
// 		}
// 	}
// }

// TODO: get rid of this!?

// ActiveStyle satisfies the ActiveStyler interface
// and returns the active style of the widget
func (wb *WidgetBase) ActiveStyle() *styles.Style {
	return &wb.Style
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
	bs := wb.Style.BoxSpace()
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
		slog.Error("Scene is nil", "widget", wb)
	}

	if wb.OverrideStyle {
		return
	}

	wb.DefaultStyleWidget()

	// todo: remove all these prof steps -- should be much less now..
	pin := prof.Start("ApplyStyleWidget-Inherit")

	if parSty := wb.ParentActiveStyle(); parSty != nil {
		wb.Style.InheritFields(parSty)
		// wb.ParentStyleRUnlock()
	}
	pin.End()

	prun := prof.Start("ApplyStyleWidget-RunStylers")
	wb.RunStylers()
	prun.End()

	// we automatically apply Prefs.DensityMul after we run all of the stylers
	wb.ApplyPrefsDensityMul()

	// note: it is critical to do this styling here so that layout getsizes
	// has the proper info for laying out items
	puc := prof.Start("ApplyStyleWidget-SetUnitContext")
	SetUnitContext(&wb.Style, wb.Sc, mat32.Vec2{}, mat32.Vec2{})
	puc.End()

	psc := prof.Start("ApplyStyleWidget-SetCurrentColor")
	if wb.Style.Inactive { // inactive can only set, not clear
		wb.SetState(true, states.Disabled)
	}
	sc.SetCurrentColor(wb.Style.Color)

	wb.ApplyStyleParts(sc)

	psc.End()
}

// DefaultStyleWidget applies the base, widget-universal default
// styles to the widget. It is called automatically in [ApplyStyleWidget]
// and should not need to be called by end-user code.
func (wb *WidgetBase) DefaultStyleWidget() {
	s := &wb.Style

	state := s.State
	*s = styles.Style{}
	s.Defaults()
	s.State = state

	s.MaxBorder.Style.Set(styles.BorderSolid)
	s.MaxBorder.Color.Set(colors.Scheme.Primary.Base)
	s.MaxBorder.Width.Set(units.Dp(1))

	// if we are disabled, we do not react to any state changes,
	// and instead always have the same gray colors
	if s.Is(states.Disabled) {
		s.Cursor = cursors.NotAllowed
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceVariant)
		s.Color = colors.Scheme.OnSurfaceVariant
		s.Opacity = 0.38
	} else {
		s.SetAbilities(wb.Tooltip != "", abilities.LongHoverable)
		// default to state layer associated with the state,
		// which the developer can override in their stylers
		s.StateLayer = s.State.StateLayer()

		if s.Is(states.Focused) {
			s.Border = s.MaxBorder
		}
		if s.Is(states.Selected) {
			s.BackgroundColor.SetSolid(colors.Scheme.Select.Container)
			s.Color = colors.Scheme.Select.OnContainer
		}
	}
}

// RunStylers runs the style functions specified in
// the StyleFuncs field in sequential ascending order.
func (wb *WidgetBase) RunStylers() {
	for _, s := range wb.Stylers {
		s(&wb.Style)
	}
}

// ApplyPrefsDensityMul multiplies all of the margin and padding
// values for the widget by the result of [Prefs.DensityMul]
func (wb *WidgetBase) ApplyPrefsDensityMul() {
	wb.Style.Margin.Top.Val *= Prefs.DensityMul()
	wb.Style.Margin.Right.Val *= Prefs.DensityMul()
	wb.Style.Margin.Bottom.Val *= Prefs.DensityMul()
	wb.Style.Margin.Left.Val *= Prefs.DensityMul()

	wb.Style.Padding.Top.Val *= Prefs.DensityMul()
	wb.Style.Padding.Right.Val *= Prefs.DensityMul()
	wb.Style.Padding.Bottom.Val *= Prefs.DensityMul()
	wb.Style.Padding.Left.Val *= Prefs.DensityMul()
}

func (wb *WidgetBase) ApplyStyleUpdate(sc *Scene) {
	wi := wb.This().(Widget)
	updt := wb.UpdateStart()
	wi.ApplyStyle(sc)
	wb.UpdateEnd(updt)
	wb.SetNeedsRender(sc, updt)
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
		} else {
			slog.Error("SetUnitContext RenderCtx is nil", "scene", sc.Name())
		}
		if sc.RenderState.Image != nil {
			sz := sc.Geom.Size // Render.Image.Bounds().Size()
			st.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, par.X, par.Y)
		}
	} else {
		slog.Error("SetUnitContext Scene nil!")
	}
	pr := prof.Start("SetUnitContext-OpenFont")
	st.Font = paint.OpenFont(st.FontRender(), &st.UnContext) // calls SetUnContext after updating metrics
	pr.End()
	ptd := prof.Start("SetUnitContext-ToDots")
	st.ToDots()
	ptd.End()
}

// ParentBackgroundColor returns the background color
// of the nearest widget parent of the widget that
// has a defined background color. If no such parent is found,
// it returns a transparent background color.
func (wb *WidgetBase) ParentBackgroundColor() colors.Full {
	// todo: this style reading requires a mutex!
	_, pwb := wb.ParentWidgetIf(func(p *WidgetBase) bool {
		// if we have a color or a state layer, we are a relevant breakpoint
		return !p.Style.BackgroundColor.IsNil() || p.Style.StateLayer > 0
	})
	if pwb == nil {
		return colors.Full{}
	}
	// If we don't have a background color ourselves (but we have a state layer),
	// we recursively get our parent's background color and apply our state layer
	// to it. This makes state layers work on transparent elements.
	if pwb.Style.BackgroundColor.IsNil() {
		return pwb.Style.StateBackgroundColor(pwb.ParentBackgroundColor())
	}
	// Otherwise, we can directly apply the state layer to our background color
	return pwb.Style.StateBackgroundColor(pwb.Style.BackgroundColor)
}

/////////////////////////////////////////////////////////////////
// Style helper methods

// SetMinPrefWidth sets minimum and preferred width;
// will get at least this amount; max unspecified.
// This adds a styler that calls [styles.Style.SetMinPrefWidth].
func (wb *WidgetBase) SetMinPrefWidth(val units.Value) Widget {
	wb.AddStyles(func(s *styles.Style) {
		s.SetMinPrefWidth(val)
	})
	return wb.This().(Widget)
}

// SetMinPrefHeight sets minimum and preferred height;
// will get at least this amount; max unspecified.
// This adds a styler that calls [styles.Style.SetMinPrefHeight].
func (wb *WidgetBase) SetMinPrefHeight(val units.Value) Widget {
	wb.AddStyles(func(s *styles.Style) {
		s.SetMinPrefHeight(val)
	})
	return wb.This().(Widget)
}

// SetStretchMaxWidth sets stretchy max width (-1);
// can grow to take up avail room.
// This adds a styler that calls [styles.Style.SetStretchMaxWidth].
func (wb *WidgetBase) SetStretchMaxWidth() Widget {
	wb.AddStyles(func(s *styles.Style) {
		s.SetStretchMaxWidth()
	})
	return wb.This().(Widget)
}

// SetStretchMaxHeight sets stretchy max height (-1);
// can grow to take up avail room.
// This adds a styler that calls [styles.Style.SetStretchMaxHeight].
func (wb *WidgetBase) SetStretchMaxHeight() Widget {
	wb.AddStyles(func(s *styles.Style) {
		s.SetStretchMaxHeight()
	})
	return wb.This().(Widget)
}

// SetStretchMax sets stretchy max width and height (-1);
// can grow to take up avail room.
// This adds a styler that calls [styles.Style.SetStretchMax].
func (wb *WidgetBase) SetStretchMax() Widget {
	wb.AddStyles(func(s *styles.Style) {
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
	wb.AddStyles(func(s *styles.Style) {
		s.SetFixedWidth(val)
	})
	return wb.This().(Widget)
}

// SetFixedHeight sets all height style options
// (Height, MinHeight, and MaxHeight) to
// the given fixed height unit value.
// This adds a styler that calls [styles.Style.SetFixedHeight].
func (wb *WidgetBase) SetFixedHeight(val units.Value) Widget {
	wb.AddStyles(func(s *styles.Style) {
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
