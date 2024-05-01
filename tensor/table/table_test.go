// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"testing"
)

func TestAdd3DCol(t *testing.T) {
	dt := NewTable(0)
	dt.AddFloat32TensorColumn("Values", []int{11, 1, 16})

	col := dt.ColumnByName("Values")
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
