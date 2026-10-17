// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package copiercheck compares reflectx.CopyStructDeep with
// github.com/jinzhu/copier on every tree.Node type in core.
package copiercheck

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unsafe"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz"
	"github.com/jinzhu/copier"
)

// filler fills values with deterministic representative data.
type filler struct {
	n        int
	maxDepth int
}

func (f *filler) next() int { f.n++; return f.n }

// settable returns a settable version of v, even for unexported fields.
func settable(v reflect.Value) reflect.Value {
	if v.CanSet() {
		return v
	}
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

var nodeBaseType = reflect.TypeFor[tree.NodeBase]()

var skipFields = map[string]bool{"This": true, "Parent": true, "Children": true}

var anyValues = []func(f *filler) any{
	func(f *filler) any { x := f.next(); return &x },
	func(f *filler) any { return map[string]int{"a": f.next(), "b": f.next()} },
	func(f *filler) any { return struct{ A []int }{A: []int{f.next(), f.next()}} },
	func(f *filler) any { return []int{f.next(), f.next()} },
	func(f *filler) any { return f.next() },
	func(f *filler) any { return nil },
	func(f *filler) any { return &struct{ B []string }{B: []string{"x"}} },
}

func newValue(t reflect.Type) (v reflect.Value, ok bool) {
	defer func() { recover() }()
	return reflect.New(t), true
}

func (f *filler) fill(v reflect.Value, depth int) {
	v = settable(v)
	n := f.next()
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(n%2 == 0)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(int64(n % 100))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v.SetUint(uint64(n % 100))
	case reflect.Float32, reflect.Float64:
		v.SetFloat(float64(n) + 0.5)
	case reflect.Complex64, reflect.Complex128:
		v.SetComplex(complex(float64(n), 1))
	case reflect.String:
		v.SetString(fmt.Sprintf("s%d", n))
	case reflect.Array:
		for i := range v.Len() {
			f.fill(v.Index(i), depth+1)
		}
	case reflect.Struct:
		t := v.Type()
		for i := range t.NumField() {
			if skipFields[t.Field(i).Name] && t.Field(i).Type.PkgPath() == "cogentcore.org/core/tree" {
				continue
			}
			if t.Field(i).Name == "This" || t.Field(i).Name == "Parent" || t.Field(i).Name == "Children" {
				continue
			}
			f.fill(v.Field(i), depth)
		}
	case reflect.Pointer:
		if depth >= f.maxDepth || n%5 == 0 {
			if n%2 == 0 {
				v.Set(reflect.Zero(v.Type()))
			}
			return
		}
		if v.IsNil() {
			nv, ok := newValue(v.Type().Elem())
			if !ok {
				return
			}
			v.Set(nv)
		}
		f.fill(v.Elem(), depth+1)
	case reflect.Slice:
		if depth >= f.maxDepth || n%7 == 0 {
			if n%2 == 0 {
				v.Set(reflect.Zero(v.Type()))
			} else {
				v.Set(reflect.MakeSlice(v.Type(), 0, 0))
			}
			return
		}
		m := 1 + n%3
		s := reflect.MakeSlice(v.Type(), m, m)
		for i := range m {
			f.fill(s.Index(i), depth+1)
		}
		v.Set(s)
	case reflect.Map:
		if depth >= f.maxDepth || n%7 == 0 {
			if n%2 == 0 {
				v.Set(reflect.Zero(v.Type()))
			} else {
				v.Set(reflect.MakeMap(v.Type()))
			}
			return
		}
		mp := reflect.MakeMap(v.Type())
		for range 2 {
			k := reflect.New(v.Type().Key()).Elem()
			f.fill(k, f.maxDepth) // keys stay shallow
			e := reflect.New(v.Type().Elem()).Elem()
			f.fill(e, depth+1)
			mp.SetMapIndex(k, e)
		}
		v.Set(mp)
	case reflect.Interface:
		if v.Type().NumMethod() == 0 {
			x := anyValues[n%len(anyValues)](f)
			if x == nil {
				v.Set(reflect.Zero(v.Type()))
			} else {
				v.Set(reflect.ValueOf(x))
			}
		} else if v.IsNil() {
			for _, p := range nodeTypes {
				if pt := reflect.TypeOf(p); pt.Implements(v.Type()) {
					v.Set(reflect.New(pt.Elem()))
					break
				}
			}
		}
	case reflect.Func:
		if v.IsNil() {
			v.Set(reflect.MakeFunc(v.Type(), func(args []reflect.Value) []reflect.Value {
				out := make([]reflect.Value, v.Type().NumOut())
				for i := range out {
					out[i] = reflect.Zero(v.Type().Out(i))
				}
				return out
			}))
		}
	case reflect.Chan:
		if v.Type().ChanDir() == reflect.BothDir {
			v.Set(reflect.MakeChan(v.Type(), 1))
		}
	}
}

