// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// style implements CSS-based styling using ki.Props to hold name / vals
// CSS style reference: https://www.w3schools.com/cssref/default.asp
// list of inherited: https://stackoverflow.com/questions/5612302/which-css-properties-are-inherited

// styling strategy:
// * indiv objects specify styles using property map -- good b/c it is fully open-ended
// * we process those properties dynamically when rendering (first pass only) into state
//   on objects that can be directly used during rendering
// * good for basic rendering -- lots of additional things that could be extended later..

// StyleTemplates are cached styles used for styling large numbers of identical
// elements in views
var StyleTemplates map[string]*Style

// StyleTemplatesMu is a mutex protecting updates to StyleTemplates
var StyleTemplatesMu sync.RWMutex

// IMPORTANT: any changes here must be updated in style_props.go StyleStyleFuncs
// and likewise for all sub-styles as fields here.

// Style has all the CSS-based style elements -- used for widget-type objects
type Style struct {
	Template string `desc:"if present, then this should use unique template name for cached style -- critical for large numbers of repeated widgets in e.g., sliceview, tableview, etc"` // if present, then this should use unique template name for cached style -- critical for large numbers of repeated widgets in e.g., sliceview, tableview, etc

	Display  bool `xml:"display" desc:"todo big enum of how to display item -- controls layout etc"` // todo big enum of how to display item -- controls layout etc
	Visible  bool `xml:"visible" desc:"is the item visible or not"`                                  // is the item visible or not
	Inactive bool `xml:"inactive" desc:"make a control inactive so it does not respond to input"`    // make a control inactive so it does not respond to input

	ZIndex         int         `xml:"z-index" desc:"prop: z-index = ordering factor for rendering depth -- lower numbers rendered first -- sort children according to this factor"`                                                                                                                                                           // prop: z-index = ordering factor for rendering depth -- lower numbers rendered first -- sort children according to this factor
	AlignH         Align       `xml:"horizontal-align" desc:"prop: horizontal-align specifies the horizontal alignment of widget elements within a *vertical* layout container (has no effect within horizontal layouts -- use space / stretch elements instead).  For text layout, use text-align. This is not a standard css property."`    // prop: horizontal-align specifies the horizontal alignment of widget elements within a *vertical* layout container (has no effect within horizontal layouts -- use space / stretch elements instead).  For text layout, use text-align. This is not a standard css property.
	AlignV         Align       `xml:"vertical-align" desc:"prop: vertical-align specifies the vertical alignment of widget elements within a *horizontal* layout container (has no effect within vertical layouts -- use space / stretch elements instead).  For text layout, use text-vertical-align.  This is not a standard css property"` // prop: vertical-align specifies the vertical alignment of widget elements within a *horizontal* layout container (has no effect within vertical layouts -- use space / stretch elements instead).  For text layout, use text-vertical-align.  This is not a standard css property
	PosX           units.Value `xml:"x" desc:"prop: x = horizontal position -- often superseded by layout but otherwise used"`                                                                                                                                                                                                                // prop: x = horizontal position -- often superseded by layout but otherwise used
	PosY           units.Value `xml:"y" desc:"prop: y = vertical position -- often superseded by layout but otherwise used"`                                                                                                                                                                                                                  // prop: y = vertical position -- often superseded by layout but otherwise used
	Width          units.Value `xml:"width" desc:"prop: width = specified size of element -- 0 if not specified"`                                                                                                                                                                                                                             // prop: width = specified size of element -- 0 if not specified
	Height         units.Value `xml:"height" desc:"prop: height = specified size of element -- 0 if not specified"`                                                                                                                                                                                                                           // prop: height = specified size of element -- 0 if not specified
	MaxWidth       units.Value `xml:"max-width" desc:"prop: max-width = specified maximum size of element -- 0  means just use other values, negative means stretch"`                                                                                                                                                                         // prop: max-width = specified maximum size of element -- 0  means just use other values, negative means stretch
	MaxHeight      units.Value `xml:"max-height" desc:"prop: max-height = specified maximum size of element -- 0 means just use other values, negative means stretch"`                                                                                                                                                                        // prop: max-height = specified maximum size of element -- 0 means just use other values, negative means stretch
	MinWidth       units.Value `xml:"min-width" desc:"prop: min-width = specified minimum size of element -- 0 if not specified"`                                                                                                                                                                                                             // prop: min-width = specified minimum size of element -- 0 if not specified
	MinHeight      units.Value `xml:"min-height" desc:"prop: min-height = specified minimum size of element -- 0 if not specified"`                                                                                                                                                                                                           // prop: min-height = specified minimum size of element -- 0 if not specified
	Margin         SideValues  `xml:"margin" desc:"prop: margin = outer-most transparent space around box element -- todo: can be specified per side"`                                                                                                                                                                                        // prop: margin = outer-most transparent space around box element -- todo: can be specified per side
	Padding        SideValues  `xml:"padding" desc:"prop: padding = transparent space around central content of box -- todo: if 4 values it is top, right, bottom, left; 3 is top, right&left, bottom; 2 is top & bottom, right and left"`                                                                                                    // prop: padding = transparent space around central content of box -- todo: if 4 values it is top, right, bottom, left; 3 is top, right&left, bottom; 2 is top & bottom, right and left
	Overflow       Overflow    `xml:"overflow" desc:"prop: overflow = what to do with content that overflows -- default is Auto add of scrollbars as needed -- todo: can have separate -x -y values"`                                                                                                                                         // prop: overflow = what to do with content that overflows -- default is Auto add of scrollbars as needed -- todo: can have separate -x -y values
	Columns        int         `xml:"columns" alt:"grid-cols" desc:"prop: columns = number of columns to use in a grid layout -- used as a constraint in layout if individual elements do not specify their row, column positions"`                                                                                                           // prop: columns = number of columns to use in a grid layout -- used as a constraint in layout if individual elements do not specify their row, column positions
	Row            int         `xml:"row" desc:"prop: row = specifies the row that this element should appear within a grid layout"`                                                                                                                                                                                                          // prop: row = specifies the row that this element should appear within a grid layout
	Col            int         `xml:"col" desc:"prop: col = specifies the column that this element should appear within a grid layout"`                                                                                                                                                                                                       // prop: col = specifies the column that this element should appear within a grid layout
	RowSpan        int         `xml:"row-span" desc:"prop: row-span = specifies the number of sequential rows that this element should occupy within a grid layout (todo: not currently supported)"`                                                                                                                                          // prop: row-span = specifies the number of sequential rows that this element should occupy within a grid layout (todo: not currently supported)
	ColSpan        int         `xml:"col-span" desc:"prop: col-span = specifies the number of sequential columns that this element should occupy within a grid layout"`                                                                                                                                                                       // prop: col-span = specifies the number of sequential columns that this element should occupy within a grid layout
	ScrollBarWidth units.Value `xml:"scrollbar-width" desc:"prop: scrollbar-width = width of a layout scrollbar"`                                                                                                                                                                                                                             // prop: scrollbar-width = width of a layout scrollbar

	Color           Color     `xml:"color" inherit:"true" desc:"prop: color (inherited) = text color -- also defines the currentColor variable value"` // prop: color (inherited) = text color -- also defines the currentColor variable value
	BackgroundColor ColorSpec `xml:"background-color" desc:"prop: background-color = background color -- not inherited, transparent by default"`       // prop: background-color = background color -- not inherited, transparent by default

	Border        Border        `xml:"border" desc:"border around the box element -- todo: can have separate ones for different sides"`                     // border around the box element -- todo: can have separate ones for different sides
	BoxShadow     Shadow        `xml:"box-shadow" desc:"prop: box-shadow = type of shadow to render around box"`                                            // prop: box-shadow = type of shadow to render around box
	Font          Font          `desc:"font parameters -- no xml prefix -- also has color, background-color"`                                               // font parameters -- no xml prefix -- also has color, background-color
	Text          Text          `desc:"text parameters -- no xml prefix"`                                                                                   // text parameters -- no xml prefix
	Outline       Border        `xml:"outline" desc:"prop: outline = draw an outline around an element -- mostly same styles as border -- default to none"` // prop: outline = draw an outline around an element -- mostly same styles as border -- default to none
	PointerEvents bool          `xml:"pointer-events" desc:"prop: pointer-events = does this element respond to pointer events -- default is true"`         // prop: pointer-events = does this element respond to pointer events -- default is true
	UnContext     units.Context `xml:"-" desc:"units context -- parameters necessary for anchoring relative units"`                                         // units context -- parameters necessary for anchoring relative units
	IsSet         bool          `desc:"has this style been set from object values yet?"`                                                                    // has this style been set from object values yet?
	PropsNil      bool          `desc:"set to true if parent node has no props -- allows optimization of styling"`                                          // set to true if parent node has no props -- allows optimization of styling
	dotsSet       bool
	lastUnCtxt    units.Context
}

