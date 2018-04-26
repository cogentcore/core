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
	"unsafe"

	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
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

// use StylePropProps for any enum (not type -- types must have their own
// props) that is useful as a styling property -- use this for selecting types
// to add to Props
var StylePropProps = ki.Props{
	"style-prop": true,
}

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

var KiT_BoxSides = kit.Enums.AddEnumAltLower(BoxN, false, StylePropProps, "Box")

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

var KiT_BorderDrawStyle = kit.Enums.AddEnumAltLower(BorderN, false, StylePropProps, "Border")

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
	UnContext     units.Context   `xml:"-" desc:"units context -- parameters necessary for anchoring relative units"`
	Layout        LayoutStyle     `desc:"layout styles -- do not prefix with any xml"`
	Border        BorderStyle     `xml:"border" desc:"border around the box element -- todo: can have separate ones for different sides"`
	BoxShadow     ShadowStyle     `xml:"box-shadow" desc:"type of shadow to render around box"`
	Font          FontStyle       `xml:"font" desc:"font parameters"`
	Text          TextStyle       `desc:"text parameters -- no xml prefix"`
	Color         Color           `xml:"color" inherit:"true" desc:"text color"`
	Background    BackgroundStyle `xml:"background" desc:"background settings"`
	Opacity       float32         `xml:"opacity" desc:"alpha value to apply to all elements"`
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

// SetStyle sets style values based on given property map (name: value pairs),
// inheriting elements as appropriate from parent
func (s *Style) SetStyle(parent *Style, props ki.Props) {
	if !s.IsSet && parent != nil { // first time
		StyleFieldsVar.Inherit(s, parent)
	}
	StyleFieldsVar.SetFromProps(s, parent, props)
	s.Text.AlignV = s.Layout.AlignV
	s.Layout.SetStylePost()
	s.Font.SetStylePost()
	s.Text.SetStylePost()
	s.IsSet = true
}

// SetUnitContext sets the unit context based on size of viewport and parent
// element (from bbox) and then cache everything out in terms of raw pixel
// dots for rendering -- call at start of render
func (s *Style) SetUnitContext(vp *Viewport2D, el Vec2D) {
	s.UnContext.Defaults() // todo: need to get screen information and true dpi
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
	s.Font.SetUnitContext(&s.UnContext)
	s.ToDots()
}

// CopyUnitContext copies unit context from another, update with our font
// info, and then cache everything out in terms of raw pixel dots for
// rendering -- call at start of render
func (s *Style) CopyUnitContext(ctxt *units.Context) {
	s.UnContext = *ctxt
	s.Font.SetUnitContext(&s.UnContext)
	s.ToDots()
}

// ToDots calls ToDots on all units.Value fields in the style (recursively) --
// need to have set the UnContext first -- only after layout at render time is
// that possible
func (s *Style) ToDots() {
	StyleFieldsVar.ToDots(s)
}

// extra space around the central content in the box model, in dots -- todo:
// must complicate this if we want different spacing on different sides
// box outside-in: margin | border | padding | content
func (s *Style) BoxSpace() float32 {
	return s.Layout.Margin.Dots + s.Border.Width.Dots + s.Layout.Padding.Dots
}

// StyleFields contains the compiled fields for the Style type
type StyleFields struct {
	Fields   map[string]*StyledField `xml:"-" desc:"the compiled stylable fields, mapped the xml and alt tags for the field"`
	Inherits []*StyledField          `xml:"-" desc:"the compiled stylable fields of the unit.Value type -- "`
	Units    []*StyledField          `xml:"-" desc:"the compiled stylable fields of the unit.Value type -- "`
}

var StyleFieldsVar = StyleFields{}

func (sf *StyleFields) Init() {
	if sf.Fields == nil {
		sf.Fields, sf.Inherits, sf.Units = CompileStyledFields(&StyleDefault)
	}
}

func (sf *StyleFields) Inherit(st, par *Style) {
	sf.Init()
	InheritStyledFields(sf.Inherits, st, par)
}

func (sf *StyleFields) SetFromProps(st, par *Style, props ki.Props) {
	sf.Init()
	StyledFieldsFromProps(sf.Fields, st, par, props)
}

func (sf *StyleFields) ToDots(st *Style) {
	sf.Init()
	UnitValsToDots(sf.Units, st, &st.UnContext)
}

////////////////////////////////////////////////////////////////////////////////////////
//   Style processing util

// this is the function to process a given field when walking the style
type WalkStyleFieldFun func(sf reflect.StructField, vf reflect.Value, tag string, baseoff uintptr)

