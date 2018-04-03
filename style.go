// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image/color"
	"log"
	"reflect"
	"strings"

	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
// Widget Styling

// using CSS style reference: https://www.w3schools.com/cssref/default.asp
// which are inherited: https://stackoverflow.com/questions/5612302/which-css-properties-are-inherited

// styling strategy:
// * indiv objects specify styles using property map -- good b/c it is fully open-ended
// * we process those properties dynamically when rendering (first pass only) into state
//   on objects that can be directly used during rendering
// * good for basic rendering -- lots of additional things that could be extended later..
// * todo: could we generalize this to not have to write the parsing code?  YES need to
//
// SVG Paint inheritance is probably NOT such a good idea for widgets??  fill = background?
// may need to figure that part out a bit more..

// todo: Animation

// Bottom = alignment too

// Clear -- no floating elements

// Clip -- clip images

// column- settings -- lots of those

// LayoutStyle is in layout.go
// FontStyle is in font.go
// TextStyle is in text.go

// List-style for lists

// Object-fit for videos

// visibility -- support more than just hidden  inherit:"true"

// Transform -- can throw in any 2D or 3D transform!  we support that!  sort of..

// transition -- animation of hover, etc

// style parameters for backgrounds
type BackgroundStyle struct {
	Color Color `xml:"color" desc:"background color"`
	// todo: all the properties not yet implemented -- mostly about images
	// Image is like a PaintServer -- includes gradients etc
	// Attachment -- how the image moves
	// Clip -- how to clip the image
	// Origin
	// Position
	// Repeat
	// Size
}

func (b *BackgroundStyle) Defaults() {
	b.Color.SetColor(color.White)
}

// sides of a box -- some properties can be specified per each side (e.g., border) or not
type BoxSides int32

const (
	BoxTop BoxSides = iota
	BoxRight
	BoxBottom
	BoxLeft
	BoxN
)

//go:generate stringer -type=BoxSides

var KiT_BoxSides = kit.Enums.AddEnumAltLower(BoxN, false, nil, "Box")

// how to draw the border
type BorderDrawStyle int32

const (
	BorderSolid BorderDrawStyle = iota
	BorderDotted
	BorderDashed
	BorderDouble
	BorderGroove
	BorderRidge
	BorderInset
	BorderOutset
	BorderNone
	BorderHidden
	BorderN
)

//go:generate stringer -type=BorderDrawStyle

var KiT_BorderDrawStyle = kit.Enums.AddEnumAltLower(BorderN, false, nil, "Border")

// style parameters for borders
type BorderStyle struct {
	Style  BorderDrawStyle `xml:"style" desc:"how to draw the border"`
	Width  units.Value     `xml:"width" desc:"width of the border"`
	Radius units.Value     `xml:"radius" desc:"rounding of the corners"`
	Color  Color           `xml:"color" desc:"color of the border"`
}

// style parameters for shadows
type ShadowStyle struct {
	HOffset units.Value `xml:".h-offset" desc:"horizontal offset of shadow -- positive = right side, negative = left side"`
	VOffset units.Value `xml:".v-offset" desc:"vertical offset of shadow -- positive = below, negative = above"`
	Blur    units.Value `xml:".blur" desc:"blur radius -- higher numbers = more blurry"`
	Spread  units.Value `xml:".spread" desc:"spread radius -- positive number increases size of shadow, negative descreases size"`
	Color   Color       `xml:".color" desc:"color of the shadow"`
	Inset   bool        `xml:".inset" desc:"shadow is inset within box instead of outset outside of box"`
}

func (s *ShadowStyle) HasShadow() bool {
	return (s.HOffset.Dots > 0 || s.VOffset.Dots > 0)
}

