// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

//go:generate goki generate

import (
	"fmt"
	"image/color"
	"io"
	"strings"

	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/enums"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
	"goki.dev/laser"
	"goki.dev/mat32/v2"
)

// style implements CSS-based styling, as in: https://www.w3schools.com/cssref/default.asp
// list of inherited: https://stackoverflow.com/questions/5612302/which-css-properties-are-inherited

// styling strategy:
//	- either direct Go code based styling functions or ki.Props style map[string]any settings.
//	- we process those properties dynamically when rendering (first pass only) into state
//   on objects that can be directly used during rendering
//	- good for basic rendering -- lots of additional things that could be extended later..

// IMPORTANT: any changes here must be updated in style_props.go StyleStyleFuncs
// and likewise for all sub-styles as fields here.

// Style has all the CSS-based style elements -- used for widget-type GUI objects.
type Style struct { //gti:add
	// State holds style-relevant state flags, for convenient styling access,
	// given that styles typically depend on element states.
	State states.States

	// Abilities specifies the abilities of this element, which determine
	// which kinds of states the element can express.
	// This is used by the goosi/events system.  Putting this info next
	// to the State info makes it easy to configure and manage.
	Abilities abilities.Abilities

	// the cursor to switch to upon hovering over the element (inherited)
	Cursor cursors.Cursor

	// Padding is the transparent space around central content of box,
	// which is _included_ in the size of the standard box rendering.
	Padding SideValues `view:"inline"`

	// Margin is the outer-most transparent space around box element,
	// which is _excluded_ from standard box rendering.
	Margin SideValues `view:"inline"`

	// Display controls how items are displayed, in terms of layout
	Display Displays

	// Direction specifies the order elements are organized:
	// Row is horizontal, Col is vertical.
	// See also [Wrap]
	Direction Directions

	// Wrap causes elements to wrap around in the CrossAxis dimension
	// to fit within sizing constraints (on by default).
	Wrap bool

	// Justify specifies the distribution of elements along the main axis,
	// i.e., the same as Direction, for Flex Display.  For Grid, the main axis is
	// given by the writing direction (e.g., Row-wise for latin based languages).
	Justify AlignSet `view:"inline"`

	// Align specifies the cross-axis alignment of elements, orthogonal to the
	// main Direction axis. For Grid, the cross-axis is orthogonal to the
	// writing direction (e.g., Column-wise for latin based languages).
	Align AlignSet `view:"inline"`

	// Min is the minimum size of the actual content, exclusive of additional space
	// from padding, border, margin; 0 = default is sum of Min for all content
	// (which _includes_ space for all sub-elements).
	// This is equivalent to the Basis for the CSS flex styling model.
	Min units.XY `view:"inline"`

	// Max is the maximum size of the actual content, exclusive of additional space
	// from padding, border, margin; 0 = default provides no Max size constraint
	Max units.XY `view:"inline"`

	// Grow is the proportional amount that the element can grow (stretch)
	// if there is more space available.  0 = default = no growth.
	// Extra available space is allocated as: Grow / sum (all Grow).
	// Important: grow elements absorb available space and thus are not
	// subject to alignment (Center, End).
	Grow mat32.Vec2

	// GrowWrap is a special case for Text elements where it grows initially
	// in the horizontal axis to allow for longer, word wrapped text to fill
	// the available space, but then it does not grow thereafter, so that alignment
	// operations still work (Grow elements do not align because they absorb all
	// available space).
	GrowWrap bool

	// FillMargin determines is whether to fill the margin with
	// the surrounding background color before rendering the element itself.
	// This is typically necessary to prevent text, border, and box shadow from rendering
	// over themselves. It should be kept at its default value of true
	// in most circumstances, but it can be set to false when the element
	// is fully managed by something that is guaranteed to render the
	// appropriate background color for the element.
	FillMargin bool

	// Overflow determines how to handle overflowing content in a layout.
	// Default is OverflowVisible.  Set to OverflowAuto to enable scrollbars.
	Overflow XY[Overflows]

	// For layout, extra space added between elements in the layout.
	Gap units.XY `view:"inline"`

	// For layout, number of columns to use in a grid layout.
	// If > 0, number of rows is computed as N elements / Columns.
	// Used as a constraint in layout if individual elements
	// do not specify their row, column positions
	Columns int

	// If this object is a replaced object (image, video, etc)
	// or has a background image, ObjectFit specifies the way
	// in which the replaced object should be fit into the element.
	ObjectFit ObjectFits

	// Border is a line border around the box element
	Border Border

	// MaxBorder is the largest border that will ever be rendered
	// around the element, the size of which is used for computing
	// the effective margin to allocate for the element.
	MaxBorder Border

	// BoxShadow is the box shadows to render around box (can have multiple)
	BoxShadow []Shadow

	// MaxBoxShadow contains the largest shadows that will ever be rendered
	// around the element, the size of which are used for computing the
	// effective margin to allocate for the element.
	MaxBoxShadow []Shadow

	// Color specifies the text / content color, and it is inherited.
	Color color.RGBA `inherit:"true"`

	// BackgroundColor specifies the background color of the element. It is not inherited,
	// and it is transparent by default.
	BackgroundColor colors.Full

	// BackgroundImage, if non-nil, specifies an [io.Reader] to read a background image from using [image.Decode].
	// If it is specified, [Style.BackgroundColor] has no effect.
	BackgroundImage io.Reader

	// prop: opacity = alpha value to apply to the foreground and background of this element and all of its children
	Opacity float32

	// StateLayer, if above zero, indicates to create a state layer over the element with this much opacity (on a scale of 0-1) and the
	// color Color (or StateColor if it defined). It is automatically set based on State, but can be overridden in stylers.
	StateLayer float32

	// StateColor, if not the zero color, is the color to use for the StateLayer instead of Color. If you want to disable state layers
	// for an element, do not use this; instead, set StateLayer to 0.
	StateColor color.RGBA

	// position is only used for Layout = Nil cases
	Pos units.XY `view:"inline"`

	// ordering factor for rendering depth -- lower numbers rendered first.
	// Sort children according to this factor
	ZIndex int

	// prop: row = specifies the row that this element should appear within a grid layout
	Row int

	// prop: col = specifies the column that this element should appear within a grid layout
	Col int

	// specifies the number of sequential rows that this element should occupy
	// within a grid layout (todo: not currently supported)
	RowSpan int

	// specifies the number of sequential columns that this element should occupy
	// within a grid layout
	ColSpan int

	// width of a layout scrollbar
	ScrollBarWidth units.Value

	// font parameters -- no xml prefix -- also has color, background-color
	Font Font

	// text parameters -- no xml prefix
	Text Text

	// units context -- parameters necessary for anchoring relative units
	UnContext units.Context
}

