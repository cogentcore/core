// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"fmt"
	"slices"

	"cogentcore.org/core/tensor"
)

// Columns is the underlying column list and number of rows for Table.
// Each column is a raw [tensor.Values] tensor, and [Table]
// provides a [tensor.Rows] indexed view onto the Columns.
type Cols struct {
	// Columns is the ordered slice of columns.
	Columns []tensor.Values

	// Names is the ordered list of names, in same order as Columns.
	Names []string

	// indexes is the key-to-index mapping.
	indexes map[string]int

	// number of rows, which is enforced to be the size of the
	// outermost row dimension of the column tensors.
	Rows int `edit:"-"`
}

// NewCols returns a new Columns.
func NewCols() *Cols {
	return &Cols{}
}

func (cl *Cols) makeIndexes() {
	cl.indexes = make(map[string]int)
}

// initIndexes ensures that the index map exists.
func (cl *Cols) initIndexes() {
	if cl.indexes == nil {
		cl.makeIndexes()
	}
}

// Reset resets the list, removing any existing elements.
func (cl *Cols) Reset() {
	cl.Columns = nil
	cl.Names = nil
	cl.makeIndexes()
}

// Set sets given key to given value, adding to the end of the list
// if not already present, and otherwise replacing with this new value.
// This is the same semantics as a Go map.
// See [List.Add] for version that only adds and does not replace.
func (cl *Cols) Set(key string, val tensor.Values) {
	cl.initIndexes()
	if idx, ok := cl.indexes[key]; ok {
		cl.Columns[idx] = val
		cl.Names[idx] = key
		return
	}
	cl.indexes[key] = len(cl.Columns)
	cl.Columns = append(cl.Columns, val)
	cl.Names = append(cl.Names, key)
}

// Add adds an item to the list with given key,
// An error is returned if the key is already on the list.
// See [List.Set] for a method that automatically replaces.
func (cl *Cols) Add(key string, val tensor.Values) error {
	cl.initIndexes()
	if _, ok := cl.indexes[key]; ok {
		return fmt.Errorf("keylist.Add: key %v is already on the list", key)
	}
	cl.indexes[key] = len(cl.Columns)
	cl.Columns = append(cl.Columns, val)
	cl.Names = append(cl.Names, key)
	return nil
}

// Insert inserts the given value with the given key at the given index.
// This is relatively slow because it needs regenerate the keys list.
// It panics if the key already exists because the behavior is undefined
// in that situation.
func (cl *Cols) Insert(idx int, key string, val tensor.Values) {
	if _, has := cl.indexes[key]; has {
		panic("keylist.Add: key is already on the list")
	}

	cl.Names = slices.Insert(cl.Names, idx, key)
	cl.Columns = slices.Insert(cl.Columns, idx, val)
	cl.makeIndexes()
	for i, k := range cl.Names {
		cl.indexes[k] = i
	}
}

// At returns the value corresponding to the given key,
// with a zero value returned for a missing key. See [List.AtTry]
// for one that returns a bool for missing keys.
// For index-based access, use [List.Values] or [List.Keys] slices directly.
func (cl *Cols) At(key string) tensor.Values {
	idx, ok := cl.indexes[key]
	if ok {
		return cl.Columns[idx]
	}
	var zv tensor.Values
	return zv
}

// AtTry returns the value corresponding to the given key,
// with false returned for a missing key, in case the zero value
// is not diagnostic.
func (cl *Cols) AtTry(key string) (tensor.Values, bool) {
	idx, ok := cl.indexes[key]
	if ok {
		return cl.Columns[idx], true
	}
	var zv tensor.Values
	return zv, false
}

// IndexIsValid returns an error if the given index is invalid.
func (cl *Cols) IndexIsValid(idx int) error {
	if idx >= len(cl.Columns) || idx < 0 {
		return fmt.Errorf("keylist.List: IndexIsValid: index %d is out of range of a list of length %d", idx, len(cl.Columns))
	}
	return nil
}

