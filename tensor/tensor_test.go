// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"reflect"
	"testing"

	"cogentcore.org/core/base/metadata"
	"github.com/stretchr/testify/assert"
)

func TestProjection2D(t *testing.T) {
	shp := NewShape(5)
	var nilINts []int
	rowShape, colShape, rowIdxs, colIdxs := Projection2DDimShapes(shp, OnedRow)
	assert.Equal(t, []int{5}, rowShape.Sizes)
	assert.Equal(t, []int{1}, colShape.Sizes)
	assert.Equal(t, []int{0}, rowIdxs)
	assert.Equal(t, nilINts, colIdxs)

	rowShape, colShape, rowIdxs, colIdxs = Projection2DDimShapes(shp, OnedColumn)
	assert.Equal(t, []int{1}, rowShape.Sizes)
	assert.Equal(t, []int{5}, colShape.Sizes)
	assert.Equal(t, nilINts, rowIdxs)
	assert.Equal(t, []int{0}, colIdxs)

	shp = NewShape(3, 4)
	rowShape, colShape, rowIdxs, colIdxs = Projection2DDimShapes(shp, OnedRow)
	assert.Equal(t, []int{3}, rowShape.Sizes)
	assert.Equal(t, []int{4}, colShape.Sizes)
	assert.Equal(t, []int{0}, rowIdxs)
	assert.Equal(t, []int{1}, colIdxs)

	shp = NewShape(3, 4, 5)
	rowShape, colShape, rowIdxs, colIdxs = Projection2DDimShapes(shp, OnedRow)
	assert.Equal(t, []int{3, 4}, rowShape.Sizes)
	assert.Equal(t, []int{5}, colShape.Sizes)
	assert.Equal(t, []int{0, 1}, rowIdxs)
	assert.Equal(t, []int{2}, colIdxs)

	shp = NewShape(3, 4, 5, 6)
	rowShape, colShape, rowIdxs, colIdxs = Projection2DDimShapes(shp, OnedRow)
	assert.Equal(t, []int{3, 5}, rowShape.Sizes)
	assert.Equal(t, []int{4, 6}, colShape.Sizes)
	assert.Equal(t, []int{0, 2}, rowIdxs)
	assert.Equal(t, []int{1, 3}, colIdxs)

	shp = NewShape(3, 4, 5, 6, 7)
	rowShape, colShape, rowIdxs, colIdxs = Projection2DDimShapes(shp, OnedRow)
	assert.Equal(t, []int{3, 4, 6}, rowShape.Sizes)
	assert.Equal(t, []int{5, 7}, colShape.Sizes)
	assert.Equal(t, []int{0, 1, 3}, rowIdxs)
	assert.Equal(t, []int{2, 4}, colIdxs)
}

