// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
	"math"
	"reflect"
	"strconv"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////////////
//   Type converter to / from type name

// Type provides JSON marshal / unmarshal with encoding of underlying type name
type Type struct {
	T reflect.Type
}

// stringer interface
func String(k Type) string {
	if k.T == nil {
		return "nil"
	}
	return k.T.Name()
}

// MarshalJSON saves only the type name
func (k Type) MarshalJSON() ([]byte, error) {
	if k.T == nil {
		b := []byte("null")
		return b, nil
	}
	nm := "\"" + k.T.Name() + "\""
	b := []byte(nm)
	return b, nil
}

// UnmarshalJSON loads the type name and looks it up in the Types registry of type names
func (k *Type) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		k.T = nil
		return nil
	}
	tn := string(bytes.Trim(bytes.TrimSpace(b), "\""))
	// fmt.Printf("loading type: %v", tn)
	typ := Types.FindType(tn)
	if typ == nil {
		return fmt.Errorf("Type UnmarshalJSON: Types type name not found: %v", tn)
	}
	k.T = typ
	return nil
}

// MarshalXML saves only the type name
func (k Type) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	tokens := []xml.Token{start}
	if k.T == nil {
		tokens = append(tokens, xml.CharData("null"))
	} else {
		tokens = append(tokens, xml.CharData(k.T.Name()))
	}
	tokens = append(tokens, xml.EndElement{start.Name})
	for _, t := range tokens {
		err := e.EncodeToken(t)
		if err != nil {
			return err
		}
	}
	err := e.Flush()
	if err != nil {
		return err
	}
	return nil
}

// UnmarshalXML loads the type name and looks it up in the Types registry of type names
func (k *Type) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	ct, ok := t.(xml.CharData)
	if ok {
		tn := string(bytes.TrimSpace([]byte(ct)))
		if tn == "null" {
			k.T = nil
		} else {
			// fmt.Printf("loading type: %v\n", tn)
			typ := Types.FindType(tn)
			if typ == nil {
				return fmt.Errorf("Type UnmarshalXML: Types type name not found: %v", tn)
			}
			k.T = typ
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		et, ok := t.(xml.EndElement)
		if ok {
			if et.Name != start.Name {
				return fmt.Errorf("Type UnmarshalXML: EndElement: %v does not match StartElement: %v", et.Name, start.Name)
			}
			return nil
		}
		return fmt.Errorf("Type UnmarshalXML: Token: %+v is not expected EndElement", et)
	}
	return fmt.Errorf("Type UnmarshalXML: Token: %+v is not expected EndElement", ct)
}

////////////////////////////////////////////////////////////////////////////////////////
//   Convenience functions for converting interface{} (e.g. properties) to given types
//     uses the "ok" bool mechanism to report failure, and is as robust and general as possible
//     WARNING: these violate many of the type-safety features of Go but OTOH give maximum
//     robustness, appropriate for the world of end-user settable properties, and deal with
//     most common-sense cases, e.g., string <-> number, etc.  nil values return !ok

// robustly convert anything to a bool
func ToBool(it interface{}) (bool, bool) {
	if it == nil {
		return false, false
	}
	v := reflect.ValueOf(it)
	vk := v.Kind()
	if vk == reflect.Ptr {
		v = v.Elem()
		vk = v.Kind()
	}
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return (v.Int() != 0), true
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return (v.Uint() != 0), true
	case vk == reflect.Bool:
		return v.Bool(), true
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return (v.Float() != 0.0), true
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return (real(v.Complex()) != 0.0), true
	case vk == reflect.String:
		r, err := strconv.ParseBool(v.String())
		if err != nil {
			return false, false
		}
		return r, true
	default:
		return false, false
	}
}

// robustly convert anything to an int64
func ToInt(it interface{}) (int64, bool) {
	if it == nil {
		return 0, false
	}
	v := reflect.ValueOf(it)
	vk := v.Kind()
	if vk == reflect.Ptr {
		v = v.Elem()
		vk = v.Kind()
	}
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return v.Int(), true
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return int64(v.Uint()), true
	case vk == reflect.Bool:
		if v.Bool() {
			return 1, true
		}
		return 0, true
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return int64(v.Float()), true
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return int64(real(v.Complex())), true
	case vk == reflect.String:
		r, err := strconv.ParseInt(v.String(), 0, 64)
		if err != nil {
			return 0, false
		}
		return r, true
	default:
		return 0, false
	}
}