// general-purpose function for walking through style structures and calling fun on each field with a valid 'xml' tag
func WalkStyleStruct(obj interface{}, outerTag string, baseoff uintptr, fun WalkStyleFieldFun) {
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
		if ft.Kind() == reflect.Struct && ft.Name() != "Value" && ft.Name() != "Color" {
			WalkStyleStruct(vfi, tag, baseoff+sf.Offset, fun)
		} else {
			if tag == "" { // non-struct = don't process
				continue
			}
			fun(sf, vf, outerTag, baseoff)
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

// CompileStyledFields gathers all the fields with xml tag != "-", plus those
// that are units.Value's for later optimized processing of styles
func CompileStyledFields(defobj interface{}) (flds map[string]*StyledField, inhflds, uvls []*StyledField) {
	valtyp := reflect.TypeOf(units.Value{})

	flds = make(map[string]*StyledField)
	inhflds = make([]*StyledField, 0, 50)
	uvls = make([]*StyledField, 0, 50)

	WalkStyleStruct(defobj, "", uintptr(0),
		func(sf reflect.StructField, vf reflect.Value, outerTag string, baseoff uintptr) {
			// fmt.Printf("complile fld: %v base: %v off: %v\n", sf.Name, baseoff, sf.Offset)
			styf := &StyledField{Field: sf, NetOff: baseoff + sf.Offset, Default: vf}
			tag := StyleEffTag(sf.Tag.Get("xml"), outerTag)
			if _, ok := flds[tag]; ok {
				fmt.Printf("CompileStyledFields: ERROR redundant tag found! will fail! %v\n", tag)
			}
			flds[tag] = styf
			atags := sf.Tag.Get("alt")
			if atags != "" {
				atag := strings.Split(atags, ",")

				for _, tg := range atag {
					tag = StyleEffTag(tg, outerTag)
					flds[tag] = styf
				}
			}
			inhs := sf.Tag.Get("inherit")
			if inhs == "true" {
				inhflds = append(inhflds, styf)
			}
			if vf.Kind() == reflect.Struct && vf.Type() == valtyp {
				uvls = append(uvls, styf)
			}
		})
	return
}

// InheritStyleFields copies all the values from par to obj for fields marked
// as "inherit" -- inherited by default
func InheritStyledFields(inhflds []*StyledField, obj, par interface{}) {
	for _, fld := range inhflds {
		vf := fld.FieldValue(obj)
		pf := fld.FieldValue(par)
		vf.Elem().Set(pf.Elem()) // copy
	}
}

// StyleFieldsFromProps styles the fields from given properties for given object
func StyledFieldsFromProps(fields map[string]*StyledField, obj, par interface{}, props ki.Props) {
	hasPar := (par != nil)
	// fewer props than fields, esp with alts!
	for key, prv := range props {
		if key[0] == '#' || key[0] == '.' || key[0] == ':' {
			continue
		}
		if pstr, ok := prv.(string); ok {
			if len(pstr) > 0 && pstr[0] == '$' {
				nkey := pstr[1:]
				if vfld, nok := fields[nkey]; nok {
					nprv := vfld.FieldValue(obj).Elem().Interface()
					if fld, fok := fields[key]; fok {
						fld.FromProps(fields, obj, par, nprv, hasPar)
						continue
					}
				}
				fmt.Printf("StyledFieldsFromProps: redirect field not found: %v for key: %v\n", nkey, key)
			}
		}
		fld, ok := fields[key]
		if !ok {
			// note: props can apply to Paint or Style and not easy to keep those
			// precisely separated, so there will be mismatch..
			// log.Printf("SetStyleFields: Property key: %v not among xml or alt field tags for styled obj: %T\n", key, obj)
			continue
		}
		fld.FromProps(fields, obj, par, prv, hasPar)
	}
}

// UnitValsToDots runs ToDots on unit values, to compile down to raw pixels
func UnitValsToDots(unvls []*StyledField, obj interface{}, uc *units.Context) {
	for _, fld := range unvls {
		vf := fld.FieldValue(obj)
		uv := vf.Interface().(*units.Value)
		uv.ToDots(uc)
	}
}

// StyledField contains the relevant data for a given stylable field in a style struct
type StyledField struct {
	Field   reflect.StructField
	NetOff  uintptr       `desc:"net accumulated offset from the overall main type, e.g., Style"`
	Default reflect.Value `desc:"value of default value of this field"`
}

// FieldValue returns a reflect.Value for a given object, computed from NetOff
func (sf *StyledField) FieldValue(obj interface{}) reflect.Value {
	ov := reflect.ValueOf(obj)
	f := unsafe.Pointer(ov.Pointer() + sf.NetOff)
	nw := reflect.NewAt(sf.Field.Type, f)
	return kit.UnhideIfaceValue(nw).Elem()
}

// FromProps styles given field from property value prv
func (fld *StyledField) FromProps(fields map[string]*StyledField, obj, par, prv interface{}, hasPar bool) {
	vf := fld.FieldValue(obj)
	var pf reflect.Value
	if hasPar {
		pf = fld.FieldValue(par)
	}
	prstr := ""
	switch prtv := prv.(type) {
	case string:
		prstr = prtv
		if prtv == "inherit" {
			if hasPar {
				vf.Set(pf)
				fmt.Printf("StyleField %v set to inherited value: %v\n", fld.Field.Name, pf.Interface())
			}
			return
		}
		if prtv == "initial" {
			vf.Set(fld.Default)
			// fmt.Printf("StyleField set tag: %v to initial default value: %v\n", tag, df)
			return
		}
	}

	// todo: support keywords such as auto, normal, which should just set to 0

	npvf := kit.NonPtrValue(vf)

	vk := npvf.Kind()
	vt := npvf.Type()

	if vk == reflect.Struct { // only a few types -- todo: could make an interface if needed
		if vt == reflect.TypeOf(Color{}) {
			vc := vf.Interface().(*Color)
			switch prtv := prv.(type) {
			case string:
				if idx := strings.Index(prtv, "$"); idx > 0 {
					oclr := prtv[idx+1:]
					prtv = prtv[:idx]
					if vfld, nok := fields[oclr]; nok {
						nclr, nok := vfld.FieldValue(obj).Interface().(*Color)
						if nok {
							vc.SetColor(nclr) // init from color
							fmt.Printf("StyleField %v initialized to other color: %v val: %v\n", fld.Field.Name, oclr, vc)
						}
					}
				}
				// todo: this is crashing due to some kind of bad color -- presumably a pointer?
				// if hasPar {
				// 	fmt.Printf("field %v pf kind: %v str %v\n",
				// 		fld.Field.Name, pf.Kind(), pf.String())
				// 	// fmt.Printf("field %v pf kind: %v if: %v typ %T ptr %p\n",
				// 	// 	fld.Field.Name, pf.Kind(), pf.Interface(), pf.Interface(), pf.Interface())
				// 	// fmt.Printf("elem kind %v elem if %v elem typ %T ptr %p\n",
				// 	// 	pf.Elem().Kind(), pf.Elem().Interface(), pf.Elem().Interface(), pf.Elem().Interface())
				// 	if pclr, pok := pf.Interface().(*Color); pok {
				// 		vc.SetString(prtv, pclr)
				// 		fmt.Printf("StyleField %v set to color string: %v, with parent value: %v\n", fld.Field.Name, prtv, pclr)
				// 		return
				// 	}
				// }
				err := vc.SetString(prtv, nil)
				if err != nil {
					log.Printf("StyleField: %v\n", err)
				}
			case color.Color:
				vc.SetColor(prtv)
			}
			return
		} else if vt == reflect.TypeOf(units.Value{}) {
			uv := vf.Interface().(*units.Value)
			switch prtv := prv.(type) {
			case string:
				uv.SetFromString(prtv)
			case units.Value:
				*uv = prtv
			default: // assume Px as an implicit default
				prvflt := reflect.ValueOf(prv).Convert(reflect.TypeOf(float32(0.0))).Interface().(float32)
				uv.Set(prvflt, units.Px)
			}
			return
		}
		return // no can do any struct otherwise
	} else if vk >= reflect.Int && vk <= reflect.Uint64 { // some kind of int
		if prstr != "" {
			tn := kit.FullTypeName(fld.Field.Type)
			if kit.Enums.Enum(tn) != nil {
				kit.Enums.SetEnumValueFromStringAltFirst(vf, prstr)
			} else {
				fmt.Printf("gi.StyleField: enum name not found %v for field %v\n", tn, fld.Field.Name)
			}
			return
		} else {
			// somehow this doesn't work:
			// vf.Set(reflect.ValueOf(prv))
			ival, ok := kit.ToInt(prv)
			if !ok {
				log.Printf("gi.StyledField.FromProps: for field: %v could not convert property to int: %v %T\n", fld.Field.Name, prv, prv)
			} else {
				kit.Enums.SetEnumValueFromInt64(vf, ival)
			}
			return
		}
	}

	// fmt.Printf("field: %v, type: %v value: %v %T\n", fld.Field.Name, fld.Field.Type.Name(), vf.Interface(), vf.Interface())

	// otherwise just set directly based on type, using standard conversions
	vf.Set(reflect.ValueOf(prv).Convert(reflect.TypeOf(vt)))
}

// manual method for getting a units value directly
func StyleUnitsValue(tag string, uv *units.Value, props ki.Props) bool {
	prv, got := props[tag]
	if !got {
		return false
	}
	switch v := prv.(type) {
	case string:
		uv.SetFromString(v)
	case float64:
		uv.Set(float32(v), units.Px) // assume px
	case float32:
		uv.Set(v, units.Px) // assume px
	case int:
		uv.Set(float32(v), units.Px) // assume px
	}
	return true
}
