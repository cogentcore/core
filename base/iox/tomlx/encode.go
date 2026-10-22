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
	"slices"
	"strconv"
	"strings"
	"time"

	"cogentcore.org/core/base/iox"
)

// The encoder writes the same format as the pelletier/go-toml/v2 encoder
// (with indented tables and multiline arrays) that tomlx used before,
// so that files written by this version can be read by older versions:
// [time.Duration] values are written as integer nanoseconds, map keys of
// non-string types are written in their text form, strings use literal
// (single-quoted) form when possible, nil slices are written as empty
// arrays, and the simple values of a table come before its sub-tables,
// in struct field order (or sorted key order for maps).

var (
	timeType          = reflect.TypeFor[time.Time]()
	textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()
)

// indentSymbol is the indentation for each level of nested tables.
const indentSymbol = "  "

// encoder implements [iox.Encoder] for TOML.
type encoder struct {
	w io.Writer
}

// NewEncoder returns a new [iox.Encoder]
func NewEncoder(w io.Writer) iox.Encoder {
	return &encoder{w: w}
}

// Encode writes the TOML encoding of v, which must be a struct or a map
// (or a pointer to one), as TOML documents are tables.
func (e *encoder) Encode(v any) error {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		rv = rv.Elem()
	}
	if (rv.Kind() != reflect.Struct && rv.Kind() != reflect.Map) || isText(rv) {
		return fmt.Errorf("toml: cannot encode %T as a TOML document: it must be a struct or a map", v)
	}
	b, err := encodeTable(nil, encodeContext{}, rv)
	if err != nil {
		return err
	}
	_, err = e.w.Write(b)
	return err
}

// encodeContext is the context in which a value is encoded.
type encodeContext struct {

	// keys is the full key path of the current table.
	keys []string

	// indent is the current indentation level.
	indent int

	// inline is whether the value is part of an inline value,
	// such that tables are written as inline tables.
	inline bool

	// nested is whether the value is in an array, which makes
	// arrays use a single line.
	nested bool
}

// entry is a key and value in a table.
type entry struct {
	key   string
	value reflect.Value
}

// table has the simple key-value entries and the sub-table entries
// (including arrays of tables) of a table, in output order.
type table struct {
	kvs    []entry
	tables []entry
}

// add adds the given entry to the table.
func (t *table) add(ctx encodeContext, key string, v reflect.Value) {
	if !ctx.inline && isTableOrArrayTable(v) {
		t.tables = append(t.tables, entry{key, v})
	} else {
		t.kvs = append(t.kvs, entry{key, v})
	}
}

// makeTable returns the table entries of the given struct or map value.
func makeTable(ctx encodeContext, v reflect.Value) (*table, error) {
	t := &table{}
	if v.Kind() == reflect.Struct {
		walkStruct(ctx, t, v)
		return t, nil
	}
	iter := v.MapRange()
	for iter.Next() {
		mv := iter.Value()
		if isNil(mv) || isUnsupported(mv) {
			continue
		}
		ks, err := keyToString(iter.Key())
		if err != nil {
			return nil, err
		}
		t.add(ctx, ks, mv)
	}
	byKey := func(a, b entry) int { return strings.Compare(a.key, b.key) }
	slices.SortFunc(t.kvs, byKey)
	slices.SortFunc(t.tables, byKey)
	return t, nil
}

// walkStruct adds the exported fields of the given struct value to the table,
// with untagged embedded structs contributing their fields directly.
func walkStruct(ctx encodeContext, t *table, v reflect.Value) {
	typ := v.Type()
	for i := range typ.NumField() {
		sf := typ.Field(i)
		name, omitEmpty, skip := parseTag(sf)
		if skip {
			continue
		}
		f := v.Field(i)
		if name == "" && sf.Anonymous {
			if f.Kind() == reflect.Pointer && sf.IsExported() && !f.IsNil() {
				f = f.Elem()
			}
			if f.Kind() == reflect.Struct {
				walkStruct(ctx, t, f)
			}
			continue
		}
		if !sf.IsExported() {
			continue
		}
		if name == "" {
			name = sf.Name
		}
		if isNil(f) || isUnsupported(f) || (omitEmpty && f.IsZero()) {
			continue
		}
		t.add(ctx, name, f)
	}
}

// parseTag returns the key name from the toml tag of the given field,
// and whether the field is omitted when empty or skipped entirely.
func parseTag(sf reflect.StructField) (name string, omitEmpty, skip bool) {
	tag := sf.Tag.Get("toml")
	if tag == "-" {
		return "", false, true
	}
	name, opts, _ := strings.Cut(tag, ",")
	for opt := range strings.SplitSeq(opts, ",") {
		if opt == "omitempty" {
			omitEmpty = true
		}
	}
	return name, omitEmpty, false
}

// encodeTable encodes the given struct or map value as a table,
// with a header unless it is the root table.
func encodeTable(b []byte, ctx encodeContext, v reflect.Value) ([]byte, error) {
	t, err := makeTable(ctx, v)
	if err != nil {
		return nil, err
	}
	if len(ctx.keys) > 0 {
		b = appendIndent(b, ctx.indent)
		b = appendKeyPath(append(b, '['), ctx.keys)
		b = append(b, "]\n"...)
		ctx.indent++
	}
	return encodeTableBody(b, ctx, t)
}

