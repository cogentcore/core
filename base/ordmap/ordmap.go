// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
package ordmap implements an ordered map that retains the order of items
added to a slice, while also providing fast key-based map lookup of items,
using the Go 1.18 generics system.

The implementation is fully visible and the API provides a minimal
subset of methods, compared to other implementations that are heavier,
so that additional functionality can be added as needed.

The slice structure holds the Key and Value for items as they are added,
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

// KeyValue represents a key-value pair.
type KeyValue[K comparable, V any] struct {
	Key   K
	Value V
}

// Map is a generic ordered map that combines the order of a slice
// and the fast key lookup of a map. A map stores an index
// into a slice that has the value and key associated with the value.
type Map[K comparable, V any] struct {

	// Order is an ordered list of values and associated keys, in the order added.
	Order []KeyValue[K, V]

	// Map is the key to index mapping.
	Map map[K]int `display:"-"`
}

// New returns a new ordered map.
func New[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		Map: make(map[K]int),
	}
}

// Make constructs a new ordered map with the given key-value pairs
func Make[K comparable, V any](vals []KeyValue[K, V]) *Map[K, V] {
	om := &Map[K, V]{
		Order: vals,
		Map:   make(map[K]int, len(vals)),
	}
	for i, v := range om.Order {
		om.Map[v.Key] = i
	}
	return om
}

// Init initializes the map if it isn't already.
func (om *Map[K, V]) Init() {
	if om.Map == nil {
		om.Map = make(map[K]int)
	}
}

// Reset resets the map, removing any existing elements.
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
		om.Order[idx] = KeyValue[K, V]{Key: key, Value: val}
	} else {
		om.Map[key] = len(om.Order)
		om.Order = append(om.Order, KeyValue[K, V]{Key: key, Value: val})
	}
}

// ReplaceIndex replaces the value at the given index
// with the given new item with the given key.
func (om *Map[K, V]) ReplaceIndex(idx int, key K, val V) {
	old := om.Order[idx]
	if key != old.Key {
		delete(om.Map, old.Key)
		om.Map[key] = idx
	}
	om.Order[idx] = KeyValue[K, V]{Key: key, Value: val}
}

// InsertAtIndex inserts the given value with the given key at the given index.
// This is relatively slow because it needs to renumber the index map above
// the inserted value.  It will panic if the key already exists because
// the behavior is undefined in that situation.
func (om *Map[K, V]) InsertAtIndex(idx int, key K, val V) {
	if _, has := om.Map[key]; has {
		panic("key already exists")
	}
	om.Init()
	sz := len(om.Order)
	for o := idx; o < sz; o++ {
		om.Map[om.Order[o].Key] = o + 1
	}
	om.Map[key] = idx
	om.Order = slices.Insert(om.Order, idx, KeyValue[K, V]{Key: key, Value: val})
}

// ValueByKey returns the value corresponding to the given key,
// with a zero value returned for a missing key. See [Map.ValueByKeyTry]
// for one that returns a bool for missing keys.
func (om *Map[K, V]) ValueByKey(key K) V {
	idx, ok := om.Map[key]
	if ok {
		return om.Order[idx].Value
	}
	var zv V
	return zv
}

// ValueByKeyTry returns the value corresponding to the given key,
// with false returned for a missing key.
func (om *Map[K, V]) ValueByKeyTry(key K) (V, bool) {
	idx, ok := om.Map[key]
	if ok {
		return om.Order[idx].Value, ok
	}
	var zv V
	return zv, false
}

// IndexIsValid returns an error if the given index is invalid
func (om *Map[K, V]) IndexIsValid(idx int) error {
	if idx >= len(om.Order) || idx < 0 {
		return fmt.Errorf("ordmap.Map: IndexIsValid: index %d is out of range of a map of length %d", idx, len(om.Order))
	}
	return nil
}

// IndexByKey returns the index of the given key, with a -1 for missing key.
// See [Map.IndexByKeyTry] for a version returning a bool for missing key.
func (om *Map[K, V]) IndexByKey(key K) int {
	idx, ok := om.Map[key]
	if !ok {
		return -1
	}
	return idx
}

// IndexByKeyTry returns the index of the given key, with false for a missing key.
func (om *Map[K, V]) IndexByKeyTry(key K) (int, bool) {
	idx, ok := om.Map[key]
	return idx, ok
}

// ValueByIndex returns the value at the given index in the ordered slice.
func (om *Map[K, V]) ValueByIndex(idx int) V {
	return om.Order[idx].Value
}

// KeyByIndex returns the key for the given index in the ordered slice.
func (om *Map[K, V]) KeyByIndex(idx int) K {
	return om.Order[idx].Key
}

// Len returns the number of items in the map.
func (om *Map[K, V]) Len() int {
	if om == nil {
		return 0
	}
	return len(om.Order)
}

// DeleteIndex deletes item(s) within the index range [i:j].
// This is relatively slow because it needs to renumber the
// index map above the deleted range.
func (om *Map[K, V]) DeleteIndex(i, j int) {
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

// DeleteKey deletes the item with the given key, returning false if it does not find it.
func (om *Map[K, V]) DeleteKey(key K) bool {
	idx, ok := om.Map[key]
	if !ok {
		return false
	}
	om.DeleteIndex(idx, idx+1)
	return true
}

// Keys returns a slice of the keys in order.
func (om *Map[K, V]) Keys() []K {
	kl := make([]K, om.Len())
	for i, kv := range om.Order {
		kl[i] = kv.Key
	}
	return kl
}

// Values returns a slice of the values in order.
func (om *Map[K, V]) Values() []V {
	vl := make([]V, om.Len())
	for i, kv := range om.Order {
		vl[i] = kv.Value
	}
	return vl
}

// Copy copies all of the entries from the given ordered map
// into this ordered map. It keeps existing entries in this
// map unless they also exist in the given map, in which case
// they are overwritten.
func (om *Map[K, V]) Copy(from *Map[K, V]) {
	for _, kv := range from.Order {
		om.Add(kv.Key, kv.Value)
	}
}

// String returns a string representation of the map.
func (om *Map[K, V]) String() string {
	return fmt.Sprintf("%v", om.Order)
}

// GoString returns the map as Go code.
func (om *Map[K, V]) GoString() string {
	var zk K
	var zv V
	res := fmt.Sprintf("ordmap.Make([]ordmap.KeyVal[%T, %T]{\n", zk, zv)
	for _, kv := range om.Order {
		res += fmt.Sprintf("{%#v, %#v},\n", kv.Key, kv.Value)
	}
	res += "})"
	return res
}
