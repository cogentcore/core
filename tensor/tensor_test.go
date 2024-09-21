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
	// tsr.SetNames("Row", "Vals")
	// assert.Equal(t, []string{"Row", "Vals"}, tsr.Shape().Names)
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

	tsr.SetShapeInts(2, 4)
	// tsr.SetNames("Vals", "Row")
	assert.Equal(t, "test", tsr.StringValue(1, 0))
	assert.Equal(t, "testing", tsr.StringValue(1, 1))

	cln.SetString1D("ctesting", 5)
	SetShapeFrom(cln, tsr)
	assert.Equal(t, "ctesting", cln.StringValue(1, 1))

	cln.CopyCellsFrom(tsr, 5, 4, 2)
	assert.Equal(t, "test", cln.StringValue(1, 1))
	assert.Equal(t, "testing", cln.StringValue(1, 2))

	tsr.SetNumRows(5)
	assert.Equal(t, 20, tsr.Len())

	tsr.Metadata().SetName("test")
	nm := tsr.Metadata().Name()
	assert.Equal(t, "test", nm)
	_, err := metadata.Get[string](*tsr.Metadata(), "type")
	assert.Error(t, err)

	cln.SetString1D("3.14", 0)
	assert.Equal(t, 3.14, cln.Float1D(0))

	af := AsFloat64Slice(cln)
	assert.Equal(t, 3.14, af[0])
	assert.Equal(t, 0.0, af[1])
}

func TestTensorFloat64(t *testing.T) {
	tsr := New[float64](4, 2)
	// tsr.SetNames("Row")
	// assert.Equal(t, []string{"Row", ""}, tsr.Shape().Names)
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

	tsr.SetShapeInts(2, 4)
	assert.Equal(t, 3.14, tsr.Float(1, 0))
	assert.Equal(t, 2.17, tsr.Float(1, 1))

	cln.SetFloat1D(9.9, 5)
	SetShapeFrom(cln, tsr)
	assert.Equal(t, 9.9, cln.Float(1, 1))

	cln.CopyCellsFrom(tsr, 5, 4, 2)
	assert.Equal(t, 3.14, cln.Float(1, 1))
	assert.Equal(t, 2.17, cln.Float(1, 2))

	tsr.SetNumRows(5)
	assert.Equal(t, 20, tsr.Len())

	cln.SetString1D("3.14", 0)
	assert.Equal(t, 3.14, cln.Float1D(0))

	af := AsFloat64Slice(cln)
	assert.Equal(t, 3.14, af[0])
	assert.Equal(t, 0.0, af[1])
}

func TestSliced(t *testing.T) {
	ft := NewFloat64(3, 4)
	for y := range 3 {
		for x := range 4 {
			v := y*10 + x
			ft.SetFloat(float64(v), y, x)
		}
	}

	res := `[3, 4]
[0]:       0       1       2       3 
[1]:      10      11      12      13 
[2]:      20      21      22      23 
`
	assert.Equal(t, res, ft.String())
	// fmt.Println(ft)

	res = `[2, 2]
[0]:      23      22 
[1]:      13      12 
`

	sl := NewSlicedIndexes(ft, []int{2, 1}, []int{3, 2})
	// fmt.Println(sl)
	assert.Equal(t, res, sl.String())

	/*

		ft := NewFloat64(2, 3, 4)
		for r := range 2 {
			for y := range 3 {
				for x := range 4 {
					v := (r+1)*100 + y*10 + x
					ft.SetFloat(float64(v), r, y, x)
				}
			}
		}

		fmt.Println(ft)

		sl := NewSliced(ft, []int{1, 0}, []int{1, 0}, []int{1, 0})
		fmt.Println(sl)

				assert.Equal(t, res, sf.Tensor.String())
	*/

}

func TestSortFilter(t *testing.T) {
	tsr := NewRows(NewFloat64(5))
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

func TestGrowRow(t *testing.T) {
	tsr := NewFloat64(1000)
	assert.Equal(t, 1000, cap(tsr.Values))
	assert.Equal(t, 1000, tsr.Len())
	tsr.SetNumRows(0)
	assert.Equal(t, 1000, cap(tsr.Values))
	assert.Equal(t, 0, tsr.Len())
	tsr.SetNumRows(1)
	assert.Equal(t, 1000, cap(tsr.Values))
	assert.Equal(t, 1, tsr.Len())

	tsr2 := NewFloat64(1000, 10, 10)
	assert.Equal(t, 100000, cap(tsr2.Values))
	assert.Equal(t, 100000, tsr2.Len())
	tsr2.SetNumRows(0)
	assert.Equal(t, 100000, cap(tsr2.Values))
	assert.Equal(t, 0, tsr2.Len())
	tsr2.SetNumRows(1)
	assert.Equal(t, 100000, cap(tsr2.Values))
	assert.Equal(t, 100, tsr2.Len())

	bits := NewBits(1000)
	assert.Equal(t, 1000, bits.Len())
	bits.SetNumRows(0)
	assert.Equal(t, 0, bits.Len())
	bits.SetNumRows(1)
	assert.Equal(t, 1, bits.Len())
}