// encodeTableBody encodes the entries of the given table,
// with the simple values first and then the sub-tables.
func encodeTableBody(b []byte, ctx encodeContext, t *table) ([]byte, error) {
	var err error
	for _, kv := range t.kvs {
		b = appendIndent(b, ctx.indent)
		b, err = encodeKeyValue(b, ctx, kv)
		if err != nil {
			return nil, err
		}
		b = append(b, '\n')
	}
	for i, st := range t.tables {
		if i > 0 || len(t.kvs) > 0 {
			b = append(b, '\n')
		}
		sctx := ctx
		sctx.keys = append(slices.Clip(ctx.keys), st.key)
		b, err = encodeSubTable(b, sctx, st.value)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

// encodeSubTable encodes the given value, which is a table or an array
// of tables, under the key path of the context.
func encodeSubTable(b []byte, ctx encodeContext, v reflect.Value) ([]byte, error) {
	v = indirect(v)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return encodeTable(b, ctx, v)
	}
	var header []byte
	header = appendIndent(header, ctx.indent)
	header = appendKeyPath(append(header, "[["...), ctx.keys)
	header = append(header, "]]\n"...)
	ctx.indent++
	for i := range v.Len() {
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, header...)
		t, err := makeTable(ctx, indirect(v.Index(i)))
		if err != nil {
			return nil, err
		}
		b, err = encodeTableBody(b, ctx, t)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

// encodeKeyValue encodes the given entry as a key = value pair.
func encodeKeyValue(b []byte, ctx encodeContext, kv entry) ([]byte, error) {
	b = appendKey(b, kv.key)
	b = append(b, " = "...)
	ctx.inline = true
	return encodeValue(b, ctx, kv.value)
}

// encodeValue encodes the given value inline, as the value of a key
// or an element of an array.
func encodeValue(b []byte, ctx encodeContext, v reflect.Value) ([]byte, error) {
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil, fmt.Errorf("toml: cannot encode a nil interface value at %q", strings.Join(ctx.keys, "."))
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v = reflect.Zero(v.Type().Elem())
		} else {
			v = v.Elem()
		}
	}
	if v.Type() == timeType {
		t := v.Interface().(time.Time)
		if t.Nanosecond() > 0 {
			return t.AppendFormat(b, time.RFC3339Nano), nil
		}
		return t.AppendFormat(b, time.RFC3339), nil
	}
	if isText(v) {
		text, err := textMarshaler(v).MarshalText()
		if err != nil {
			return nil, err
		}
		return appendString(b, string(text)), nil
	}

	switch v.Kind() {
	case reflect.Map, reflect.Struct:
		return encodeInlineTable(b, ctx, v)
	case reflect.Slice, reflect.Array:
		return encodeArray(b, ctx, v)
	case reflect.String:
		return appendString(b, v.String()), nil
	case reflect.Bool:
		return strconv.AppendBool(b, v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.AppendInt(b, v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if v.Uint() > math.MaxInt64 {
			return nil, fmt.Errorf("toml: cannot encode uint %d greater than the maximum int64", v.Uint())
		}
		return strconv.AppendUint(b, v.Uint(), 10), nil
	case reflect.Float32:
		return appendFloat(b, v.Float(), 32), nil
	case reflect.Float64:
		return appendFloat(b, v.Float(), 64), nil
	}
	return nil, fmt.Errorf("toml: cannot encode a value of type %s", v.Type())
}

// encodeInlineTable encodes the given struct or map as an inline table.
func encodeInlineTable(b []byte, ctx encodeContext, v reflect.Value) ([]byte, error) {
	t, err := makeTable(ctx, v)
	if err != nil {
		return nil, err
	}
	b = append(b, '{')
	for i, kv := range t.kvs {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = encodeKeyValue(b, ctx, kv)
		if err != nil {
			return nil, err
		}
	}
	return append(b, '}'), nil
}

// encodeArray encodes the given slice or array as an array, with one element
// per line unless it is empty or nested in another array.
func encodeArray(b []byte, ctx encodeContext, v reflect.Value) ([]byte, error) {
	n := v.Len()
	if n == 0 {
		return append(b, "[]"...), nil
	}
	multiline := !ctx.nested
	ectx := ctx
	ectx.nested = true
	b = append(b, '[')
	var err error
	for i := range n {
		if i > 0 {
			b = append(b, ',')
			if !multiline {
				b = append(b, ' ')
			}
		}
		if multiline {
			b = appendIndent(append(b, '\n'), ctx.indent+1)
		}
		b, err = encodeValue(b, ectx, v.Index(i))
		if err != nil {
			return nil, err
		}
	}
	if multiline {
		b = appendIndent(append(b, '\n'), ctx.indent)
	}
	return append(b, ']'), nil
}

// isTableOrArrayTable returns whether the given value is encoded as a table
// or as an array of tables: a non-empty array of tables.
func isTableOrArrayTable(v reflect.Value) bool {
	v = indirect(v)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return isTable(v)
	}
	if v.Len() == 0 {
		return false
	}
	for i := range v.Len() {
		if !isTable(v.Index(i)) {
			return false
		}
	}
	return true
}

// isTable returns whether the given value is encoded as a table.
func isTable(v reflect.Value) bool {
	v = indirect(v)
	if !v.IsValid() || v.Type() == timeType || isText(v) {
		return false
	}
	return v.Kind() == reflect.Struct || v.Kind() == reflect.Map
}

// indirect returns the value that the given value points to, following
// pointers and interfaces, or an invalid value for nil.
func indirect(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

// isText returns whether the given value implements [encoding.TextMarshaler],
// directly or through a pointer if it is addressable.
func isText(v reflect.Value) bool {
	return v.Type().Implements(textMarshalerType) || (v.CanAddr() && reflect.PointerTo(v.Type()).Implements(textMarshalerType))
}

// textMarshaler returns the [encoding.TextMarshaler] of the given value,
// for which [isText] is true.
func textMarshaler(v reflect.Value) encoding.TextMarshaler {
	if !v.Type().Implements(textMarshalerType) {
		v = v.Addr()
	}
	return v.Interface().(encoding.TextMarshaler)
}

// isNil returns whether the given value is a nil pointer, interface, or map,
// which are not encoded. Nil slices are encoded as empty arrays.
func isNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map:
		return v.IsNil()
	}
	return false
}

