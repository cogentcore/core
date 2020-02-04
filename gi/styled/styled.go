// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package styled implements styling via generic walking of relfect type info.
It is somewhat slower than the explicit manual code which is after
all not that hard to write -- maintenance may be more of an issue,
but given how time-critical styling is, it is worth it overall.

So basically this is legacy code here, maintained for future reference
given that it took a bit of work.
*/
package styled

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"unsafe"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//   Fields

// // StyleFields contain the Fields for Style type
// var StyleFields = initStyle()
//
// func initStyle() *Fields {
// 	StyleDefault.Defaults()
// 	sf := &Fields{}
// 	sf.Init(&StyleDefault)
// 	return sf
// }

// Fields contains fields of a struct that are styled -- create one
// instance of this for each type that has styled fields (Style, Paint, and a
// few with ad-hoc styled fields)
type Fields struct {
	Fields   map[string]*Field `desc:"the compiled stylable fields, mapped for the xml and alt tags for the field"`
	Inherits []*Field          `desc:"the compiled stylable fields that have inherit:true tags and should thus be inherited from parent objects"`
	Units    []*Field          `desc:"the compiled stylable fields of the unit.Value type, which should have ToDots run on them"`
	Default  interface{}       `desc:"points to the Default instance of this type, initialized with the default values used for 'initial' keyword"`
}

