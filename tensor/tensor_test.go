// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"reflect"
	"testing"

	"cogentcore.org/core/base/metadata"
	"github.com/stretchr/testify/assert"
)

func TestTensorString(t *testing.T) {
	tsr := New[string](4, 2)
	tsr.SetNames("Row", "Vals")
	assert.Equal(t, 8, tsr.Len())
	assert.Equal(t, true, tsr.IsString())
	assert.Equal(t, reflect.String, tsr.DataType())
	assert.Equal(t, 2, tsr.SubSpace(0).Len())
	r, c := tsr.RowCellSize()
	assert.Equal(t, 4, r)
	assert.Equal(t, 2, c)

	tsr.SetString("test", 2, 0)
	assert.Equal(t, "test", tsr.StringValue(2, 0))
	tsr.SetString1D("testing", 5)
	assert.Equal(t, "testing", tsr.StringValue(2, 1))
	assert.Equal(t, "test", tsr.String1D(4))

	assert.Equal(t, "test", tsr.StringRowCell(2, 0))
	assert.Equal(t, "testing", tsr.StringRowCell(2, 1))
	assert.Equal(t, "", tsr.StringRowCell(3, 0))

	cln := tsr.Clone()
	assert.Equal(t, "testing", cln.StringValue(2, 1))

	cln.SetZeros()
	assert.Equal(t, "", cln.StringValue(2, 1))
	assert.Equal(t, "testing", tsr.StringValue(2, 1))

	tsr.SetShape(2, 4)
	tsr.SetNames("Vals", "Row")
	assert.Equal(t, "test", tsr.StringValue(1, 0))
	assert.Equal(t, "testing", tsr.StringValue(1, 1))

	cln.SetString1D("ctesting", 5)
	cln.SetShapeFrom(tsr)
	assert.Equal(t, "ctesting", cln.StringValue(1, 1))

	cln.CopyCellsFrom(tsr, 5, 4, 2)
	assert.Equal(t, "test", cln.StringValue(1, 1))
	assert.Equal(t, "testing", cln.StringValue(1, 2))

	tsr.SetNumRows(5)
	assert.Equal(t, 20, tsr.Len())

	tsr.Metadata().Set("name", "test")
	nm, err := metadata.Get[string](*tsr.Metadata(), "name")
	assert.Equal(t, "test", nm)
	assert.NoError(t, err)
	_, err = metadata.Get[string](*tsr.Metadata(), "type")
	assert.Error(t, err)

	var flt []float64
	cln.SetString1D("3.14", 0)
	assert.Equal(t, 3.14, cln.Float1D(0))

	cln.Floats(&flt)
	assert.Equal(t, 3.14, flt[0])
	assert.Equal(t, 0.0, flt[1])
}

func TestTensorFloat64(t *testing.T) {
	tsr := New[float64](4, 2)
	tsr.SetNames("Row", "Vals")
	assert.Equal(t, 8, tsr.Len())
	assert.Equal(t, false, tsr.IsString())
	assert.Equal(t, reflect.Float64, tsr.DataType())
	assert.Equal(t, 2, tsr.SubSpace(0).Len())
	r, c := tsr.RowCellSize()
	assert.Equal(t, 4, r)
	assert.Equal(t, 2, c)

	tsr.SetFloat(3.14, 2, 0)
	assert.Equal(t, 3.14, tsr.Float(2, 0))
	tsr.SetFloat1D(2.17, 5)
	assert.Equal(t, 2.17, tsr.Float(2, 1))
	assert.Equal(t, 3.14, tsr.Float1D(4))

	assert.Equal(t, 3.14, tsr.FloatRowCell(2, 0))
	assert.Equal(t, 2.17, tsr.FloatRowCell(2, 1))
	assert.Equal(t, 0.0, tsr.FloatRowCell(3, 0))

	cln := tsr.Clone()
	assert.Equal(t, 2.17, cln.Float(2, 1))

	cln.SetZeros()
	assert.Equal(t, 0.0, cln.Float(2, 1))
	assert.Equal(t, 2.17, tsr.Float(2, 1))

	tsr.SetShape(2, 4)
	tsr.SetNames("Vals", "Row")
	assert.Equal(t, 3.14, tsr.Float(1, 0))
	assert.Equal(t, 2.17, tsr.Float(1, 1))

	cln.SetFloat1D(9.9, 5)
	cln.SetShapeFrom(tsr)
	assert.Equal(t, 9.9, cln.Float(1, 1))

	cln.CopyCellsFrom(tsr, 5, 4, 2)
	assert.Equal(t, 3.14, cln.Float(1, 1))
	assert.Equal(t, 2.17, cln.Float(1, 2))

	tsr.SetNumRows(5)
	assert.Equal(t, 20, tsr.Len())

	var flt []float64
	cln.SetString1D("3.14", 0)
	assert.Equal(t, 3.14, cln.Float1D(0))

	cln.Floats(&flt)
	assert.Equal(t, 3.14, flt[0])
	assert.Equal(t, 0.0, flt[1])
}
