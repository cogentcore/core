// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

//go:generate core generate

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/states"
	"cogentcore.org/core/units"
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

	// Direction specifies the way in which elements are laid out, or
	// the dimension on which an element is longer / travels in.
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

	// For layouts, extra space added between elements in the layout.
	Gap units.XY `view:"inline"`

	// For grid layouts, the number of columns to use.
	// If > 0, number of rows is computed as N elements / Columns.
	// Used as a constraint in layout if individual elements
	// do not specify their row, column positions
	Columns int

	// If this object is a replaced object (image, video, etc)
	// or has a background image, ObjectFit specifies the way
	// in which the replaced object should be fit into the element.
	ObjectFit ObjectFits

	// If this object is a replaced object (image, video, etc)
	// or has a background image, ObjectPosition specifies the
	// X,Y position of the object within the space allocated for
	// the object (see ObjectFit).
	ObjectPosition units.XY

	// Border is a rendered border around the element.
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
	Color image.Image

	// Background specifies the background of the element. It is not inherited,
	// and it is nil (transparent) by default.
	Background image.Image

	// alpha value between 0 and 1 to apply to the foreground and background of this element and all of its children
	Opacity float32

	// StateLayer, if above zero, indicates to create a state layer over the element with this much opacity (on a scale of 0-1) and the
	// color Color (or StateColor if it defined). It is automatically set based on State, but can be overridden in stylers.
	StateLayer float32

	// StateColor, if not nil, is the color to use for the StateLayer instead of Color. If you want to disable state layers
	// for an element, do not use this; instead, set StateLayer to 0.
	StateColor image.Image

	// ActualBackground is the computed actual background rendered for the element,
	// taking into account its Background, Opacity, StateLayer, and parent
	// ActualBackground. It is automatically computed and should not be set manually.
	ActualBackground image.Image

	// VirtualKeyboard is the virtual keyboard to display, if any,
	// on mobile platforms when this element is focused.
	VirtualKeyboard VirtualKeyboards

	// position is only used for Layout = Nil cases
	Pos units.XY `view:"inline"`

	// ordering factor for rendering depth -- lower numbers rendered first.
	// Sort children according to this factor
	ZIndex int

	// specifies the row that this element should appear within a grid layout
	Row int

	// specifies the column that this element should appear within a grid layout
	Col int

	// specifies the number of sequential rows that this element should occupy
	// within a grid layout (todo: not currently supported)
	RowSpan int

	// specifies the number of sequential columns that this element should occupy
	// within a grid layout
	ColSpan int

	// width of a layout scrollbar
	ScrollBarWidth units.Value

	// font styling parameters
	Font Font

	// text styling parameters
	Text Text

	// unit context: parameters necessary for anchoring relative units
	UnitContext units.Context
}

func (s *Style) Defaults() {
	// mostly all the defaults are 0 initial values, except these..
	s.UnitContext.Defaults()
	s.LayoutDefaults()
	s.Color = colors.C(colors.Scheme.OnSurface)
	s.Opacity = 1
	s.FillMargin = true
	s.Font.Defaults()
	s.Text.Defaults()
}

// VirtualKeyboards are all of the supported virtual keyboard types
// to display on mobile platforms.
type VirtualKeyboards int32 //enums:enum -trim-prefix Keyboard -transform kebab

const (
	// KeyboardNone indicates to display no virtual keyboard.
	KeyboardNone VirtualKeyboards = iota

	// KeyboardSingleLine indicates to display a virtual keyboard
	// with a default input style and a "Done" return key.
	KeyboardSingleLine

	// KeyboardMultiLine indicates to display a virtual keyboard
	// with a default input style and a "Return" return key.
	KeyboardMultiLine

	// KeyboardNumber indicates to display a virtual keyboard
	// for inputting a number.
	KeyboardNumber

	// KeyboardPassword indicates to display a virtual keyboard
	// for inputting a password.
	KeyboardPassword

	// KeyboardEmail indicates to display a virtual keyboard
	// for inputting an email address.
	KeyboardEmail

	// KeyboardPhone indicates to display a virtual keyboard
	// for inputting a phone number.
	KeyboardPhone

	// KeyboardURL indicates to display a virtual keyboard for
	// inputting a URL / URI / web address.
	KeyboardURL
)

// todo: Animation

// Clear -- no floating elements

// Clip -- clip images

// column- settings -- lots of those

// List-style for lists

// Object-fit for videos

// visibility -- support more than just hidden

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

