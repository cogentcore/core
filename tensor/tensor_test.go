// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTensorString(t *testing.T) {
	shp := []int{4, 2}
	nms := []string{"Row", "Vals"}
	tsr := New[string](shp, nms...)
	assert.Equal(t, 8, tsr.Len())
	assert.Equal(t, true, tsr.IsString())
	assert.Equal(t, reflect.String, tsr.DataType())
	assert.Equal(t, 2, tsr.SubSpace([]int{0}).Len())
	r, c := tsr.RowCellSize()
	assert.Equal(t, 4, r)
	assert.Equal(t, 2, c)

	tsr.SetString([]int{2, 0}, "test")
	assert.Equal(t, "test", tsr.StringValue([]int{2, 0}))
	tsr.SetString1D(5, "testing")
	assert.Equal(t, "testing", tsr.StringValue([]int{2, 1}))
	assert.Equal(t, "test", tsr.String1D(4))

	assert.Equal(t, "test", tsr.StringRowCell(2, 0))
	assert.Equal(t, "testing", tsr.StringRowCell(2, 1))
	assert.Equal(t, "", tsr.StringRowCell(3, 0))

	cln := tsr.Clone()
	assert.Equal(t, "testing", cln.StringValue([]int{2, 1}))

	cln.SetZeros()
	assert.Equal(t, "", cln.StringValue([]int{2, 1}))
	assert.Equal(t, "testing", tsr.StringValue([]int{2, 1}))

	tsr.SetShape([]int{2, 4}, "Vals", "Row")
	assert.Equal(t, "test", tsr.StringValue([]int{1, 0}))
	assert.Equal(t, "testing", tsr.StringValue([]int{1, 1}))

	cln.SetString1D(5, "ctesting")
	cln.CopyShapeFrom(tsr)
	assert.Equal(t, "ctesting", cln.StringValue([]int{1, 1}))

	cln.CopyCellsFrom(tsr, 5, 4, 2)
	assert.Equal(t, "test", cln.StringValue([]int{1, 1}))
	assert.Equal(t, "testing", cln.StringValue([]int{1, 2}))

	tsr.SetNumRows(5)
	assert.Equal(t, 20, tsr.Len())

	tsr.SetMetaData("name", "test")
	nm, has := tsr.MetaData("name")
	assert.Equal(t, "test", nm)
	assert.Equal(t, true, has)
	_, has = tsr.MetaData("type")
	assert.Equal(t, false, has)

	var flt []float64
	cln.SetString1D(0, "3.14")
	assert.Equal(t, 3.14, cln.Float1D(0))

	cln.Floats(&flt)
	assert.Equal(t, 3.14, flt[0])
	assert.Equal(t, 0.0, flt[1])
}

func TestTensorFloat64(t *testing.T) {
	shp := []int{4, 2}
	nms := []string{"Row", "Vals"}
	tsr := New[float64](shp, nms...)
	assert.Equal(t, 8, tsr.Len())
	assert.Equal(t, false, tsr.IsString())
	assert.Equal(t, reflect.Float64, tsr.DataType())
	assert.Equal(t, 2, tsr.SubSpace([]int{0}).Len())
	r, c := tsr.RowCellSize()
	assert.Equal(t, 4, r)
	assert.Equal(t, 2, c)

	tsr.SetFloat([]int{2, 0}, 3.14)
	assert.Equal(t, 3.14, tsr.Float([]int{2, 0}))
	tsr.SetFloat1D(5, 2.17)
	assert.Equal(t, 2.17, tsr.Float([]int{2, 1}))
	assert.Equal(t, 3.14, tsr.Float1D(4))

	assert.Equal(t, 3.14, tsr.FloatRowCell(2, 0))
	assert.Equal(t, 2.17, tsr.FloatRowCell(2, 1))
	assert.Equal(t, 0.0, tsr.FloatRowCell(3, 0))

	cln := tsr.Clone()
	assert.Equal(t, 2.17, cln.Float([]int{2, 1}))

	cln.SetZeros()
	assert.Equal(t, 0.0, cln.Float([]int{2, 1}))
	assert.Equal(t, 2.17, tsr.Float([]int{2, 1}))

	tsr.SetShape([]int{2, 4}, "Vals", "Row")
	assert.Equal(t, 3.14, tsr.Float([]int{1, 0}))
	assert.Equal(t, 2.17, tsr.Float([]int{1, 1}))

	cln.SetFloat1D(5, 9.9)
	cln.CopyShapeFrom(tsr)
	assert.Equal(t, 9.9, cln.Float([]int{1, 1}))

	cln.CopyCellsFrom(tsr, 5, 4, 2)
	assert.Equal(t, 3.14, cln.Float([]int{1, 1}))
	assert.Equal(t, 2.17, cln.Float([]int{1, 2}))

	tsr.SetNumRows(5)
	assert.Equal(t, 20, tsr.Len())

	tsr.SetMetaData("name", "test")
	nm, has := tsr.MetaData("name")
	assert.Equal(t, "test", nm)
	assert.Equal(t, true, has)
	_, has = tsr.MetaData("type")
	assert.Equal(t, false, has)

	var flt []float64
	cln.SetString1D(0, "3.14")
	assert.Equal(t, 3.14, cln.Float1D(0))

	cln.Floats(&flt)
	assert.Equal(t, 3.14, flt[0])
	assert.Equal(t, 0.0, flt[1])
}