// robustly convert anything to a Float64
func ToFloat(it interface{}) (float64, bool) {
	if it == nil {
		return 0.0, false
	}
	v := reflect.ValueOf(it)
	vk := v.Kind()
	if vk == reflect.Ptr {
		v = v.Elem()
		vk = v.Kind()
	}
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return float64(v.Int()), true
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return float64(v.Uint()), true
	case vk == reflect.Bool:
		if v.Bool() {
			return 1.0, true
		}
		return 0.0, true
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return v.Float(), true
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return real(v.Complex()), true
	case vk == reflect.String:
		r, err := strconv.ParseFloat(v.String(), 64)
		if err != nil {
			return 0.0, false
		}
		return r, true
	default:
		return 0.0, false
	}
}

// robustly convert anything to a String
func ToString(it interface{}) (string, bool) {
	if it == nil {
		return "", false
	}
	v := reflect.ValueOf(it)
	vk := v.Kind()
	if vk == reflect.Ptr {
		v = v.Elem()
		vk = v.Kind()
	}
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), true
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10), true
	case vk == reflect.Bool:
		return strconv.FormatBool(v.Bool()), true
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'G', -1, 64), true
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		cv := v.Complex()
		rv := strconv.FormatFloat(real(cv), 'G', -1, 64) + "," + strconv.FormatFloat(imag(cv), 'G', -1, 64)
		return rv, true
	case vk == reflect.String: // todo: what about []byte?
		return v.String(), true
	default:
		return "", false
	}
}

// math provides Max/Min for 64bit -- these are for specific subtypes

func Max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func Min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// minimum excluding 0
func MinPos(a, b float64) float64 {
	if a > 0.0 && b > 0.0 {
		return math.Min(a, b)
	} else if a > 0.0 {
		return a
	} else if b > 0.0 {
		return b
	}
	return a
}

////////////////////////////////////////////////////////////////////////////////////////
//   Types TypeRegistry

// TypeRegistry is a map from type name to reflect.Type -- need to explicitly register each new type by calling AddType in the process of creating a new global variable, as in:
// 	var KiT_TypeName = ki.Types.AddType(&TypeName{})
// 	where TypeName is the name of the type -- note that it is ESSENTIAL to pass a pointer
//  so that the type is considered addressable, even after we get Elem() of it
type TypeRegistry struct {
	// to get a type from its name
	Types map[string]reflect.Type
	// type properties -- nodes can get default properties from their types and then optionally override them with their own settings
	Props map[string]map[string]interface{}
}

// Types is master registry of types that embed Ki Nodes
var Types TypeRegistry

// AddType adds a given type to the registry -- requires an empty object to grab type info from -- must be passed as a pointer to ensure that it is an addressable, settable type -- also optional properties that can be associated with the type and accessible e.g. for view-specific properties etc
func (tr *TypeRegistry) AddType(obj interface{}, props map[string]interface{}) reflect.Type {
	if tr.Types == nil {
		tr.Types = make(map[string]reflect.Type)
		tr.Props = make(map[string]map[string]interface{})
	}

	typ := reflect.TypeOf(obj).Elem()
	tn := typ.Name()
	tr.Types[tn] = typ
	// fmt.Printf("added type: %v\n", tn)
	if props != nil {
		// fmt.Printf("added props: %v\n", tn)
		tr.Props[tn] = props
	}
	return typ
}

// FindType finds a type based on its name -- returns nil if not found
func (tr *TypeRegistry) FindType(name string) reflect.Type {
	return tr.Types[name]
}

// Properties returns properties for this type -- makes props map if not already made
func (tr *TypeRegistry) Properties(typeName string) map[string]interface{} {
	tp, ok := tr.Props[typeName]
	if !ok {
		tp = make(map[string]interface{})
		tr.Props[typeName] = tp
	}
	return tp
}

// Prop safely finds a type property from type name and property key -- nil if not found
func (tr *TypeRegistry) Prop(typeName, propKey string) interface{} {
	tp, ok := tr.Props[typeName]
	if !ok {
		// fmt.Printf("no props for type: %v\n", typeName)
		return nil
	}
	p, ok := tp[propKey]
	if !ok {
		// fmt.Printf("no props for key: %v\n", propKey)
		return nil
	}
	return p
}

////////////////////////////////////////////////////////////////////////////////////////
//   EnumRegistry and Enum <-> string support

// todo: suport bit-flag enums and composition of names as | of values, etc

