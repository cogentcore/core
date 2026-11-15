// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ordmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// checkIndexes checks that the map has the index of each key.
func checkIndexes[K comparable, V any](t *testing.T, om *Map[K, V]) {
	t.Helper()
	assert.Len(t, om.Map, len(om.Order))
	for i, kv := range om.Order {
		assert.Equal(t, i, om.Map[kv.Key], "key %v", kv.Key)
	}
}

func TestMapOps(t *testing.T) {
	var om Map[string, int]
	assert.Equal(t, 0, om.Len())
	var nom *Map[string, int]
	assert.Equal(t, 0, nom.Len(), "nil")
	om.Add("a", 1)
	om.Add("b", 2)
	om.Add("c", 3)
	om.Add("b", 20)
	assert.Equal(t, []string{"a", "b", "c"}, om.Keys())
	assert.Equal(t, []int{1, 20, 3}, om.Values(), "adding an existing key replaces it")
	checkIndexes(t, &om)

	v, ok := om.ValueByKeyTry("c")
	assert.True(t, ok)
	assert.Equal(t, 3, v)
	_, ok = om.ValueByKeyTry("z")
	assert.False(t, ok)
	assert.Equal(t, 0, om.ValueByKey("z"))
	idx, ok := om.IndexByKeyTry("c")
	assert.True(t, ok)
	assert.Equal(t, 2, idx)
	assert.Equal(t, -1, om.IndexByKey("z"))
	assert.NoError(t, om.IndexIsValid(2))
	assert.Error(t, om.IndexIsValid(3))
	assert.Error(t, om.IndexIsValid(-1))

	om.ReplaceIndex(1, "B", 200)
	assert.Equal(t, []string{"a", "B", "c"}, om.Keys())
	assert.Equal(t, -1, om.IndexByKey("b"))
	om.ReplaceIndex(1, "B", 201)
	assert.Equal(t, 201, om.ValueByKey("B"))
	checkIndexes(t, &om)

	om.InsertAtIndex(0, "first", 0)
	assert.Equal(t, []string{"first", "a", "B", "c"}, om.Keys())
	checkIndexes(t, &om)
	assert.Panics(t, func() { om.InsertAtIndex(0, "a", 0) })

	assert.True(t, om.DeleteKey("a"))
	assert.False(t, om.DeleteKey("a"))
	assert.Equal(t, []string{"first", "B", "c"}, om.Keys())
	checkIndexes(t, &om)
	assert.Panics(t, func() { om.DeleteIndex(1, 1) })

	cp := New[string, int]()
	cp.Add("c", -1)
	cp.Copy(&om)
	assert.Equal(t, []string{"c", "first", "B"}, cp.Keys())
	assert.Equal(t, 3, cp.ValueByKey("c"), "copied over")

	assert.Equal(t, "[{first 0} {B 201} {c 3}]", om.String())
	assert.Equal(t, "ordmap.Make([]ordmap.KeyValue[string, int]{\n{\"first\", 0},\n{\"B\", 201},\n{\"c\", 3},\n})", om.GoString())

	om.Reset()
	assert.Equal(t, 0, om.Len())
	om.InsertAtIndex(0, "x", 1)
	assert.Equal(t, 1, om.ValueByKey("x"), "insert into a reset map")
}
