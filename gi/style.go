// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log"
	"strings"
	"sync"

	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/prof"
)

// style implements CSS-based styling using ki.Props to hold name / vals
// CSS style reference: https://www.w3schools.com/cssref/default.asp
// list of inherited: https://stackoverflow.com/questions/5612302/which-css-properties-are-inherited

// styling strategy:
// * indiv objects specify styles using property map -- good b/c it is fully open-ended
// * we process those properties dynamically when rendering (first pass only) into state
//   on objects that can be directly used during rendering
// * good for basic rendering -- lots of additional things that could be extended later..

// styleTemplates are cached styles used for styling large numbers of identical
// elements in views
var styleTemplates map[string]*Style

// styleTemplatesMu is a mutex protecting updates to styleTemplates
var styleTemplatesMu sync.RWMutex

// IMPORTANT: any changes here must be updated in stylefuncs.go StyleStyleFuncs
// and likewise for all sub-styles as fields here.

// Style has all the CSS-based style elements -- used for widget-type objects
type Style struct {
	Template      string        `desc:"if present, then this should use unique template name for cached style -- critical for large numbers of repeated widgets in e.g., sliceview, tableview, etc"`
	Display       bool          `xml:"display" desc:"todo big enum of how to display item -- controls layout etc"`
	Visible       bool          `xml:"visible" desc:"is the item visible or not"`
	Inactive      bool          `xml:"inactive" desc:"make a control inactive so it does not respond to input"`
	Layout        LayoutStyle   `desc:"layout styles -- do not prefix with any xml"`
	Border        BorderStyle   `xml:"border" desc:"border around the box element -- todo: can have separate ones for different sides"`
	BoxShadow     ShadowStyle   `xml:"box-shadow" desc:"prop: box-shadow = type of shadow to render around box"`
	Font          FontStyle     `desc:"font parameters -- no xml prefix -- also has color, background-color"`
	Text          TextStyle     `desc:"text parameters -- no xml prefix"`
	Outline       BorderStyle   `xml:"outline" desc:"prop: outline = draw an outline around an element -- mostly same styles as border -- default to none"`
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

// LayoutStyle is in layoutstyles.go
// FontStyle is in font.go
// TextStyle is in textstyles.go
// Border, BoxShadow, Outline all in boxstyles.go

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
	styleTemplatesMu.RLock()
	defer styleTemplatesMu.RUnlock()
	if ts, has := styleTemplates[s.Template]; has {
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
	styleTemplatesMu.Lock()
	if styleTemplates == nil {
		styleTemplates = make(map[string]*Style)
	}
	styleTemplates[s.Template] = ts
	styleTemplatesMu.Unlock()
}

// InheritFields from parent: Manual inheriting of values is much faster than
// automatic version!
func (s *Style) InheritFields(par *Style) {
	s.Font.InheritFields(&par.Font)
	s.Text.InheritFields(&par.Text)
}

// SetStyleProps sets style values based on given property map (name: value pairs),
// inheriting elements as appropriate from parent
func (s *Style) SetStyleProps(par *Style, props ki.Props, vp *Viewport2D) {
	if !s.IsSet && par != nil { // first time
		// StyleFields.Inherit(s, par) // very slow for some mysterious reason
		s.InheritFields(par)
	}
	// StyleFields.Style(s, par, props, vp)
	s.StyleFromProps(par, props, vp)
	s.Text.AlignV = s.Layout.AlignV
	if s.Layout.Margin.Val > 0 && s.Text.ParaSpacing.Val == 0 {
		s.Text.ParaSpacing = s.Layout.Margin
	}
	s.Layout.SetStylePost(props)
	s.Font.SetStylePost(props)
	s.Text.SetStylePost(props)
	s.PropsNil = (len(props) == 0)
	s.IsSet = true
}

// Use activates the style settings in this style for any general settings and
// parameters -- typically specific style values are used directly in
// particular rendering steps, but some settings also impact global variables,
// such as CurrentColor -- this is automatically called for a successful
// PushBounds in Node2DBase
func (s *Style) Use(vp *Viewport2D) {
	vp.SetCurrentColor(s.Font.Color)
}

// SetUnitContext sets the unit context based on size of viewport and parent
// element (from bbox) and then cache everything out in terms of raw pixel
// dots for rendering -- call at start of render
func (s *Style) SetUnitContext(vp *Viewport2D, el Vec2D) {
	pr := prof.Start("Style.SetUnitContext")
	defer pr.End()
	// s.UnContext.Defaults()
	if vp != nil {
		if vp.Win != nil {
			s.UnContext.DPI = vp.Win.LogicalDPI()
			// fmt.Printf("set dpi: %v\n", s.UnContext.DPI)
			// } else {
			// 	fmt.Printf("No win for vp: %v\n", vp.PathUnique())
		}
		if vp.Render.Image != nil {
			sz := vp.Render.Image.Bounds().Size()
			s.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y)
		}
	}
	// todo font.computemetrics here is slow element here -- use caching..
	prf := prof.Start("Style.OpenFont")
	s.Font.OpenFont(&s.UnContext) // calls SetUnContext after updating metrics
	prf.End()

	// skipping this doesn't seem to be good:
	// if !(s.dotsSet && s.UnContext == s.lastUnCtxt && s.PropsNil) {
	// fmt.Printf("dotsSet: %v unctx: %v lst: %v  == %v  pn: %v\n", s.dotsSet, s.UnContext, s.lastUnCtxt, (s.UnContext == s.lastUnCtxt), s.PropsNil)
	prd := prof.Start("Style.ToDots")
	s.ToDots(&s.UnContext)
	prd.End()
	s.dotsSet = true
	s.lastUnCtxt = s.UnContext
	// } else {
	// 	fmt.Println("skipped")
	// }
}

