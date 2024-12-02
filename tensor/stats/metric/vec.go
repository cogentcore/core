// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"cogentcore.org/core/tensor"
)

// VectorizeOut64 is the general compute function for metric.
// This version makes a Float64 output tensor for aggregating
// and computing values, and then copies the results back to the
// original output. This allows metric functions to operate directly
// on integer valued inputs and produce sensible results.
// It returns the Float64 output tensor for further processing as needed.
// a and b are already enforced to be the same shape.
func VectorizeOut64(a, b tensor.Tensor, out tensor.Values, ini float64, fun func(a, b, agg float64) float64) *tensor.Float64 {
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
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					agg = fun(x.Float1D(i), y.Float1D(i), agg)
				}
			case *tensor.Float32:
				for i := range rows {
					agg = fun(x.Float1D(i), y.Float1D(i), agg)
				}
			default:
				for i := range rows {
					agg = fun(x.Float1D(i), b.Float1D(i), agg)
				}
			}
		case *tensor.Float32:
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					agg = fun(x.Float1D(i), y.Float1D(i), agg)
				}
			case *tensor.Float32:
				for i := range rows {
					agg = fun(x.Float1D(i), y.Float1D(i), agg)
				}
			default:
				for i := range rows {
					agg = fun(x.Float1D(i), b.Float1D(i), agg)
				}
			}
		default:
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					agg = fun(a.Float1D(i), y.Float1D(i), agg)
				}
			case *tensor.Float32:
				for i := range rows {
					agg = fun(a.Float1D(i), y.Float1D(i), agg)
				}
			default:
				for i := range rows {
					agg = fun(a.Float1D(i), b.Float1D(i), agg)
				}
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
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(x.Float1D(si+j), y.Float1D(si+j), o64.Float1D(j)), j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(x.Float1D(si+j), y.Float1D(si+j), o64.Float1D(j)), j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(x.Float1D(si+j), b.Float1D(si+j), o64.Float1D(j)), j)
				}
			}
		}
	case *tensor.Float32:
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(x.Float1D(si+j), y.Float1D(si+j), o64.Float1D(j)), j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(x.Float1D(si+j), y.Float1D(si+j), o64.Float1D(j)), j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(x.Float1D(si+j), b.Float1D(si+j), o64.Float1D(j)), j)
				}
			}
		}
	default:
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(a.Float1D(si+j), y.Float1D(si+j), o64.Float1D(j)), j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(a.Float1D(si+j), y.Float1D(si+j), o64.Float1D(j)), j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(a.Float1D(si+j), b.Float1D(si+j), o64.Float1D(j)), j)
				}
			}
		}
	}
	for j := range cells {
		out.SetFloat1D(o64.Float1D(j), j)
	}
	return o64
}

