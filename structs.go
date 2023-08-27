// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package laser

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
)

// FlatFieldsTypeFunc calls a function on all the primary fields of a given
// struct type, including those on anonymous embedded structs that this struct
// has, passing the current (embedded) type and StructField -- effectively
// flattens the reflect field list -- if fun returns false then iteration
// stops -- overall rval is false if iteration was stopped or there was an
// error (logged), true otherwise
func FlatFieldsTypeFunc(typ reflect.Type, fun func(typ reflect.Type, field reflect.StructField) bool) bool {
	typ = NonPtrType(typ)
	if typ.Kind() != reflect.Struct {
		log.Printf("kit.FlatFieldsTypeFunc: Must call on a struct type, not: %v\n", typ)
		return false
	}
	rval := true
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			rval = FlatFieldsTypeFunc(f.Type, fun) // no err here
			if !rval {
				break
			}
		} else {
			rval = fun(typ, f)
			if !rval {
				break
			}
		}
	}
	return rval
}

// AllFieldsTypeFunc calls a function on all the fields of a given struct type,
// including those on *any* embedded structs that this struct has -- if fun
// returns false then iteration stops -- overall rval is false if iteration
// was stopped or there was an error (logged), true otherwise.
func AllFieldsTypeFunc(typ reflect.Type, fun func(typ reflect.Type, field reflect.StructField) bool) bool {
	typ = NonPtrType(typ)
	if typ.Kind() != reflect.Struct {
		log.Printf("kit.AllFieldsTypeFunc: Must call on a struct type, not: %v\n", typ)
		return false
	}
	rval := true
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Struct {
			rval = AllFieldsTypeFunc(f.Type, fun) // no err here
			if !rval {
				break
			}
		} else {
			rval = fun(typ, f)
			if !rval {
				break
			}
		}
	}
	return rval
}

// FlatFieldsValueFunc calls a function on all the primary fields of a
// given struct value (must pass a pointer to the struct) including those on
// anonymous embedded structs that this struct has, passing the current
// (embedded) type and StructField -- effectively flattens the reflect field
// list
func FlatFieldsValueFunc(stru any, fun func(stru any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool) bool {
	vv := reflect.ValueOf(stru)
	if stru == nil || vv.Kind() != reflect.Ptr {
		log.Printf("kit.FlatFieldsValueFunc: must pass a non-nil pointer to the struct: %v\n", stru)
		return false
	}
	v := NonPtrValue(vv)
	if !v.IsValid() {
		return true
	}
	typ := v.Type()
	if typ.Kind() != reflect.Struct {
		// log.Printf("kit.FlatFieldsValueFunc: non-pointer type is not a struct: %v\n", typ.String())
		return false
	}
	rval := true
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		vf := v.Field(i)
		if !vf.CanInterface() {
			continue
		}
		vfi := vf.Interface()
		if vfi == stru {
			continue
		}
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			// key to take addr here so next level is addressable
			rval = FlatFieldsValueFunc(PtrValue(vf).Interface(), fun)
			if !rval {
				break
			}
		} else {
			rval = fun(vfi, typ, f, vf)
			if !rval {
				break
			}
		}
	}
	return rval
}

// FlatFields returns a slice list of all the StructField type information for
// fields of given type and any embedded types -- returns nil on error
// (logged)
func FlatFields(typ reflect.Type) []reflect.StructField {
	ff := make([]reflect.StructField, 0)
	falseErr := FlatFieldsTypeFunc(typ, func(typ reflect.Type, field reflect.StructField) bool {
		ff = append(ff, field)
		return true
	})
	if falseErr == false {
		return nil
	}
	return ff
}

// AllFields returns a slice list of all the StructField type information for
// all elemental fields of given type and all embedded types -- returns nil on
// error (logged)
func AllFields(typ reflect.Type) []reflect.StructField {
	ff := make([]reflect.StructField, 0)
	falseErr := AllFieldsTypeFunc(typ, func(typ reflect.Type, field reflect.StructField) bool {
		ff = append(ff, field)
		return true
	})
	if falseErr == false {
		return nil
	}
	return ff
}

