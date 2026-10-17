// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type copyInner struct {
	Ints  []int
	Map   map[string]int
	Ptr   *int
	Any   any
	inner int
}

type CopyBase struct {
	Name    string `copier:"-"`
	Skip    int    `copier:"-"`
	Base    int
	BaseAny any
}

type copyNamer interface{ String() string }

type copyStringer string

func (s copyStringer) String() string { return string(s) }

type copyOuter struct {
	CopyBase
	*copyInner
	Embed copyInner

	Struct  copyInner
	Ptr     *copyInner
	NilPtr  *copyInner
	Slice   []copyInner
	Ptrs    []*copyInner
	Nil     []int
	NilMap  map[string]int
	PtrMap  map[string]*int
	Array   [2][]int
	Func    func() int
	Chan    chan int
	Any     any
	AnyPtr  any
	Anys    []any
	Namers  []copyNamer
	Time    time.Time
	private []int
}

func TestCopyStructDeep(t *testing.T) {
	one := 1
	src := &copyOuter{
		CopyBase:  CopyBase{Name: "src", Skip: 1, Base: 2, BaseAny: []int{1}},
		copyInner: &copyInner{Ints: []int{3}},
		Struct:    copyInner{Ints: []int{1, 2}, Map: map[string]int{"a": 1}, Ptr: &one, Any: []int{4}, inner: 5},
		Ptr:       &copyInner{Ints: []int{6}},
		Slice:     []copyInner{{Ints: []int{7}}},
		Ptrs:      []*copyInner{{Ints: []int{8}}, nil},
		PtrMap:    map[string]*int{"a": &one, "nil": nil},
		Array:     [2][]int{{9}, {10}},
		Func:      func() int { return 11 },
		Chan:      make(chan int),
		Any:       map[string]int{"b": 2},
		AnyPtr:    &one,
		Anys:      []any{[]int{12}},
		Namers:    []copyNamer{copyStringer("x")},
		Time:      time.Now(),
		private:   []int{13},
	}
	dstPtr := &copyInner{Ints: []int{0, 0, 0}}
	dst := &copyOuter{CopyBase: CopyBase{Name: "dst", Skip: 2}, Ptr: dstPtr, Anys: []any{1, 2}}
	CopyStructDeep(dst, src)

	// tags
	assert.Equal(t, "dst", dst.Name)
	assert.Equal(t, 2, dst.Skip)
	assert.Equal(t, 2, dst.Base)

	// deep copies
	assert.Equal(t, src.Struct.Ints, dst.Struct.Ints)
	assert.NotSame(t, &src.Struct.Ints[0], &dst.Struct.Ints[0])
	assert.Equal(t, src.Struct.Map, dst.Struct.Map)
	dst.Struct.Map["a"] = 2
	assert.Equal(t, 1, src.Struct.Map["a"])
	assert.NotSame(t, src.Struct.Ptr, dst.Struct.Ptr)
	assert.Equal(t, 1, *dst.Struct.Ptr)
	assert.Equal(t, 5, dst.Struct.inner)
	assert.Equal(t, []int{4}, dst.Struct.Any)
	assert.NotSame(t, &src.Struct.Any.([]int)[0], &dst.Struct.Any.([]int)[0])
	assert.NotSame(t, &src.Ptrs[0].Ints[0], &dst.Ptrs[0].Ints[0])
	assert.Nil(t, dst.Ptrs[1])
	assert.Nil(t, dst.NilPtr)
	assert.Equal(t, src.Slice[0].Ints, dst.Slice[0].Ints)
	assert.NotSame(t, &src.Slice[0].Ints[0], &dst.Slice[0].Ints[0])
	assert.Equal(t, src.Time, dst.Time)

	// existing dst pointers and slices are reused, keeping extra elements
	assert.Same(t, dstPtr, dst.Ptr)
	assert.Equal(t, []int{6, 0, 0}, dst.Ptr.Ints)

	// nil slices and maps become empty, nil map pointers point to zero
	assert.NotNil(t, dst.Nil)
	assert.Len(t, dst.Nil, 0)
	assert.NotNil(t, dst.NilMap)
	assert.Equal(t, 0, *dst.PtrMap["nil"])
	assert.NotSame(t, &one, dst.PtrMap["a"])

	// assigned values
	assert.Same(t, &src.Array[0][0], &dst.Array[0][0])
	assert.Equal(t, 11, dst.Func())
	assert.Equal(t, src.Chan, dst.Chan)
	assert.Same(t, &src.private[0], &dst.private[0])
	assert.Same(t, &one, dst.AnyPtr)

	// interfaces: deep when direct and nil in dst, shallow when promoted
	dst.Any.(map[string]int)["b"] = 3
	assert.Equal(t, 2, src.Any.(map[string]int)["b"])
	assert.Same(t, &src.BaseAny.([]int)[0], &dst.BaseAny.([]int)[0])

	// slices of interfaces are new slices
	assert.Equal(t, []any{[]int{12}}, dst.Anys)
	assert.NotSame(t, &src.Anys[0].([]int)[0], &dst.Anys[0].([]int)[0])
	assert.Equal(t, src.Namers, dst.Namers)

	// unexported embedded fields are shallow
	assert.Same(t, src.copyInner, dst.copyInner) // unexported embedded
	assert.Equal(t, []int{3}, dst.copyInner.Ints)
}

type copyShadow struct {
	CopyBase
	Skip int // ignored because CopyBase.Skip has copier:"-"
}

func TestCopyStructDeepShadow(t *testing.T) {
	src := &copyShadow{Skip: 1}
	dst := &copyShadow{Skip: 2}
	CopyStructDeep(dst, src)
	assert.Equal(t, 2, dst.Skip)
}
