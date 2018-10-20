// Copyright (c) 2018, The GoKi Authors. All rights reserved.
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

	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
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

// Style has all the CSS-based style elements -- used for widget-type objects
type Style struct {
	Display       bool          `xml:"display" desc:"todo big enum of how to display item -- controls layout etc"`
	Visible       bool          `xml:"visible" desc:"todo big enum of how to display item -- controls layout etc"`
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

// todo: Animation

// Clear -- no floating elements

// Clip -- clip images

// column- settings -- lots of those

// LayoutStyle is in layout.go
// FontStyle is in font.go
// TextStyle is in text.go

// List-style for lists

// Object-fit for videos

// visibility -- support more than just hidden  inherit:"true"

// transition -- animation of hover, etc

// RebuildDefaultStyles is a global state var used by Prefs to trigger rebuild
// of all the default styles, which are otherwise compiled and not updated
var RebuildDefaultStyles bool

// CurStyleNode2D is always set to the current node that is being styled --
// used for finding url references -- only active during a Style pass
var CurStyleNode2D Node2D

// SetCurStyleNode2D sets the current styling node to given node, or nil when done
func SetCurStyleNode2D(node Node2D) {
	CurStyleNode2D = node
}

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

////////////////////////////////////////////////////////////////////////////////////////
// Style structs

// note: background-color is in FontStyle as it is needed to make that the
// only style needed for text render styling

// // BackgroundStyle has style parameters for backgrounds
// type BackgroundStyle struct {
// 	// todo: all the properties not yet implemented -- mostly about images
// 	// Image is like a PaintServer -- includes gradients etc
// 	// Attachment -- how the image moves
// 	// Clip -- how to clip the image
// 	// Origin
// 	// Position
// 	// Repeat
// 	// Size
// }

// func (b *BackgroundStyle) Defaults() {
// 	b.Color.SetColor(color.White)
// }

// BoxSides specifies sides of a box -- some properties can be specified per
// each side (e.g., border) or not
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

func (ev BoxSides) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *BoxSides) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// BorderDrawStyle determines how to draw the border
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

func (ev BorderDrawStyle) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *BorderDrawStyle) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// BorderStyle contains style parameters for borders
type BorderStyle struct {
	Style  BorderDrawStyle `xml:"style" desc:"prop: border-style = how to draw the border"`
	Width  units.Value     `xml:"width" desc:"prop: border-width = width of the border"`
	Radius units.Value     `xml:"radius" desc:"prop: border-radius = rounding of the corners"`
	Color  Color           `xml:"color" desc:"prop: border-color = color of the border"`
}

// style parameters for shadows
type ShadowStyle struct {
	HOffset units.Value `xml:".h-offset" desc:"prop: .h-offset = horizontal offset of shadow -- positive = right side, negative = left side"`
	VOffset units.Value `xml:".v-offset" desc:"prop: .v-offset = vertical offset of shadow -- positive = below, negative = above"`
	Blur    units.Value `xml:".blur" desc:"prop: .blur = blur radius -- higher numbers = more blurry"`
	Spread  units.Value `xml:".spread" desc:"prop: .spread = spread radius -- positive number increases size of shadow, negative descreases size"`
	Color   Color       `xml:".color" desc:"prop: .color = color of the shadow"`
	Inset   bool        `xml:".inset" desc:"prop: .inset = shadow is inset within box instead of outset outside of box"`
}

func (s *ShadowStyle) HasShadow() bool {
	return (s.HOffset.Dots > 0 || s.VOffset.Dots > 0)
}

// CurrentColor is automatically updated from the Color setting of a Style and
// accessible as a color name in any other style as currentColor
var CurrentColor Color

func (s *Style) Defaults() {
	// mostly all the defaults are 0 initial values, except these..
	s.IsSet = false
	s.UnContext.Defaults()
	s.Outline.Style = BorderNone
	s.PointerEvents = true
	s.Layout.Defaults()
	s.Font.Defaults()
	s.Text.Defaults()
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
	*s = *cp
	s.Font.BgColor = cp.Font.BgColor
	s.IsSet = is
	s.PropsNil = pn
	s.dotsSet = ds
	s.lastUnCtxt = lu
}

// InheritFields from parent: Manual inheriting of values is much faster than
// automatic version!
func (s *Style) InheritFields(par *Style) {
	s.Font.InheritFields(&par.Font)
	s.Text.InheritFields(&par.Text)
}