// isUnsupported returns whether the given value has a kind that
// cannot be represented in TOML, so it is skipped.
func isUnsupported(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Chan, reflect.UnsafePointer, reflect.Complex64, reflect.Complex128:
		return true
	}
	return false
}

// keyToString returns the string form of the given map key.
func keyToString(k reflect.Value) (string, error) {
	if k.Kind() == reflect.Interface {
		k = k.Elem()
	}
	switch {
	case k.Kind() == reflect.String:
		return k.String(), nil
	case k.Type().Implements(textMarshalerType):
		text, err := k.Interface().(encoding.TextMarshaler).MarshalText()
		return string(text), err
	}
	switch k.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(k.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(k.Uint(), 10), nil
	case reflect.Float32:
		return strconv.FormatFloat(k.Float(), 'f', -1, 32), nil
	case reflect.Float64:
		return strconv.FormatFloat(k.Float(), 'f', -1, 64), nil
	case reflect.Bool:
		return strconv.FormatBool(k.Bool()), nil
	}
	return "", fmt.Errorf("toml: cannot encode a map key of type %s", k.Type())
}

// appendFloat appends the given float, which always has a decimal point
// so that it is read as a float.
func appendFloat(b []byte, f float64, bits int) []byte {
	switch {
	case math.IsNaN(f):
		return append(b, "nan"...)
	case math.IsInf(f, 1):
		return append(b, "inf"...)
	case math.IsInf(f, -1):
		return append(b, "-inf"...)
	case math.Trunc(f) == f:
		return strconv.AppendFloat(b, f, 'f', 1, bits)
	}
	return strconv.AppendFloat(b, f, 'f', -1, bits)
}

// appendIndent appends the indentation for the given level.
func appendIndent(b []byte, level int) []byte {
	for range level {
		b = append(b, indentSymbol...)
	}
	return b
}

// appendKeyPath appends the given dotted key path.
func appendKeyPath(b []byte, keys []string) []byte {
	for i, k := range keys {
		if i > 0 {
			b = append(b, '.')
		}
		b = appendKey(b, k)
	}
	return b
}

// appendKey appends the given key, which is bare if it only has
// letters, digits, - and _, and quoted otherwise.
func appendKey(b []byte, k string) []byte {
	bare := k != ""
	for _, c := range k {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			bare = false
			break
		}
	}
	if bare {
		return append(b, k...)
	}
	return appendString(b, k)
}

// appendString appends the given string as a literal (single-quoted)
// string if possible, and as a basic (double-quoted) string otherwise.
func appendString(b []byte, s string) []byte {
	if !needsQuoting(s) {
		b = append(b, '\'')
		b = append(b, s...)
		return append(b, '\'')
	}
	const hex = "0123456789ABCDEF"
	b = append(b, '"')
	for i := range len(s) {
		c := s[i]
		switch c {
		case '\\':
			b = append(b, `\\`...)
		case '"':
			b = append(b, `\"`...)
		case '\b':
			b = append(b, `\b`...)
		case '\f':
			b = append(b, `\f`...)
		case '\n':
			b = append(b, `\n`...)
		case '\r':
			b = append(b, `\r`...)
		case '\t':
			b = append(b, `\t`...)
		default:
			if c < 0x20 || c == 0x7f {
				b = append(b, `\u00`...)
				b = append(b, hex[c>>4], hex[c&0x0f])
			} else {
				b = append(b, c)
			}
		}
	}
	return append(b, '"')
}

// needsQuoting returns whether the given string cannot be
// written as a literal string.
func needsQuoting(s string) bool {
	for i := range len(s) {
		c := s[i]
		if c == '\'' || (c < 0x20 && c != '\t') || c == 0x7f {
			return true
		}
	}
	return false
}