// Bit flags are setup just using the ordinal count iota, and the only diff is the methods
// which do 1 << flag when operating on them

// todo: can add a global debug level setting and test for overflow in bits --
// or maybe better in the enum type registry constructor?
// see also https://github.com/sirupsen/logrus

// we assume 64bit bitflags by default -- 32 bit methods specifically marked

// set a bit value based on the ordinal flag value
func SetBitFlag(bits *int64, flag int) {
	*bits |= 1 << uint32(flag)
}

// set or clear a bit value depending on state (on / off) based on the ordinal flag value
func SetBitFlagState(bits *int64, flag int, state bool) {
	if state {
		SetBitFlag(bits, flag)
	} else {
		ClearBitFlag(bits, flag)
	}
}

// clear bit value based on the ordinal flag value
func ClearBitFlag(bits *int64, flag int) {
	*bits = *bits & ^(1 << uint32(flag)) // note: ^ is unary bitwise negation, not ~ as in C
}

// toggle state of bit value based on the ordinal flag value -- returns new state
func ToggleBitFlag(bits *int64, flag int) bool {
	if HasBitFlag(*bits, flag) {
		ClearBitFlag(bits, flag)
		return false
	} else {
		SetBitFlag(bits, flag)
		return true
	}
}

// check if given bit value is set for given flag
func HasBitFlag(bits int64, flag int) bool {
	return bits&(1<<uint32(flag)) != 0
}

// check if any of a set of flags are set
func HasBitFlags(bits int64, flags ...int) bool {
	for _, flg := range flags {
		if HasBitFlag(bits, flg) {
			return true
		}
	}
	return false
}

// make a mask for checking multiple different flags
func MakeBitMask(flags ...int) int64 {
	var mask int64
	for _, flg := range flags {
		SetBitFlag(&mask, flg)
	}
	return mask
}

// check if any of the bits in mask are set
func HasBitMask(bits, mask int64) bool {
	return bits&mask != 0
}

//////////////////////////////
//   32 bit

// set a bit value based on the ordinal flag value
func SetBitFlag32(bits *int32, flag int) {
	*bits |= 1 << uint32(flag)
}

// clear bit value based on the ordinal flag value
func ClearBitFlag32(bits *int32, flag int) {
	*bits = *bits & ^(1 << uint32(flag)) // note: ^ is unary bitwise negation, not ~ as in C
}

// toggle state of bit value based on the ordinal flag value -- returns new state
func ToggleBitFlag32(bits *int32, flag int) bool {
	if HasBitFlag32(*bits, flag) {
		ClearBitFlag32(bits, flag)
		return false
	} else {
		SetBitFlag32(bits, flag)
		return true
	}
}

// check if given bit value is set for given flag
func HasBitFlag32(bits int32, flag int) bool {
	return bits&(1<<uint32(flag)) != 0
}

// check if any of a set of flags are set
func HasBitFlags32(bits int32, flags ...int) bool {
	for _, flg := range flags {
		if HasBitFlag32(bits, flg) {
			return true
		}
	}
	return false
}

// make a mask for checking multiple different flags
func MakeBitMask32(flags ...int) int32 {
	var mask int32
	for _, flg := range flags {
		SetBitFlag32(&mask, flg)
	}
	return mask
}

// check if any of the bits in mask are set
func HasBitMask32(bits, mask int32) bool {
	return bits&mask != 0
}

// design notes: for methods that return string, not passing error b/c you can
// easily check for null string, and registering errors in log for setter
// methods, returning error and also logging so it is safe to ignore err if
// you don't care

// EnumRegistry is a map from an enum-style const int type name to a
// corresponding reflect.Type and conversion methods generated by (modified)
// stringer that convert to / from strings -- need to explicitly register each
// new type by calling AddEnum in the process of creating a new global
// variable, as in:
// var KiT_TypeName = ki.Enums.AddEnum(&TypeName{}, bitFlag true/false,
//    TypeNameProps (or nil))
// where TypeName is the name of the type, see below for BitFlag, and
// TypeNameProps is nil or a map[string]interface{} of properties, OR:
// var KiT_TypeName = ki.Enums.AddEnumAltLower(&TypeName{}, bitFlag true/false,
//    TypeNameProps, "Prefix", MyEnumN)
// which automatically registers alternative names as lower-case versions of
// const names with given prefix removed -- often what is used in e.g., json
// or xml kinds of formats
// special properties:
// * "BitFlag": true -- each value represents a bit in a set of bit flags, so
// the string rep of a value contains an or-list of names for each bit set,
// separated by |
// * "AltStrings": map[int64]string -- provides an alternative string mapping for
// the enum values
type EnumRegistry struct {
	Enums map[string]reflect.Type
	// properties that can be associated with each enum type -- e.g., "BitFlag": true  --  "AltStrings" : map[int64]string, or other custom settings
	Props map[string]map[string]interface{}
}

