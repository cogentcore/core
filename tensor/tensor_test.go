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

	cln.SetString1D("3.14", 0)
	assert.Equal(t, 3.14, cln.Float1D(0))

	af := AsFloat64(cln)
	assert.Equal(t, 3.14, af.Float1D(0))
	assert.Equal(t, 0.0, af.Float1D(1))
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

	cln.SetString1D("3.14", 0)
	assert.Equal(t, 3.14, cln.Float1D(0))

	af := AsFloat64(cln)
	assert.Equal(t, 3.14, af.Float1D(0))
	assert.Equal(t, 0.0, af.Float1D(1))
}

func TestSlice(t *testing.T) {
	ft := NewFloat64Indexed(3, 4, 5)
	for r := range 3 {
		for y := range 4 {
			for x := range 5 {
				v := (r+1)*100 + y*10 + x
				ft.SetFloat(float64(v), r, y, x)
			}
		}
	}
	// fmt.Println(ft.String())
	sf := NewFloat64Indexed()
	Slice(ft, sf, Range{}, Range{Start: 1, End: 2})
	// fmt.Println(sf.String())
	res := `Tensor: [3, 1, 5]
[0 0]:     110     111     112     113     114 
[1 0]:     210     211     212     213     214 
[2 0]:     310     311     312     313     314 
`
	assert.Equal(t, res, sf.Tensor.String())

	Slice(ft, sf, Range{}, Range{}, Range{Start: 1, End: 2})
	// fmt.Println(sf.String())
	res = `Tensor: [3, 4, 1]
[0 0]:     101 
[0 1]:     111 
[0 2]:     121 
[0 3]:     131 
[1 0]:     201 
[1 1]:     211 
[1 2]:     221 
[1 3]:     231 
[2 0]:     301 
[2 1]:     311 
[2 2]:     321 
[2 3]:     331 
`
	assert.Equal(t, res, sf.Tensor.String())
}

func TestSortFilter(t *testing.T) {
	tsr := NewFloat64Indexed(5)
	for i := range 5 {
		tsr.SetFloatRowCell(float64(i), i, 0)
	}
	tsr.Sort(Ascending)
	assert.Equal(t, []int{0, 1, 2, 3, 4}, tsr.Indexes)
	tsr.Sort(Descending)
	assert.Equal(t, []int{4, 3, 2, 1, 0}, tsr.Indexes)

	tsr.Sequential()
	tsr.FilterString("1", Include, Equals, UseCase)
	assert.Equal(t, []int{1}, tsr.Indexes)

	tsr.Sequential()
	tsr.FilterString("1", Exclude, Equals, UseCase)
	assert.Equal(t, []int{0, 2, 3, 4}, tsr.Indexes)
}
