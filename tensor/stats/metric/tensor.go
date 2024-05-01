// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"math"

	"cogentcore.org/core/tensor"
)

// ClosestRow32 returns the closest fit between probe pattern and patterns in
// an tensor with float32 data where the outer-most dimension is assumed to be a row
// (e.g., as a column in an table), using the given metric function,
// *which must have the Increasing property* -- i.e., larger = further.
// returns the row and metric value for that row.
// Col cell sizes must match size of probe (panics if not).
func ClosestRow32(probe tensor.Tensor, col tensor.Tensor, mfun Func32) (int, float32) {
	pr := probe.(*tensor.Float32)
	cl := col.(*tensor.Float32)
	rows := col.Shape().DimSize(0)
	csz := col.Len() / rows
	if csz != probe.Len() {
		panic("metric.ClosestRow32: probe size != cell size of tensor column!\n")
	}
	ci := -1
	minv := float32(math.MaxFloat32)
	for ri := 0; ri < rows; ri++ {
		st := ri * csz
		rvals := cl.Values[st : st+csz]
		v := mfun(pr.Values, rvals)
		if v < minv {
			ci = ri
			minv = v
		}
	}
	return ci, minv
}

// ClosestRow64 returns the closest fit between probe pattern and patterns in
// a tensor with float64 data where the outer-most dimension is assumed to be a row
// (e.g., as a column in an table), using the given metric function,
// *which must have the Increasing property* -- i.e., larger = further.
// returns the row and metric value for that row.
// Col cell sizes must match size of probe (panics if not).
func ClosestRow64(probe tensor.Tensor, col tensor.Tensor, mfun Func64) (int, float64) {
	pr := probe.(*tensor.Float64)
	cl := col.(*tensor.Float64)
	rows := col.DimSize(0)
	csz := col.Len() / rows
	if csz != probe.Len() {
		panic("metric.ClosestRow64: probe size != cell size of tensor column!\n")
	}
	ci := -1
	minv := math.MaxFloat64
	for ri := 0; ri < rows; ri++ {
		st := ri * csz
		rvals := cl.Values[st : st+csz]
		v := mfun(pr.Values, rvals)
		if v < minv {
			ci = ri
			minv = v
		}
	}
	return ci, minv
}

// ClosestRow32Py returns the closest fit between probe pattern and patterns in
// an tensor.Float32 where the outer-most dimension is assumed to be a row
// (e.g., as a column in an table), using the given metric function,
// *which must have the Increasing property* -- i.e., larger = further.
// returns the row and metric value for that row.
// Col cell sizes must match size of probe (panics if not).
// Py version is for Python, returns a slice with row, cor, takes std metric
func ClosestRow32Py(probe tensor.Tensor, col tensor.Tensor, std StdMetrics) []float32 {
	row, cor := ClosestRow32(probe, col, StdFunc32(std))
	return []float32{float32(row), cor}
}

// ClosestRow64Py returns the closest fit between probe pattern and patterns in
// an tensor.Tensor where the outer-most dimension is assumed to be a row
// (e.g., as a column in an table), using the given metric function,
// *which must have the Increasing property* -- i.e., larger = further.
// returns the row and metric value for that row.
// Col cell sizes must match size of probe (panics if not).
// Optimized for tensor.Float64 but works for any tensor.
// Py version is for Python, returns a slice with row, cor, takes std metric
func ClosestRow64Py(probe tensor.Tensor, col tensor.Tensor, std StdMetrics) []float64 {
	row, cor := ClosestRow64(probe, col, StdFunc64(std))
	return []float64{float64(row), cor}
}