func (sf *Fields) Init(df interface{}) {
	sf.Default = df
	sf.CompileFields(df)
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
func (sf *Fields) AddField(df interface{}, fieldName string) error {
	valtyp := reflect.TypeOf(units.Value{})

	if sf.Fields == nil {
		sf.Fields = make(map[string]*Field, 5)
		sf.Inherits = make([]*Field, 0, 5)
		sf.Units = make([]*Field, 0, 5)
	}
	otp := reflect.TypeOf(df)
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
	vo := reflect.ValueOf(df).Elem()
	struf, ok := ot.FieldByName(fieldName)
	if !ok {
		err := fmt.Errorf("gi.StyleFields.AddField: field name: %v not found in type %v\n", fieldName, ot.Name())
		log.Print(err)
		return err
	}

	vf := vo.FieldByName(fieldName)

	styf := &Field{Field: struf, NetOff: struf.Offset, Default: vf}
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
func (sf *Fields) CompileFields(df interface{}) {
	valtyp := reflect.TypeOf(units.Value{})

	sf.Fields = make(map[string]*Field, 50)
	sf.Inherits = make([]*Field, 0, 50)
	sf.Units = make([]*Field, 0, 50)

	WalkStyleStruct(df, "", uintptr(0),
		func(struf reflect.StructField, vf reflect.Value, outerTag string, baseoff uintptr) {
			styf := &Field{Field: struf, NetOff: baseoff + struf.Offset, Default: vf}
			tag := StyleEffTag(struf.Tag.Get("xml"), outerTag)
			if _, ok := sf.Fields[tag]; ok {
				fmt.Printf("gi.Fileds.CompileFields: ERROR redundant tag found -- please only use unique tags! %v\n", tag)
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
func (sf *Fields) Inherit(obj, par interface{}, vp *gi.Viewport2D) {
	// pr := prof.Start("StyleFields.Inherit")
	// defer pr.End()
	objptr := reflect.ValueOf(obj).Pointer()
	hasPar := !kit.IfaceIsNil(par)
	var parptr uintptr
	if hasPar {
		parptr = reflect.ValueOf(par).Pointer()
	}
	for _, fld := range sf.Inherits {
		pfi := fld.FieldIface(parptr)
		fld.FromProps(sf.Fields, objptr, parptr, pfi, hasPar, vp)
		// fmt.Printf("inh: %v\n", fld.Field.Name)
	}
}

// Style applies styles to the fields from given properties for given object
func (sf *Fields) Style(obj, par interface{}, props ki.Props, vp *gi.Viewport2D) {
	if props == nil {
		return
	}
	// pr := prof.Start("StyleFields.Style")
	// defer pr.End()
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
						fld.FromProps(sf.Fields, objptr, parptr, nval, hasPar, vp)
						continue
					}
				}
				fmt.Printf("gi.Fields.Style: redirect field not found: %v for key: %v\n", nkey, key)
			}
		}
		fld, ok := sf.Fields[key]
		if !ok {
			// note: props can apply to Paint or Style and not easy to keep those
			// precisely separated, so there will be mismatch..
			// log.Printf("SetStyleFields: Property key: %v not among xml or alt field tags for styled obj: %T\n", key, obj)
			continue
		}
		fld.FromProps(sf.Fields, objptr, parptr, val, hasPar, vp)
	}
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (sf *Fields) ToDots(obj interface{}, uc *units.Context) {
	// pr := prof.Start("StyleFields.ToDots")
	objptr := reflect.ValueOf(obj).Pointer()
	for _, fld := range sf.Units {
		uv := fld.UnitsValue(objptr)
		uv.ToDots(uc)
	}
	// pr.End()
}

////////////////////////////////////////////////////////////////////////////////////////
//   Field

// Field contains the relevant data for a given stylable field in a struct
type Field struct {
	Field   reflect.StructField
	NetOff  uintptr       `desc:"net accumulated offset from the overall main type, e.g., Style"`
	Default reflect.Value `desc:"value of default value of this field"`
}

// FieldValue returns a reflect.Value for a given object, computed from NetOff
// -- this is VERY expensive time-wise -- need to figure out a better solution..
func (sf *Field) FieldValue(objptr uintptr) reflect.Value {
	f := unsafe.Pointer(objptr + sf.NetOff)
	nw := reflect.NewAt(sf.Field.Type, f)
	return kit.UnhideIfaceValue(nw).Elem()
}

// FieldIface returns an interface{} for a given object, computed from NetOff
// -- much faster -- use this
func (sf *Field) FieldIface(objptr uintptr) interface{} {
	npt := kit.NonPtrType(sf.Field.Type)
	npk := npt.Kind()
	switch {
	case npt == gi.KiT_Color:
		return (*gi.Color)(unsafe.Pointer(objptr + sf.NetOff))
	case npt == gi.KiT_ColorSpec:
		return (*gi.ColorSpec)(unsafe.Pointer(objptr + sf.NetOff))
	// case npt == mat32.KiT_Mat2:
	// 	return (*mat32.Mat2)(unsafe.Pointer(objptr + sf.NetOff))
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
		fmt.Printf("Field: %v type %v not processed in Field.FieldIface -- fixme!\n", sf.Field.Name, npt.String())
		return nil
	}
}

// UnitsValue returns a units.Value for a field, which must be of that type..
func (sf *Field) UnitsValue(objptr uintptr) *units.Value {
	uv := (*units.Value)(unsafe.Pointer(objptr + sf.NetOff))
	return uv
}

// FromProps styles given field from property value val, with optional parent object obj
func (fld *Field) FromProps(fields map[string]*Field, objptr, parptr uintptr, val interface{}, hasPar bool, vp *gi.Viewport2D) {
	errstr := "gi.Field FromProps: Field:"
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
	case *gi.ColorSpec:
		fiv.SetIFace(val, vp, fld.Field.Name)
	case *gi.Color:
		fiv.SetIFace(val, vp, fld.Field.Name)
	case *units.Value:
		fiv.SetIFace(val, fld.Field.Name)
	case *mat32.Mat2:
		switch valv := val.(type) {
		case string:
			fiv.SetString(valv)
		case *mat32.Mat2:
			*fiv = *valv
		}
	case *[]float64:
		switch valv := val.(type) {
		case string:
			*fiv = gi.ParseDashesString(valv)
		case *[]float64:
			*fiv = *valv
		}
	default:
		if npk >= reflect.Int && npk <= reflect.Uint64 {
			switch valv := val.(type) {
			case string:
				tn := kit.ShortTypeName(fld.Field.Type)
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
	units.KiT_Value:  {},
	gi.KiT_Color:     {},
	gi.KiT_ColorSpec: {},
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