// SetStyleProps sets style values based on given property map (name: value pairs),
// inheriting elements as appropriate from parent
func (s *Style) SetStyleProps(par *Style, props ki.Props) {
	if !s.IsSet && par != nil { // first time
		// StyleFields.Inherit(s, par) // very slow for some mysterious reason
		s.InheritFields(par)
	}
	StyleFields.Style(s, par, props)
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
func (s *Style) Use() {
	CurrentColor = s.Font.Color
}

// SetUnitContext sets the unit context based on size of viewport and parent
// element (from bbox) and then cache everything out in terms of raw pixel
// dots for rendering -- call at start of render
func (s *Style) SetUnitContext(vp *Viewport2D, el Vec2D) {
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
	s.Font.OpenFont(&s.UnContext) // calls SetUnContext after updating metrics

	// skipping this doesn't seem to be good:
	// if !(s.dotsSet && s.UnContext == s.lastUnCtxt && s.PropsNil) {
	// fmt.Printf("dotsSet: %v unctx: %v lst: %v  == %v  pn: %v\n", s.dotsSet, s.UnContext, s.lastUnCtxt, (s.UnContext == s.lastUnCtxt), s.PropsNil)
	s.ToDots()
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
	s.Font.SetUnitContext(&s.UnContext)
	// this seems to work fine
	if !(s.dotsSet && s.UnContext == s.lastUnCtxt && s.PropsNil) {
		s.ToDots()
		s.dotsSet = true
		s.lastUnCtxt = s.UnContext
		// } else {
		// 	fmt.Println("skipped")
	}
}

// ToDots calls ToDots on all units.Value fields in the style (recursively) --
// need to have set the UnContext first
func (s *Style) ToDots() {
	StyleFields.ToDots(s, &s.UnContext)
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
func (s *Style) ApplyCSS(node Node2D, css ki.Props, key, selector string) bool {
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
	s.SetStyleProps(parSty, pmap)
	return true
}

// StyleCSS applies css style properties to given Widget node, parsing out
// type, .class, and #name selectors, along with optional sub-selector
// (:hover, :active etc)
func (s *Style) StyleCSS(node Node2D, css ki.Props, selector string) {
	tyn := strings.ToLower(node.Type().Name()) // type is most general, first
	s.ApplyCSS(node, css, tyn, selector)
	cln := "." + strings.ToLower(node.AsNode2D().Class) // then class
	s.ApplyCSS(node, css, cln, selector)
	idnm := "#" + strings.ToLower(node.Name()) // then name
	s.ApplyCSS(node, css, idnm, selector)
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

// StyleFields contain the StyledFields for Style type
var StyleFields = initStyle()

func initStyle() *StyledFields {
	StyleDefault.Defaults()
	sf := &StyledFields{}
	sf.Init(&StyleDefault)
	return sf
}

////////////////////////////////////////////////////////////////////////////////////////
//   StyledFields

// StyledFields contains fields of a struct that are styled -- create one
// instance of this for each type that has styled fields (Style, Paint, and a
// few with ad-hoc styled fields)
type StyledFields struct {
	Fields   map[string]*StyledField `desc:"the compiled stylable fields, mapped for the xml and alt tags for the field"`
	Inherits []*StyledField          `desc:"the compiled stylable fields that have inherit:true tags and should thus be inherited from parent objects"`
	Units    []*StyledField          `desc:"the compiled stylable fields of the unit.Value type, which should have ToDots run on them"`
	Default  interface{}             `desc:"points to the Default instance of this type, initialized with the default values used for 'initial' keyword"`
}

func (sf *StyledFields) Init(def interface{}) {
	sf.Default = def
	sf.CompileFields(def)
}

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

// AddField adds a single field -- must be a direct field on the object and
// not a field on an embedded type -- used for Widget objects where only one
// or a few fields are styled
func (sf *StyledFields) AddField(def interface{}, fieldName string) error {
	valtyp := reflect.TypeOf(units.Value{})

	if sf.Fields == nil {
		sf.Fields = make(map[string]*StyledField, 5)
		sf.Inherits = make([]*StyledField, 0, 5)
		sf.Units = make([]*StyledField, 0, 5)
	}
	otp := reflect.TypeOf(def)
	if otp.Kind() != reflect.Ptr {
		err := fmt.Errorf("gi.StyleFields.AddField: must pass pointers to the structs, not type: %v kind %v\n", otp, otp.Kind())
		log.Print(err)
		return err
	}
	ot := otp.Elem()
	if ot.Kind() != reflect.Struct {
		err := fmt.Errorf("gi.StyleFields.AddField: only works on structs, not type: %v kind %v\n", ot, ot.Kind())
		log.Print(err)
		return err
	}
	vo := reflect.ValueOf(def).Elem()
	struf, ok := ot.FieldByName(fieldName)
	if !ok {
		err := fmt.Errorf("gi.StyleFields.AddField: field name: %v not found in type %v\n", fieldName, ot.Name())
		log.Print(err)
		return err
	}

	vf := vo.FieldByName(fieldName)

	styf := &StyledField{Field: struf, NetOff: struf.Offset, Default: vf}
	tag := struf.Tag.Get("xml")
	sf.Fields[tag] = styf
	atags := struf.Tag.Get("alt")
	if atags != "" {
		atag := strings.Split(atags, ",")

		for _, tg := range atag {
			sf.Fields[tg] = styf
		}
	}
	inhs := struf.Tag.Get("inherit")
	if inhs == "true" {
		sf.Inherits = append(sf.Inherits, styf)
	}
	if vf.Kind() == reflect.Struct && vf.Type() == valtyp {
		sf.Units = append(sf.Units, styf)
	}
	return nil
}

// CompileFields gathers all the fields with xml tag != "-", plus those
// that are units.Value's for later optimized processing of styles
func (sf *StyledFields) CompileFields(def interface{}) {
	valtyp := reflect.TypeOf(units.Value{})

	sf.Fields = make(map[string]*StyledField, 50)
	sf.Inherits = make([]*StyledField, 0, 50)
	sf.Units = make([]*StyledField, 0, 50)

	WalkStyleStruct(def, "", uintptr(0),
		func(struf reflect.StructField, vf reflect.Value, outerTag string, baseoff uintptr) {
			styf := &StyledField{Field: struf, NetOff: baseoff + struf.Offset, Default: vf}
			tag := StyleEffTag(struf.Tag.Get("xml"), outerTag)
			if _, ok := sf.Fields[tag]; ok {
				fmt.Printf("gi.StyledFileds.CompileFields: ERROR redundant tag found -- please only use unique tags! %v\n", tag)
			}
			sf.Fields[tag] = styf
			atags := struf.Tag.Get("alt")
			if atags != "" {
				atag := strings.Split(atags, ",")

				for _, tg := range atag {
					tag = StyleEffTag(tg, outerTag)
					sf.Fields[tag] = styf
				}
			}
			inhs := struf.Tag.Get("inherit")
			if inhs == "true" {
				sf.Inherits = append(sf.Inherits, styf)
			}
			if vf.Kind() == reflect.Struct && vf.Type() == valtyp {
				sf.Units = append(sf.Units, styf)
			}
		})
	return
}

// Inherit copies all the values from par to obj for fields marked as
// "inherit" -- inherited by default.  NOTE: No longer using this -- doing it
// manually -- much faster
func (sf *StyledFields) Inherit(obj, par interface{}) {
	// pr := prof.Start("StyleFields.Inherit")
	objptr := reflect.ValueOf(obj).Pointer()
	hasPar := !kit.IfaceIsNil(par)
	var parptr uintptr
	if hasPar {
		parptr = reflect.ValueOf(par).Pointer()
	}
	for _, fld := range sf.Inherits {
		pfi := fld.FieldIface(parptr)
		fld.FromProps(sf.Fields, objptr, parptr, pfi, hasPar)
		// fmt.Printf("inh: %v\n", fld.Field.Name)
	}
	// pr.End()
}

// Style applies styles to the fields from given properties for given object
func (sf *StyledFields) Style(obj, par interface{}, props ki.Props) {
	if props == nil {
		return
	}
	pr := prof.Start("StyleFields.Style")
	objptr := reflect.ValueOf(obj).Pointer()
	hasPar := !kit.IfaceIsNil(par)
	var parptr uintptr
	if hasPar {
		parptr = reflect.ValueOf(par).Pointer()
	}
	// fewer props than fields, esp with alts!
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		if vstr, ok := val.(string); ok {
			if len(vstr) > 0 && vstr[0] == '$' { // special case to use other value
				nkey := vstr[1:] // e.g., border-color has "$background-color" value
				if vfld, nok := sf.Fields[nkey]; nok {
					nval := vfld.FieldIface(objptr)
					if fld, fok := sf.Fields[key]; fok {
						fld.FromProps(sf.Fields, objptr, parptr, nval, hasPar)
						continue
					}
				}
				fmt.Printf("gi.StyledFields.Style: redirect field not found: %v for key: %v\n", nkey, key)
			}
		}
		fld, ok := sf.Fields[key]
		if !ok {
			// note: props can apply to Paint or Style and not easy to keep those
			// precisely separated, so there will be mismatch..
			// log.Printf("SetStyleFields: Property key: %v not among xml or alt field tags for styled obj: %T\n", key, obj)
			continue
		}
		fld.FromProps(sf.Fields, objptr, parptr, val, hasPar)
	}
	pr.End()
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (sf *StyledFields) ToDots(obj interface{}, uc *units.Context) {
	pr := prof.Start("StyleFields.ToDots")
	objptr := reflect.ValueOf(obj).Pointer()
	for _, fld := range sf.Units {
		uv := fld.UnitsValue(objptr)
		uv.ToDots(uc)
	}
	pr.End()
}

////////////////////////////////////////////////////////////////////////////////////////
//   StyledField

// StyledField contains the relevant data for a given stylable field in a struct
type StyledField struct {
	Field   reflect.StructField
	NetOff  uintptr       `desc:"net accumulated offset from the overall main type, e.g., Style"`
	Default reflect.Value `desc:"value of default value of this field"`
}

// FieldValue returns a reflect.Value for a given object, computed from NetOff
// -- this is VERY expensive time-wise -- need to figure out a better solution..
func (sf *StyledField) FieldValue(objptr uintptr) reflect.Value {
	f := unsafe.Pointer(objptr + sf.NetOff)
	nw := reflect.NewAt(sf.Field.Type, f)
	return kit.UnhideIfaceValue(nw).Elem()
}

// FieldIface returns an interface{} for a given object, computed from NetOff
// -- much faster -- use this
func (sf *StyledField) FieldIface(objptr uintptr) interface{} {
	npt := kit.NonPtrType(sf.Field.Type)
	npk := npt.Kind()
	switch {
	case npt == KiT_Color:
		return (*Color)(unsafe.Pointer(objptr + sf.NetOff))
	case npt == KiT_ColorSpec:
		return (*ColorSpec)(unsafe.Pointer(objptr + sf.NetOff))
	case npt == KiT_Matrix2D:
		return (*Matrix2D)(unsafe.Pointer(objptr + sf.NetOff))
	case npt.Name() == "Value":
		return (*units.Value)(unsafe.Pointer(objptr + sf.NetOff))
	case npk >= reflect.Int && npk <= reflect.Uint64:
		return sf.FieldValue(objptr).Interface() // no choice for enums
	case npk == reflect.Float32:
		return (*float32)(unsafe.Pointer(objptr + sf.NetOff))
	case npk == reflect.Float64:
		return (*float64)(unsafe.Pointer(objptr + sf.NetOff))
	case npk == reflect.Bool:
		return (*bool)(unsafe.Pointer(objptr + sf.NetOff))
	case npk == reflect.String:
		return (*string)(unsafe.Pointer(objptr + sf.NetOff))
	case sf.Field.Name == "Dashes":
		return (*[]float64)(unsafe.Pointer(objptr + sf.NetOff))
	default:
		fmt.Printf("Field: %v type %v not processed in StyledField.FieldIface -- fixme!\n", sf.Field.Name, npt.String())
		return nil
	}
}

// UnitsValue returns a units.Value for a field, which must be of that type..
func (sf *StyledField) UnitsValue(objptr uintptr) *units.Value {
	uv := (*units.Value)(unsafe.Pointer(objptr + sf.NetOff))
	return uv
}

// FromProps styles given field from property value val, with optional parent object obj
func (fld *StyledField) FromProps(fields map[string]*StyledField, objptr, parptr uintptr, val interface{}, hasPar bool) {
	errstr := "gi.StyledField FromProps: Field:"
	fi := fld.FieldIface(objptr)
	if kit.IfaceIsNil(fi) {
		fmt.Printf("%v %v of type %v has nil value\n", errstr, fld.Field.Name, fld.Field.Type.String())
		return
	}
	switch valv := val.(type) {
	case string:
		if valv == "inherit" {
			if hasPar {
				val = fld.FieldIface(parptr)
				// fmt.Printf("%v %v set to inherited value: %v\n", errstr, fld.Field.Name, val)
			} else {
				// fmt.Printf("%v %v tried to inherit but par null: %v\n", errstr, fld.Field.Name, val)
				return
			}
		}
		if valv == "initial" {
			val = fld.Default.Interface()
			// fmt.Printf("%v set tag: %v to initial default value: %v\n", errstr, tag, df)
		}
	}
	// todo: support keywords such as auto, normal, which should just set to 0

	npt := kit.NonPtrType(reflect.TypeOf(fi))
	npk := npt.Kind()

	switch fiv := fi.(type) {
	case *ColorSpec:
		switch valv := val.(type) {
		case string:
			fiv.SetString(valv)
		case *Color:
			fiv.SetColor(*valv)
		case *ColorSpec:
			*fiv = *valv
		case color.Color:
			fiv.SetColor(valv)
		}
	case *Color:
		switch valv := val.(type) {
		case string:
			if idx := strings.Index(valv, "$"); idx > 0 {
				oclr := valv[idx+1:]
				valv = valv[:idx]
				if vfld, nok := fields[oclr]; nok {
					nclr, nok := vfld.FieldIface(objptr).(*Color)
					if nok {
						fiv.SetColor(nclr) // init from color
						fmt.Printf("%v %v initialized to other color: %v val: %v\n", errstr, fld.Field.Name, oclr, fiv)
					}
				}
			}
			err := fiv.SetString(valv, nil)
			if err != nil {
				log.Printf("StyleField: %v\n", err)
			}
		case *Color:
			*fiv = *valv
		case color.Color:
			fiv.SetColor(valv)
		default:
			fmt.Printf("%v %v could not set Color from prop: %v type: %T\n", errstr, fld.Field.Name, val, val)
		}
	case *units.Value:
		err := fiv.SetIFace(val)
		if err != nil {
			fmt.Printf("%v %v %v\n", errstr, fld.Field.Name, err)
		}
	case *Matrix2D:
		switch valv := val.(type) {
		case string:
			fiv.SetString(valv)
		case *Matrix2D:
			*fiv = *valv
		}
	case *[]float64:
		switch valv := val.(type) {
		case string:
			*fiv = ParseDashesString(valv)
		case *[]float64:
			*fiv = *valv
		}
	default:
		if npk >= reflect.Int && npk <= reflect.Uint64 {
			switch valv := val.(type) {
			case string:
				tn := kit.FullTypeName(fld.Field.Type)
				if kit.Enums.Enum(tn) != nil {
					kit.Enums.SetAnyEnumIfaceFromString(fi, valv)
				} else if tn == "..int" {
					kit.SetRobust(fi, val)
				} else {
					fmt.Printf("%v enum name not found %v for field %v\n", errstr, tn, fld.Field.Name)
				}
			default:
				ival, ok := kit.ToInt(val)
				if !ok {
					log.Printf("%v for field: %v could not convert property to int: %v %T\n", errstr, fld.Field.Name, val, val)
				} else {
					kit.SetEnumIfaceFromInt64(fi, ival, npt)
				}
			}
		} else {
			kit.SetRobust(fi, val)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//   WalkStyleStruct

// this is the function to process a given field when walking the style
type WalkStyleFieldFunc func(struf reflect.StructField, vf reflect.Value, tag string, baseoff uintptr)

// StyleValueTypes is a map of types that are used as value types in style structures
var StyleValueTypes = map[reflect.Type]struct{}{
	units.KiT_Value: {},
	KiT_Color:       {},
	KiT_ColorSpec:   {},
	KiT_Matrix2D:    {},
}

// WalkStyleStruct walks through a struct, calling a function on fields with
// xml tags that are not "-", recursively through all the fields
func WalkStyleStruct(obj interface{}, outerTag string, baseoff uintptr, fun WalkStyleFieldFunc) {
	otp := reflect.TypeOf(obj)
	if otp.Kind() != reflect.Ptr {
		log.Printf("gi.WalkStyleStruct -- you must pass pointers to the structs, not type: %v kind %v\n", otp, otp.Kind())
		return
	}
	ot := otp.Elem()
	if ot.Kind() != reflect.Struct {
		log.Printf("gi.WalkStyleStruct -- only works on structs, not type: %v kind %v\n", ot, ot.Kind())
		return
	}
	vo := reflect.ValueOf(obj).Elem()
	for i := 0; i < ot.NumField(); i++ {
		struf := ot.Field(i)
		if struf.PkgPath != "" { // skip unexported fields
			continue
		}
		tag := struf.Tag.Get("xml")
		if tag == "-" {
			continue
		}
		ft := struf.Type
		// note: need Addrs() to pass pointers to fields, not fields themselves
		// fmt.Printf("processing field named: %v\n", struf.Nm)
		vf := vo.Field(i)
		vfi := vf.Addr().Interface()
		_, styvaltype := StyleValueTypes[ft]
		if ft.Kind() == reflect.Struct && !styvaltype {
			WalkStyleStruct(vfi, tag, baseoff+struf.Offset, fun)
		} else {
			if tag == "" { // non-struct = don't process
				continue
			}
			fun(struf, vf, outerTag, baseoff)
		}
	}
}
