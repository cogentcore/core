// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
package keylist implements an ordered list (slice) of items,
with a map from a key (e.g., names) to indexes,
to support fast lookup by name.
Compared to the [ordmap] package, this is not as efficient
for operations such as deletion and insertion, but it
has the advantage of providing a simple slice of the target
items that can be used directly in many cases.
Thus, it is more suitable for largely static lists, which
are constructed by adding items to the end of the list.
*/
package keylist

import (
	"fmt"
	"slices"
)

// TODO: probably want to consolidate ordmap and keylist

// List implements an ordered list (slice) of items,
// with a map from a key (e.g., names) to indexes,
// to support fast lookup by name.
type List[K comparable, V any] struct { //types:add
	// List is the ordered slice of items.
	List []V

	// indexes is the key-to-index mapping.
	indexes map[K]int
}

// New returns a new Key List.  Zero value
// is usable without initialization, so this is
// just a simple standard convenience method.
func New[K comparable, V any]() *List[K, V] {
	return &List[K, V]{}
}

func (kl *List[K, V]) newIndexes() {
	kl.indexes = make(map[K]int)
}

// initIndexes ensures that the index map exists.
func (kl *List[K, V]) initIndexes() {
	if kl.indexes == nil {
		kl.newIndexes()
	}
}

// Reset resets the list, removing any existing elements.
func (kl *List[K, V]) Reset() {
	kl.List = nil
	kl.newIndexes()
}

// Keys returns the list of keys in List sequential order.
func (kl *List[K, V]) Keys() []K {
	keys := make([]K, len(kl.indexes))
	for k, i := range kl.indexes {
		keys[i] = k
	}
	return keys
}

// Add adds an item to the list with given key.
// An error is returned if the key is already on the list.
// See [AddReplace] for a method that automatically replaces.
func (kl *List[K, V]) Add(key K, val V) error {
	kl.initIndexes()
	if _, ok := kl.indexes[key]; ok {
		return fmt.Errorf("keylist.Add: key %v is already on the list", key)
	}
	kl.indexes[key] = len(kl.List)
	kl.List = append(kl.List, val)
	return nil
}

// AddReplace adds an item to the list with given key,
// replacing any existing item with the same key.
func (kl *List[K, V]) AddReplace(key K, val V) {
	kl.initIndexes()
	if idx, ok := kl.indexes[key]; ok {
		kl.List[idx] = val
		return
	}
	kl.indexes[key] = len(kl.List)
	kl.List = append(kl.List, val)
}

// Insert inserts the given value with the given key at the given index.
// This is relatively slow because it needs regenerate the keys list.
// It returns an error if the key already exists because
// the behavior is undefined in that situation.
func (kl *List[K, V]) Insert(idx int, key K, val V) error {
	if _, has := kl.indexes[key]; has {
		return fmt.Errorf("keylist.Add: key %v is already on the list", key)
	}

	keys := kl.Keys()
	keys = slices.Insert(keys, idx, key)
	kl.List = slices.Insert(kl.List, idx, val)
	kl.newIndexes()
	for i, k := range keys {
		kl.indexes[k] = i
	}
	return nil
}

// ValueByKey returns the value corresponding to the given key,
// with a zero value returned for a missing key. See [List.ValueByKeyTry]
// for one that returns a bool for missing keys.
func (kl *List[K, V]) ValueByKey(key K) V {
	idx, ok := kl.indexes[key]
	if ok {
		return kl.List[idx]
	}
	var zv V
	return zv
}

// ValueByKeyTry returns the value corresponding to the given key,
// with false returned for a missing key, in case the zero value
// is not diagnostic.
func (kl *List[K, V]) ValueByKeyTry(key K) (V, bool) {
	idx, ok := kl.indexes[key]
	if ok {
		return kl.List[idx], true
	}
	var zv V
	return zv, false
}

// IndexIsValid returns an error if the given index is invalid
func (kl *List[K, V]) IndexIsValid(idx int) error {
	if idx >= len(kl.List) || idx < 0 {
		return fmt.Errorf("keylist.List: IndexIsValid: index %d is out of range of a list of length %d", idx, len(kl.List))
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
	return len(kl.List)
}

// DeleteIndex deletes item(s) within the index range [i:j].
// This is relatively slow because it needs to regenerate the
// index map.
func (kl *List[K, V]) DeleteIndex(i, j int) {
	ndel := j - i
	if ndel <= 0 {
		panic("index range is <= 0")
	}
	keys := kl.Keys()
	keys = slices.Delete(keys, i, j)
	kl.List = slices.Delete(kl.List, i, j)
	kl.newIndexes()
	for i, k := range keys {
		kl.indexes[k] = i
	}

}

// DeleteKey deletes the item with the given key,
// returning false if it does not find it.
// This is relatively slow because it needs to regenerate the
// index map.
func (kl *List[K, V]) DeleteKey(key K) bool {
	idx, ok := kl.indexes[key]
	if !ok {
		return false
	}
	kl.DeleteIndex(idx, idx+1)
	return true
}

// Copy copies all of the entries from the given keyed list
// into this list. It keeps existing entries in this
// list unless they also exist in the given list, in which case
// they are overwritten.  Use [Reset] first to get an exact copy.
func (kl *List[K, V]) Copy(from *List[K, V]) {
	keys := from.Keys()
	for i, v := range from.List {
		kl.AddReplace(keys[i], v)
	}
}

// String returns a string representation of the list.
func (kl *List[K, V]) String() string {
	sv := "{"
	keys := kl.Keys()
	for i, v := range kl.List {
		sv += fmt.Sprintf("%v", keys[i]) + ": " + fmt.Sprintf("%v", v) + ", "
	}
	sv += "}"
	return sv
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
