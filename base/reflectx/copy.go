// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"reflect"
	"strings"
	"sync"
)

// CopyStructDeep copies the struct pointed to by src into the struct
// pointed to by dst, which must have the same type. It is the field
// copying behind [cogentcore.org/core/tree.NodeBase.CopyFieldsFrom].
// It has the semantics of github.com/jinzhu/copier with DeepCopy and
// CaseSensitive for copies between values of the same type, which
// Cogent Core used before, except that it never looks up or calls
// methods. Methods found by name with reflection keep every method of
// every type alive in a whole-program dead code eliminating build.
//
//   - Exported fields are copied deeply, except for fields with a
//     `copier:"-"` struct tag, which keep their dst values. A field is
//     also skipped when a field promoted into its struct through
//     embedding has the same name and a `copier:"-"` tag.
//   - Unexported fields are copied shallowly.
//   - Pointers, slices and maps are copied into the existing dst values:
//     a non-nil dst pointer is reused, dst slice elements beyond the
//     length of src are kept, and dst map entries whose keys are not in
//     src are kept. Nil src slices and maps become empty ones when dst
//     is nil, and nil pointer map values become pointers to zero values.
//   - An interface holding a struct, map or slice is copied deeply when
//     the dst interface is nil, unless the interface is in a field
//     promoted through embedding. Other interfaces are assigned.
//   - Arrays, funcs, channels and all other values are assigned.
func CopyStructDeep(dst, src any) {
	d := reflect.ValueOf(dst).Elem()
	s := reflect.ValueOf(src).Elem()
	info := structInfoFor(d.Type())
	copyStruct(d, s, info, false, info.promoted)
}

// copyValue deeply copies src into dst, which must be settable and
// have the same type as src. If shallow, interfaces are assigned;
// copier assigns them in fields promoted through embedding, as it
// copies those fields a second time from the outer struct, once the
// dst interfaces are no longer nil.
func copyValue(dst, src reflect.Value, shallow bool) {
	switch src.Kind() {
	case reflect.Struct:
		info := structInfoFor(src.Type())
		if info.plain {
			dst.Set(src)
			return
		}
		copyStruct(dst, src, info, shallow, info.promoted)
	case reflect.Pointer:
		if src.IsNil() {
			dst.Set(reflect.Zero(dst.Type()))
			return
		}
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		copyValue(dst.Elem(), src.Elem(), shallow)
	case reflect.Slice:
		copySlice(dst, src, shallow)
	case reflect.Map:
		copyMap(dst, src)
	case reflect.Interface:
		if shallow || src.IsNil() || !dst.IsNil() {
			dst.Set(src)
			return
		}
		e := src.Elem()
		switch e.Kind() {
		case reflect.Struct, reflect.Map, reflect.Slice:
			nv := reflect.New(e.Type()).Elem()
			copyValue(nv, e, false)
			dst.Set(nv)
		default:
			dst.Set(src)
		}
	default:
		dst.Set(src)
	}
}

func copySlice(dst, src reflect.Value, shallow bool) {
	n := src.Len()
	et := dst.Type().Elem()
	if et.Kind() == reflect.Interface {
		// copier always makes a new slice of interfaces.
		dst.Set(reflect.MakeSlice(dst.Type(), n, src.Cap()))
		shallow = false
	} else if dst.IsNil() {
		dst.Set(reflect.MakeSlice(dst.Type(), n, src.Cap()))
	} else if dst.Len() < n {
		if dst.Cap() >= n {
			dst.SetLen(n)
		} else {
			ns := reflect.MakeSlice(dst.Type(), n, n)
			reflect.Copy(ns, dst)
			dst.Set(ns)
		}
	}
	if isShallow(et) {
		reflect.Copy(dst, src)
		return
	}
	for i := range n {
		copyValue(dst.Index(i), src.Index(i), shallow)
	}
}

func copyMap(dst, src reflect.Value) {
	if dst.IsNil() {
		dst.Set(reflect.MakeMapWithSize(dst.Type(), src.Len()))
	}
	et := dst.Type().Elem()
	shallow := isShallow(et)
	it := src.MapRange()
	for it.Next() {
		if shallow {
			dst.SetMapIndex(it.Key(), it.Value())
			continue
		}
		nv := reflect.New(et).Elem()
		v := it.Value()
		if et.Kind() == reflect.Pointer && v.IsNil() {
			nv.Set(reflect.New(et.Elem()))
		} else {
			copyValue(nv, v, false)
		}
		dst.SetMapIndex(it.Key(), nv)
	}
}

// copyStruct copies src into dst. The promoted node marks the fields
// that copier copies a second time from an outer struct.
func copyStruct(dst, src reflect.Value, info *structInfo, shallow bool, promoted *pathNode) {
	if info.hasUnexported {
		// Unexported fields can only be set through the whole struct.
		tmp := reflect.New(dst.Type()).Elem()
		tmp.Set(src)
		for _, i := range info.exported {
			tmp.Field(i).Set(dst.Field(i))
		}
		dst.Set(tmp)
	}
	for _, f := range info.fields {
		if f.shallow {
			dst.Field(f.index).Set(src.Field(f.index))
			continue
		}
		fsh := shallow
		var sub *pathNode
		if promoted != nil {
			sub = promoted.kids[f.index]
			if sub != nil && sub.hit {
				fsh = true
			}
		}
		if !f.embedded || sub == nil {
			copyValue(dst.Field(f.index), src.Field(f.index), fsh)
			continue
		}
		d, s := dst.Field(f.index), src.Field(f.index)
		if s.Kind() == reflect.Pointer {
			if s.IsNil() {
				d.Set(reflect.Zero(d.Type()))
				continue
			}
			if d.IsNil() {
				d.Set(reflect.New(d.Type().Elem()))
			}
			d, s = d.Elem(), s.Elem()
		}
		copyStruct(d, s, structInfoFor(s.Type()), fsh, sub)
	}
}