func (s *Style) Defaults() {
	// mostly all the defaults are 0 initial values, except these..
	s.UnContext.Defaults()
	s.LayoutDefaults()
	s.Color = colors.Black
	s.Opacity = 1
	s.FillMargin = true
	s.Font.Defaults()
	s.Text.Defaults()
}

// todo: Animation

// Clear -- no floating elements

// Clip -- clip images

// column- settings -- lots of those

// List-style for lists

// Object-fit for videos

// visibility -- support more than just hidden  inherit:"true"

// transition -- animation of hover, etc

// SetStylePropsXML sets style props from XML style string, which contains ';'
// separated name: value pairs
func SetStylePropsXML(style string, props *map[string]any) {
	st := strings.Split(style, ";")
	for _, s := range st {
		kv := strings.Split(s, ":")
		if len(kv) >= 2 {
			k := strings.TrimSpace(strings.ToLower(kv[0]))
			v := strings.TrimSpace(kv[1])
			if *props == nil {
				*props = make(map[string]any)
			}
			(*props)[k] = v
		}
	}
}

// StylePropsXML returns style props for XML style string, which contains ';'
// separated name: value pairs
func StylePropsXML(props map[string]any) string {
	var sb strings.Builder
	for k, v := range props {
		if k == "transform" {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s:%s;", k, laser.ToString(v)))
	}
	return sb.String()
}

// NewStyle returns a new [Style] object with default values
func NewStyle() *Style {
	s := &Style{}
	s.Defaults()
	return s
}

// Is returns whether the given [states.States] flag is set
func (s *Style) Is(st states.States) bool {
	return s.State.HasFlag(st)
}

// AbilityIs returns whether the given [abilities.Abilities] flag is set
func (s *Style) AbilityIs(able abilities.Abilities) bool {
	return s.Abilities.HasFlag(able)
}

// SetState sets the given [states.State] flags to the given value
func (s *Style) SetState(on bool, state ...states.States) *Style {
	bfs := make([]enums.BitFlag, len(state))
	for i, st := range state {
		bfs[i] = st
	}
	s.State.SetFlag(on, bfs...)
	return s
}

// SetAbilities sets the given [states.State] flags to the given value
func (s *Style) SetAbilities(on bool, able ...abilities.Abilities) {
	bfs := make([]enums.BitFlag, len(able))
	for i, st := range able {
		bfs[i] = st
	}
	s.Abilities.SetFlag(on, bfs...)
}

// CopyFrom copies from another style, while preserving relevant local state
func (s *Style) CopyFrom(cp *Style) {
	*s = *cp
	s.BackgroundColor = cp.BackgroundColor
}

// InheritFields from parent: Manual inheriting of values is much faster than
// automatic version!
func (s *Style) InheritFields(par *Style) {
	// fmt.Println("Inheriting from", *par)
	s.Color = par.Color
	// we only inherit the parent's state layer if we don't have one for ourself
	if s.StateLayer == 0 {
		s.StateLayer = par.StateLayer
	}
	s.Font.InheritFields(&par.Font)
	s.Text.InheritFields(&par.Text)
}

// ToDotsImpl runs ToDots on unit values, to compile down to raw pixels
func (s *Style) ToDotsImpl(uc *units.Context) {
	s.LayoutToDots(uc)
	s.Font.ToDots(uc)
	s.Text.ToDots(uc)
	s.Border.ToDots(uc)
	s.MaxBorder.ToDots(uc)
	s.BoxShadowToDots(uc)
}

// ToDots caches all style elements in terms of raw pixel
// dots for rendering.
func (s *Style) ToDots() {
	s.ToDotsImpl(&s.UnContext)
}

// BoxSpace returns extra space around the central content in the box model, in dots
func (s *Style) BoxSpace() SideFloats {
	return s.TotalMargin().Add(s.Padding.Dots())
}

// TotalMargin returns the total effective margin of the element
// holding the style, using the sum of the actual margin, the max
// border width, and the max box shadow effective margin. If the
// values for the max border width / box shadow are unset, the
// current values are used instead, which allows for the omission
// of the max properties when the values do not change.
func (s *Style) TotalMargin() SideFloats {
	mbw := s.MaxBorder.Width.Dots()
	if mbw.IsZero() {
		mbw = s.Border.Width.Dots()
	}
	mbsm := s.MaxBoxShadowMargin()
	if mbsm.IsZero() {
		mbsm = s.BoxShadowMargin()
	}
	return s.Margin.Dots().Add(mbw).Add(mbsm)
}

// SubProps returns a sub-property map from given prop map for a given styling
// selector (property name) -- e.g., :normal :active :hover etc -- returns
// false if not found
func SubProps(prp map[string]any, selector string) (map[string]any, bool) {
	sp, ok := prp[selector]
	if !ok {
		return nil, false
	}
	spm, ok := sp.(map[string]any)
	if ok {
		return spm, true
	}
	kpm, ok := sp.(ki.Props)
	if ok {
		return kpm, true
	}
	return nil, false
}

// StyleDefault is default style can be used when property specifies "default"
var StyleDefault Style

// StateBackgroundColor returns the stateful, effective version of
// the given background color by applying [Style.StateLayer] based on
// [Style.Color] and [Style.StateColor]. It also applies [Style.Opacity]
// to the color. It does not modify the underlying style object.
func (s *Style) StateBackgroundColor(bg colors.Full) colors.Full {
	if s.StateLayer <= 0 && s.Opacity >= 1 {
		return bg
	}
	if bg.Gradient == nil {
		if s.StateLayer > 0 {
			clr := s.Color
			if !colors.IsNil(s.StateColor) {
				clr = s.StateColor
			}
			bg.Solid = colors.AlphaBlend(bg.Solid, colors.WithAF32(clr, s.StateLayer))
		}
		if s.Opacity < 1 {
			bg.Solid = colors.WithA(bg.Solid, uint8(s.Opacity*255)*bg.Solid.A)
		}
		return bg
	}
	// still need to copy because underlying gradient isn't automatically copied
	res := colors.Full{}
	res.CopyFrom(&bg)
	for i, stop := range res.Gradient.Stops {
		if s.StateLayer > 0 {
			clr := s.Color
			if !colors.IsNil(s.StateColor) {
				clr = s.StateColor
			}
			res.Gradient.Stops[i].Color = colors.AlphaBlend(stop.Color, colors.WithAF32(clr, s.StateLayer))
		}
		if s.Opacity < 1 {
			res.Gradient.Stops[i].Color = colors.WithA(stop.Color, uint8(s.Opacity*255)*stop.Color.A)
		}
	}
	return res
}

func (st *Style) IsFlexWrap() bool {
	return st.Wrap && st.Display == Flex
}

// SetTextWrap sets the Text.WhiteSpace and GrowWrap properties in
// a coordinated manner.  If wrap == true, then WhiteSpaceNormal
// and GrowWrap = true; else WhiteSpaceNowrap and GrowWrap = false, which
// are typically the two desired stylings.
func (st *Style) SetTextWrap(wrap bool) {
	if wrap {
		st.Text.WhiteSpace = WhiteSpaceNormal
		st.GrowWrap = true
	} else {
		st.Text.WhiteSpace = WhiteSpaceNowrap
		st.GrowWrap = false
	}
}

// SetNonSelectable turns off the Selectable and DoubleClicable
// abilities and sets the Cursor to None.
func (st *Style) SetNonSelectable() {
	st.SetAbilities(false, abilities.Selectable, abilities.DoubleClickable)
	st.Cursor = cursors.None
}