func TestPrintf(t *testing.T) {
	ft := NewFloat64(4)
	for x := range 4 {
		ft.SetFloat(float64(x), x)
	}
	// fmt.Println(ft.String())
	res := `[4] 0 1 2 3 
`
	assert.Equal(t, res, ft.String())

	ft = NewFloat64(40)
	for x := range 40 {
		ft.SetFloat(float64(x), x)
	}
	// fmt.Println(ft.String())
	res = `[40]  0  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 
     25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 
`
	assert.Equal(t, res, ft.String())

	ft = NewFloat64(4, 1)
	for x := range 4 {
		ft.SetFloat(float64(x), x, 0)
	}
	// fmt.Println(ft.String())
	res = `[4 1]
[0] 0 
[1] 1 
[2] 2 
[3] 3 
`
	assert.Equal(t, res, ft.String())

	ft = NewFloat64(4, 3)
	for y := range 4 {
		for x := range 3 {
			v := y*10 + x
			ft.SetFloat(float64(v), y, x)
		}
	}
	// fmt.Println(ft.String())
	res = `[4 3]
    [0] [1] [2] 
[0]   0   1   2 
[1]  10  11  12 
[2]  20  21  22 
[3]  30  31  32 
`
	assert.Equal(t, res, ft.String())

	ft = NewFloat64(4, 3, 2)
	for z := range 4 {
		for y := range 3 {
			for x := range 2 {
				v := z*100 + y*10 + x
				ft.SetFloat(float64(v), z, y, x)
			}
		}
	}
	// fmt.Println(ft.String())
	res = `[4 3 2]
[r r c] [0] [1] 
[0 0]     0   1 
[0 1]    10  11 
[0 2]    20  21 
[1 0]   100 101 
[1 1]   110 111 
[1 2]   120 121 
[2 0]   200 201 
[2 1]   210 211 
[2 2]   220 221 
[3 0]   300 301 
[3 1]   310 311 
[3 2]   320 321 
`
	assert.Equal(t, res, ft.String())

	ft = NewFloat64(5, 4, 3, 2)
	for z := range 5 {
		for y := range 4 {
			for x := range 3 {
				for w := range 2 {
					v := z*1000 + y*100 + x*10 + w
					ft.SetFloat(float64(v), z, y, x, w)
				}
			}
		}
	}
	// fmt.Println(ft.String())
	res = `[5 4 3 2]
[r c r c] [0 0] [0 1] [1 0] [1 1] [2 0] [2 1] [3 0] [3 1] 
[0 0]         0     1   100   101   200   201   300   301 
[0 1]        10    11   110   111   210   211   310   311 
[0 2]        20    21   120   121   220   221   320   321 
[1 0]      1000  1001  1100  1101  1200  1201  1300  1301 
[1 1]      1010  1011  1110  1111  1210  1211  1310  1311 
[1 2]      1020  1021  1120  1121  1220  1221  1320  1321 
[2 0]      2000  2001  2100  2101  2200  2201  2300  2301 
[2 1]      2010  2011  2110  2111  2210  2211  2310  2311 
[2 2]      2020  2021  2120  2121  2220  2221  2320  2321 
[3 0]      3000  3001  3100  3101  3200  3201  3300  3301 
[3 1]      3010  3011  3110  3111  3210  3211  3310  3311 
[3 2]      3020  3021  3120  3121  3220  3221  3320  3321 
[4 0]      4000  4001  4100  4101  4200  4201  4300  4301 
[4 1]      4010  4011  4110  4111  4210  4211  4310  4311 
[4 2]      4020  4021  4120  4121  4220  4221  4320  4321 
`
	assert.Equal(t, res, ft.String())

	ft = NewFloat64(6, 5, 4, 3, 2)
	for z := range 6 {
		for y := range 5 {
			for x := range 4 {
				for w := range 3 {
					for q := range 2 {
						v := z*10000 + y*1000 + x*100 + w*10 + q
						ft.SetFloat(float64(v), z, y, x, w, q)
					}
				}
			}
		}
	}
	// fmt.Println(ft.String())
	res = `[6 5 4 3 2]
[r r c r c] [0 0] [0 1] [1 0] [1 1] [2 0] [2 1] [3 0] [3 1] 
[0 0 0]         0     1   100   101   200   201   300   301 
[0 0 1]        10    11   110   111   210   211   310   311 
[0 0 2]        20    21   120   121   220   221   320   321 
[0 1 0]      1000  1001  1100  1101  1200  1201  1300  1301 
[0 1 1]      1010  1011  1110  1111  1210  1211  1310  1311 
[0 1 2]      1020  1021  1120  1121  1220  1221  1320  1321 
[0 2 0]      2000  2001  2100  2101  2200  2201  2300  2301 
[0 2 1]      2010  2011  2110  2111  2210  2211  2310  2311 
[0 2 2]      2020  2021  2120  2121  2220  2221  2320  2321 
[0 3 0]      3000  3001  3100  3101  3200  3201  3300  3301 
[0 3 1]      3010  3011  3110  3111  3210  3211  3310  3311 
[0 3 2]      3020  3021  3120  3121  3220  3221  3320  3321 
[0 4 0]      4000  4001  4100  4101  4200  4201  4300  4301 
[0 4 1]      4010  4011  4110  4111  4210  4211  4310  4311 
[0 4 2]      4020  4021  4120  4121  4220  4221  4320  4321 
[1 0 0]     10000 10001 10100 10101 10200 10201 10300 10301 
[1 0 1]     10010 10011 10110 10111 10210 10211 10310 10311 
[1 0 2]     10020 10021 10120 10121 10220 10221 10320 10321 
[1 1 0]     11000 11001 11100 11101 11200 11201 11300 11301 
[1 1 1]     11010 11011 11110 11111 11210 11211 11310 11311 
[1 1 2]     11020 11021 11120 11121 11220 11221 11320 11321 
[1 2 0]     12000 12001 12100 12101 12200 12201 12300 12301 
[1 2 1]     12010 12011 12110 12111 12210 12211 12310 12311 
[1 2 2]     12020 12021 12120 12121 12220 12221 12320 12321 
[1 3 0]     13000 13001 13100 13101 13200 13201 13300 13301 
[1 3 1]     13010 13011 13110 13111 13210 13211 13310 13311 
[1 3 2]     13020 13021 13120 13121 13220 13221 13320 13321 
[1 4 0]     14000 14001 14100 14101 14200 14201 14300 14301 
[1 4 1]     14010 14011 14110 14111 14210 14211 14310 14311 
[1 4 2]     14020 14021 14120 14121 14220 14221 14320 14321 
[2 0 0]     20000 20001 20100 20101 20200 20201 20300 20301 
[2 0 1]     20010 20011 20110 20111 20210 20211 20310 20311 
[2 0 2]     20020 20021 20120 20121 20220 20221 20320 20321 
[2 1 0]     21000 21001 21100 21101 21200 21201 21300 21301 
[2 1 1]     21010 21011 21110 21111 21210 21211 21310 21311 
[2 1 2]     21020 21021 21120 21121 21220 21221 21320 21321 
[2 2 0]     22000 22001 22100 22101 22200 22201 22300 22301 
[2 2 1]     22010 22011 22110 22111 22210 22211 22310 22311 
[2 2 2]     22020 22021 22120 22121 22220 22221 22320 22321 
[2 3 0]     23000 23001 23100 23101 23200 23201 23300 23301 
[2 3 1]     23010 23011 23110 23111 23210 23211 23310 23311 
[2 3 2]     23020 23021 23120 23121 23220 23221 23320 23321 
[2 4 0]     24000 24001 24100 24101 24200 24201 24300 24301 
[2 4 1]     24010 24011 24110 24111 24210 24211 24310 24311 
[2 4 2]     24020 24021 24120 24121 24220 24221 24320 24321 
[3 0 0]     30000 30001 30100 30101 30200 30201 30300 30301 
[3 0 1]     30010 30011 30110 30111 30210 30211 30310 30311 
[3 0 2]     30020 30021 30120 30121 30220 30221 30320 30321 
[3 1 0]     31000 31001 31100 31101 31200 31201 31300 31301 
[3 1 1]     31010 31011 31110 31111 31210 31211 31310 31311 
[3 1 2]     31020 31021 31120 31121 31220 31221 31320 31321 
[3 2 0]     32000 32001 32100 32101 32200 32201 32300 32301 
[3 2 1]     32010 32011 32110 32111 32210 32211 32310 32311 
[3 2 2]     32020 32021 32120 32121 32220 32221 32320 32321 
[3 3 0]     33000 33001 33100 33101 33200 33201 33300 33301 
[3 3 1]     33010 33011 33110 33111 33210 33211 33310 33311 
[3 3 2]     33020 33021 33120 33121 33220 33221 33320 33321 
[3 4 0]     34000 34001 34100 34101 34200 34201 34300 34301 
[3 4 1]     34010 34011 34110 34111 34210 34211 34310 34311 
[3 4 2]     34020 34021 34120 34121 34220 34221 34320 34321 
[4 0 0]     40000 40001 40100 40101 40200 40201 40300 40301 
[4 0 1]     40010 40011 40110 40111 40210 40211 40310 40311 
[4 0 2]     40020 40021 40120 40121 40220 40221 40320 40321 
[4 1 0]     41000 41001 41100 41101 41200 41201 41300 41301 
[4 1 1]     41010 41011 41110 41111 41210 41211 41310 41311 
[4 1 2]     41020 41021 41120 41121 41220 41221 41320 41321 
[4 2 0]     42000 42001 42100 42101 42200 42201 42300 42301 
[4 2 1]     42010 42011 42110 42111 42210 42211 42310 42311 
[4 2 2]     42020 42021 42120 42121 42220 42221 42320 42321 
[4 3 0]     43000 43001 43100 43101 43200 43201 43300 43301 
[4 3 1]     43010 43011 43110 43111 43210 43211 43310 43311 
[4 3 2]     43020 43021 43120 43121 43220 43221 43320 43321 
[4 4 0]     44000 44001 44100 44101 44200 44201 44300 44301 
[4 4 1]     44010 44011 44110 44111 44210 44211 44310 44311 
[4 4 2]     44020 44021 44120 44121 44220 44221 44320 44321 
[5 0 0]     50000 50001 50100 50101 50200 50201 50300 50301 
[5 0 1]     50010 50011 50110 50111 50210 50211 50310 50311 
[5 0 2]     50020 50021 50120 50121 50220 50221 50320 50321 
[5 1 0]     51000 51001 51100 51101 51200 51201 51300 51301 
[5 1 1]     51010 51011 51110 51111 51210 51211 51310 51311 
[5 1 2]     51020 51021 51120 51121 51220 51221 51320 51321 
[5 2 0]     52000 52001 52100 52101 52200 52201 52300 52301 
[5 2 1]     52010 52011 52110 52111 52210 52211 52310 52311 
[5 2 2]     52020 52021 52120 52121 52220 52221 52320 52321 
[5 3 0]     53000 53001 53100 53101 53200 53201 53300 53301 
[5 3 1]     53010 53011 53110 53111 53210 53211 53310 53311 
[5 3 2]     53020 53021 53120 53121 53220 53221 53320 53321 
[5 4 0]     54000 54001 54100 54101 54200 54201 54300 54301 
[5 4 1]     54010 54011 54110 54111 54210 54211 54310 54311 
[5 4 2]     54020 54021 54120 54121 54220 54221 54320 54321 
`
	assert.Equal(t, res, ft.String())
}

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

	assert.Equal(t, "test", tsr.StringRow(2, 0))
	assert.Equal(t, "testing", tsr.StringRow(2, 1))
	assert.Equal(t, "", tsr.StringRow(3, 0))

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

	metadata.SetName(tsr, "test")
	nm := metadata.Name(tsr)
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

	assert.Equal(t, 3.14, tsr.FloatRow(2, 0))
	assert.Equal(t, 2.17, tsr.FloatRow(2, 1))
	assert.Equal(t, 0.0, tsr.FloatRow(3, 0))

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
	[0]:	[1]:	[2]:	[3]:	