// AllFieldsN returns number of elemental fields in given type
func AllFieldsN(typ reflect.Type) int {
	n := 0
	falseErr := AllFieldsTypeFunc(typ, func(typ reflect.Type, field reflect.StructField) bool {
		n++
		return true
	})
	if falseErr == false {
		return 0
	}
	return n
}

// FlatFieldsVals returns a slice list of all the field reflect.Value's for
// fields of given struct (must pass a pointer to the struct) and any of its
// embedded structs -- returns nil on error (logged)
func FlatFieldVals(stru any) []reflect.Value {
	ff := make([]reflect.Value, 0)
	falseErr := FlatFieldsValueFunc(stru, func(stru any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		ff = append(ff, fieldVal)
		return true
	})
	if falseErr == false {
		return nil
	}
	return ff
}

// FlatFieldInterfaces returns a slice list of all the field interface{}
// values *as pointers to the field value* (i.e., calling Addr() on the Field
// Value) for fields of given struct (must pass a pointer to the struct) and
// any of its embedded structs -- returns nil on error (logged)
func FlatFieldInterfaces(stru any) []any {
	ff := make([]any, 0)
	falseErr := FlatFieldsValueFunc(stru, func(stru any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		ff = append(ff, PtrValue(fieldVal).Interface())
		return true
	})
	if falseErr == false {
		return nil
	}
	return ff
}

// FlatFieldByName returns field in type or embedded structs within type, by
// name -- native function already does flat version, so this is just for
// reference and consistency
func FlatFieldByName(typ reflect.Type, nm string) (reflect.StructField, bool) {
	return typ.FieldByName(nm)
}

// FieldByPath returns field in type or embedded structs within type, by a
// dot-separated path -- finds field by name for each level of the path, and
// recurses.
func FieldByPath(typ reflect.Type, path string) (reflect.StructField, bool) {
	pels := strings.Split(path, ".")
	ctyp := typ
	plen := len(pels)
	for i, pe := range pels {
		fld, ok := ctyp.FieldByName(pe)
		if !ok {
			log.Printf("kit.FieldByPath: field: %v not found in type: %v, starting from path: %v, in type: %v\n", pe, ctyp.String(), path, typ.String())
			return fld, false
		}
		if i == plen-1 {
			return fld, true
		}
		ctyp = fld.Type
	}
	return reflect.StructField{}, false
}

// FieldValueByPath returns field interface in type or embedded structs within
// type, by a dot-separated path -- finds field by name for each level of the
// path, and recurses.
func FieldValueByPath(stru any, path string) (reflect.Value, bool) {
	pels := strings.Split(path, ".")
	sval := reflect.ValueOf(stru)
	cval := sval
	typ := sval.Type()
	ctyp := typ
	plen := len(pels)
	for i, pe := range pels {
		_, ok := ctyp.FieldByName(pe)
		if !ok {
			log.Printf("kit.FieldValueByPath: field: %v not found in type: %v, starting from path: %v, in type: %v\n", pe, cval.Type().String(), path, typ.String())
			return cval, false
		}
		fval := cval.FieldByName(pe)
		if i == plen-1 {
			return fval, true
		}
		cval = fval
		ctyp = fval.Type()
	}
	return reflect.Value{}, false
}

// FlatFieldTag returns given tag value in field in type or embedded structs
// within type, by name -- empty string if not set or field not found
func FlatFieldTag(typ reflect.Type, nm, tag string) string {
	fld, ok := typ.FieldByName(nm)
	if !ok {
		return ""
	}
	return fld.Tag.Get(tag)
}

// FlatFieldValueByName finds field in object and embedded objects, by name,
// returning reflect.Value of field -- native version of Value function
// already does flat find, so this just provides a convenient wrapper
func FlatFieldValueByName(stru any, nm string) reflect.Value {
	vv := reflect.ValueOf(stru)
	if stru == nil || vv.Kind() != reflect.Ptr {
		log.Printf("kit.FlatFieldsValueFunc: must pass a non-nil pointer to the struct: %v\n", stru)
		return reflect.Value{}
	}
	v := NonPtrValue(vv)
	return v.FieldByName(nm)
}

// FlatFieldInterfaceByName finds field in object and embedded objects, by
// name, returning interface{} to pointer of field, or nil if not found
func FlatFieldInterfaceByName(stru any, nm string) any {
	ff := FlatFieldValueByName(stru, nm)
	if !ff.IsValid() {
		return nil
	}
	return PtrValue(ff).Interface()
}