// all the CSS-based style elements -- used for widget-type objects
type Style struct {
	IsSet         bool            `desc:"has this style been set from object values yet?"`
	Display       bool            `xml:display" desc:"todo big enum of how to display item -- controls layout etc"`
	Visible       bool            `xml:visible" desc:"todo big enum of how to display item -- controls layout etc"`
	UnContext     units.Context   `desc:"units context -- parameters necessary for anchoring relative units"`
	Layout        LayoutStyle     `desc:"layout styles -- do not prefix with any xml"`
	Border        BorderStyle     `xml:"border" desc:"border around the box element -- todo: can have separate ones for different sides"`
	BoxShadow     ShadowStyle     `xml:"box-shadow" desc:"type of shadow to render around box"`
	Font          FontStyle       `xml:"font" desc:"font parameters"`
	Text          TextStyle       `desc:"text parameters -- no xml prefix"`
	Color         Color           `xml:"color" inherit:"true" desc:"text color"`
	Background    BackgroundStyle `xml:"background" desc:"background settings"`
	Opacity       float64         `xml:"opacity" desc:"alpha value to apply to all elements"`
	Outline       BorderStyle     `xml:"outline" desc:"draw an outline around an element -- mostly same styles as border -- default to none"`
	PointerEvents bool            `xml:"pointer-events" desc:"does this element respond to pointer events -- default is true"`
	// todo: also see above for more notes on missing style elements
}

func (s *Style) Defaults() {
	// mostly all the defaults are 0 initial values, except these..
	s.IsSet = false
	s.UnContext.Defaults()
	s.Opacity = 1.0
	s.Outline.Style = BorderNone
	s.PointerEvents = true
	s.Color.SetColor(color.Black)
	s.Background.Defaults()
	s.Layout.Defaults()
	s.Font.Defaults()
	s.Text.Defaults()
}

func NewStyle() Style {
	s := Style{}
	s.Defaults()
	return s
}

// default style can be used when property specifies "default"
var StyleDefault = NewStyle()

// set style values based on given property map (name: value pairs), inheriting elements as appropriate from parent, and also having a default style for the "initial" setting
func (s *Style) SetStyle(parent, defs *Style, props map[string]interface{}) {
	// nil interface is special and != interface{} of a nil ptr!
	pfi := interface{}(nil)
	dfi := interface{}(nil)
	if parent != nil {
		pfi = interface{}(parent)
	}
	if defs != nil {
		dfi = interface{}(defs)
	}
	inherit := !s.IsSet // we only inherit if not already set
	WalkStyleStruct(s, pfi, dfi, "", inherit, props, StyleField)
	s.Layout.SetStylePost()
	s.Font.SetStylePost()
	s.Text.SetStylePost()
	s.IsSet = true
}

// set the unit context based on size of viewport and parent element (from bbox)
// and then cache everything out in terms of raw pixel dots for rendering -- call at start of
// render
func (s *Style) SetUnitContext(vp *Viewport2D, el Vec2D) {
	s.UnContext.Defaults() // todo: need to get screen information and true dpi
	if vp != nil && vp.Render.Image != nil {
		sz := vp.Render.Image.Bounds().Size()
		s.UnContext.SetSizes(float64(sz.X), float64(sz.Y), el.X, el.Y)
	}
	s.Font.SetUnitContext(&s.UnContext)
	s.ToDots()
}

// copy units context from another, update with our font info, and then cache
// everything out in terms of raw pixel dots for rendering -- call at start of
// render
func (s *Style) CopyUnitContext(ctxt *units.Context) {
	s.UnContext = *ctxt
	s.Font.SetUnitContext(&s.UnContext)
	s.ToDots()
}

// call ToDots on all units.Value fields in the style (recursively) -- need to have set the
// UnContext first -- only after layout at render time is that possible
func (s *Style) ToDots() {
	valtyp := reflect.TypeOf(units.Value{})

	WalkStyleStruct(s, nil, nil, "", false, nil,
		func(sf reflect.StructField, vf, pf, df reflect.Value,
			hasPar bool, tag string, inherit bool, props map[string]interface{}) {
			if vf.Kind() == reflect.Struct && vf.Type() == valtyp {
				uv := vf.Addr().Interface().(*units.Value)
				uv.ToDots(&s.UnContext)
			}
		})
}