// SetEnabled sets the Disabled State flag according to given bool
func (s *Style) SetEnabled(on bool) *Style {
	s.State.SetFlag(!on, states.Disabled)
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

// InheritFields from parent
func (s *Style) InheritFields(parent *Style) {
	s.Color = parent.Color
	s.Opacity = parent.Opacity
	s.Font.InheritFields(&parent.Font)
	s.Text.InheritFields(&parent.Text)
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
	s.ToDotsImpl(&s.UnitContext)
}

// BoxSpace returns the extra space around the central content in the box model in dots.
// It rounds all of the sides first.
func (s *Style) BoxSpace() SideFloats {
	return s.TotalMargin().Add(s.Padding.Dots()).Round()
}

// TotalMargin returns the total effective margin of the element
// holding the style, using the sum of the actual margin, the max
// border width, and the max box shadow effective margin. If the
// values for the max border width / box shadow are unset, the
// current values are used instead, which allows for the omission
// of the max properties when the values do not change.
func (s *Style) TotalMargin() SideFloats {
	mbw := s.MaxBorder.Width.Dots()
	if SidesAreZero(mbw.Sides) {
		mbw = s.Border.Width.Dots()
	}
	mbo := s.MaxBorder.Offset.Dots()
	if SidesAreZero(mbo.Sides) {
		mbo = s.Border.Offset.Dots()
	}
	mbw = mbw.Add(mbo)

	if s.Border.Style.Top == BorderNone {
		mbw.Top = 0
	}
	if s.Border.Style.Right == BorderNone {
		mbw.Right = 0
	}
	if s.Border.Style.Bottom == BorderNone {
		mbw.Bottom = 0
	}
	if s.Border.Style.Left == BorderNone {
		mbw.Left = 0
	}

	mbsm := s.MaxBoxShadowMargin()
	if SidesAreZero(mbsm.Sides) {
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

// ComputeActualBackground sets [Style.ActualBackground] based on the
// given parent actual background and the properties of the style object.
func (s *Style) ComputeActualBackground(pabg image.Image) {
	s.ActualBackground = s.ComputeActualBackgroundFor(s.Background, pabg)
}

// ComputeActualBackgroundFor returns the actual background for
// the given background based on the given parent actual background
// and the properties of the style object.
func (s *Style) ComputeActualBackgroundFor(bg, pabg image.Image) image.Image {
	if bg == nil {
		bg = pabg
	} else if u, ok := bg.(*image.Uniform); ok && colors.IsNil(u.C) {
		bg = pabg
	}

	if s.Opacity >= 1 && s.StateLayer <= 0 {
		// we have no transformations to apply
		return bg
	}

	// TODO(kai): maybe improve this function to handle all
	// use cases correctly (image parents, image state colors, etc)

	upabg := colors.ToUniform(pabg)

	if s.Opacity < 1 {
		bg = gradient.Apply(bg, func(c color.Color) color.Color {
			// we take our opacity-applied background color and then overlay it onto our surrounding color
			obg := colors.ApplyOpacity(c, s.Opacity)
			return colors.AlphaBlend(upabg, obg)
		})
	}
	if s.StateLayer > 0 {
		sc := s.Color
		if s.StateColor != nil {
			sc = s.StateColor
		}
		// we take our state-layer-applied state color and then overlay it onto our background color
		sclr := colors.WithAF32(colors.ToUniform(sc), s.StateLayer)
		bg = gradient.Apply(bg, func(c color.Color) color.Color {
			return colors.AlphaBlend(c, sclr)
		})
	}
	return bg
}

func (s *Style) IsFlexWrap() bool {
	return s.Wrap && s.Display == Flex
}

// SetTextWrap sets the Text.WhiteSpace and GrowWrap properties in
// a coordinated manner.  If wrap == true, then WhiteSpaceNormal
// and GrowWrap = true; else WhiteSpaceNowrap and GrowWrap = false, which
// are typically the two desired stylings.
func (s *Style) SetTextWrap(wrap bool) {
	if wrap {
		s.Text.WhiteSpace = WhiteSpaceNormal
		s.GrowWrap = true
	} else {
		s.Text.WhiteSpace = WhiteSpaceNowrap
		s.GrowWrap = false
	}
}

// SetNonSelectable turns off the Selectable and DoubleClicable
// abilities and sets the Cursor to None.
func (s *Style) SetNonSelectable() {
	s.SetAbilities(false, abilities.Selectable, abilities.DoubleClickable)
	s.Cursor = cursors.None
}
