// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdd3DCol(t *testing.T) {
	dt := NewTable()
	dt.AddFloat32TensorColumn("Values", []int{11, 1, 16})

	col, err := dt.ColumnByName("Values")
	if err != nil {
		t.Error(err)
	}
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
	for i := 0; i < dt.Rows; i++ {
		dt.SetString("Str", i, strconv.Itoa(i))
		dt.SetFloat("Flt64", i, float64(i))
		dt.SetFloat("Int", i, float64(i))
	}
	return dt
}

func TestAppendRows(t *testing.T) {
	st := NewTestTable()
	dt := NewTestTable()
	dt.AppendRows(st)
	dt.AppendRows(st)
	dt.AppendRows(st)
	for j := 0; j < 3; j++ {
		for i := 0; i < st.Rows; i++ {
			sr := j*3 + i
			ss := st.StringValue("Str", i)
			ds := dt.StringValue("Str", sr)
			assert.Equal(t, ss, ds)

			sf := st.Float("Flt64", i)
			df := dt.Float("Flt64", sr)
			assert.Equal(t, sf, df)

			sf = st.Float("Int", i)
			df = dt.Float("Int", sr)
			assert.Equal(t, sf, df)
		}
	}
}