// extra space around the central content in the box model, in dots -- todo:
// must complicate this if we want different spacing on different sides
// box outside-in: margin | border | padding | content
func (s *Style) BoxSpace() float64 {
	return s.Layout.Margin.Dots + s.Border.Width.Dots + s.Layout.Padding.Dots
}

////////////////////////////////////////////////////////////////////////////////////////
//   Style processing util

// this is the function to process a given field when walking the style
type WalkStyleFieldFun func(sf reflect.StructField, vf, pf, df reflect.Value, hasPar bool, tag string, inherit bool, props map[string]interface{})

// general-purpose function for walking through style structures and calling fun on each field with a valid 'xml' tag
func WalkStyleStruct(obj interface{}, parent interface{}, defs interface{}, outerTag string,
	inherit bool, props map[string]interface{}, fun WalkStyleFieldFun) {
	otp := reflect.TypeOf(obj)
	if otp.Kind() != reflect.Ptr {
		log.Printf("gi.StyleStruct -- you must pass pointers to the structs, not type: %v kind %v\n", otp, otp.Kind())
		return
	}
	ot := otp.Elem()
	if ot.Kind() != reflect.Struct {
		log.Printf("gi.StyleStruct -- only works on structs, not type: %v kind %v\n", ot, ot.Kind())
		return
	}
	var pt reflect.Type
	if parent != nil {
		pt = reflect.TypeOf(parent).Elem()
		if pt != ot {
			log.Printf("gi.StyleStruct -- inheritance only works for objs of same type: %v != %v\n", ot, pt)
			parent = nil
		}
	}
	vo := reflect.ValueOf(obj).Elem()
	for i := 0; i < ot.NumField(); i++ {
		sf := ot.Field(i)
		if sf.PkgPath != "" { // skip unexported fields
			continue
		}
		tag := sf.Tag.Get("xml")
		if tag == "-" {
			continue
		}
		ft := sf.Type
		// note: need Addrs() to pass pointers to fields, not fields themselves
		// fmt.Printf("processing field named: %v\n", sf.Nm)
		vf := vo.Field(i)
		vfi := vf.Addr().Interface()
		var pf reflect.Value
		var df reflect.Value
		pfi := interface{}(nil)
		dfi := interface{}(nil)
		if parent != nil {
			pf = reflect.ValueOf(parent).Elem().Field(i)
			pfi = pf.Addr().Interface()
		}
		if defs != nil {
			df = reflect.ValueOf(defs).Elem().Field(i)
			dfi = df.Addr().Interface()
		}
		if ft.Kind() == reflect.Struct && ft.Name() != "Value" && ft.Name() != "Color" {
			WalkStyleStruct(vfi, pfi, dfi, tag, inherit, props, fun)
		} else {
			if tag == "" { // non-struct = don't process
				continue
			}
			fun(sf, vf, pf, df, parent != nil, outerTag, inherit, props)
		}
	}
}

// todo:
// * need to be able to process entire chunks at a time: box-shadow: val val val

// get the full effective tag based on outer tag plus given tag
func StyleEffTag(tag, outerTag string) string {
	tagEff := tag
	if outerTag != "" && len(tag) > 0 {
		if tag[0] == '.' {
			tagEff = outerTag + tag
		} else {
			tagEff = outerTag + "-" + tag
		}
	}
	return tagEff
}

// process the field tags for any that fit with values in props, trying xml and xml_alt options
func StyleFieldProps(sf reflect.StructField, outerTag string, props map[string]interface{}) (interface{}, bool) {
	tag := StyleEffTag(sf.Tag.Get("xml"), outerTag)
	prv, got := props[tag]
	if got {
		return prv, got
	}
	atags := sf.Tag.Get("alt")
	if atags == "" {
		return nil, false
	}
	atag := strings.Split(atags, ",")

	for _, tg := range atag {
		tag = StyleEffTag(tg, outerTag)
		prv, got = props[tag]
		if got {
			// fmt.Printf("Got alt tag: %v for field %v\n", tag, sf.Name)
			return prv, true
			// } else {
			// 	fmt.Printf("Did NOT get xml_alt tag: %v for field %v\n", tag, sf.Name)
		}
	}
	return nil, false
}

