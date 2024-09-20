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

	if col.DimSize(0) != 0 {
		t.Errorf("Add4DCol: dim 0 len != 0, was: %v\n", col.DimSize(0))
	}

	if col.DimSize(1) != 11 {
		t.Errorf("Add4DCol: dim 0 len != 11, was: %v\n", col.DimSize(1))
	}

	if col.DimSize(2) != 1 {
		t.Errorf("Add4DCol: dim 0 len != 1, was: %v\n", col.DimSize(2))
	}

	if col.DimSize(3) != 16 {
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
		dt.Column("Str").SetStringRow(strconv.Itoa(i), i)
		dt.Column("Flt64").SetFloatRow(float64(i), i)
		dt.Column("Int").SetFloatRow(float64(i), i)
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
			ss := st.Column("Str").StringRow(i)
			ds := dt.Column("Str").StringRow(sr)
			assert.Equal(t, ss, ds)

			sf := st.Column("Flt64").FloatRow(i)
			df := dt.Column("Flt64").FloatRow(sr)
			assert.Equal(t, sf, df)

			sf = st.Column("Int").FloatRow(i)
			df = dt.Column("Int").FloatRow(sr)
			assert.Equal(t, sf, df)
		}
	}
	dt.Sequential()
	dt.SortColumn("Int", tensor.Descending)
	assert.Equal(t, []int{2, 5, 8, 11, 1, 4, 7, 10, 0, 3, 6, 9}, dt.Indexes)

	dt.Sequential()
	dt.SortColumns(tensor.Descending, true, "Int", "Flt64")
	assert.Equal(t, []int{2, 5, 8, 11, 1, 4, 7, 10, 0, 3, 6, 9}, dt.Indexes)

	dt.Sequential()
	dt.FilterString("Int", "1", tensor.Include, tensor.Contains, tensor.IgnoreCase)
	assert.Equal(t, []int{1, 4, 7, 10}, dt.Indexes)

	dt.Sequential()
	dt.Filter(func(dt *Table, row int) bool {
		return dt.Column("Flt64").FloatRow(row) > 1
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

func TestCells(t *testing.T) {
	dt := NewTable()
	err := dt.OpenCSV("../stats/cluster/testdata/faces.dat", tensor.Tab)
	assert.NoError(t, err)
	in := dt.Column("Input")
	for i := range 10 {
		vals := make([]float32, 16)
		for j := range 16 {
			vals[j] = float32(in.FloatRowCell(i, j))
		}
		// fmt.Println(s)
		ss := in.Tensor.SubSpace(i).(*tensor.Float32)
		// fmt.Println(ss.Values[:16])
		cl := in.Cells1D(i).Tensor.(*tensor.Float32)
		// fmt.Println(cl.Values[:16])
		assert.Equal(t, vals, ss.Values[:16])
		assert.Equal(t, vals, cl.Values[:16])
	}
}