// CopyUnitContext copies unit context from another, update with our font
// info, and then cache everything out in terms of raw pixel dots for
// rendering -- call at start of render
func (s *Style) CopyUnitContext(ctxt *units.Context) {
	s.UnContext = *ctxt
	// s.Font.SetUnitContext(&s.UnContext)
	// this seems to work fine
	if !(s.dotsSet && s.UnContext == s.lastUnCtxt && s.PropsNil) {
		s.ToDots(&s.UnContext)
		s.dotsSet = true
		s.lastUnCtxt = s.UnContext
		// } else {
		// 	fmt.Println("skipped")
	}
}

// BoxSpace returns extra space around the central content in the box model,
// in dots -- todo: must complicate this if we want different spacing on
// different sides box outside-in: margin | border | padding | content
func (s *Style) BoxSpace() float32 {
	return s.Layout.Margin.Dots + s.Border.Width.Dots + s.Layout.Padding.Dots
}

// ApplyCSS applies css styles for given node, using key to select sub-props
// from overall properties list, and optional selector to select a further
// :name selector within that key
func (s *Style) ApplyCSS(node Node2D, css ki.Props, key, selector string, vp *Viewport2D) bool {
	pp, got := css[key]
	if !got {
		return false
	}
	pmap, ok := pp.(ki.Props) // must be a props map
	if !ok {
		return false
	}
	if selector != "" {
		pmap, ok = SubProps(pmap, selector)
		if !ok {
			return false
		}
	}
	parSty := node.AsNode2D().ParentStyle()
	s.SetStyleProps(parSty, pmap, vp)
	return true
}

// StyleCSS applies css style properties to given Widget node, parsing out
// type, .class, and #name selectors, along with optional sub-selector
// (:hover, :active etc)
func (s *Style) StyleCSS(node Node2D, css ki.Props, selector string, vp *Viewport2D) {
	pr := prof.Start("StyleCSS")
	tyn := strings.ToLower(node.Type().Name()) // type is most general, first
	s.ApplyCSS(node, css, tyn, selector, vp)
	classes := strings.Split(strings.ToLower(node.AsNode2D().Class), " ")
	for _, cl := range classes {
		cln := "." + strings.TrimSpace(cl)
		s.ApplyCSS(node, css, cln, selector, vp)
	}
	idnm := "#" + strings.ToLower(node.Name()) // then name
	s.ApplyCSS(node, css, idnm, selector, vp)
	pr.End()
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