// standard field processing function for WalkStyleStruct
func StyleField(sf reflect.StructField, vf, pf, df reflect.Value, hasPar bool, outerTag string, inherit bool, props map[string]interface{}) {

	if inherit { // first process inherit flag -- only inherit on FIRST pass!!
		inhs := sf.Tag.Get("inherit")
		if inhs == "true" {
			if hasPar {
				// fmt.Printf("StyleField %v inherited property: %v\n", sf.Name, pf)
				vf.Set(pf) // copy
			}
		} else if inhs != "" && inhs != "false" {
			log.Printf("gi.StyleField -- bad inherit tag -- can only be true or false: %v\n", inhs)
		}
	}
	// fmt.Printf("StyleField %v tag: %v\n", vf, tag)

	prv, got := StyleFieldProps(sf, outerTag, props)
	if !got {
		// fmt.Printf("StyleField didn't find tag: %v\n", tag)
		return
	}
	// fmt.Printf("StyleField %v got value %v\n", sf.Name, prv)

	prstr := ""
	switch prtv := prv.(type) {
	case string:
		prstr = prtv
		if prtv == "inherit" && hasPar {
			vf.Set(pf)
			// fmt.Printf("StyleField %v set to inherited value: %v\n", sf.Name, pf)
			return
		}
		if prtv == "initial" && hasPar {
			vf.Set(df)
			// fmt.Printf("StyleField set tag: %v to initial default value: %v\n", tag, df)
			return
		}
	}

	// todo: support keywords such as auto, normal, which should just set to 0

	vk := vf.Kind()
	vt := vf.Type()

	if vk == reflect.Struct { // only a few types
		if vt == reflect.TypeOf(Color{}) {
			vc := vf.Addr().Interface().(*Color)
			err := vc.SetFromString(prstr)
			if err != nil {
				log.Printf("StyleField: %v\n", err)
			}
			return
		} else if vt == reflect.TypeOf(units.Value{}) {
			uv := vf.Addr().Interface().(*units.Value)
			switch prtv := prv.(type) {
			case string:
				uv.SetFromString(prtv)
			case units.Value:
				*uv = prtv
			default: // assume Px as an implicit default
				prvflt := reflect.ValueOf(prv).Convert(reflect.TypeOf(0.0)).Interface().(float64)
				uv.Set(prvflt, units.Px)
			}
			return
		}
		return // no can do any struct otherwise
	} else if vk >= reflect.Int && vk <= reflect.Uint64 { // some kind of int
		// fmt.Printf("int field: %v, type: %v\n", sf.Name, sf.Type.Name())
		if prstr != "" {
			tn := kit.FullTypeName(sf.Type)
			if kit.Enums.FindEnum(tn) != nil {
				kit.Enums.SetEnumValueFromAltString(vf, prstr)
			} else {
				fmt.Printf("gi.StyleField: enum name not found %v for field %v\n", tn, sf.Name)
			}
			return
		} else {
			// note: cannot use convert on const values apparently.. but this works
			vf.Set(reflect.ValueOf(prv))
			return
		}
	}

	// otherwise just set directly based on type, using standard conversions
	vf.Set(reflect.ValueOf(prv).Convert(reflect.TypeOf(vt)))
}

// manual method for getting a units value directly
func StyleUnitsValue(tag string, uv *units.Value, props map[string]interface{}) bool {
	prv, got := props[tag]
	if !got {
		return false
	}
	switch v := prv.(type) {
	case string:
		uv.SetFromString(v)
	case float64:
		uv.Set(v, units.Px) // assume px
	case float32:
		uv.Set(float64(v), units.Px) // assume px
	case int:
		uv.Set(float64(v), units.Px) // assume px
	}
	return true
}
