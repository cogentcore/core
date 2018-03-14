// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	// "github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/gi/units"
	"log"
	"reflect"
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

// visibility -- support more than just hidden ,inherit:"true"

// Transform -- can throw in any 2D or 3D transform!  we support that!  sort of..

// transition -- animation of hover, etc

// style parameters for backgrounds
type BackgroundStyle struct {
	Color Color `xml:"color",desc:"background color"`
	// todo: all the properties not yet implemented -- mostly about images
	// Image is like a PaintServer -- includes gradients etc
	// Attachment -- how the image moves
	// Clip -- how to clip the image
	// Origin
	// Position
	// Repeat
	// Size
}

// sides of a box -- some properties can be specified per each side (e.g., border) or not
type BoxSides int32

const (
	BoxTop BoxSides = iota
	BoxRight
	BoxBottom
	BoxLeft
)

//go:generate stringer -type=BoxSides

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
)

//go:generate stringer -type=BorderDrawStyle

// style parameters for shadows
type ShadowStyle struct {
	HOffset units.Value `xml:".h-offset",desc:"horizontal offset of shadow -- positive = right side, negative = left side"`
	VOffset units.Value `xml:".v-offset",desc:"vertical offset of shadow -- positive = below, negative = above"`
	Blur    units.Value `xml:".blur",desc:"blur radius -- higher numbers = more blurry"`
	Spread  units.Value `xml:".spread",desc:"spread radius -- positive number increases size of shadow, negative descreases size"`
	Color   Color       `xml:".color",desc:"color of the shadow"`
	Inset   bool        `xml:".inset",desc:"shadow is inset within box instead of outset outside of box"`
}

// style parameters for borders
type BorderStyle struct {
	Style  BorderDrawStyle `xml:"style",desc:"how to draw the border"`
	Width  units.Value     `xml:"width",desc:"width of the border"`
	Radius units.Value     `xml:"radius",desc:"rounding of the corners"`
	Color  Color           `xml:"color",desc:"color of the border"`
}

// all the CSS-based style elements
type Style struct {
	IsSet         bool            `desc:"has this style been set from object values yet?"`
	Layout        LayoutStyle     `desc:"layout styles -- do not prefix with any xml"`
	Border        []BorderStyle   `xml:"border",desc:"border around the box element -- can have separate ones for different sides"`
	Shadow        ShadowStyle     `xml:"box-shadow",desc:"type of shadow to render around box"`
	Padding       []units.Value   `xml:"padding",desc:"transparent space around central content of box -- if 4 values it is top, right, bottom, left; 3 is top, right&left, bottom; 2 is top & bottom, right and left"`
	Font          FontStyle       `xml:"font",desc:"font parameters"`
	Text          TextStyle       `desc:"text parameters -- no xml prefix"`
	Color         Color           `xml:"color",inherit:"true",desc:"text color"`
	Background    BackgroundStyle `xml:"background",desc:"background settings"`
	Opacity       float64         `xml:"opacity",desc:"alpha value to apply to all elements"`
	Outline       BorderStyle     `xml:"outline",desc:"draw an outline around an element -- mostly same styles as border -- default to none"`
	PointerEvents bool            `xml:"pointer-events",desc:"does this element respond to pointer events -- default is true"`
	// todo: also see above for more notes on missing style elements
	// Display    bool            `xml:display",desc:"big enum of how to display item -- controls layout etc"`
}

func (s *Style) Defaults() {
	// mostly all the defaults are 0 initial values, except these..
	s.Opacity = 1.0
	s.Outline.Style = BorderNone
	s.PointerEvents = true
}

// todo: css parser just needs to consolidate the css into a final set of props

