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
	Template      string        `desc:"if present, then this should use unique template name for cached style -- critical for large numbers of repeated widgets in e.g., sliceview, tableview, etc"`
	Display       bool          `xml:"display" desc:"todo big enum of how to display item -- controls layout etc"`
	Visible       bool          `xml:"visible" desc:"is the item visible or not"`
	Inactive      bool          `xml:"inactive" desc:"make a control inactive so it does not respond to input"`
	Layout        Layout        `desc:"layout styles -- do not prefix with any xml"`
	Border        Border        `xml:"border" desc:"border around the box element -- todo: can have separate ones for different sides"`
	BoxShadow     Shadow        `xml:"box-shadow" desc:"prop: box-shadow = type of shadow to render around box"`
	Font          Font          `desc:"font parameters -- no xml prefix -- also has color, background-color"`
	Text          Text          `desc:"text parameters -- no xml prefix"`
	Outline       Border        `xml:"outline" desc:"prop: outline = draw an outline around an element -- mostly same styles as border -- default to none"`
	PointerEvents bool          `xml:"pointer-events" desc:"prop: pointer-events = does this element respond to pointer events -- default is true"`
	UnContext     units.Context `xml:"-" desc:"units context -- parameters necessary for anchoring relative units"`
	IsSet         bool          `desc:"has this style been set from object values yet?"`
	PropsNil      bool          `desc:"set to true if parent node has no props -- allows optimization of styling"`
	dotsSet       bool
	lastUnCtxt    units.Context
}

func (s *Style) Defaults() {
	// mostly all the defaults are 0 initial values, except these..
	s.IsSet = false
	s.UnContext.Defaults()
	s.Outline.Style = BorderNone
	s.Display = true
	s.PointerEvents = true
	s.Layout.Defaults()
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

// Styler defines an interface for anything that has a Style on it
type Styler interface {
	Style() *Style

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
	s.Font.BgColor = cp.Font.BgColor
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
	if s.Layout.Margin.Val > 0 && s.Text.ParaSpacing.Val == 0 {
		s.Text.ParaSpacing = s.Layout.Margin
	}
	s.Layout.SetStylePost(props)
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
	s.Layout.ToDots(uc)
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
func (s *Style) BoxSpace() float32 {
	return s.Layout.Margin.Dots + s.Border.Width.Dots + s.Layout.Padding.Dots
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
