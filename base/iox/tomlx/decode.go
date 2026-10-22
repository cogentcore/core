// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tomlx

import (
	"encoding"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/iox"
	"github.com/BurntSushi/toml"
)

// The decoder parses TOML with [toml.Decode] into generic values, and then
// sets them on the target value, so that it reads all of the files written
// by the pelletier/go-toml/v2 encoder that tomlx used before: map keys of
// any type with a text form (strings, numbers, and types implementing
// [encoding.TextUnmarshaler], such as enums) are supported, and
// [time.Duration] values can be integer nanoseconds or duration strings.

var (
	durationType        = reflect.TypeFor[time.Duration]()
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
)

// decoder implements [iox.Decoder] for TOML.
type decoder struct {
	r io.Reader
}

// NewDecoder returns a new [iox.Decoder]
func NewDecoder(r io.Reader) iox.Decoder { return &decoder{r: r} }

// Decode reads all of the TOML data from the reader and sets the
// corresponding values on v, which must be a non-nil pointer.
// Struct fields and map entries that are not in the data are not changed,
// keys that do not match a field are ignored, and slices are replaced.
func (d *decoder) Decode(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("toml: cannot decode into non-pointer or nil %T", v)
	}
	b, err := io.ReadAll(d.r)
	if err != nil {
		return err
	}
	var data map[string]any
	if _, err := toml.Decode(string(b), &data); err != nil {
		return err
	}
	return decodeValue(rv, data, "")
}