// VectorizePreOut64 is a version of [VectorizeOut64] that takes additional
// tensor.Float64 inputs of pre-computed values, e.g., the means of each output cell.
func VectorizePreOut64(a, b tensor.Tensor, out tensor.Values, ini float64, preA, preB *tensor.Float64, fun func(a, b, preA, preB, agg float64) float64) *tensor.Float64 {
	rows, cells := a.Shape().RowCellSize()
	o64 := tensor.NewFloat64(cells)
	if rows <= 0 {
		return o64
	}
	if cells == 1 {
		out.SetShapeSizes(1)
		agg := ini
		prevA := preA.Float1D(0)
		prevB := preB.Float1D(0)
		switch x := a.(type) {
		case *tensor.Float64:
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					agg = fun(x.Float1D(i), y.Float1D(i), prevA, prevB, agg)
				}
			case *tensor.Float32:
				for i := range rows {
					agg = fun(x.Float1D(i), y.Float1D(i), prevA, prevB, agg)
				}
			default:
				for i := range rows {
					agg = fun(x.Float1D(i), b.Float1D(i), prevA, prevB, agg)
				}
			}
		case *tensor.Float32:
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					agg = fun(x.Float1D(i), y.Float1D(i), prevA, prevB, agg)
				}
			case *tensor.Float32:
				for i := range rows {
					agg = fun(x.Float1D(i), y.Float1D(i), prevA, prevB, agg)
				}
			default:
				for i := range rows {
					agg = fun(x.Float1D(i), b.Float1D(i), prevA, prevB, agg)
				}
			}
		default:
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					agg = fun(a.Float1D(i), y.Float1D(i), prevA, prevB, agg)
				}
			case *tensor.Float32:
				for i := range rows {
					agg = fun(a.Float1D(i), y.Float1D(i), prevA, prevB, agg)
				}
			default:
				for i := range rows {
					agg = fun(a.Float1D(i), b.Float1D(i), prevA, prevB, agg)
				}
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
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(x.Float1D(si+j), y.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), o64.Float1D(j)), j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(x.Float1D(si+j), y.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), o64.Float1D(j)), j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(x.Float1D(si+j), b.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), o64.Float1D(j)), j)
				}
			}
		}
	case *tensor.Float32:
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(x.Float1D(si+j), y.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), o64.Float1D(j)), j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(x.Float1D(si+j), y.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), o64.Float1D(j)), j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(x.Float1D(si+j), b.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), o64.Float1D(j)), j)
				}
			}
		}
	default:
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(a.Float1D(si+j), y.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), o64.Float1D(j)), j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(a.Float1D(si+j), y.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), o64.Float1D(j)), j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					o64.SetFloat1D(fun(a.Float1D(si+j), b.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), o64.Float1D(j)), j)
				}
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
func Vectorize2Out64(a, b tensor.Tensor, iniX, iniY float64, fun func(a, b, ox, oy float64) (float64, float64)) (ox64, oy64 *tensor.Float64) {
	rows, cells := a.Shape().RowCellSize()
	ox64 = tensor.NewFloat64(cells)
	oy64 = tensor.NewFloat64(cells)
	if rows <= 0 {
		return
	}
	if cells == 1 {
		ox := iniX
		oy := iniY
		switch x := a.(type) {
		case *tensor.Float64:
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					ox, oy = fun(x.Float1D(i), y.Float1D(i), ox, oy)
				}
			case *tensor.Float32:
				for i := range rows {
					ox, oy = fun(x.Float1D(i), y.Float1D(i), ox, oy)
				}
			default:
				for i := range rows {
					ox, oy = fun(x.Float1D(i), b.Float1D(i), ox, oy)
				}
			}
		case *tensor.Float32:
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					ox, oy = fun(x.Float1D(i), y.Float1D(i), ox, oy)
				}
			case *tensor.Float32:
				for i := range rows {
					ox, oy = fun(x.Float1D(i), y.Float1D(i), ox, oy)
				}
			default:
				for i := range rows {
					ox, oy = fun(x.Float1D(i), b.Float1D(i), ox, oy)
				}
			}
		default:
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					ox, oy = fun(a.Float1D(i), y.Float1D(i), ox, oy)
				}
			case *tensor.Float32:
				for i := range rows {
					ox, oy = fun(a.Float1D(i), y.Float1D(i), ox, oy)
				}
			default:
				for i := range rows {
					ox, oy = fun(a.Float1D(i), b.Float1D(i), ox, oy)
				}
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
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy := fun(x.Float1D(si+j), y.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy := fun(x.Float1D(si+j), y.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy := fun(x.Float1D(si+j), b.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
				}
			}
		}
	case *tensor.Float32:
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy := fun(x.Float1D(si+j), y.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy := fun(x.Float1D(si+j), y.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy := fun(x.Float1D(si+j), b.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
				}
			}
		}
	default:
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy := fun(a.Float1D(si+j), y.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy := fun(a.Float1D(si+j), y.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy := fun(a.Float1D(si+j), b.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
				}
			}
		}
	}
	return
}

