// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package keylist implements an ordered list (slice) of items,
with a map from a key (e.g., names) to indexes,
to support fast lookup by name.
This is a different implementation of the [ordmap] package,
that has separate slices for Values and Keys, instead of
using a tuple list of both. The awkwardness of value access
through the tuple is the major problem with ordmap.
*/
package keylist

import (
	"fmt"
	"slices"
)

// TODO: probably want to consolidate ordmap and keylist: https://github.com/cogentcore/core/issues/1224

// List implements an ordered list (slice) of Values,
// with a map from a key (e.g., names) to indexes,
// to support fast lookup by name.
type List[K comparable, V any] struct { //types:add
	// List is the ordered slice of items.
	Values []V

	// Keys is the ordered list of keys, in same order as [List.Values]
	Keys []K

	// indexes is the key-to-index mapping.
	indexes map[K]int
}

// New returns a new [List].  The zero value
// is usable without initialization, so this is
// just a simple standard convenience method.
func New[K comparable, V any]() *List[K, V] {
	return &List[K, V]{}
}

func (kl *List[K, V]) makeIndexes() {
	kl.indexes = make(map[K]int)
}

// initIndexes ensures that the index map exists.
func (kl *List[K, V]) initIndexes() {
	if kl.indexes == nil {
		kl.makeIndexes()
	}
}

// Reset resets the list, removing any existing elements.
func (kl *List[K, V]) Reset() {
	kl.Values = nil
	kl.Keys = nil
	kl.makeIndexes()
}

// Set sets given key to given value, adding to the end of the list
// if not already present, and otherwise replacing with this new value.
// This is the same semantics as a Go map.
// See [List.Add] for version that only adds and does not replace.
func (kl *List[K, V]) Set(key K, val V) {
	kl.initIndexes()
	if idx, ok := kl.indexes[key]; ok {
		kl.Values[idx] = val
		kl.Keys[idx] = key
		return
	}
	kl.indexes[key] = len(kl.Values)
	kl.Values = append(kl.Values, val)
	kl.Keys = append(kl.Keys, key)
}

// Add adds an item to the list with given key,
// An error is returned if the key is already on the list.
// See [List.Set] for a method that automatically replaces.
func (kl *List[K, V]) Add(key K, val V) error {
	kl.initIndexes()
	if _, ok := kl.indexes[key]; ok {
		return fmt.Errorf("keylist.Add: key %v is already on the list", key)
	}
	kl.indexes[key] = len(kl.Values)
	kl.Values = append(kl.Values, val)
	kl.Keys = append(kl.Keys, key)
	return nil
}

// Insert inserts the given value with the given key at the given index.
// This is relatively slow because it needs regenerate the keys list.
// It panics if the key already exists because the behavior is undefined
// in that situation.
func (kl *List[K, V]) Insert(idx int, key K, val V) {
	if _, has := kl.indexes[key]; has {
		panic("keylist.Add: key is already on the list")
	}

	kl.Keys = slices.Insert(kl.Keys, idx, key)
	kl.Values = slices.Insert(kl.Values, idx, val)
	kl.makeIndexes()
	for i, k := range kl.Keys {
		kl.indexes[k] = i
	}
}

// At returns the value corresponding to the given key,
// with a zero value returned for a missing key. See [List.AtTry]
// for one that returns a bool for missing keys.
// For index-based access, use [List.Values] or [List.Keys] slices directly.
func (kl *List[K, V]) At(key K) V {
	idx, ok := kl.indexes[key]
	if ok {
		return kl.Values[idx]
	}
	var zv V
	return zv
}

// AtTry returns the value corresponding to the given key,
// with false returned for a missing key, in case the zero value
// is not diagnostic.
func (kl *List[K, V]) AtTry(key K) (V, bool) {
	idx, ok := kl.indexes[key]
	if ok {
		return kl.Values[idx], true
	}
	var zv V
	return zv, false
}

// IndexIsValid returns an error if the given index is invalid.
func (kl *List[K, V]) IndexIsValid(idx int) error {
	if idx >= len(kl.Values) || idx < 0 {
		return fmt.Errorf("keylist.List: IndexIsValid: index %d is out of range of a list of length %d", idx, len(kl.Values))
	}
	return nil
}

// IndexByKey returns the index of the given key, with a -1 for missing key.
func (kl *List[K, V]) IndexByKey(key K) int {
	idx, ok := kl.indexes[key]
	if !ok {
		return -1
	}
	return idx
}

// Len returns the number of items in the list.
func (kl *List[K, V]) Len() int {
	if kl == nil {
		return 0
	}
	return len(kl.Values)
}

// DeleteByIndex deletes item(s) within the index range [i:j].
// This is relatively slow because it needs to regenerate the
// index map.
func (kl *List[K, V]) DeleteByIndex(i, j int) {
	ndel := j - i
	if ndel <= 0 {
		panic("index range is <= 0")
	}
	kl.Keys = slices.Delete(kl.Keys, i, j)
	kl.Values = slices.Delete(kl.Values, i, j)
	kl.makeIndexes()
	for i, k := range kl.Keys {
		kl.indexes[k] = i
	}

}

// DeleteByKey deletes the item with the given key,
// returning false if it does not find it.
// This is relatively slow because it needs to regenerate the
// index map.
func (kl *List[K, V]) DeleteByKey(key K) bool {
	idx, ok := kl.indexes[key]
	if !ok {
		return false
	}
	kl.DeleteByIndex(idx, idx+1)
	return true
}

// RenameIndex renames the item at given index to new key.
func (kl *List[K, V]) RenameIndex(i int, key K) {
	old := kl.Keys[i]
	delete(kl.indexes, old)
	kl.Keys[i] = key
	kl.indexes[key] = i
}

// Copy copies all of the entries from the given key list
// into this list. It keeps existing entries in this
// list unless they also exist in the given list, in which case
// they are overwritten.  Use [List.Reset] first to get an exact copy.
func (kl *List[K, V]) Copy(from *List[K, V]) {
	for i, v := range from.Values {
		kl.Set(kl.Keys[i], v)
	}
}

// String returns a string representation of the list.
func (kl *List[K, V]) String() string {
	sv := "{"
	for i, v := range kl.Values {
		sv += fmt.Sprintf("%v", kl.Keys[i]) + ": " + fmt.Sprintf("%v", v) + ", "
	}
	sv += "}"
	return sv
}

// UpdateIndexes updates the Indexes from Keys and Values.
// This must be called after loading Values from a file, for example,
// where Keys can be populated from Values or are also otherwise available.
func (kl *List[K, V]) UpdateIndexes() {
	kl.makeIndexes()
	for i := range kl.Values {
		k := kl.Keys[i]
		kl.indexes[k] = i
	}
}

/*
// GoString returns the list as Go code.
func (kl *List[K, V]) GoString() string {
	var zk K
	var zv V
	res := fmt.Sprintf("ordlist.Make([]ordlist.KeyVal[%T, %T]{\n", zk, zv)
	for _, kv := range kl.Order {
		res += fmt.Sprintf("{%#v, %#v},\n", kv.Key, kv.Value)
	}
	res += "})"
	return res
}
*/