[0]:	0	1	2	3	
[1]:	10	11	12	13	
[2]:	20	21	22	23	
`
	assert.Equal(t, res, ft.String())

	res = `[2, 2]
	[0]:	[1]:	
[0]:	23	22	
[1]:	13	12	
`
	sl := NewSliced(ft, []int{2, 1}, []int{3, 2})
	assert.Equal(t, res, sl.String())

	vl := sl.AsValues()
	assert.Equal(t, res, vl.String())
	res = `[3, 1]
[0]:	2	
[1]:	12	
[2]:	22	
`
	sl2 := Reslice(ft, FullAxis, Slice{2, 3, 0})
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
	[0]:	[1]:	[2]:	[3]:	
[0]:	0	1	2	3	
[1]:	10	11	12	13	
[2]:	20	21	22	23	
`
	assert.Equal(t, res, ms.String())

	ms.Filter(func(tsr Tensor, idx int) bool {
		val := tsr.Float1D(idx)
		return int(val)%10 == 2
	})
	res = `[3, 4]
	[0]:	[1]:	[2]:	[3]:	
[0]:	NaN	NaN	2	NaN	
[1]:	NaN	NaN	12	NaN	
[2]:	NaN	NaN	22	NaN	
`
	assert.Equal(t, res, ms.String())

	res = `[3] 2	12	22	
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
	[0 0]:	[0 1]:	[0 0]:	[0 1]:	
[0]:	1	1	11	11	
[0]:	2	2	22	22	
`
	fmt.Println(ix.String())
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
	[0]:	[1]:	[2]:	