// Enums is master registry of enum types -- can also create your own package-specific ones
var Enums EnumRegistry

// AddEnum adds a given type to the registry -- requires an empty object to
// grab type info from -- if bitFlag then sets BitFlag property, and each
// value represents a bit in a set of bit flags, so the string rep of a value
// contains an or-list of names for each bit set, separated by | -- can also
// add additional properties
func (tr *EnumRegistry) AddEnum(obj interface{}, bitFlag bool, props map[string]interface{}) reflect.Type {
	if tr.Enums == nil {
		tr.Enums = make(map[string]reflect.Type)
		tr.Props = make(map[string]map[string]interface{})
	}

	// get the pointer-to version and elem so it is a settable type!
	typ := reflect.PtrTo(reflect.TypeOf(obj)).Elem()
	tn := typ.Name()
	tr.Enums[tn] = typ
	if props != nil {
		tr.Props[tn] = props
	}
	if bitFlag {
		tp := tr.Properties(tn)
		tp["BitFlag"] = true
	}
	// fmt.Printf("added enum: %v\n", tn)
	return typ
}

// AddEnumAltLower adds a given type to the registry -- requires an empty object to
// grab type info from -- automatically initializes AltStrings alternative string map
// based on the name with given prefix removed (e.g., a type name-based prefix)
// and lower-cased -- also requires the number of enums -- assumes starts at 0
func (tr *EnumRegistry) AddEnumAltLower(obj interface{}, bitFlag bool, props map[string]interface{}, prefix string, n int64) reflect.Type {
	typ := tr.AddEnum(obj, bitFlag, props)
	tn := typ.Name()
	alts := make(map[int64]string)
	tp := tr.Properties(tn)
	tp["AltStrings"] = alts
	for i := int64(0); i < n; i++ {
		str := EnumInt64ToString(i, typ)
		str = strings.ToLower(strings.TrimPrefix(str, prefix))
		// fmt.Printf("adding enum: %v\n", str)
		alts[i] = str
	}
	return typ
}

// FindEnum finds an enum type based on its type name -- returns nil if not found
func (tr *EnumRegistry) FindEnum(name string) reflect.Type {
	return tr.Enums[name]
}

// Props returns properties for this type -- makes props map if not already made
func (tr *EnumRegistry) Properties(enumName string) map[string]interface{} {
	tp, ok := tr.Props[enumName]
	if !ok {
		tp = make(map[string]interface{})
		tr.Props[enumName] = tp
	}
	return tp
}

// Prop safely finds an enum type property from enum type name and property key -- nil if not found
func (tr *EnumRegistry) Prop(enumName, propKey string) interface{} {
	tp, ok := tr.Props[enumName]
	if !ok {
		// fmt.Printf("no props for enum type: %v\n", enumName)
		return nil
	}
	p, ok := tp[propKey]
	if !ok {
		// fmt.Printf("no props for key: %v\n", propKey)
		return nil
	}
	return p
}

// get optional alternative string map for enums -- e.g., lower-case, without
// prefixes etc -- can put multiple such alt strings in the one string with
// your own separator, in a predefined order, if necessary, and just call
// strings.Split on those and get the one you want -- nil if not set
func (tr *EnumRegistry) AltStrings(enumName string) map[int64]string {
	ps := tr.Prop(enumName, "AltStrings")
	if ps == nil {
		return nil
	}
	m, ok := ps.(map[int64]string)
	if !ok {
		log.Printf("ki.EnumRegistry AltStrings error: AltStrings property must be a map[int64]string type, is not -- is instead: %T\n", m)
		return nil
	}
	return m
}

// check if this enum is for bit flags instead of mutually-exclusive int
// values -- checks BitFlag property -- if true string rep of a value contains
// an or-list of names for each bit set, separated by |
func (tr *EnumRegistry) IsBitFlag(enumName string) bool {
	b, _ := ToBool(tr.Prop(enumName, "BitFlag"))
	return b
}

