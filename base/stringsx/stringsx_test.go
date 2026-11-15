// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stringsx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimCR(t *testing.T) {
	assert.Equal(t, "", TrimCR(""))
	assert.Equal(t, "a", TrimCR("a\r"))
	assert.Equal(t, "a\rb", TrimCR("a\rb"))
	assert.Equal(t, "a\r", TrimCR("a\r\r"), "only one")
	assert.Equal(t, []byte{}, ByteTrimCR([]byte{}))
	assert.Equal(t, []byte("a"), ByteTrimCR([]byte("a\r")))
	assert.Equal(t, []byte("ab"), ByteTrimCR([]byte("ab")))
}

func TestSplitLines(t *testing.T) {
	assert.Equal(t, []string{"a", "b", ""}, SplitLines("a\r\nb\n"))
	assert.Equal(t, []string{""}, SplitLines(""))
	assert.Equal(t, [][]byte{[]byte("a"), []byte("b")}, ByteSplitLines([]byte("a\r\nb\r")))
}

func TestInsertFirstUnique(t *testing.T) {
	var strs []string
	InsertFirstUnique(&strs, "a", 3)
	InsertFirstUnique(&strs, "b", 3)
	InsertFirstUnique(&strs, "c", 3)
	assert.Equal(t, []string{"c", "b", "a"}, strs)
	InsertFirstUnique(&strs, "d", 3)
	assert.Equal(t, []string{"d", "c", "b"}, strs, "the oldest is dropped at the max")
	InsertFirstUnique(&strs, "b", 3)
	assert.Equal(t, []string{"b", "d", "c"}, strs, "an existing item moves to the top")
	InsertFirstUnique(&strs, "b", 3)
	assert.Equal(t, []string{"b", "d", "c"}, strs)

	long := []string{"1", "2", "3", "4"}
	InsertFirstUnique(&long, "4", 2)
	assert.Equal(t, []string{"4", "1"}, long, "a list longer than the max is cut first")

	zero := []string{"a"}
	assert.NotPanics(t, func() { InsertFirstUnique(&zero, "b", 0) })
	assert.Empty(t, zero, "nothing is kept with a max of 0")
}

func TestUniqueList(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c"}, UniqueList([]string{"a", "b", "a", "c", "b"}))
	assert.Equal(t, []string{"a"}, UniqueList([]string{"a", "a", "a"}))
	assert.Empty(t, UniqueList(nil))
	assert.Equal(t, []string{"x"}, UniqueList([]string{"x"}))
}
