// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
)

// Quantiles returns the given quantile(s) of non-NaN elements in given
// 1D tensor. Because sorting uses indexes, this only works for 1D case.
// If needed for a sub-space of values, that can be extracted through slicing
// and then used. Logs an error if not 1D.
// qs are 0-1 values, 0 = min, 1 = max, .5 = median, etc.
// Uses linear interpolation.
// Because this requires a sort, it is more efficient to get as many quantiles
// as needed in one pass.
func Quantiles(in, qs tensor.Tensor) tensor.Values {
	return tensor.CallOut2(QuantilesOut, in, qs)
}

// QuantilesOut returns the given quantile(s) of non-NaN elements in given
// 1D tensor. Because sorting uses indexes, this only works for 1D case.
// If needed for a sub-space of values, that can be extracted through slicing
// and then used. Returns and logs an error if not 1D.
// qs are 0-1 values, 0 = min, 1 = max, .5 = median, etc.
// Uses linear interpolation.
// Because this requires a sort, it is more efficient to get as many quantiles
// as needed in one pass.
func QuantilesOut(in, qs tensor.Tensor, out tensor.Values) error {
	if in.NumDims() != 1 {
		return errors.Log(errors.New("stats.QuantilesFunc: only 1D input tensors allowed"))
	}
	if qs.NumDims() != 1 {
		return errors.Log(errors.New("stats.QuantilesFunc: only 1D quantile tensors allowed"))
	}
	tensor.SetShapeFrom(out, in)
	sin := tensor.AsRows(in.AsValues())
	sin.ExcludeMissing()
	sin.Sort(tensor.Ascending)
	sz := len(sin.Indexes) - 1 // length of our own index list
	if sz <= 0 {
		out.(tensor.Values).SetZeros()
		return nil
	}
	fsz := float64(sz)
	nq := qs.Len()
	for i := range nq {
		q := qs.Float1D(i)
		val := 0.0
		qi := q * fsz
		lwi := math.Floor(qi)
		lwii := int(lwi)
		if lwii >= sz {
			val = sin.FloatRow(sz)
		} else if lwii < 0 {
			val = sin.FloatRow(0)
		} else {
			phi := qi - lwi
			lwv := sin.FloatRow(lwii)
			hiv := sin.FloatRow(lwii + 1)
			val = (1-phi)*lwv + phi*hiv
		}
		out.SetFloat1D(val, i)
	}
	return nil
}

// Median computes the median (50% quantile) of tensor values.
// See [StatsFunc] for general information.
func Median(in tensor.Tensor) tensor.Values {
	return Quantiles(in, tensor.NewFloat64Scalar(.5))
}

// Q1 computes the first quantile (25%) of tensor values.
// See [StatsFunc] for general information.
func Q1(in tensor.Tensor) tensor.Values {
	return Quantiles(in, tensor.NewFloat64Scalar(.25))
}

// Q3 computes the third quantile (75%) of tensor values.
// See [StatsFunc] for general information.
func Q3(in tensor.Tensor) tensor.Values {
	return Quantiles(in, tensor.NewFloat64Scalar(.75))
}

// MedianOut computes the median (50% quantile) of tensor values.
// See [StatsFunc] for general information.
func MedianOut(in tensor.Tensor, out tensor.Values) error {
	return QuantilesOut(in, tensor.NewFloat64Scalar(.5), out)
}

// Q1Out computes the first quantile (25%) of tensor values.
// See [StatsFunc] for general information.
func Q1Out(in tensor.Tensor, out tensor.Values) error {
	return QuantilesOut(in, tensor.NewFloat64Scalar(.25), out)
}

// Q3Out computes the third quantile (75%) of tensor values.
// See [StatsFunc] for general information.
func Q3Out(in tensor.Tensor, out tensor.Values) error {
	return QuantilesOut(in, tensor.NewFloat64Scalar(.75), out)
}
