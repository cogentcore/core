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
	r, c := tsr.Shape().RowCellSize()
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

	tsr.SetShapeSizes(2, 4)
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
	r, c := tsr.Shape().RowCellSize()
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

	tsr.SetShapeSizes(2, 4)
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
	sl := NewSliced(ft, []int{2, 1}, []int{3, 2})
	// fmt.Println(sl)
	assert.Equal(t, res, sl.String())

	vl := sl.AsValues()
	assert.Equal(t, res, vl.String())
	res = `[3, 1]
[0]:       2 
[1]:      12 
[2]:      22 
`
	sl2 := Reslice(ft, FullAxis, Slice{2, 3, 0})
	// fmt.Println(sl)
	assert.Equal(t, res, sl2.String())

	vl = sl2.AsValues()
	assert.Equal(t, res, vl.String())
}

func TestMasked(t *testing.T) {
	ft := NewFloat64(3, 4)
	for y := range 3 {
		for x := range 4 {
			v := y*10 + x
			ft.SetFloat(float64(v), y, x)
		}
	}
	ms := NewMasked(ft)

	res := `[3, 4]
[0]:       0       1       2       3 
[1]:      10      11      12      13 
[2]:      20      21      22      23 
`
	assert.Equal(t, res, ms.String())

	ms.Filter(func(tsr Tensor, idx int) bool {
		val := tsr.Float1D(idx)
		return int(val)%10 == 2
	})
	res = `[3, 4]
[0]:     NaN     NaN       2     NaN 
[1]:     NaN     NaN      12     NaN 
[2]:     NaN     NaN      22     NaN 
`
	// fmt.Println(ms.String())
	assert.Equal(t, res, ms.String())

	res = `[3]       2      12      22 
`

	vl := ms.AsValues()
	assert.Equal(t, res, vl.String())
}

func TestIndexed(t *testing.T) {
	ft := NewFloat64(3, 4)
	for y := range 3 {
		for x := range 4 {
			v := y*10 + x
			ft.SetFloat(float64(v), y, x)
		}
	}
	ixs := NewIntFromValues(
		0, 1,
		0, 1,
		0, 2,
		0, 2,
		1, 1,
		1, 1,
		2, 2,
		2, 2,
	)

	ixs.SetShapeSizes(2, 2, 2, 2)
	ix := NewIndexed(ft, ixs)

	res := `[2, 2, 2]
[0]:       1       1      11      11 
[0]:       2       2      22      22 
`
	// fmt.Println(ix.String())
	assert.Equal(t, res, ix.String())

	vl := ix.AsValues()
	assert.Equal(t, res, vl.String())
}

func TestReshaped(t *testing.T) {
	ft := NewFloat64(3, 4)
	for y := range 3 {
		for x := range 4 {
			v := y*10 + x
			ft.SetFloat(float64(v), y, x)
		}
	}

	res := `[4, 3]
[0]:       0       1       2 
[1]:       3      10      11 
[2]:      12      13      20 
[3]:      21      22      23 
`
	rs := NewReshaped(ft, 4, 3)
	// fmt.Println(rs)
	assert.Equal(t, res, rs.String())

	res = `[1, 3, 4]
[0]:       0       1       2       3 
[0]:      10      11      12      13 
[0]:      20      21      22      23 
`
	rs = NewReshaped(ft, int(NewAxis), 3, 4)
	assert.Equal(t, res, rs.String())

	res = `[12]
[0]:       0       1       2       3      10      11      12      13      20      21      22      23 
`
	rs = NewReshaped(ft, -1)
	assert.Equal(t, res, rs.String())

	res = `[4, 3]
[0]:       0       1       2 
[1]:       3      10      11 
[2]:      12      13      20 
[3]:      21      22      23 
`
	rs = NewReshaped(ft, 4, -1)
	// fmt.Println(rs)
	assert.Equal(t, res, rs.String())

	err := rs.SetShapeSizes(5, -1)
	assert.Error(t, err)
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
	tsr.FilterString("1", FilterOptions{})
	assert.Equal(t, []int{1}, tsr.Indexes)

	tsr.Sequential()
	tsr.FilterString("1", FilterOptions{Exclude: true})
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

	bits := NewBool(1000)
	assert.Equal(t, 1000, bits.Len())
	bits.SetNumRows(0)
	assert.Equal(t, 0, bits.Len())
	bits.SetNumRows(1)
	assert.Equal(t, 1, bits.Len())
}
