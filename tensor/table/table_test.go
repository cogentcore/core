// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"strconv"
	"testing"

	"cogentcore.org/core/tensor"
	"github.com/stretchr/testify/assert"
)

func TestAdd3DCol(t *testing.T) {
	dt := NewTable()
	dt.AddFloat32Column("Values", 11, 1, 16)

	col := dt.Column("Values").Tensor
	if col.NumDims() != 4 {
		t.Errorf("Add4DCol: # of dims != 4\n")
	}

	if col.Shape().DimSize(0) != 1 {
		t.Errorf("Add4DCol: dim 0 len != 1, was: %v\n", col.Shape().DimSize(0))
	}

	if col.Shape().DimSize(1) != 11 {
		t.Errorf("Add4DCol: dim 0 len != 11, was: %v\n", col.Shape().DimSize(1))
	}

	if col.Shape().DimSize(2) != 1 {
		t.Errorf("Add4DCol: dim 0 len != 1, was: %v\n", col.Shape().DimSize(2))
	}

	if col.Shape().DimSize(3) != 16 {
		t.Errorf("Add4DCol: dim 0 len != 16, was: %v\n", col.Shape().DimSize(3))
	}
}

func NewTestTable() *Table {
	dt := NewTable()
	dt.AddStringColumn("Str")
	dt.AddFloat64Column("Flt64")
	dt.AddIntColumn("Int")
	dt.SetNumRows(3)
	for i := range dt.NumRows() {
		dt.Column("Str").SetStringRowCell(strconv.Itoa(i), i, 0)
		dt.Column("Flt64").SetFloatRowCell(float64(i), i, 0)
		dt.Column("Int").SetFloatRowCell(float64(i), i, 0)
	}
	return dt
}

func TestAppendRowsEtc(t *testing.T) {
	st := NewTestTable()
	dt := NewTestTable()
	dt.AppendRows(st)
	dt.AppendRows(st)
	dt.AppendRows(st)
	for j := range 3 {
		for i := range st.NumRows() {
			sr := j*3 + i
			ss := st.Column("Str").StringRowCell(i, 0)
			ds := dt.Column("Str").StringRowCell(sr, 0)
			assert.Equal(t, ss, ds)

			sf := st.Column("Flt64").FloatRowCell(i, 0)
			df := dt.Column("Flt64").FloatRowCell(sr, 0)
			assert.Equal(t, sf, df)

			sf = st.Column("Int").FloatRowCell(i, 0)
			df = dt.Column("Int").FloatRowCell(sr, 0)
			assert.Equal(t, sf, df)
		}
	}
	ixs := dt.RowsByString("Str", "1", tensor.Equals, tensor.UseCase)
	assert.Equal(t, []int{1, 4, 7, 10}, ixs)

	dt.Sequential()
	dt.IndexesNeeded()
	ic := dt.Column("Int")
	ic.Sort(tensor.Descending)
	dt.IndexesFromTensor(ic)
	assert.Equal(t, []int{2, 5, 8, 11, 1, 4, 7, 10, 0, 3, 6, 9}, dt.Indexes)

	dt.Sequential()
	dt.SortColumns(tensor.Descending, true, "Int", "Flt64")
	assert.Equal(t, []int{2, 5, 8, 11, 1, 4, 7, 10, 0, 3, 6, 9}, dt.Indexes)

	dt.Sequential()
	ic = dt.Column("Int") // note: important to re-get with current indexes
	ic.FilterString("1", tensor.Include, tensor.Contains, tensor.IgnoreCase)
	dt.IndexesFromTensor(ic)
	assert.Equal(t, []int{1, 4, 7, 10}, dt.Indexes)

	dt.Sequential()
	dt.Filter(func(dt *Table, row int) bool {
		return dt.Column("Flt64").FloatRowCell(row, 0) > 1
	})
	assert.Equal(t, []int{2, 5, 8, 11}, dt.Indexes)
}

func TestSetNumRows(t *testing.T) {
	st := NewTestTable()
	dt := NewTestTable()
	dt.AppendRows(st)
	dt.AppendRows(st)
	dt.AppendRows(st)
	dt.IndexesNeeded()
	dt.SetNumRows(3)
	assert.Equal(t, []int{0, 1, 2}, dt.Indexes)
}

func TestInsertDeleteRows(t *testing.T) {
	dt := NewTestTable()
	dt.IndexesNeeded()
	dt.InsertRows(1, 2)
	assert.Equal(t, []int{0, 3, 4, 1, 2}, dt.Indexes)
	dt.DeleteRows(1, 2)
	assert.Equal(t, []int{0, 1, 2}, dt.Indexes)
}