func (s *Style) Defaults() {
	// mostly all the defaults are 0 initial values, except these..
	s.IsSet = false
	s.UnContext.Defaults()
	s.Outline.Style.Set(BorderNone)
	s.Display = true
	s.PointerEvents = true

	s.LayoutDefaults()
	s.Color = Black
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

// StylePropProps should be set as type props for any enum (not struct types,
// which must have their own props) that is useful as a styling property --
// use this for selecting types to add to Props
var StylePropProps = ki.Props{
	"style-prop": true,
}

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
func SetStylePropsXML(style string, props *ki.Props) {
	st := strings.Split(style, ";")
	for _, s := range st {
		kv := strings.Split(s, ":")
		if len(kv) >= 2 {
			k := strings.TrimSpace(strings.ToLower(kv[0]))
			v := strings.TrimSpace(kv[1])
			if *props == nil {
				*props = make(ki.Props)
			}
			(*props)[k] = v
		}
	}
}

// StylePropsXML returns style props for XML style string, which contains ';'
// separated name: value pairs
func StylePropsXML(props ki.Props) string {
	var sb strings.Builder
	for k, v := range props {
		if k == "transform" {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s:%s;", k, kit.ToString(v)))
	}
	return sb.String()
}

func NewStyle() Style {
	s := Style{}
	s.Defaults()
	return s
}

// CopyFrom copies from another style, while preserving relevant local state
func (s *Style) CopyFrom(cp *Style) {
	is := s.IsSet
	pn := s.PropsNil
	ds := s.dotsSet
	lu := s.lastUnCtxt
	tm := s.Template
	*s = *cp
	s.BackgroundColor = cp.BackgroundColor
	s.IsSet = is
	s.PropsNil = pn
	s.dotsSet = ds
	s.lastUnCtxt = lu
	s.Template = tm
}

// FromTemplate checks if there is a template for this style, returning
// false for hasTemplate if not (in which case usual styling should proceed).
// If there is a template, and it has already been saved, the style is copied
// from the existing template.  If there is a template name set but no
// existing template has yet been saved, then saveTemplate = true and
// the SaveTemplate call should be made on this style after it has gone through
// the usual styling process.
func (s *Style) FromTemplate() (hasTemplate bool, saveTemplate bool) {
	if s.Template == "" {
		return false, false
	}
	if RebuildDefaultStyles {
		return false, true
	}
	StyleTemplatesMu.RLock()
	defer StyleTemplatesMu.RUnlock()
	if ts, has := StyleTemplates[s.Template]; has {
		s.CopyFrom(ts)
		s.IsSet = true
		s.dotsSet = true
		s.PropsNil = ts.PropsNil
		return true, false
	}
	return true, true // need to call save when done
}

// SaveTemplate should only be called for styles that have template
// but none has yet been saved, as determined by FromTemplate call.
func (s *Style) SaveTemplate() {
	ts := &Style{}
	ts.CopyFrom(s)
	ts.lastUnCtxt = s.lastUnCtxt
	ts.PropsNil = s.PropsNil
	StyleTemplatesMu.Lock()
	if StyleTemplates == nil {
		StyleTemplates = make(map[string]*Style)
	}
	StyleTemplates[s.Template] = ts
	StyleTemplatesMu.Unlock()
}

// InheritFields from parent: Manual inheriting of values is much faster than
// automatic version!
func (s *Style) InheritFields(par *Style) {
	// fmt.Println("Inheriting from", *par)
	s.Color = par.Color
	s.Font.InheritFields(&par.Font)
	s.Text.InheritFields(&par.Text)
}

// SetStyleProps sets style values based on given property map (name: value pairs),
// inheriting elements as appropriate from parent
func (s *Style) SetStyleProps(par *Style, props ki.Props, ctxt Context) {
	if !s.IsSet && par != nil { // first time
		s.InheritFields(par)
	}
	s.StyleFromProps(par, props, ctxt)
	if s.Margin.Bottom.Dots > 0 && s.Text.ParaSpacing.Val == 0 {
		s.Text.ParaSpacing = s.Margin.Bottom
	}
	s.LayoutSetStylePost(props)
	s.Font.SetStylePost(props)
	s.Text.SetStylePost(props)
	s.PropsNil = (len(props) == 0)
	s.IsSet = true
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
	s.BoxShadow.ToDots(uc)
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
	return NewSideFloats(
		s.Margin.Top.Dots+s.Border.Width.Top.Dots+s.Padding.Top.Dots,
		s.Margin.Right.Dots+s.Border.Width.Right.Dots+s.Padding.Right.Dots,
		s.Margin.Bottom.Dots+s.Border.Width.Bottom.Dots+s.Padding.Bottom.Dots,
		s.Margin.Left.Dots+s.Border.Width.Left.Dots+s.Padding.Left.Dots,
	)
}

// SubProps returns a sub-property map from given prop map for a given styling
// selector (property name) -- e.g., :normal :active :hover etc -- returns
// false if not found
func SubProps(prp ki.Props, selector string) (ki.Props, bool) {
	sp, ok := prp[selector]
	if !ok {
		return nil, false
	}
	spm, ok := sp.(ki.Props)
	if ok {
		return spm, true
	}
	log.Printf("gi.SubProps: looking for a ki.Props for style selector: %v, instead got type: %T\n", selector, spm)
	return nil, false
}

// StyleDefault is default style can be used when property specifies "default"
var StyleDefault Style
