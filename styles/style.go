// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"fmt"
	"image/color"
	"strings"

	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/enums"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
	"goki.dev/laser"
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
type Style struct {
	// State holds style-relevant state flags, for convenient styling access,
	// given that styles typically depend on element states.
	State states.States

	// Abilities specifies the abilities of this element, which determine
	// which kinds of states the element can express.
	// This is used by the goosi/events system.  Putting this info next
	// to the State info makes it easy to configure and manage.
	Abilities abilities.Abilities

	// todo big enum of how to display item -- controls layout etc
	Display bool `xml:"display"`

	// is the item visible or not
	Visible bool `xml:"visible"`

	// the cursor to switch to upon hovering over the element (inherited)
	Cursor cursors.Cursor

	// prop: z-index = ordering factor for rendering depth -- lower numbers rendered first -- sort children according to this factor
	ZIndex int `xml:"z-index"`

	// prop: horizontal-align specifies the horizontal alignment of widget elements within a *vertical* layout container (has no effect within horizontal layouts -- use space / stretch elements instead).  For text layout, use text-align. This is not a standard css property.
	AlignH Align `xml:"horizontal-align"`

	// prop: vertical-align specifies the vertical alignment of widget elements within a *horizontal* layout container (has no effect within vertical layouts -- use space / stretch elements instead).  For text layout, use text-vertical-align.  This is not a standard css property
	AlignV Align `xml:"vertical-align"`

	// prop: x = horizontal position -- often superseded by layout but otherwise used
	PosX units.Value `xml:"x"`

	// prop: y = vertical position -- often superseded by layout but otherwise used
	PosY units.Value `xml:"y"`

	// prop: width = specified size of element -- 0 if not specified
	Width units.Value `xml:"width"`

	// prop: height = specified size of element -- 0 if not specified
	Height units.Value `xml:"height"`

	// prop: max-width = specified maximum size of element -- 0  means just use other values, negative means stretch
	MaxWidth units.Value `xml:"max-width"`

	// prop: max-height = specified maximum size of element -- 0 means just use other values, negative means stretch
	MaxHeight units.Value `xml:"max-height"`

	// prop: min-width = specified minimum size of element -- 0 if not specified
	MinWidth units.Value `xml:"min-width"`

	// prop: min-height = specified minimum size of element -- 0 if not specified
	MinHeight units.Value `xml:"min-height"`

	// prop: margin = outer-most transparent space around box element
	Margin SideValues `xml:"margin"`

	// prop: padding = transparent space around central content of box
	Padding SideValues `xml:"padding"`

	// if this is a layout, extra space to add between elements in the layout
	Spacing units.Value `xml:"spacing"`

	// prop: overflow = what to do with content that overflows -- default is Auto add of scrollbars as needed -- todo: can have separate -x -y values
	Overflow Overflow `xml:"overflow"`

	// prop: columns = number of columns to use in a grid layout -- used as a constraint in layout if individual elements do not specify their row, column positions
	Columns int `xml:"columns" alt:"grid-cols"`

	// prop: row = specifies the row that this element should appear within a grid layout
	Row int `xml:"row"`

	// prop: col = specifies the column that this element should appear within a grid layout
	Col int `xml:"col"`

	// prop: row-span = specifies the number of sequential rows that this element should occupy within a grid layout (todo: not currently supported)
	RowSpan int `xml:"row-span"`

	// prop: col-span = specifies the number of sequential columns that this element should occupy within a grid layout
	ColSpan int `xml:"col-span"`

	// prop: scrollbar-width = width of a layout scrollbar
	ScrollBarWidth units.Value `xml:"scrollbar-width"`

	// prop: color (inherited) = text color -- also defines the currentColor variable value
	Color color.RGBA `xml:"color" inherit:"true"`

	// prop: background-color = background color -- not inherited, transparent by default
	BackgroundColor colors.Full `xml:"background-color"`

	// prop: opacity = alpha value to apply to the foreground and background of this element and all of its children
	Opacity float32 `xml:"opacity"`

	// StateLayer, if above zero, indicates to create a state layer over the element with this much opacity (on a scale of 0-1) and the
	// color Color (or StateColor if it defined). It is automatically set based on State, but can be overridden in stylers.
	StateLayer float32

	// StateColor, if not the zero color, is the color to use for the StateLayer instead of Color. If you want to disable state layers
	// for an element, do not use this; instead, set StateLayer to 0.
	StateColor color.RGBA

	// FillSurround is whether to fill a box of the surrounding background
	// color before rendering the element itself, which is typically
	// necessary to prevent text, border, and box shadow from rendering
	// over themselves. It should be kept at its default value of true
	// in most circumstances, but it can be set to false when the element
	// is fully managed by something that is guaranteed to render the
	// appropriate background color for the element.
	FillSurround bool

	// border around the box element
	Border Border `xml:"border"`

	// MaxBorder is the largest border that will ever be rendered around the element, the size of which is used for computing the effective margin to allocate for the element
	MaxBorder Border

	// prop: box-shadow = the box shadows to render around box (can have multiple)
	BoxShadow []Shadow `xml:"box-shadow"`

	// MaxBoxShadow contains the largest shadows that will ever be rendered around the element, the size of which are used for computing the effective margin to allocate for the element
	MaxBoxShadow []Shadow

	// font parameters -- no xml prefix -- also has color, background-color
	Font Font

	// text parameters -- no xml prefix
	Text Text

	// prop: outline = draw an outline around an element -- mostly same styles as border -- default to none
	Outline Border `xml:"outline"`

	// prop: pointer-events = does this element respond to pointer events -- default is true
	PointerEvents bool `xml:"pointer-events"`

	// units context -- parameters necessary for anchoring relative units
	UnContext units.Context `xml:"-"`

	// has this style been set from object values yet?
	IsSet bool

	// set to true if parent node has no props -- allows optimization of styling
	PropsNil   bool
	dotsSet    bool
	lastUnCtxt units.Context
}