[0]:	0	1	2	
[1]:	3	10	11	
[2]:	12	13	20	
[3]:	21	22	23	
`
	rs := NewReshaped(ft, 4, 3)
	assert.Equal(t, res, rs.String())

	res = `[1, 3, 4]
	[0 0]:	[0 1]:	[0 2]:	[0 3]:	
[0]:	0	1	2	3	
[0]:	10	11	12	13	
[0]:	20	21	22	23	
`
	rs = NewReshaped(ft, int(NewAxis), 3, 4)
	assert.Equal(t, res, rs.String())

	res = `[12]
[0]:	0	1	2	3	10	11	12	13	20	21	22	23	
`
	rs = NewReshaped(ft, -1)
	assert.Equal(t, res, rs.String())

	res = `[4, 3]
	[0]:	[1]:	[2]:	
[0]:	0	1	2	
[1]:	3	10	11	
[2]:	12	13	20	
[3]:	21	22	23	
`
	rs = NewReshaped(ft, 4, -1)
	assert.Equal(t, res, rs.String())

	err := rs.SetShapeSizes(5, -1)
	assert.Error(t, err)

	res = `[3, 4]
	[0]:	[3]:	[2]:	[1]:	
[0]:	0	3	12	21	
[0]:	1	10	13	22	
[0]:	2	11	20	23	
`
	tr := Transpose(ft)
	// fmt.Println(tr)
	assert.Equal(t, res, tr.String())

}

func TestSortFilter(t *testing.T) {
	tsr := NewRows(NewFloat64(5))
	for i := range 5 {
		tsr.SetFloatRow(float64(i), i, 0)
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