// this will recursively style obj (must be a *pointer to* struct), inheriting elements as appropriate from parent, and also having a default style for the "initial" seting (all must be pointers) -- based on property map (name: value pairs)
func StyleStruct(obj interface{}, parent interface{}, defs interface{}, props map[string]interface{}, outerTag string) {
	otp := reflect.TypeOf(obj)
	if otp.Kind() != reflect.Ptr {
		log.Printf("gi.StyleStruct -- you must pass pointers to the structs, not type: %v kind %v\n", otp, otp.Kind())
		return
	}
	ot := reflect.TypeOf(obj).Elem()
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
		tag := sf.Tag.Get("xml")
		if tag == "-" {
			continue
		}
		tagEff := tag
		if outerTag != "" && len(tag) > 0 {
			if tag[0] == '.' {
				tagEff = outerTag + tag
			} else {
				tagEff = outerTag + "-" + tag
			}
		}
		ft := sf.Type
		// note: need Addrs() to pass pointers to fields, not fields themselves
		// fmt.Printf("processing field named: %v\n", sf.Name)
		vf := vo.Field(i)
		pf := reflect.Value{}
		df := reflect.ValueOf(defs).Elem().Field(i)
		if parent != nil {
			pf = reflect.ValueOf(parent).Elem().Field(i)
		}
		if ft.Kind() == reflect.Struct && ft.Name() != "Value" && ft.Name() != "Color" {
			if parent != nil {
				pf = reflect.ValueOf(parent).Elem().Field(i)
				StyleStruct(vf.Addr().Interface(), pf.Addr().Interface(), df.Addr().Interface(), props, tag)
			} else {
				// fmt.Printf("StyleField Descending into struct type: %v on field %v tag: %v\n", ft.Name(), sf, tag)
				StyleStruct(vf.Addr().Interface(), nil, df.Addr().Interface(), props, tag)
			}
		} else {
			if tag == "" { // non-struct = don't process
				continue
			}
			inh := false
			inhs := sf.Tag.Get("inherit")
			if inhs == "true" {
				inh = true
			} else if inhs != "" && inhs != "false" {
				log.Printf("gi.StyleStruct -- bad inherit tag -- can only be true or false: %v\n", inhs)
			}
			if parent != nil {
				if inh {
					vf.Set(pf) // copy
				}
				StyleField(vf, pf, df, true, tagEff, props)
			} else {
				StyleField(vf, pf, df, false, tagEff, props)
			}
		}
	}
}

// todo:
// * need to be able to process entire chunks at a time: box-shadow: val val val
// * deal with slices -- border
// * deal with enums!

func StyleField(vf reflect.Value, pf reflect.Value, df reflect.Value, hasPar bool, tag string, props map[string]interface{}) {
	// fmt.Printf("StyleField %v tag: %v\n", vf, tag)
	k := ""
	prv := interface{}(nil)
	got := false
	for k, prv = range props {
		if k == tag {
			got = true
			break
		}
	}
	if !got {
		// fmt.Printf("StyleField didn't find tag: %v\n", tag)
		return
	}
	fmt.Printf("StyleField got tag: %v, value %v\n", tag, prv)

	prstr := ""
	switch prtv := prv.(type) {
	case string:
		prstr = prtv
		if prtv == "inherit" && hasPar {
			vf.Set(pf)
			fmt.Printf("StyleField set tag: %v to inherited value: %v\n", tag, pf)
			return
		}
		if prtv == "initial" && hasPar {
			vf.Set(df)
			fmt.Printf("StyleField set tag: %v to initial default value: %v\n", tag, df)
			return
		}
	}

	if vf.Kind() == reflect.Struct { // only a few types
		if vf.Type() == reflect.TypeOf(Color{}) {
			vc := vf.Addr().Interface().(*Color)
			err := vc.FromString(prstr)
			if err != nil {
				log.Printf("StyleField: %v\n", err)
			}
			return
		} else if vf.Type() == reflect.TypeOf(units.Value{}) {
			vc := vf.Addr().Interface().(*units.Value)
			if prstr != "" {
				*vc = units.StringToValue(prstr)
			} else { // assume Px as an implicit default
				prvflt := reflect.ValueOf(prv).Convert(reflect.TypeOf(0.0)).Interface().(float64)
				*vc = units.Value{prvflt, units.Px}
			}
			return
		}
	}

	switch vf.Interface().(type) {
	case string:
		vf.Set(reflect.ValueOf(prv).Convert(reflect.TypeOf("")))
	case float64:
		vf.Set(reflect.ValueOf(prv).Convert(reflect.TypeOf(0.0)))
	}
}
