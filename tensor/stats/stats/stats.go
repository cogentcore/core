// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
)

//go:generate core generate

func init() {
	tensor.AddFunc(Count.String(), CountFunc, 1)
	tensor.AddFunc(Sum.String(), SumFunc, 1)
	tensor.AddFunc(SumAbs.String(), SumAbsFunc, 1)
	tensor.AddFunc(L1Norm.String(), SumAbsFunc, 1)
	tensor.AddFunc(Prod.String(), ProdFunc, 1)
	tensor.AddFunc(Min.String(), MinFunc, 1)
	tensor.AddFunc(Max.String(), MaxFunc, 1)
	tensor.AddFunc(MinAbs.String(), MinAbsFunc, 1)
	tensor.AddFunc(MaxAbs.String(), MaxAbsFunc, 1)
	tensor.AddFunc(Mean.String(), MeanFunc, 1)
	tensor.AddFunc(Var.String(), VarFunc, 1)
	tensor.AddFunc(Std.String(), StdFunc, 1)
	tensor.AddFunc(Sem.String(), SemFunc, 1)
	tensor.AddFunc(SumSq.String(), SumSqFunc, 1)
	tensor.AddFunc(L2Norm.String(), L2NormFunc, 1)
	tensor.AddFunc(VarPop.String(), VarPopFunc, 1)
	tensor.AddFunc(StdPop.String(), StdPopFunc, 1)
	tensor.AddFunc(SemPop.String(), SemPopFunc, 1)
	tensor.AddFunc(Median.String(), MedianFunc, 1)
	tensor.AddFunc(Q1.String(), Q1Func, 1)
	tensor.AddFunc(Q3.String(), Q3Func, 1)
}

// Stat calls a standard Stats enum function on given tensors.
// Output results are in the out tensor.
func Stat(stat Stats, in, out *tensor.Indexed) {
	tensor.Call(stat.String(), in, out)
}

// StatOut calls a standard Stats enum function on given tensor,
// returning output as a newly created tensor.
func StatOut(stat Stats, in *tensor.Indexed) *tensor.Indexed {
	return errors.Log1(tensor.CallOut(stat.String(), in))[0] // note: error should never happen
}

// Stats is a list of different standard aggregation functions, which can be used
// to choose an aggregation function
type Stats int32 //enums:enum

const (
	// count of number of elements.
	Count Stats = iota

	// sum of elements.
	Sum

	// sum of absolute-value-of elements (= L1Norm).
	SumAbs

	// L1 Norm: sum of absolute values (= SumAbs).
	L1Norm

	// product of elements.
	Prod

	// minimum value.
	Min

	// maximum value.
	Max

	// minimum of absolute values.
	MinAbs

	// maximum of absolute values.
	MaxAbs

	// mean value = sum / count.
	Mean

	// sample variance (squared deviations from mean, divided by n-1).
	Var

	// sample standard deviation (sqrt of Var).
	Std

	// sample standard error of the mean (Std divided by sqrt(n)).
	Sem

	// sum of squared values.
	SumSq

	// L2 Norm: square-root of sum-of-squares.
	L2Norm

	// population variance (squared diffs from mean, divided by n).
	VarPop

	// population standard deviation (sqrt of VarPop).
	StdPop

	// population standard error of the mean (StdPop divided by sqrt(n)).
	SemPop

	// middle value in sorted ordering.
	Median

	// Q1 first quartile = 25%ile value = .25 quantile value.
	Q1

	// Q3 third quartile = 75%ile value = .75 quantile value.
	Q3
)