func (s *Style) Defaults() {
	// mostly all the defaults are 0 initial values, except these..
	s.IsSet = false
	s.UnContext.Defaults()
	s.Outline.Style.Set(BorderNone)
	s.Display = true
	s.PointerEvents = true

	s.LayoutDefaults()
	s.Color = colors.Black
	s.Opacity = 1
	s.FillSurround = true
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

// ActiveStyler defines an interface for anything
// that can report its active style
type ActiveStyler interface {
	ActiveStyle() *Style

	// StyleRLock does a read-lock for reading the style
	StyleRLock()

	// StyleRUnlock unlocks the read-lock
	StyleRUnlock()
}

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

func NewStyle() Style {
	s := Style{}
	s.Defaults()
	return s
}

// Is returns true if the State flag is set
func (s *Style) Is(st states.States) bool {
	return s.State.HasFlag(st)
}

// SetAbilities sets the abilities flags
func (s *Style) SetAbilities(on bool, able ...enums.BitFlag) {
	s.Abilities.SetFlag(on, able...)
}

// CopyFrom copies from another style, while preserving relevant local state
func (s *Style) CopyFrom(cp *Style) {
	is := s.IsSet
	pn := s.PropsNil
	ds := s.dotsSet
	lu := s.lastUnCtxt
	*s = *cp
	s.BackgroundColor = cp.BackgroundColor
	s.IsSet = is
	s.PropsNil = pn
	s.dotsSet = ds
	s.lastUnCtxt = lu
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

// StyleToDots runs ToDots on unit values, to compile down to raw pixels
func (s *Style) StyleToDots(uc *units.Context) {
	// none
}

// ToDotsImpl runs ToDots on unit values, to compile down to raw pixels
func (s *Style) ToDotsImpl(uc *units.Context) {
	s.StyleToDots(uc)

	s.LayoutToDots(uc)
	s.Font.ToDots(uc)
	s.Text.ToDots(uc)
	s.Border.ToDots(uc)
	s.MaxBorder.ToDots(uc)
	s.Outline.ToDots(uc)
	s.BoxShadowToDots(uc)
}

// ToDots caches all style elements in terms of raw pixel
// dots for rendering.
func (s *Style) ToDots() {
	s.ToDotsImpl(&s.UnContext)
	s.dotsSet = true
	s.lastUnCtxt = s.UnContext
}

// CopyUnitContext copies unit context from another, update with our font
// info, and then cache everything out in terms of raw pixel dots for
// rendering -- call at start of render
func (s *Style) CopyUnitContext(ctxt *units.Context) {
	s.UnContext = *ctxt
	if !(s.dotsSet && s.UnContext == s.lastUnCtxt && s.PropsNil) {
		s.ToDots()
	}
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
			bg.Solid = colors.AlphaBlend(bg.Solid, colors.SetAF32(clr, s.StateLayer))
		}
		if s.Opacity < 1 {
			bg.Solid = colors.SetA(bg.Solid, uint8(s.Opacity*255)*bg.Solid.A)
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
			res.Gradient.Stops[i].Color = colors.AlphaBlend(stop.Color, colors.SetAF32(clr, s.StateLayer))
		}
		if s.Opacity < 1 {
			res.Gradient.Stops[i].Color = colors.SetA(stop.Color, uint8(s.Opacity*255)*stop.Color.A)
		}
	}
	return res
}
