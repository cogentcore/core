// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keylist

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListOps(t *testing.T) {
	kl := New[string, int]()
	assert.Equal(t, 0, kl.Len())
	var nkl *List[string, int]
	assert.Equal(t, 0, nkl.Len(), "nil")

	kl.Set("a", 1)
	kl.Set("b", 2)
	kl.Set("a", 10)
	assert.Equal(t, []string{"a", "b"}, kl.Keys)
	assert.Equal(t, []int{10, 2}, kl.Values, "setting an existing key replaces it")
	assert.Error(t, kl.Add("a", 0))
	assert.NoError(t, kl.Add("c", 3))
	v, ok := kl.AtTry("c")
	assert.True(t, ok)
	assert.Equal(t, 3, v)
	_, ok = kl.AtTry("z")
	assert.False(t, ok)
	assert.Equal(t, 0, kl.At("z"))
	assert.NoError(t, kl.IndexIsValid(2))
	assert.Error(t, kl.IndexIsValid(3))
	assert.Error(t, kl.IndexIsValid(-1))
	assert.Equal(t, -1, kl.IndexByKey("z"))

	kl.Insert(1, "x", 5)
	assert.Equal(t, []string{"a", "x", "b", "c"}, kl.Keys)
	assert.Equal(t, 2, kl.IndexByKey("b"), "the indexes are updated")
	assert.Panics(t, func() { kl.Insert(0, "a", 0) })

	kl.RenameIndex(1, "y")
	assert.Equal(t, 5, kl.At("y"))
	assert.Equal(t, -1, kl.IndexByKey("x"))

	assert.True(t, kl.DeleteByKey("a"))
	assert.False(t, kl.DeleteByKey("a"))
	assert.Equal(t, []string{"y", "b", "c"}, kl.Keys)
	assert.Equal(t, 0, kl.IndexByKey("y"))
	assert.Panics(t, func() { kl.DeleteByIndex(1, 0) })
	assert.Equal(t, "{y: 5, b: 2, c: 3, }", kl.String())

	cp := New[string, int]()
	cp.Copy(kl)
	assert.Equal(t, kl.Keys, cp.Keys, "a copy into an empty list")
	assert.Equal(t, kl.Values, cp.Values)
	other := New[string, int]()
	other.Set("c", -1)
	other.Set("z", 26)
	other.Copy(kl)
	assert.Equal(t, []string{"c", "z", "y", "b"}, other.Keys)
	assert.Equal(t, 3, other.At("c"), "an existing key is overwritten")

	// the indexes of keys and values set directly (as when loaded)
	loaded := &List[string, int]{Keys: []string{"p", "q"}, Values: []int{1, 2}}
	assert.Equal(t, -1, loaded.IndexByKey("q"))
	loaded.UpdateIndexes()
	assert.Equal(t, 1, loaded.IndexByKey("q"))

	kl.Reset()
	assert.Equal(t, 0, kl.Len())
	assert.Equal(t, -1, kl.IndexByKey("b"))
}
