// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/girl/paint"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/cursor"
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
var CustomConfigStyles func(w *WidgetBase)

// Styler is a fuction that can be used to style an element.
// They are the building blocks of the GoGi styling system.
// They can be used as a closure and capture surrounding context,
// but they are passed the widget base and style for convenience
// and so that they can be used for multiple elements if desired;
// you can get all of the information you need from the function.
// A Styler should be added to a widget through the [WidgetBase.AddStyler]
// method. We use stylers for styling because they give you complete
// control and full programming logic without any CSS-selector magic.
type Styler func(w *WidgetBase, s *styles.Style)

func (sc *Scene) SetDefaultStyle() {
	sc.Frame.Style.BackgroundColor.SetSolid(colors.Scheme.Background)
	sc.Frame.Style.Color = colors.Scheme.OnBackground
}

////////////////////////////////////////////////////////////////////
// 	Widget Styling functions

// AddStyler adds the given styler to the
// widget's stylers, initializing them if necessary.
// This function can be called by both internal
// and end-user code.
// It should only be done before showing the scene
// during initial configuration -- otherwise requries
// a StyMu mutex lock.
func (wb *WidgetBase) AddStyler(s Styler) Widget {
	if wb.Stylers == nil {
		wb.Stylers = []Styler{}
	}
	wb.Stylers = append(wb.Stylers, s)
	return wb.This().(Widget)
}

// todo: should reserve ApplyStyle for user and rename existing to something
// more internal.
// note: cannot Change AddStyler signature to return Widget, so need a new method

func (wb *WidgetBase) SetStyle(s Styler) Widget {
	wb.AddStyler(s)
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
// 			wb.AddStyler(func(w *WidgetBase, s *styles.Style) {
// 				f(wb)
// 			})
// 		}
// 	}
// }

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
		ps.StyleRLock()
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

	if wb.OverrideStyle {
		return
	}

	wb.Style = styles.Style{}
	wb.Style.Defaults()

	// todo: remove all these prof steps -- should be much less now..
	pin := prof.Start("ApplyStyleWidget-Inherit")

	if parSty := wb.ParentActiveStyle(); parSty != nil {
		wb.Style.InheritFields(parSty)
		wb.ParentStyleRUnlock()
	}
	pin.End()

	prun := prof.Start("ApplyStyleWidget-RunStylers")
	wb.RunStylers()
	prun.End()

	puc := prof.Start("ApplyStyleWidget-SetUnitContext")
	SetUnitContext(&wb.Style, wb.Sc, mat32.Vec2{}, mat32.Vec2{})
	puc.End()

	psc := prof.Start("ApplyStyleWidget-SetCurrentColor")
	if wb.Style.Inactive { // inactive can only set, not clear
		wb.SetFlag(true, Disabled)
	}
	sc.SetCurrentColor(wb.Style.Color)

	wb.ApplyStyleParts(sc)

	psc.End()
}

// RunStylers runs the style functions specified in
// the StyleFuncs field in sequential ascending order.
func (wb *WidgetBase) RunStylers() {
	for _, s := range wb.Stylers {
		s(wb, &wb.Style)
	}
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
		}
		if sc.RenderState.Image != nil {
			sz := sc.Geom.Size // Render.Image.Bounds().Size()
			st.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, par.X, par.Y)
		}
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
// it returns a new [colors.Full] with a solid
// color of [colors.Scheme.Background].
func (wb *WidgetBase) ParentBackgroundColor() colors.Full {
	// todo: this style reading requires a mutex!
	_, pwb := wb.ParentWidgetIf(func(p *WidgetBase) bool {
		return !p.Style.BackgroundColor.IsNil()
	})
	if pwb == nil {
		cs := colors.Full{}
		cs.SetColor(colors.Scheme.Background)
		return cs
	}
	return pwb.Style.BackgroundColor
}

// ParentCursor returns the cursor of the nearest
// widget parent of the widget that has a a non-default
// cursor. If no such parent is found, it returns the given
// cursor. This function can be used for elements like labels
// that have a default cursor ([cursor.IBeam]) but should
// not override the cursor of a parent.
func (wb *WidgetBase) ParentCursor(cur cursors.Cursor) cursor.Cursor {
	_, pwb := wb.ParentWidgetIf(func(p *WidgetBase) bool {
		// return p.Style.Cursor != cursor.Arrow
		return true
	})
	if pwb == nil {
		return cur
	}
	// return pwb.Style.Cursor
	return cursor.Arrow
}

/////////////////////////////////////////////////////////////////
// Style helper methods

// SetMinPrefWidth sets minimum and preferred width;
// will get at least this amount; max unspecified.
// This adds a styler that calls [styles.Style.SetMinPrefWidth].
func (wb *WidgetBase) SetMinPrefWidth(val units.Value) Widget {
	wb.AddStyler(func(w *WidgetBase, s *styles.Style) {
		s.SetMinPrefWidth(val)
	})
	return wb.This().(Widget)
}

// SetMinPrefHeight sets minimum and preferred height;
// will get at least this amount; max unspecified.
// This adds a styler that calls [styles.Style.SetMinPrefHeight].
func (wb *WidgetBase) SetMinPrefHeight(val units.Value) Widget {
	wb.AddStyler(func(w *WidgetBase, s *styles.Style) {
		s.SetMinPrefHeight(val)
	})
	return wb.This().(Widget)
}

