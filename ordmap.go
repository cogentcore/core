// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
package ordmap implements an ordered map that retains the order of items
added to a slice, while also providing fast key-based map lookup of items,
using the Go 1.18 generics system.

The implementation is fully visible and the API provides a minimal
subset of methods, compared to other implementations that are heavier,
so that additional functionality can be added as needed.

The slice structure holds the Key and Val for items as they are added,
enabling direct updating of the corresponding map, which holds the
index into the slice.  Adding and access are fast, while deleting
and inserting are relatively slow, requiring updating of the index map,
but these are already slow due to the slice updating.
*/
package ordmap

import (
	"fmt"

	"slices"
)

// KeyVal represents the Key and Value
type KeyVal[K comparable, V any] struct {
	Key K
	Val V
}

// Map is a generic ordered map that combines the order of a slice
// and the fast key lookup of a map.  A map stores an index
// into a slice that has the value and key associated with the value.
type Map[K comparable, V any] struct {

	// ordered list of values and associated keys -- in order added
	Order []KeyVal[K, V] `desc:"ordered list of values and associated keys -- in order added"`

	// key to index mapping
	Map map[K]int `desc:"key to index mapping"`
}

// New returns a new ordered map
func New[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		Map: make(map[K]int),
	}
}

// Constr constructs a new ordered map with the given key, value pairs
func Constr[K comparable, V any](vals ...KeyVal[K, V]) *Map[K, V] {
	om := &Map[K, V]{
		Order: vals,
		Map:   make(map[K]int, len(vals)),
	}
	for i, v := range om.Order {
		om.Map[v.Key] = i
	}
	return om
}

// Init initializes the map if not done yet
func (om *Map[K, V]) Init() {
	if om.Map == nil {
		om.Map = make(map[K]int)
	}
}

// Reset resets the map, removing any existing elements
func (om *Map[K, V]) Reset() {
	om.Map = nil
	om.Order = nil
}

// Add adds a new value for given key.
// If key already exists in map, it replaces the item at that existing index,
// otherwise it is added to the end.
func (om *Map[K, V]) Add(key K, val V) {
	om.Init()
	if idx, has := om.Map[key]; has {
		om.Map[key] = idx
		om.Order[idx] = KeyVal[K, V]{Key: key, Val: val}
	} else {
		om.Map[key] = len(om.Order)
		om.Order = append(om.Order, KeyVal[K, V]{Key: key, Val: val})
	}
}

// ReplaceIdx replaces value at given index with new item with given key
func (om *Map[K, V]) ReplaceIdx(idx int, key K, val V) {
	old := om.Order[idx]
	if key != old.Key {
		delete(om.Map, old.Key)
		om.Map[key] = idx
	}
	om.Order[idx] = KeyVal[K, V]{Key: key, Val: val}
}

// InsertAtIdx inserts value with key at given index
// This is relatively slow because it needs to renumber the index map above
// the inserted value.  It will panic if the key already exists because
// the behavior is undefined in that situation.
func (om *Map[K, V]) InsertAtIdx(idx int, key K, val V) {
	if _, has := om.Map[key]; has {
		panic("key already exists")
	}
	om.Init()
	sz := len(om.Order)
	for o := idx; o < sz; o++ {
		om.Map[om.Order[o].Key] = o + 1
	}
	om.Map[key] = idx
	om.Order = slices.Insert(om.Order, idx, KeyVal[K, V]{Key: key, Val: val})
}

// ValByKey returns value based on Key, along with bool reflecting
// presence of key.
func (om *Map[K, V]) ValByKey(key K) (V, bool) {
	idx, ok := om.Map[key]
	if ok {
		return om.Order[idx].Val, ok
	}
	var zv V
	return zv, false
}

// IdxIsValid returns error if index is invalid
func (om *Map[K, V]) IdxIsValid(idx int) error {
	if idx >= len(om.Order) || idx < 0 {
		return fmt.Errorf("ordmap.Map: IdxIsValid -- index %d is out of range of length: %d", idx, len(om.Order))
	}
	return nil
}

// IdxByKey returns index of given Key, along with bool reflecting
// presence of key.
func (om *Map[K, V]) IdxByKey(key K) (int, bool) {
	idx, ok := om.Map[key]
	return idx, ok
}

// ValByIdx returns value at given index, in ordered slice.
func (om *Map[K, V]) ValByIdx(idx int) V {
	return om.Order[idx].Val
}

// KeyByIdx returns key for given index, in ordered slice.
func (om *Map[K, V]) KeyByIdx(idx int) K {
	return om.Order[idx].Key
}

// Len returns the number of items in the map
func (om *Map[K, V]) Len() int {
	return len(om.Order)
}

// DeleteIdx deletes item(s) within index range [i:j]
// This is relatively slow because it needs to renumber the index map above
// the deleted range.
func (om *Map[K, V]) DeleteIdx(i, j int) {
	sz := len(om.Order)
	ndel := j - i
	if ndel <= 0 {
		panic("index range is <= 0")
	}
	for o := j; o < sz; o++ {
		om.Map[om.Order[o].Key] = o - ndel
	}
	for o := i; o < j; o++ {
		delete(om.Map, om.Order[o].Key)
	}
	om.Order = slices.Delete(om.Order, i, j)
}

// DeleteKey deletes item by given key, returns true if found
func (om *Map[K, V]) DeleteKey(key K) bool {
	idx, ok := om.Map[key]
	if !ok {
		return false
	}
	om.DeleteIdx(idx, idx+1)
	return true
}

// Keys returns a slice of keys in order
func (om *Map[K, V]) Keys() []K {
	kl := make([]K, om.Len())
	for i, kv := range om.Order {
		kl[i] = kv.Key
	}
	return kl
}

// Vals returns a slice of vals in order
func (om *Map[K, V]) Vals() []V {
	vl := make([]V, om.Len())
	for i, kv := range om.Order {
		vl[i] = kv.Val
	}
	return vl
}