// comparer compares a copier result a with a CopyStructDeep result b,
// both copied from s.
type comparer struct {
	diffs   []string
	visited map[[2]uintptr]bool
}

func (c *comparer) errorf(path, format string, args ...any) {
	if len(c.diffs) < 20 {
		c.diffs = append(c.diffs, path+": "+fmt.Sprintf(format, args...))
	}
}

func aliases(x, s reflect.Value) bool {
	if !s.IsValid() || s.Kind() != x.Kind() || x.IsNil() || s.IsNil() {
		return false
	}
	if x.Kind() == reflect.Slice && (x.Cap() == 0 || s.Cap() == 0) {
		return false
	}
	return x.Pointer() == s.Pointer()
}

func sub(s reflect.Value, f func(reflect.Value) reflect.Value) reflect.Value {
	if !s.IsValid() {
		return s
	}
	defer func() { recover() }()
	return f(s)
}

func (c *comparer) compare(path string, a, b, s reflect.Value) {
	if a.Kind() != b.Kind() {
		c.errorf(path, "kind %v vs %v", a.Kind(), b.Kind())
		return
	}
	switch a.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice:
		if a.IsNil() != b.IsNil() {
			c.errorf(path, "nil %v vs %v", a.IsNil(), b.IsNil())
			return
		}
		if a.IsNil() {
			return
		}
		if aliases(a, s) != aliases(b, s) {
			c.errorf(path, "aliases source: copier %v, new %v", aliases(a, s), aliases(b, s))
		}
		if s.IsValid() && s.Kind() == a.Kind() && !s.IsNil() && aliases(a, s) {
			return // same object, nothing more to compare
		}
	}
	switch a.Kind() {
	case reflect.Pointer:
		key := [2]uintptr{a.Pointer(), b.Pointer()}
		if c.visited[key] {
			return
		}
		c.visited[key] = true
		c.compare(path, a.Elem(), b.Elem(), sub(s, reflect.Value.Elem))
	case reflect.Struct:
		t := a.Type()
		for i := range t.NumField() {
			if t == nodeBaseType && (t.Field(i).Name == "This" || t.Field(i).Name == "Parent") {
				continue
			}
			c.compare(path+"."+t.Field(i).Name, a.Field(i), b.Field(i), sub(s, func(s reflect.Value) reflect.Value { return s.Field(i) }))
		}
	case reflect.Slice, reflect.Array:
		if a.Len() != b.Len() {
			c.errorf(path, "len %d vs %d", a.Len(), b.Len())
			return
		}
		for i := range a.Len() {
			if s.IsValid() && s.Kind() == a.Kind() && i >= s.Len() {
				break // dst elements beyond src are kept
			}
			c.compare(fmt.Sprintf("%s[%d]", path, i), a.Index(i), b.Index(i), sub(s, func(s reflect.Value) reflect.Value { return s.Index(i) }))
		}
	case reflect.Map:
		if a.Len() != b.Len() {
			c.errorf(path, "map len %d vs %d", a.Len(), b.Len())
			return
		}
		for _, k := range a.MapKeys() {
			bv := b.MapIndex(k)
			if !bv.IsValid() {
				c.errorf(path, "missing key %v", k)
				continue
			}
			c.compare(fmt.Sprintf("%s[%v]", path, k), a.MapIndex(k), bv, sub(s, func(s reflect.Value) reflect.Value { return s.MapIndex(k) }))
		}
	case reflect.Interface:
		if a.IsNil() != b.IsNil() {
			c.errorf(path, "nil interface %v vs %v", a.IsNil(), b.IsNil())
			return
		}
		if a.IsNil() {
			return
		}
		if a.Elem().Type() != b.Elem().Type() {
			c.errorf(path, "interface type %v vs %v", a.Elem().Type(), b.Elem().Type())
			return
		}
		c.compare(path, a.Elem(), b.Elem(), sub(s, reflect.Value.Elem))
	case reflect.Chan:
		sa := s.IsValid() && s.Kind() == a.Kind() && s.Pointer() == a.Pointer()
		sb := s.IsValid() && s.Kind() == b.Kind() && s.Pointer() == b.Pointer()
		if sa != sb {
			c.errorf(path, "chan aliases source: copier %v, new %v", sa, sb)
		}
	case reflect.Func, reflect.UnsafePointer:
		if a.Pointer() != b.Pointer() {
			c.errorf(path, "%v pointer differs", a.Kind())
		}
	case reflect.Float32, reflect.Float64:
		if a.Float() != b.Float() && !(a.Float() != a.Float() && b.Float() != b.Float()) {
			c.errorf(path, "%v vs %v", a.Float(), b.Float())
		}
	case reflect.Complex64, reflect.Complex128:
		if a.Complex() != b.Complex() {
			c.errorf(path, "%v vs %v", a.Complex(), b.Complex())
		}
	case reflect.Bool:
		if a.Bool() != b.Bool() {
			c.errorf(path, "%v vs %v", a.Bool(), b.Bool())
		}
	case reflect.String:
		if a.String() != b.String() {
			c.errorf(path, "%q vs %q", a.String(), b.String())
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if a.Int() != b.Int() {
			c.errorf(path, "%v vs %v", a.Int(), b.Int())
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if a.Uint() != b.Uint() {
			c.errorf(path, "%v vs %v", a.Uint(), b.Uint())
		}
	}
}

func newNode(proto any) tree.Node {
	n := reflect.New(reflect.TypeOf(proto).Elem()).Interface().(tree.Node)
	tree.InitNode(n)
	return n
}

func copierCopy(to, from tree.Node) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return copier.CopyWithOption(to, from, copier.Option{CaseSensitive: true, DeepCopy: true})
}

