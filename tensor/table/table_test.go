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
	for i := range dt.Rows() {
		dt.Column("Str").SetStringRowCell(strconv.Itoa(i), i, 0)
		dt.Column("Flt64").SetFloatRowCell(float64(i), i, 0)
		dt.Column("Int").SetFloatRowCell(float64(i), i, 0)
	}
	return dt
}

func TestAppendRows(t *testing.T) {
	st := NewTestTable()
	dt := NewTestTable()
	dt.AppendRows(st)
	dt.AppendRows(st)
	dt.AppendRows(st)
	for j := range 3 {
		for i := range st.Rows() {
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
}
