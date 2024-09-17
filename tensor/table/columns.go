// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"cogentcore.org/core/tensor"
)

// Columns is the underlying column list and number of rows for Table.
// [Table] is an Indexed view onto the Columns.
type Columns struct {
	tensor.List

	// number of rows, which is enforced to be the size of the
	// outermost row dimension of the column tensors.
	Rows int `edit:"-"`
}

// NewColumns returns a new Columns.
func NewColumns() *Columns {
	return &Columns{}
}

// SetNumRows sets the number of rows in the table, across all columns.
// It is safe to set this to 0. For incrementally growing tables (e.g., a log)
// it is best to first set the anticipated full size, which allocates the
// full amount of memory, and then set to 0 and grow incrementally.
func (cl *Columns) SetNumRows(rows int) *Columns { //types:add
	cl.Rows = rows // can be 0
	for _, tsr := range cl.Values {
		tsr.SetNumRows(rows)
	}
	return cl
}

// AddColumn adds the given tensor as a column,
// returning an error and not adding if the name is not unique.
// Automatically adjusts the shape to fit the current number of rows,
// and calls the metadata SetName with column name.
func (cl *Columns) AddColumn(name string, tsr tensor.Tensor) error {
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
func (cl *Columns) InsertColumn(idx int, name string, tsr tensor.Tensor) error {
	err := cl.Insert(idx, name, tsr)
	if err != nil {
		return err
	}
	tsr.SetNumRows(cl.Rows)
	return nil
}

// Clone returns a complete copy of this set of columns.
func (cl *Columns) Clone() *Columns {
	cp := NewColumns().SetNumRows(cl.Rows)
	for i, nm := range cl.Keys {
		tsr := cl.Values[i]
		cp.AddColumn(nm, tsr.Clone())
	}
	return cl
}

// AppendRows appends shared columns in both tables with input table rows.
func (cl *Columns) AppendRows(cl2 *Columns) {
	for i, nm := range cl.Keys {
		c2 := cl2.ValueByKey(nm)
		if c2 == nil {
			continue
		}
		c1 := cl.Values[i]
		c1.AppendFrom(c2)
	}
	cl.SetNumRows(cl.Rows + cl2.Rows)
}