// EnumToInt64 converts an enum into an int64 using reflect -- just use int64(eval) when you
// have the enum in hand -- this is when you just have a generic item
func EnumToInt64(eval interface{}) int64 {
	if reflect.TypeOf(eval).Kind() == reflect.Ptr {
		eval = reflect.ValueOf(eval).Elem() // deref the pointer
	}
	var ival int64
	reflect.ValueOf(&ival).Elem().Set(reflect.ValueOf(eval).Convert(reflect.TypeOf(ival)))
	return ival
}

// set enum value from int64 value -- must pass a pointer to the enum and also needs raw type
// of the enum as well -- can't get it from the interface{} reliably
func EnumFromInt64(eval interface{}, ival int64, typ reflect.Type) error {
	if reflect.TypeOf(eval).Kind() != reflect.Ptr {
		err := fmt.Errorf("ki.EnumFromInt64: must pass a pointer to the enum: Type: %v, Kind: %v\n", reflect.TypeOf(eval).Name(), reflect.TypeOf(eval).Kind())
		log.Printf("%v", err)
		return err
	}
	reflect.ValueOf(eval).Elem().Set(reflect.ValueOf(ival).Convert(typ))
	return nil
}

// First convert an int64 to enum of given type, and then convert to string value
func EnumInt64ToString(ival int64, typ reflect.Type) string {
	// evpi, evi := NewEnumFromType(typ) // note: same code, but works here and not in fun..
	evnp := reflect.New(reflect.PtrTo(typ))
	evpi := evnp.Interface()
	evn := reflect.New(typ)
	evi := evn.Interface()
	evpi = &evi
	EnumFromInt64(evpi, ival, typ)
	return EnumToString(evi)
}

// Enum to String converts an enum value to its corresponding string value --
// you could just call fmt.Sprintf("%v") too but this is slightly faster
func EnumToString(eval interface{}) string {
	if reflect.TypeOf(eval).Kind() == reflect.Ptr {
		eval = reflect.ValueOf(eval).Elem() // deref the pointer
	}
	strer, ok := eval.(fmt.Stringer) // will fail if not impl
	if !ok {
		log.Printf("ki.EnumToString: fmt.Stringer interface not supported by type %v\n", reflect.TypeOf(eval).Name())
		return ""
	}
	return strer.String()
}

// note: convenience methods b/c it is easier to find on registry type

// Enum to String converts an enum value to its corresponding string value --
// you could just call fmt.Sprintf("%v") too but this is slightly faster
func (tr *EnumRegistry) EnumToString(eval interface{}) string {
	return EnumToString(eval)
}

// First convert an int64 to enum of given type, and then convert to string value
func (tr *EnumRegistry) EnumInt64ToString(ival int64, typ reflect.Type) string {
	return EnumInt64ToString(ival, typ)
}

// Enum to alternative String value converts an enum value to its
// corresponding alternative string value
func (tr *EnumRegistry) EnumToAltString(eval interface{}) string {
	if reflect.TypeOf(eval).Kind() == reflect.Ptr {
		eval = reflect.ValueOf(eval).Elem() // deref the pointer
	}
	et := reflect.TypeOf(eval)
	alts := tr.AltStrings(et.Name())
	if alts == nil {
		log.Printf("ki.EnumToAltString: no alternative string map for type %v\n", et.Name())
		return ""
	}
	// convert to int64 for lookup
	ival := EnumToInt64(eval)
	return alts[ival]
}

// Enum to alternative String value converts an int64 to corresponding
// alternative string value, for given type name
func (tr *EnumRegistry) EnumInt64ToAltString(ival int64, typnm string) string {
	alts := tr.AltStrings(typnm)
	if alts == nil {
		log.Printf("ki.EnumInt64ToAltString: no alternative string map for type %v\n", typnm)
		return ""
	}
	return alts[ival]
}

// Enum from String Sets enum value from string -- must pass a *pointer* to
// the enum item. IMPORTANT: requires the modified stringer go generate utility
// that generates a StringToTypeName method
func SetEnumFromString(eptr interface{}, str string) error {
	etp := reflect.TypeOf(eptr)
	if etp.Kind() != reflect.Ptr {
		err := fmt.Errorf("ki.EnumFromString -- you must pass a pointer enum, not type: %v kind %v\n", etp, etp.Kind())
		log.Printf("%v", err)
		return err
	}
	et := etp.Elem()
	sv := reflect.ValueOf(str)
	methnm := "StringTo" + et.Name()
	meth := sv.MethodByName(methnm)
	if meth.IsNil() {
		err := fmt.Errorf("ki.EnumFromString: stringer-generated StringToX method not found: %v\n", methnm)
		log.Printf("%v", err)
		return err
	}
	args := make([]reflect.Value, 0, 1)
	args = append(args, sv)
	rv := meth.Call(args)
	reflect.ValueOf(eptr).Elem().Set(rv[0])
	return nil
}