// TypeEmbeds checks if given type embeds another type, at any level of
// recursive embedding (including being the type itself)
func TypeEmbeds(typ, embed reflect.Type) bool {
	typ = NonPtrType(typ)
	embed = NonPtrType(embed)
	if typ == embed {
		return true
	}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			// fmt.Printf("typ %v anon struct %v\n", typ.Name(), f.Name)
			if f.Type == embed {
				return true
			}
			return TypeEmbeds(f.Type, embed)
		}
	}
	return false
}

// Embed returns the embedded struct of given type within given struct
func Embed(stru any, embed reflect.Type) any {
	if AnyIsNil(stru) {
		return nil
	}
	v := NonPtrValue(reflect.ValueOf(stru))
	typ := v.Type()
	if typ == embed {
		return PtrValue(v).Interface()
	}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Anonymous { // anon only avail on StructField fm typ
			vf := v.Field(i)
			vfpi := PtrValue(vf).Interface()
			if f.Type == embed {
				return vfpi
			}
			rv := Embed(vfpi, embed)
			if rv != nil {
				return rv
			}
		}
	}
	return nil
}

// EmbedImplements checks if given type implements given interface, or
// it embeds a type that does so -- must pass a type constructed like this:
// reflect.TypeOf((*gi.Node2D)(nil)).Elem() or just reflect.TypeOf(ki.BaseIface())
func EmbedImplements(typ, iface reflect.Type) bool {
	if iface.Kind() != reflect.Interface {
		log.Printf("kit.TypeRegistry EmbedImplements -- type is not an interface: %v\n", iface)
		return false
	}
	if typ.Implements(iface) {
		return true
	}
	if reflect.PtrTo(typ).Implements(iface) { // typically need the pointer type to impl
		return true
	}
	typ = NonPtrType(typ)
	if typ.Implements(iface) { // try it all possible ways..
		return true
	}
	if typ.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			rv := EmbedImplements(f.Type, iface)
			if rv {
				return true
			}
		}
	}
	return false
}

// SetFromDefaultTags sets values of fields in given struct based on
// `def:` default value field tags.
func SetFromDefaultTags(obj any) error {
	if AnyIsNil(obj) {
		return nil
	}
	ov := reflect.ValueOf(obj)
	if ov.Kind() == reflect.Pointer && ov.IsNil() {
		return nil
	}
	val := NonPtrValue(ov)
	typ := val.Type()
	var err error
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fv := val.Field(i)
		def, ok := f.Tag.Lookup("def")
		if NonPtrType(f.Type).Kind() == reflect.Struct && (!ok || def == "") {
			SetFromDefaultTags(PtrValue(fv).Interface())
			continue
		}
		if !ok || def == "" {
			continue
		}
		if def[0] == '{' || def[0] == '[' { // complex type
			def = strings.ReplaceAll(def, `'`, `"`) // allow single quote to work as double quote for JSON format
		} else {
			def = strings.Split(def, ",")[0]
			if strings.Contains(def, ":") { // don't do ranges
				continue
			}
		}
		ok = SetRobust(PtrValue(fv).Interface(), def) // overkill but whatever
		if !ok {
			err = fmt.Errorf("SetFromDefaultTags: was not able to set field: %s in object of type: %s from val: %s", f.Name, typ.Name(), def)
			log.Println(err)
		}
	}
	return err
}

// StructTags returns a map[string]string of the tag string from a reflect.StructTag value
// e.g., from StructField.Tag
func StructTags(tags reflect.StructTag) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	flds := strings.Fields(string(tags))
	smap := make(map[string]string, len(flds))
	for _, fld := range flds {
		cli := strings.Index(fld, ":")
		if cli < 0 || len(fld) < cli+3 {
			continue
		}
		vl := strings.TrimSuffix(fld[cli+2:], `"`)
		smap[fld[:cli]] = vl
	}
	return smap
}

// StringJSON returns a JSON representation of item, as a string
// e.g., for printing / debugging etc.
func StringJSON(it any) string {
	b, _ := json.MarshalIndent(it, "", "  ")
	return string(b)
}