// structInfo is the cached copying information for a struct type.
type structInfo struct {
	// fields are the exported fields to copy.
	fields []fieldInfo

	// exported are the indexes of all exported fields.
	exported []int

	// promoted has the index paths of the fields promoted through
	// embedding that copier copies a second time, from this struct
	// or from embedded structs.
	promoted *pathNode

	// hasUnexported is whether the struct has unexported fields.
	hasUnexported bool

	// plain is whether the struct can be copied by assignment.
	plain bool
}

type fieldInfo struct {
	index    int
	shallow  bool
	embedded bool
}

// pathNode is a node in a tree of field index paths.
type pathNode struct {
	hit  bool
	kids map[int]*pathNode
}

// addPath returns p with the given path added.
func addPath(p *pathNode, path []int) *pathNode {
	if p == nil {
		p = &pathNode{}
	}
	if len(path) == 0 {
		p.hit = true
		return p
	}
	if p.kids == nil {
		p.kids = map[int]*pathNode{}
	}
	p.kids[path[0]] = addPath(p.kids[path[0]], path[1:])
	return p
}

// addTree returns p with all of the paths in o added under the given
// prefix index.
func addTree(p *pathNode, index int, o *pathNode) *pathNode {
	if o == nil {
		return p
	}
	if p == nil {
		p = &pathNode{}
	}
	if p.kids == nil {
		p.kids = map[int]*pathNode{}
	}
	p.kids[index] = mergePaths(p.kids[index], o)
	return p
}

// mergePaths returns p with all of the paths in o added.
func mergePaths(p, o *pathNode) *pathNode {
	if p == nil {
		p = &pathNode{}
	}
	if o.hit {
		p.hit = true
	}
	for i, k := range o.kids {
		if p.kids == nil {
			p.kids = map[int]*pathNode{}
		}
		p.kids[i] = mergePaths(p.kids[i], k)
	}
	return p
}

var (
	structInfosMu sync.RWMutex
	structInfos   = map[reflect.Type]*structInfo{}
)

func structInfoFor(t reflect.Type) *structInfo {
	structInfosMu.RLock()
	info := structInfos[t]
	structInfosMu.RUnlock()
	if info != nil {
		return info
	}
	info = newStructInfo(t)
	structInfosMu.Lock()
	structInfos[t] = info
	structInfosMu.Unlock()
	return info
}

func newStructInfo(t reflect.Type) *structInfo {
	ignored := map[string]bool{}
	var names []string
	addDeepFields(t, ignored, &names, map[reflect.Type]bool{})
	info := &structInfo{plain: true}
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			info.hasUnexported = true
			continue
		}
		info.exported = append(info.exported, i)
		if ignored[f.Name] {
			info.plain = false
			continue
		}
		sh := isShallow(f.Type)
		if !sh {
			info.plain = false
		}
		info.fields = append(info.fields, fieldInfo{index: i, shallow: sh, embedded: f.Anonymous})
		if f.Anonymous && !sh {
			et := f.Type
			if et.Kind() == reflect.Pointer {
				et = et.Elem()
			}
			if et.Kind() == reflect.Struct {
				info.promoted = addTree(info.promoted, i, structInfoFor(et).promoted)
			}
		}
	}
	// copier copies each promoted field again by name from this struct.
	for _, name := range names {
		if ignored[name] {
			continue
		}
		sf, ok := t.FieldByName(name)
		if ok && len(sf.Index) > 1 && exportedPath(t, sf.Index) {
			info.promoted = addPath(info.promoted, sf.Index)
		}
	}
	return info
}

// exportedPath returns whether all fields along the index path are exported.
func exportedPath(t reflect.Type, index []int) bool {
	for _, i := range index {
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		f := t.Field(i)
		if !f.IsExported() {
			return false
		}
		t = f.Type
	}
	return true
}

// addDeepFields adds the names of the exported fields of t, including
// the fields promoted through exported embedded fields, and records
// whether each is ignored by its copier tag; the last tagged field with
// a given name decides.
func addDeepFields(t reflect.Type, ignored map[string]bool, names *[]string, visited map[reflect.Type]bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || visited[t] {
		return
	}
	visited[t] = true
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		*names = append(*names, f.Name)
		if tag := f.Tag.Get("copier"); tag != "" {
			ignored[f.Name] = tagIgnores(tag)
		}
		if f.Anonymous {
			addDeepFields(f.Type, ignored, names, visited)
		}
	}
	visited[t] = false
}

func tagIgnores(tag string) bool {
	for part := range strings.SplitSeq(tag, ",") {
		if part == "-" {
			return true
		}
	}
	return false
}

// isShallow returns whether values of the given type are copied by assignment.
func isShallow(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
		return false
	case reflect.Struct:
		return structInfoFor(t).plain
	}
	return true
}