// SetStretchMaxWidth sets stretchy max width (-1);
// can grow to take up avail room.
// This adds a styler that calls [styles.Style.SetStretchMaxWidth].
func (wb *WidgetBase) SetStretchMaxWidth() Widget {
	wb.AddStyler(func(w *WidgetBase, s *styles.Style) {
		s.SetStretchMaxWidth()
	})
	return wb.This().(Widget)
}

// SetStretchMaxHeight sets stretchy max height (-1);
// can grow to take up avail room.
// This adds a styler that calls [styles.Style.SetStretchMaxHeight].
func (wb *WidgetBase) SetStretchMaxHeight() Widget {
	wb.AddStyler(func(w *WidgetBase, s *styles.Style) {
		s.SetStretchMaxHeight()
	})
	return wb.This().(Widget)
}

// SetStretchMax sets stretchy max width and height (-1);
// can grow to take up avail room.
// This adds a styler that calls [styles.Style.SetStretchMax].
func (wb *WidgetBase) SetStretchMax() Widget {
	wb.AddStyler(func(w *WidgetBase, s *styles.Style) {
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
	wb.AddStyler(func(w *WidgetBase, s *styles.Style) {
		s.SetFixedWidth(val)
	})
	return wb.This().(Widget)
}

// SetFixedHeight sets all height style options
// (Height, MinHeight, and MaxHeight) to
// the given fixed height unit value.
// This adds a styler that calls [styles.Style.SetFixedHeight].
func (wb *WidgetBase) SetFixedHeight(val units.Value) Widget {
	wb.AddStyler(func(w *WidgetBase, s *styles.Style) {
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

////////////////////////////////////////////////////////////////////
// 	Default Style Vars

// Pre-configured box shadow values, based on
// those in Material 3. They are in gi because
// they need to access the color scheme.
var (
	// BoxShadow0 contains the shadows
	// to be used on Elevation 0 elements.
	// There are no shadows part of BoxShadow0,
	// so applying it is purely semantic.
	BoxShadow0 = []styles.Shadow{}
	// BoxShadow1 contains the shadows
	// to be used on Elevation 1 elements.
	BoxShadow1 = []styles.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(3),
			Blur:    units.Px(1),
			Spread:  units.Px(-2),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(2),
			Blur:    units.Px(2),
			Spread:  units.Px(0),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(1),
			Blur:    units.Px(5),
			Spread:  units.Px(0),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.12),
		},
	}
	// BoxShadow2 contains the shadows
	// to be used on Elevation 2 elements.
	BoxShadow2 = []styles.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(2),
			Blur:    units.Px(4),
			Spread:  units.Px(-1),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(4),
			Blur:    units.Px(5),
			Spread:  units.Px(0),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(1),
			Blur:    units.Px(10),
			Spread:  units.Px(0),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.12),
		},
	}
	// TODO: figure out why 3 and 4 are the same

	// BoxShadow3 contains the shadows
	// to be used on Elevation 3 elements.
	BoxShadow3 = []styles.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(5),
			Blur:    units.Px(5),
			Spread:  units.Px(-3),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(8),
			Blur:    units.Px(10),
			Spread:  units.Px(1),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(3),
			Blur:    units.Px(14),
			Spread:  units.Px(2),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.12),
		},
	}
	// BoxShadow4 contains the shadows
	// to be used on Elevation 4 elements.
	BoxShadow4 = []styles.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(5),
			Blur:    units.Px(5),
			Spread:  units.Px(-3),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(8),
			Blur:    units.Px(10),
			Spread:  units.Px(1),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(3),
			Blur:    units.Px(14),
			Spread:  units.Px(2),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.12),
		},
	}
	// BoxShadow5 contains the shadows
	// to be used on Elevation 5 elements.
	BoxShadow5 = []styles.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(8),
			Blur:    units.Px(10),
			Spread:  units.Px(-6),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(16),
			Blur:    units.Px(24),
			Spread:  units.Px(2),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(6),
			Blur:    units.Px(30),
			Spread:  units.Px(5),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.12),
		},
	}
)

/*

// StyleProps returns a property that contains another map of properties for a
// given styling selector, such as :normal :active :hover etc -- the
// convention is to prefix this selector with a : and use lower-case names, so
// we follow that.
func (nb *NodeBase) StyleProps(selector string) ki.Props {
	sp, ok := nb.PropInherit(selector, ki.NoInherit, ki.TypeProps) // yeah, use type's
	if !ok {
		return nil
	}
	spm, ok := sp.(ki.Props)
	if ok {
		return spm
	}
	log.Printf("styles.StyleProps: looking for a ki.Props for style selector: %v, instead got type: %T, for node: %v\n", selector, spm, nb.Path())
	return nil
}

// AggCSS aggregates css properties
func AggCSS(agg *ki.Props, css ki.Props) {
	if *agg == nil {
		*agg = make(ki.Props, len(css))
	}
	for key, val := range css {
		(*agg)[key] = val
	}
}

// ParentCSSAgg returns parent's CSSAgg styles or nil if not avail
func (nb *NodeBase) ParentCSSAgg() *ki.Props {
	if nb.Par == nil {
		return nil
	}
	pn := nb.Par.Embed(TypeNodeBase)
	if pn == nil {
		return nil
	}
	return &pn.(*NodeBase).CSSAgg
}

*/