// Set Enum from String Sets enum value from string -- must pass a *pointer* to
// the enum item. IMPORTANT: requires the modified stringer go generate utility
// that generates a StringToTypeName method
func (tr *EnumRegistry) SetEnumFromString(eptr interface{}, str string) error {
	return SetEnumFromString(eptr, str)
}

// Set  Enum from String using reflect.Value
// IMPORTANT: requires the modified stringer go generate utility
// that generates a StringToTypeName method
func SetEnumValueFromString(eval reflect.Value, str string) error {
	et := eval.Type()
	sv := reflect.ValueOf(str)
	methnm := "StringTo" + et.Name()
	meth := sv.MethodByName(methnm)
	if meth.IsNil() {
		err := fmt.Errorf("ki.EnumFromString: stringer-generated StringToX method not found: %v\n", methnm)
		log.Printf("%v", err)
		return err
	}
	args := make([]reflect.Value, 0, 1)
	args = append(args, sv)
	rv := meth.Call(args)
	eval.Set(rv[0])
	return nil
}

// Sets enum value from string, into a reflect.Value
// IMPORTANT: requires the modified stringer go generate utility
// that generates a StringToTypeName method
func (tr *EnumRegistry) SetEnumValueFromString(eval reflect.Value, str string) error {
	return SetEnumValueFromString(eval, str)
}

// Set Enum from alternative String -- must pass a *pointer* to the enum item.
func (tr *EnumRegistry) SetEnumFromAltString(eptr interface{}, str string) error {
	etp := reflect.TypeOf(eptr)
	if etp.Kind() != reflect.Ptr {
		err := fmt.Errorf("ki.EnumFromString -- you must pass a pointer enum, not type: %v kind %v\n", etp, etp.Kind())
		log.Printf("%v", err)
		return err
	}
	et := etp.Elem()
	alts := tr.AltStrings(et.Name())
	if alts == nil {
		err := fmt.Errorf("ki.EnumFromAltString: no alternative string map for type %v\n", et.Name())
		log.Printf("%v", err)
		return err
	}
	for i, v := range alts {
		if v == str {
			reflect.ValueOf(eptr).Elem().Set(reflect.ValueOf(i).Convert(et))
			return nil
		}
	}
	err := fmt.Errorf("ki.EnumFromAltString: string: %v not found in alt list of strings for type%v\n", str, et.Name())
	log.Printf("%v", err)
	return err
}

// Set Enum from alternative String using a reflect.Value -- must pass a *pointer* to the enum item.
func (tr *EnumRegistry) SetEnumValueFromAltString(eval reflect.Value, str string) error {
	et := eval.Type()
	alts := tr.AltStrings(et.Name())
	if alts == nil {
		err := fmt.Errorf("ki.SetEnumValueFromAltString: no alternative string map for type %v\n", et.Name())
		log.Printf("%v", err)
		return err
	}
	for i, v := range alts {
		if v == str {
			eval.Set(reflect.ValueOf(i).Convert(et))
			return nil
		}
	}
	err := fmt.Errorf("ki.SetEnumValueFromAltString: string: %v not found in alt list of strings for type%v\n", str, et.Name())
	log.Printf("%v", err)
	return err
}

// Set Enum from int64 value
func (tr *EnumRegistry) SetEnumValueFromInt64(eval reflect.Value, ival int64) error {
	evi := eval.Interface()
	et := eval.Type()
	return EnumFromInt64(&evi, ival, et)
}

// get a pointer and the enum itself as interface{}'s, based on the type
// todo: can't seem to get this to work -- just replicate the code per below
// func NewEnumFromType(typ reflect.Type) (interface{}, interface{}) {
// 	evnp := reflect.New(reflect.PtrTo(typ))
// 	evpi := evnp.Interface()
// 	evn := reflect.New(typ)
// 	evi := evn.Interface()
// 	evpi = &evi
// 	return evpi, evi
// }