// Vectorize3Out64 is a version of [VectorizeOut64] that has 3 outputs instead of 1.
func Vectorize3Out64(a, b tensor.Tensor, iniX, iniY, iniZ float64, fun func(a, b, ox, oy, oz float64) (float64, float64, float64)) (ox64, oy64, oz64 *tensor.Float64) {
	rows, cells := a.Shape().RowCellSize()
	ox64 = tensor.NewFloat64(cells)
	oy64 = tensor.NewFloat64(cells)
	oz64 = tensor.NewFloat64(cells)
	if rows <= 0 {
		return
	}
	if cells == 1 {
		ox := iniX
		oy := iniY
		oz := iniZ
		switch x := a.(type) {
		case *tensor.Float64:
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					ox, oy, oz = fun(x.Float1D(i), y.Float1D(i), ox, oy, oz)
				}
			case *tensor.Float32:
				for i := range rows {
					ox, oy, oz = fun(x.Float1D(i), y.Float1D(i), ox, oy, oz)
				}
			default:
				for i := range rows {
					ox, oy, oz = fun(x.Float1D(i), b.Float1D(i), ox, oy, oz)
				}
			}
		case *tensor.Float32:
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					ox, oy, oz = fun(x.Float1D(i), y.Float1D(i), ox, oy, oz)
				}
			case *tensor.Float32:
				for i := range rows {
					ox, oy, oz = fun(x.Float1D(i), y.Float1D(i), ox, oy, oz)
				}
			default:
				for i := range rows {
					ox, oy, oz = fun(x.Float1D(i), b.Float1D(i), ox, oy, oz)
				}
			}
		default:
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					ox, oy, oz = fun(a.Float1D(i), y.Float1D(i), ox, oy, oz)
				}
			case *tensor.Float32:
				for i := range rows {
					ox, oy, oz = fun(a.Float1D(i), y.Float1D(i), ox, oy, oz)
				}
			default:
				for i := range rows {
					ox, oy, oz = fun(a.Float1D(i), b.Float1D(i), ox, oy, oz)
				}
			}
		}
		ox64.SetFloat1D(ox, 0)
		oy64.SetFloat1D(oy, 0)
		oz64.SetFloat1D(oz, 0)
		return
	}
	for j := range cells {
		ox64.SetFloat1D(iniX, j)
		oy64.SetFloat1D(iniY, j)
		oz64.SetFloat1D(iniZ, j)
	}
	switch x := a.(type) {
	case *tensor.Float64:
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(x.Float1D(si+j), y.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(x.Float1D(si+j), y.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(x.Float1D(si+j), b.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		}
	case *tensor.Float32:
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(x.Float1D(si+j), y.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(x.Float1D(si+j), y.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(x.Float1D(si+j), b.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		}
	default:
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(a.Float1D(si+j), y.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(a.Float1D(si+j), y.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(a.Float1D(si+j), b.Float1D(si+j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		}
	}
	return
}

// VectorizePre3Out64 is a version of [VectorizePreOut64] that takes additional
// tensor.Float64 inputs of pre-computed values, e.g., the means of each output cell,
// and has 3 outputs instead of 1.
func VectorizePre3Out64(a, b tensor.Tensor, iniX, iniY, iniZ float64, preA, preB *tensor.Float64, fun func(a, b, preA, preB, ox, oy, oz float64) (float64, float64, float64)) (ox64, oy64, oz64 *tensor.Float64) {
	rows, cells := a.Shape().RowCellSize()
	ox64 = tensor.NewFloat64(cells)
	oy64 = tensor.NewFloat64(cells)
	oz64 = tensor.NewFloat64(cells)
	if rows <= 0 {
		return
	}
	if cells == 1 {
		ox := iniX
		oy := iniY
		oz := iniZ
		prevA := preA.Float1D(0)
		prevB := preB.Float1D(0)
		switch x := a.(type) {
		case *tensor.Float64:
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					ox, oy, oz = fun(x.Float1D(i), y.Float1D(i), prevA, prevB, ox, oy, oz)
				}
			case *tensor.Float32:
				for i := range rows {
					ox, oy, oz = fun(x.Float1D(i), y.Float1D(i), prevA, prevB, ox, oy, oz)
				}
			default:
				for i := range rows {
					ox, oy, oz = fun(x.Float1D(i), b.Float1D(i), prevA, prevB, ox, oy, oz)
				}
			}
		case *tensor.Float32:
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					ox, oy, oz = fun(x.Float1D(i), y.Float1D(i), prevA, prevB, ox, oy, oz)
				}
			case *tensor.Float32:
				for i := range rows {
					ox, oy, oz = fun(x.Float1D(i), y.Float1D(i), prevA, prevB, ox, oy, oz)
				}
			default:
				for i := range rows {
					ox, oy, oz = fun(x.Float1D(i), b.Float1D(i), prevA, prevB, ox, oy, oz)
				}
			}
		default:
			switch y := b.(type) {
			case *tensor.Float64:
				for i := range rows {
					ox, oy, oz = fun(a.Float1D(i), y.Float1D(i), prevA, prevB, ox, oy, oz)
				}
			case *tensor.Float32:
				for i := range rows {
					ox, oy, oz = fun(a.Float1D(i), y.Float1D(i), prevA, prevB, ox, oy, oz)
				}
			default:
				for i := range rows {
					ox, oy, oz = fun(a.Float1D(i), b.Float1D(i), prevA, prevB, ox, oy, oz)
				}
			}
		}
		ox64.SetFloat1D(ox, 0)
		oy64.SetFloat1D(oy, 0)
		oz64.SetFloat1D(oz, 0)
		return
	}
	for j := range cells {
		ox64.SetFloat1D(iniX, j)
		oy64.SetFloat1D(iniY, j)
		oz64.SetFloat1D(iniZ, j)
	}
	switch x := a.(type) {
	case *tensor.Float64:
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(x.Float1D(si+j), y.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(x.Float1D(si+j), y.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(x.Float1D(si+j), b.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		}
	case *tensor.Float32:
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(x.Float1D(si+j), y.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(x.Float1D(si+j), y.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(x.Float1D(si+j), b.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		}
	default:
		switch y := b.(type) {
		case *tensor.Float64:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(a.Float1D(si+j), y.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		case *tensor.Float32:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(a.Float1D(si+j), y.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		default:
			for i := range rows {
				si := i * cells
				for j := range cells {
					ox, oy, oz := fun(a.Float1D(si+j), b.Float1D(si+j), preA.Float1D(j), preB.Float1D(j), ox64.Float1D(j), oy64.Float1D(j), oz64.Float1D(j))
					ox64.SetFloat1D(ox, j)
					oy64.SetFloat1D(oy, j)
					oz64.SetFloat1D(oz, j)
				}
			}
		}
	}
	return
}