func TestCopierEquivalence(t *testing.T) {
	for _, filledDst := range []bool{false, true} {
		for _, depth := range []int{2, 4} {
			for _, proto := range nodeTypes {
				name := fmt.Sprintf("%T/depth%d/filledDst=%v", proto, depth, filledDst)
				f := &filler{maxDepth: depth}
				src := newNode(proto)
				f.fill(reflect.ValueOf(src).Elem(), 0)
				fd := &filler{maxDepth: depth, n: 1000}
				da := newNode(proto)
				db := newNode(proto)
				if filledDst {
					fd.fill(reflect.ValueOf(da).Elem(), 0)
					fd.n = 1000
					fd.fill(reflect.ValueOf(db).Elem(), 0)
				}
				if err := copierCopy(da, src); err != nil {
					t.Errorf("%s: copier: %v", name, err)
					continue
				}
				reflectx.CopyStructDeep(db, src)
				c := &comparer{visited: map[[2]uintptr]bool{}}
				c.compare("", reflect.ValueOf(da).Elem(), reflect.ValueOf(db).Elem(), reflect.ValueOf(src).Elem())
				for _, d := range c.diffs {
					t.Errorf("%s: %s", name, d)
				}
			}
		}
	}
}

// TestMethodFieldNames lists the cases where copier would copy the
// result of a method into a field of the same name: a method with no
// arguments and one result named like an exported field of the type
// or a field promoted into it.
func TestMethodFieldNames(t *testing.T) {
	var found []string
	seen := map[reflect.Type]bool{}
	var check func(typ reflect.Type)
	check = func(typ reflect.Type) {
		for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array || typ.Kind() == reflect.Map {
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Struct || seen[typ] {
			return
		}
		seen[typ] = true
		pt := reflect.PointerTo(typ)
		// copier's deepFields: exported fields, recursing into exported
		// embedded fields, including fields hidden by shallower ones.
		var deep func(st reflect.Type, prefix string)
		deep = func(st reflect.Type, prefix string) {
			for st.Kind() == reflect.Pointer {
				st = st.Elem()
			}
			if st.Kind() != reflect.Struct {
				return
			}
			for i := range st.NumField() {
				sf := st.Field(i)
				if !sf.IsExported() {
					continue
				}
				if m, ok := pt.MethodByName(sf.Name); ok && m.Type.NumIn() == 1 && m.Type.NumOut() == 1 {
					found = append(found, fmt.Sprintf("%v: field %s%s, method %v", typ, prefix, sf.Name, m.Type))
				}
				check(sf.Type)
				if sf.Anonymous {
					deep(sf.Type, prefix+sf.Name+".")
				}
			}
		}
		deep(typ, "")
	}
	for _, proto := range nodeTypes {
		check(reflect.TypeOf(proto))
	}
	sort.Strings(found)
	t.Logf("method/field name matches:\n%s", strings.Join(found, "\n"))
}

func TestSliceIfaceFields(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var found []string
	var walk func(typ reflect.Type, path string)
	walk = func(typ reflect.Type, path string) {
		switch typ.Kind() {
		case reflect.Pointer, reflect.Array, reflect.Map:
			walk(typ.Elem(), path+"*")
		case reflect.Slice:
			e := typ.Elem()
			for e.Kind() == reflect.Pointer || e.Kind() == reflect.Slice {
				e = e.Elem()
			}
			if e.Kind() == reflect.Interface && e.NumMethod() > 0 {
				found = append(found, path+" "+typ.String())
			}
			walk(typ.Elem(), path+"[]")
		case reflect.Struct:
			if seen[typ] {
				return
			}
			seen[typ] = true
			for i := range typ.NumField() {
				f := typ.Field(i)
				if f.IsExported() && !strings.Contains(string(f.Tag), `copier:"-"`) {
					walk(f.Type, typ.String()+"."+f.Name)
				}
			}
		}
	}
	for _, p := range nodeTypes {
		walk(reflect.TypeOf(p), "")
	}
	t.Logf("slices of non-empty interfaces:\n%s", strings.Join(found, "\n"))
}

// TestSceneLights shows the one deliberate difference from copier:
// copier replaces a slice of non-empty interfaces with a slice of nils
// of the same length, which breaks the lights, meshes and textures of a
// copied xyz.Scene. CopyStructDeep copies the interface values.
func TestSceneLights(t *testing.T) {
	src := tree.New[xyz.Scene]()
	xyz.NewAmbient(src, "amb", 1, xyz.DirectSun)
	da := tree.New[xyz.Scene]()
	db := tree.New[xyz.Scene]()
	if err := copierCopy(da, src); err != nil {
		t.Fatal(err)
	}
	reflectx.CopyStructDeep(db, src)
	if len(da.Lights.Values) != 1 || da.Lights.Values[0] != nil {
		t.Errorf("copier lights: %v", da.Lights.Values)
	}
	if len(db.Lights.Values) != 1 || db.Lights.Values[0] != src.Lights.Values[0] {
		t.Errorf("CopyStructDeep lights: %v", db.Lights.Values)
	}
	if db.Lights.At("amb") != src.Lights.Values[0] {
		t.Errorf("CopyStructDeep light lookup: %v", db.Lights.At("amb"))
	}
}

func benchmarkCopy(b *testing.B, copy func(to, from tree.Node)) {
	for _, proto := range []any{&core.Button{}, &core.TextField{}, &svg.Path{}} {
		b.Run(fmt.Sprintf("%T", proto), func(b *testing.B) {
			src := newNode(proto)
			dst := newNode(proto)
			for b.Loop() {
				copy(dst, src)
			}
		})
	}
}

func BenchmarkCopier(b *testing.B) {
	benchmarkCopy(b, func(to, from tree.Node) { copierCopy(to, from) })
}

func BenchmarkCopyStructDeep(b *testing.B) {
	benchmarkCopy(b, func(to, from tree.Node) { reflectx.CopyStructDeep(to, from) })
}
