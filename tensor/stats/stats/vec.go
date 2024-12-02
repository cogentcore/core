// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"cogentcore.org/core/tensor"
)

// VectorizeOut64 is the general compute function for stats.
// This version makes a Float64 output tensor for aggregating
// and computing values, and then copies the results back to the
// original output. This allows stats functions to operate directly
// on integer valued inputs and produce sensible results.
// It returns the Float64 output tensor for further processing as needed.
func VectorizeOut64(a tensor.Tensor, out tensor.Values, ini float64, fun func(val, agg float64) float64) *tensor.Float64 {
	rows, cells := a.Shape().RowCellSize()
	o64 := tensor.NewFloat64(cells)
	if rows <= 0 {
		return o64
	}
	if cells == 1 {
		out.SetShapeSizes(1)
		agg := ini
		switch x := a.(type) {
		case *tensor.Float64:
			for i := range rows {
				agg = fun(x.Float1D(i), agg)
			}
		case *tensor.Float32:
			for i := range rows {
				agg = fun(x.Float1D(i), agg)
			}
		default:
			for i := range rows {
				agg = fun(a.Float1D(i), agg)
			}
		}
		o64.SetFloat1D(agg, 0)
		out.SetFloat1D(agg, 0)
		return o64
	}
	osz := tensor.CellsSize(a.ShapeSizes())
	out.SetShapeSizes(osz...)
	for i := range cells {
		o64.SetFloat1D(ini, i)
	}
	switch x := a.(type) {
	case *tensor.Float64:
		for i := range rows {
			for j := range cells {
				o64.SetFloat1D(fun(x.Float1D(i*cells+j), o64.Float1D(j)), j)
			}
		}
	case *tensor.Float32:
		for i := range rows {
			for j := range cells {
				o64.SetFloat1D(fun(x.Float1D(i*cells+j), o64.Float1D(j)), j)
			}
		}
	default:
		for i := range rows {
			for j := range cells {
				o64.SetFloat1D(fun(a.Float1D(i*cells+j), o64.Float1D(j)), j)
			}
		}
	}
	for j := range cells {
		out.SetFloat1D(o64.Float1D(j), j)
	}
	return o64
}

// VectorizePreOut64 is a version of [VectorizeOut64] that takes an additional
// tensor.Float64 input of pre-computed values, e.g., the means of each output cell.
func VectorizePreOut64(a tensor.Tensor, out tensor.Values, ini float64, pre *tensor.Float64, fun func(val, pre, agg float64) float64) *tensor.Float64 {
	rows, cells := a.Shape().RowCellSize()
	o64 := tensor.NewFloat64(cells)
	if rows <= 0 {
		return o64
	}
	if cells == 1 {
		out.SetShapeSizes(1)
		agg := ini
		prev := pre.Float1D(0)
		switch x := a.(type) {
		case *tensor.Float64:
			for i := range rows {
				agg = fun(x.Float1D(i), prev, agg)
			}
		case *tensor.Float32:
			for i := range rows {
				agg = fun(x.Float1D(i), prev, agg)
			}
		default:
			for i := range rows {
				agg = fun(a.Float1D(i), prev, agg)
			}
		}
		o64.SetFloat1D(agg, 0)
		out.SetFloat1D(agg, 0)
		return o64
	}
	osz := tensor.CellsSize(a.ShapeSizes())
	out.SetShapeSizes(osz...)
	for j := range cells {
		o64.SetFloat1D(ini, j)
	}
	switch x := a.(type) {
	case *tensor.Float64:
		for i := range rows {
			for j := range cells {
				o64.SetFloat1D(fun(x.Float1D(i*cells+j), pre.Float1D(j), o64.Float1D(j)), j)
			}
		}
	case *tensor.Float32:
		for i := range rows {
			for j := range cells {
				o64.SetFloat1D(fun(x.Float1D(i*cells+j), pre.Float1D(j), o64.Float1D(j)), j)
			}
		}
	default:
		for i := range rows {
			for j := range cells {
				o64.SetFloat1D(fun(a.Float1D(i*cells+j), pre.Float1D(j), o64.Float1D(j)), j)
			}
		}
	}
	for i := range cells {
		out.SetFloat1D(o64.Float1D(i), i)
	}
	return o64
}

// Vectorize2Out64 is a version of [VectorizeOut64] that separately aggregates
// two output values, x and y as tensor.Float64.
func Vectorize2Out64(a tensor.Tensor, iniX, iniY float64, fun func(val, ox, oy float64) (float64, float64)) (ox64, oy64 *tensor.Float64) {
	rows, cells := a.Shape().RowCellSize()
	ox64 = tensor.NewFloat64(cells)
	oy64 = tensor.NewFloat64(cells)
	if rows <= 0 {
		return ox64, oy64
	}
	if cells == 1 {
		ox := iniX
		oy := iniY
		switch x := a.(type) {
		case *tensor.Float64:
			for i := range rows {
				ox, oy = fun(x.Float1D(i), ox, oy)
			}
		case *tensor.Float32:
			for i := range rows {
				ox, oy = fun(x.Float1D(i), ox, oy)
			}
		default:
			for i := range rows {
				ox, oy = fun(a.Float1D(i), ox, oy)
			}
		}
		ox64.SetFloat1D(ox, 0)
		oy64.SetFloat1D(oy, 0)
		return
	}
	for j := range cells {
		ox64.SetFloat1D(iniX, j)
		oy64.SetFloat1D(iniY, j)
	}
	switch x := a.(type) {
	case *tensor.Float64:
		for i := range rows {
			for j := range cells {
				ox, oy := fun(x.Float1D(i*cells+j), ox64.Float1D(j), oy64.Float1D(j))
				ox64.SetFloat1D(ox, j)
				oy64.SetFloat1D(oy, j)
			}
		}
	case *tensor.Float32:
		for i := range rows {
			for j := range cells {
				ox, oy := fun(x.Float1D(i*cells+j), ox64.Float1D(j), oy64.Float1D(j))
				ox64.SetFloat1D(ox, j)
				oy64.SetFloat1D(oy, j)
			}
		}
	default:
		for i := range rows {
			for j := range cells {
				ox, oy := fun(a.Float1D(i*cells+j), ox64.Float1D(j), oy64.Float1D(j))
				ox64.SetFloat1D(ox, j)
				oy64.SetFloat1D(oy, j)
			}
		}
	}
	return
}
