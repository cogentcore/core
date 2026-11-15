// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slicesx

//core:js-test

import (
	"testing"

	"cogentcore.org/core/base/goscript"

	"github.com/stretchr/testify/assert"
)

func TestGrowTo(t *testing.T) {
	s := make([]int, 2, 3)
	s[1] = 7
	g := GrowTo(s, 10)
	assert.Len(t, g, 2, "the length is kept")
	assert.GreaterOrEqual(t, cap(g), 10)
	assert.Equal(t, 7, g[1])
	assert.Equal(t, 3, cap(GrowTo(s, 2)), "enough capacity: unchanged")
	assert.Panics(t, func() { GrowTo(s, -1) })
}

func TestCopyFrom(t *testing.T) {
	dst := []int{9, 9, 9, 9}
	dst = CopyFrom(dst, []int{1, 2})
	assert.Equal(t, []int{1, 2}, dst)
	dst = CopyFrom(dst, []int{1, 2, 3, 4, 5})
	assert.Equal(t, []int{1, 2, 3, 4, 5}, dst)
	var nd []int
	nd = CopyFrom(nd, []int{4})
	assert.Equal(t, []int{4}, nd)
}

func TestMoveSwap(t *testing.T) {
	s := []string{"a", "b", "c", "d"}
	s = Move(s, 0, 2)
	assert.Equal(t, []string{"b", "c", "a", "d"}, s)
	s = Move(s, 3, 0)
	assert.Equal(t, []string{"d", "b", "c", "a"}, s)
	s = Move(s, 1, 1)
	assert.Equal(t, []string{"d", "b", "c", "a"}, s)
	Swap(s, 0, 3)
	assert.Equal(t, []string{"a", "b", "c", "d"}, s)
}

type myInt int

func TestAs(t *testing.T) {
	in := []any{1, 2, 3}
	assert.Equal(t, []int{1, 2, 3}, As[any, int](in))
	assert.Equal(t, []any{myInt(4)}, As[myInt, any]([]myInt{4}))
	assert.Empty(t, As[any, int](nil))
}

func TestSearch(t *testing.T) {
	s := []int{0, 10, 20, 30, 40, 50, 60}
	is := func(v int) func(int) bool { return func(e int) bool { return e == v } }
	for _, v := range s {
		for _, start := range []int{-1, 0, 3, 6, 100} {
			assert.Equal(t, v/10, Search(s, is(v), start), "v: %d start: %d", v, start)
		}
		assert.Equal(t, v/10, Search(s, is(v)), "no start")
	}
	assert.Equal(t, -1, Search(s, is(5)))
	assert.Equal(t, -1, Search(s, is(5), 0))
	assert.Equal(t, -1, Search(s, is(5), 6))
	assert.Equal(t, -1, Search([]int{}, is(5)))
	// the closest match to the start is found first
	dup := []int{1, 0, 0, 1, 0, 0, 0, 1}
	assert.Equal(t, 3, Search(dup, is(1), 4))
	assert.Equal(t, 7, Search(dup, is(1), 6))
}

func TestToBytes(t *testing.T) {
	goscript.Skip(t, "unsafe.Sizeof")
	assert.Nil(t, ToBytes([]uint32{}))
	b := ToBytes([]uint16{0x0102, 0x0304})
	assert.Len(t, b, 4)
	assert.ElementsMatch(t, []byte{1, 2, 3, 4}, b, "the same bytes, in machine order")
}