// decodeValue sets the given value from the given generic TOML data
// (as returned by [toml.Decode]) at the given dotted key path.
func decodeValue(v reflect.Value, data any, path string) error {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return decodeValue(v.Elem(), data, path)
	case reflect.Interface:
		return decodeInterface(v, data, path)
	}
	if s, ok := data.(string); ok && v.CanAddr() && reflect.PointerTo(v.Type()).Implements(textUnmarshalerType) {
		if err := v.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(s)); err != nil {
			return fmt.Errorf("toml: %s: %w", path, err)
		}
		return nil
	}
	switch v.Type() {
	case durationType:
		return decodeDuration(v, data, path)
	case timeType:
		if t, ok := data.(time.Time); ok {
			v.Set(reflect.ValueOf(t))
			return nil
		}
		return typeError(v, data, path)
	}

	switch v.Kind() {
	case reflect.Struct:
		return decodeStruct(v, data, path)
	case reflect.Map:
		return decodeMap(v, data, path)
	case reflect.Slice:
		return decodeSlice(v, data, path)
	case reflect.Array:
		return decodeArray(v, data, path)
	case reflect.String:
		if s, ok := data.(string); ok {
			v.SetString(s)
			return nil
		}
	case reflect.Bool:
		if b, ok := data.(bool); ok {
			v.SetBool(b)
			return nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i, ok := toInt(data); ok {
			if v.OverflowInt(i) {
				return fmt.Errorf("toml: %s: %d overflows %s", path, i, v.Type())
			}
			v.SetInt(i)
			return nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if i, ok := toInt(data); ok {
			if i < 0 || v.OverflowUint(uint64(i)) {
				return fmt.Errorf("toml: %s: %d overflows %s", path, i, v.Type())
			}
			v.SetUint(uint64(i))
			return nil
		}
	case reflect.Float32, reflect.Float64:
		switch x := data.(type) {
		case float64:
			v.SetFloat(x)
			return nil
		case int64:
			v.SetFloat(float64(x))
			return nil
		}
	case reflect.Func, reflect.Chan, reflect.UnsafePointer, reflect.Complex64, reflect.Complex128:
		return nil // not represented in TOML
	}
	return typeError(v, data, path)
}

// typeError returns an error for TOML data that cannot be set on the value.
func typeError(v reflect.Value, data any, path string) error {
	return fmt.Errorf("toml: %s: cannot decode TOML %s into %s", path, tomlTypeName(data), v.Type())
}

// tomlTypeName returns the TOML type name of the given generic data.
func tomlTypeName(data any) string {
	switch data.(type) {
	case string:
		return "string"
	case int64:
		return "integer"
	case float64:
		return "float"
	case bool:
		return "boolean"
	case time.Time:
		return "datetime"
	case []any:
		return "array"
	case []map[string]any:
		return "array of tables"
	case map[string]any:
		return "table"
	}
	return fmt.Sprintf("%T", data)
}

// toInt returns the given integer data, or float data with an integer value.
func toInt(data any) (int64, bool) {
	switch x := data.(type) {
	case int64:
		return x, true
	case float64:
		if x == math.Trunc(x) && x >= math.MinInt64 && x < math.MaxInt64 {
			return int64(x), true
		}
	}
	return 0, false
}

// decodeDuration sets the given [time.Duration] value from integer
// nanoseconds (as written by tomlx) or a duration string such as "1.5s".
func decodeDuration(v reflect.Value, data any, path string) error {
	if s, ok := data.(string); ok {
		d, err := time.ParseDuration(s)
		if err != nil {
			return fmt.Errorf("toml: %s: %w", path, err)
		}
		v.SetInt(int64(d))
		return nil
	}
	if i, ok := toInt(data); ok {
		v.SetInt(i)
		return nil
	}
	return typeError(v, data, path)
}

// decodeInterface sets the given interface value. If it holds a non-nil
// pointer, the data is decoded into that; otherwise, the generic value is
// set, with arrays of tables as []any.
func decodeInterface(v reflect.Value, data any, path string) error {
	if !v.IsNil() && v.Elem().Kind() == reflect.Pointer && !v.Elem().IsNil() {
		return decodeValue(v.Elem(), data, path)
	}
	gv := reflect.ValueOf(genericValue(data))
	if !gv.Type().AssignableTo(v.Type()) {
		return typeError(v, data, path)
	}
	v.Set(gv)
	return nil
}

// genericValue returns the given data with arrays of tables
// converted to []any, so that all arrays are []any.
func genericValue(data any) any {
	switch x := data.(type) {
	case []map[string]any:
		a := make([]any, len(x))
		for i, m := range x {
			a[i] = genericValue(m)
		}
		return a
	case []any:
		for i, e := range x {
			x[i] = genericValue(e)
		}
	case map[string]any:
		for k, e := range x {
			x[k] = genericValue(e)
		}
	}
	return data
}

// decodeStruct sets the fields of the given struct value from table data.
// Keys match field names (or toml tag names) exactly, or else ignoring case.
func decodeStruct(v reflect.Value, data any, path string) error {
	m, ok := data.(map[string]any)
	if !ok {
		return typeError(v, data, path)
	}
	fields := structFields(v.Type())
	for k, d := range m {
		fi, ok := fields.byName[k]
		if !ok {
			fi, ok = fields.byFold[strings.ToLower(k)]
		}
		if !ok {
			continue
		}
		f, ok := fieldByIndex(v, fi)
		if !ok {
			continue
		}
		if err := decodeValue(f, d, joinPath(path, k)); err != nil {
			return err
		}
	}
	return nil
}

// fieldByIndex returns the field of the given struct value with the given
// index path, allocating nil embedded struct pointers along the way.
func fieldByIndex(v reflect.Value, index []int) (reflect.Value, bool) {
	for i, x := range index {
		if i > 0 && v.Kind() == reflect.Pointer {
			if v.IsNil() {
				if !v.CanSet() {
					return reflect.Value{}, false
				}
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(x)
	}
	return v, v.CanSet()
}

// fieldMap maps TOML keys to the index paths of struct fields.
type fieldMap struct {
	byName map[string][]int
	byFold map[string][]int
}

// fieldMaps caches the [fieldMap] for each struct type.
var fieldMaps sync.Map

// structFields returns the [fieldMap] for the given struct type.
// The fields of untagged embedded structs are included as if they
// were fields of the outer struct, unless an outer field has the same name.
// The embedded structs themselves are included by their type names,
// with the lowest priority.
func structFields(typ reflect.Type) *fieldMap {
	if fm, ok := fieldMaps.Load(typ); ok {
		return fm.(*fieldMap)
	}
	fm := &fieldMap{byName: map[string][]int{}, byFold: map[string][]int{}}
	addStructFields(fm, typ, nil)
	fieldMaps.Store(typ, fm)
	return fm
}

// addStructFields adds the fields of the given struct type to the field map,
// with the given index path prefix, outer fields before embedded ones.
func addStructFields(fm *fieldMap, typ reflect.Type, prefix []int) {
	var embedded []reflect.StructField
	for i := range typ.NumField() {
		sf := typ.Field(i)
		name, _, skip := parseTag(sf)
		if skip {
			continue
		}
		if name == "" && sf.Anonymous {
			ft := sf.Type
			if ft.Kind() == reflect.Pointer && sf.IsExported() {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				embedded = append(embedded, sf)
			}
			continue
		}
		if !sf.IsExported() {
			continue
		}
		if name == "" {
			name = sf.Name
		}
		index := append(append([]int{}, prefix...), i)
		if _, ok := fm.byName[name]; !ok {
			fm.byName[name] = index
		}
		if _, ok := fm.byFold[strings.ToLower(name)]; !ok {
			fm.byFold[strings.ToLower(name)] = index
		}
	}
	for _, sf := range embedded {
		ft := sf.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		addStructFields(fm, ft, append(append([]int{}, prefix...), sf.Index...))
	}
	// An embedded struct can also be set as a table with its type name, as
	// reflectx.NonDefaultFields makes for settings, if no field has that name.
	for _, sf := range embedded {
		if !sf.IsExported() {
			continue
		}
		index := append(append([]int{}, prefix...), sf.Index...)
		if _, ok := fm.byName[sf.Name]; !ok {
			fm.byName[sf.Name] = index
		}
		if _, ok := fm.byFold[strings.ToLower(sf.Name)]; !ok {
			fm.byFold[strings.ToLower(sf.Name)] = index
		}
	}
}

// joinPath returns the given dotted key path with the given key added.
func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// decodeMap sets entries of the given map value from table data,
// making the map if it is nil. Each entry is decoded into a new value.
func decodeMap(v reflect.Value, data any, path string) error {
	m, ok := data.(map[string]any)
	if !ok {
		return typeError(v, data, path)
	}
	if v.IsNil() {
		v.Set(reflect.MakeMapWithSize(v.Type(), len(m)))
	}
	kt, et := v.Type().Key(), v.Type().Elem()
	for k, d := range m {
		kv := reflect.New(kt).Elem()
		if err := setMapKey(kv, k); err != nil {
			return fmt.Errorf("toml: %s: map key %q: %w", path, k, err)
		}
		ev := reflect.New(et).Elem()
		if err := decodeValue(ev, d, joinPath(path, k)); err != nil {
			return err
		}
		v.SetMapIndex(kv, ev)
	}
	return nil
}

// setMapKey sets the given map key value from its string form.
func setMapKey(k reflect.Value, s string) error {
	if reflect.PointerTo(k.Type()).Implements(textUnmarshalerType) {
		return k.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(s))
	}
	switch k.Kind() {
	case reflect.String:
		k.SetString(s)
	case reflect.Interface:
		k.Set(reflect.ValueOf(s))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, k.Type().Bits())
		if err != nil {
			return err
		}
		k.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		i, err := strconv.ParseUint(s, 10, k.Type().Bits())
		if err != nil {
			return err
		}
		k.SetUint(i)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, k.Type().Bits())
		if err != nil {
			return err
		}
		k.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		k.SetBool(b)
	default:
		return fmt.Errorf("unsupported map key type %s", k.Type())
	}
	return nil
}

