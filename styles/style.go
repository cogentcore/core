// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"fmt"
	"image/color"
	"strings"

	"goki.dev/colors"
	"goki.dev/girl/states"
	"goki.dev/girl/units"
	"goki.dev/laser"
)

// style implements CSS-based styling, as in: https://www.w3schools.com/cssref/default.asp
// list of inherited: https://stackoverflow.com/questions/5612302/which-css-properties-are-inherited

// styling strategy:
// * either direct Go code based styling functions or ki.Props style map[string]any settings.
// * we process those properties dynamically when rendering (first pass only) into state
//   on objects that can be directly used during rendering
// * good for basic rendering -- lots of additional things that could be extended later..

// IMPORTANT: any changes here must be updated in style_props.go StyleStyleFuncs
// and likewise for all sub-styles as fields here.

// Style has all the CSS-based style elements -- used for widget-type GUI objects.
type Style struct {
	// State holds style-relevant state flags, for convenient styling access,
	// given that styles typically depend on element states.
	State states.States

	// Abilities specifies the abilities of this element, which determine
	// which kinds of states the element can express.
	// This is used by the goosi.events system.  Putting this info next
	// to the State info makes it easy to configure and manage.
	Abilities states.Abilities

	// todo big enum of how to display item -- controls layout etc
	Display bool `xml:"display" desc:"todo big enum of how to display item -- controls layout etc"`

	// is the item visible or not
	Visible bool `xml:"visible" desc:"is the item visible or not"`

	// make a control inactive so it does not respond to input
	Inactive bool `xml:"inactive" desc:"make a control inactive so it does not respond to input"`

	// the cursor to switch to upon hovering over the element (inherited)
	// Cursor cursor.Shapes `desc:"the cursor to switch to upon hovering over the element (inherited)"`

	// prop: z-index = ordering factor for rendering depth -- lower numbers rendered first -- sort children according to this factor
	ZIndex int `xml:"z-index" desc:"prop: z-index = ordering factor for rendering depth -- lower numbers rendered first -- sort children according to this factor"`

	// prop: horizontal-align specifies the horizontal alignment of widget elements within a *vertical* layout container (has no effect within horizontal layouts -- use space / stretch elements instead).  For text layout, use text-align. This is not a standard css property.
	AlignH Align `xml:"horizontal-align" desc:"prop: horizontal-align specifies the horizontal alignment of widget elements within a *vertical* layout container (has no effect within horizontal layouts -- use space / stretch elements instead).  For text layout, use text-align. This is not a standard css property."`

	// prop: vertical-align specifies the vertical alignment of widget elements within a *horizontal* layout container (has no effect within vertical layouts -- use space / stretch elements instead).  For text layout, use text-vertical-align.  This is not a standard css property
	AlignV Align `xml:"vertical-align" desc:"prop: vertical-align specifies the vertical alignment of widget elements within a *horizontal* layout container (has no effect within vertical layouts -- use space / stretch elements instead).  For text layout, use text-vertical-align.  This is not a standard css property"`

	// prop: x = horizontal position -- often superseded by layout but otherwise used
	PosX units.Value `xml:"x" desc:"prop: x = horizontal position -- often superseded by layout but otherwise used"`

	// prop: y = vertical position -- often superseded by layout but otherwise used
	PosY units.Value `xml:"y" desc:"prop: y = vertical position -- often superseded by layout but otherwise used"`

	// prop: width = specified size of element -- 0 if not specified
	Width units.Value `xml:"width" desc:"prop: width = specified size of element -- 0 if not specified"`

	// prop: height = specified size of element -- 0 if not specified
	Height units.Value `xml:"height" desc:"prop: height = specified size of element -- 0 if not specified"`

	// prop: max-width = specified maximum size of element -- 0  means just use other values, negative means stretch
	MaxWidth units.Value `xml:"max-width" desc:"prop: max-width = specified maximum size of element -- 0  means just use other values, negative means stretch"`

	// prop: max-height = specified maximum size of element -- 0 means just use other values, negative means stretch
	MaxHeight units.Value `xml:"max-height" desc:"prop: max-height = specified maximum size of element -- 0 means just use other values, negative means stretch"`

	// prop: min-width = specified minimum size of element -- 0 if not specified
	MinWidth units.Value `xml:"min-width" desc:"prop: min-width = specified minimum size of element -- 0 if not specified"`

	// prop: min-height = specified minimum size of element -- 0 if not specified
	MinHeight units.Value `xml:"min-height" desc:"prop: min-height = specified minimum size of element -- 0 if not specified"`

	// prop: margin = outer-most transparent space around box element
	Margin SideValues `xml:"margin" desc:"prop: margin = outer-most transparent space around box element"`

	// prop: padding = transparent space around central content of box
	Padding SideValues `xml:"padding" desc:"prop: padding = transparent space around central content of box"`

	// prop: overflow = what to do with content that overflows -- default is Auto add of scrollbars as needed -- todo: can have separate -x -y values
	Overflow Overflow `xml:"overflow" desc:"prop: overflow = what to do with content that overflows -- default is Auto add of scrollbars as needed -- todo: can have separate -x -y values"`

	// prop: columns = number of columns to use in a grid layout -- used as a constraint in layout if individual elements do not specify their row, column positions
	Columns int `xml:"columns" alt:"grid-cols" desc:"prop: columns = number of columns to use in a grid layout -- used as a constraint in layout if individual elements do not specify their row, column positions"`

	// prop: row = specifies the row that this element should appear within a grid layout
	Row int `xml:"row" desc:"prop: row = specifies the row that this element should appear within a grid layout"`

	// prop: col = specifies the column that this element should appear within a grid layout
	Col int `xml:"col" desc:"prop: col = specifies the column that this element should appear within a grid layout"`

	// prop: row-span = specifies the number of sequential rows that this element should occupy within a grid layout (todo: not currently supported)
	RowSpan int `xml:"row-span" desc:"prop: row-span = specifies the number of sequential rows that this element should occupy within a grid layout (todo: not currently supported)"`

	// prop: col-span = specifies the number of sequential columns that this element should occupy within a grid layout
	ColSpan int `xml:"col-span" desc:"prop: col-span = specifies the number of sequential columns that this element should occupy within a grid layout"`

	// prop: scrollbar-width = width of a layout scrollbar
	ScrollBarWidth units.Value `xml:"scrollbar-width" desc:"prop: scrollbar-width = width of a layout scrollbar"`

	// prop: color (inherited) = text color -- also defines the currentColor variable value
	Color color.RGBA `xml:"color" inherit:"true" desc:"prop: color (inherited) = text color -- also defines the currentColor variable value"`

	// prop: background-color = background color -- not inherited, transparent by default
	BackgroundColor colors.Full `xml:"background-color" desc:"prop: background-color = background color -- not inherited, transparent by default"`

	// border around the box element
	Border Border `xml:"border" desc:"border around the box element"`

	// prop: box-shadow = the box shadows to render around box (can have multiple)
	BoxShadow []Shadow `xml:"box-shadow" desc:"prop: box-shadow = the box shadows to render around box (can have multiple)"`

	// font parameters -- no xml prefix -- also has color, background-color
	Font Font `desc:"font parameters -- no xml prefix -- also has color, background-color"`

	// text parameters -- no xml prefix
	Text Text `desc:"text parameters -- no xml prefix"`

	// prop: outline = draw an outline around an element -- mostly same styles as border -- default to none
	Outline Border `xml:"outline" desc:"prop: outline = draw an outline around an element -- mostly same styles as border -- default to none"`

	// prop: pointer-events = does this element respond to pointer events -- default is true
	PointerEvents bool `xml:"pointer-events" desc:"prop: pointer-events = does this element respond to pointer events -- default is true"`

	// units context -- parameters necessary for anchoring relative units
	UnContext units.Context `xml:"-" desc:"units context -- parameters necessary for anchoring relative units"`

	// has this style been set from object values yet?
	IsSet bool `desc:"has this style been set from object values yet?"`

	// set to true if parent node has no props -- allows optimization of styling
	PropsNil   bool `desc:"set to true if parent node has no props -- allows optimization of styling"`
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

// RebuildDefaultStyles is a global state var used by Prefs to trigger rebuild
// of all the default styles, which are otherwise compiled and not updated
var RebuildDefaultStyles bool

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
	// s.Cursor = par.Cursor
	s.Color = par.Color
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

// BoxSpace returns extra space around the central content in the box model,
// in dots -- todo: must complicate this if we want different spacing on
// different sides box outside-in: margin | border | padding | content
func (s *Style) BoxSpace() SideFloats {
	return s.EffMargin().Add(s.Border.Width.Dots()).Add(s.Padding.Dots())
}

// EffMargin returns the effective margin of the element
// holding the style, using the maximum of the actual
// margin and the box shadow margin.
func (s *Style) EffMargin() SideFloats {
	// TODO: this does not follow the CSS spec
	// (shadow is supposed to be outside of the box model),
	// but that is incompatible with the way we position things.
	// Maybe we could come up with a better way to do this at some point.
	return s.Margin.Dots().Max(s.BoxShadowMargin())
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
	return nil, false
}

// StyleDefault is default style can be used when property specifies "default"
var StyleDefault Style