// IndexByKey returns the index of the given key, with a -1 for missing key.
func (cl *Cols) IndexByKey(key string) int {
	idx, ok := cl.indexes[key]
	if !ok {
		return -1
	}
	return idx
}

// Len returns the number of items in the list.
func (cl *Cols) Len() int {
	if cl == nil {
		return 0
	}
	return len(cl.Columns)
}

// DeleteByIndex deletes item(s) within the index range [i:j].
// This is relatively slow because it needs to regenerate the
// index map.
func (cl *Cols) DeleteByIndex(i, j int) {
	ndel := j - i
	if ndel <= 0 {
		panic("index range is <= 0")
	}
	cl.Names = slices.Delete(cl.Names, i, j)
	cl.Columns = slices.Delete(cl.Columns, i, j)
	cl.makeIndexes()
	for i, k := range cl.Names {
		cl.indexes[k] = i
	}

}

// DeleteByKey deletes the item with the given key,
// returning false if it does not find it.
// This is relatively slow because it needs to regenerate the
// index map.
func (cl *Cols) DeleteByKey(key string) bool {
	idx, ok := cl.indexes[key]
	if !ok {
		return false
	}
	cl.DeleteByIndex(idx, idx+1)
	return true
}

// Copy copies all of the entries from the given key list
// into this list. It keeps existing entries in this
// list unless they also exist in the given list, in which case
// they are overwritten.  Use [List.Reset] first to get an exact copy.
func (cl *Cols) Copy(from *Cols) {
	for i, v := range from.Columns {
		cl.Set(cl.Names[i], v)
	}
}

// String returns a string representation of the list.
func (cl *Cols) String() string {
	sv := "{"
	for i, v := range cl.Columns {
		sv += fmt.Sprintf("%v", cl.Names[i]) + ": " + fmt.Sprintf("%v", v) + ", "
	}
	sv += "}"
	return sv
}

// SetNumRows sets the number of rows in the table, across all columns.
// It is safe to set this to 0. For incrementally growing tables (e.g., a log)
// it is best to first set the anticipated full size, which allocates the
// full amount of memory, and then set to 0 and grow incrementally.
func (cl *Cols) SetNumRows(rows int) *Cols { //types:add
	cl.Rows = rows // can be 0
	for _, tsr := range cl.Columns {
		tsr.SetNumRows(rows)
	}
	return cl
}

// AddColumn adds the given tensor (as a [tensor.Values]) as a column,
// returning an error and not adding if the name is not unique.
// Automatically adjusts the shape to fit the current number of rows,
// and calls the metadata SetName with column name.
func (cl *Cols) AddColumn(name string, tsr tensor.Values) error {
	err := cl.Add(name, tsr)
	if err != nil {
		return err
	}
	tsr.SetNumRows(cl.Rows)
	tsr.Metadata().SetName(name)
	return nil
}

// InsertColumn inserts the given tensor as a column at given index,
// returning an error and not adding if the name is not unique.
// Automatically adjusts the shape to fit the current number of rows.
func (cl *Cols) InsertColumn(idx int, name string, tsr tensor.Values) error {
	cl.Insert(idx, name, tsr)
	tsr.SetNumRows(cl.Rows)
	return nil
}

// Clone returns a complete copy of this set of columns.
func (cl *Cols) Clone() *Cols {
	cp := NewCols().SetNumRows(cl.Rows)
	for i, nm := range cl.Names {
		tsr := cl.Columns[i]
		cp.AddColumn(nm, tsr.Clone())
	}
	return cl
}

// AppendRows appends shared columns in both tables with input table rows.
func (cl *Cols) AppendRows(cl2 *Cols) {
	for i, nm := range cl.Names {
		c2 := cl2.At(nm)
		if c2 == nil {
			continue
		}
		c1 := cl.Columns[i]
		c1.AppendFrom(c2)
	}
	cl.SetNumRows(cl.Rows + cl2.Rows)
}