// arrayData returns the elements of the given array data.
func arrayData(data any) ([]any, bool) {
	switch x := data.(type) {
	case []any:
		return x, true
	case []map[string]any:
		a := make([]any, len(x))
		for i, m := range x {
			a[i] = m
		}
		return a, true
	}
	return nil, false
}

// decodeSlice sets the given slice value to a new slice from array data,
// or from a string for a byte slice.
func decodeSlice(v reflect.Value, data any, path string) error {
	if s, ok := data.(string); ok && v.Type().Elem().Kind() == reflect.Uint8 {
		v.Set(reflect.ValueOf([]byte(s)).Convert(v.Type()))
		return nil
	}
	a, ok := arrayData(data)
	if !ok {
		return typeError(v, data, path)
	}
	sv := reflect.MakeSlice(v.Type(), len(a), len(a))
	for i, d := range a {
		if err := decodeValue(sv.Index(i), d, path+"["+strconv.Itoa(i)+"]"); err != nil {
			return err
		}
	}
	v.Set(sv)
	return nil
}

// decodeArray sets the elements of the given array value from array data;
// elements beyond the data are set to zero, and data beyond the array
// length is ignored.
func decodeArray(v reflect.Value, data any, path string) error {
	a, ok := arrayData(data)
	if !ok {
		return typeError(v, data, path)
	}
	for i := range v.Len() {
		ev := v.Index(i)
		ev.SetZero()
		if i < len(a) {
			if err := decodeValue(ev, a[i], path+"["+strconv.Itoa(i)+"]"); err != nil {
				return err
			}
		}
	}
	return nil
}
