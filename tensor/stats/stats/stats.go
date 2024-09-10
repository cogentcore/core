// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import "cogentcore.org/core/tensor"

//go:generate core generate

// StatsFuncs is a registry of named stats functions,
// which can then be called by standard enum or
// string name for custom functions.
var StatsFuncs map[string]StatsFunc

func init() {
	StatsFuncs = make(map[string]StatsFunc)
	StatsFuncs[Count.String()] = CountFunc
	StatsFuncs[Sum.String()] = SumFunc
	StatsFuncs[SumAbs.String()] = SumAbsFunc
	StatsFuncs[L1Norm.String()] = SumAbsFunc
	StatsFuncs[Prod.String()] = ProdFunc
	StatsFuncs[Min.String()] = MinFunc
	StatsFuncs[Max.String()] = MaxFunc
	StatsFuncs[MinAbs.String()] = MinAbsFunc
	StatsFuncs[MaxAbs.String()] = MaxAbsFunc
	StatsFuncs[Mean.String()] = MeanFunc
	StatsFuncs[Var.String()] = VarFunc
	StatsFuncs[Std.String()] = StdFunc
	StatsFuncs[Sem.String()] = SemFunc
	StatsFuncs[SumSq.String()] = SumSqFunc
	StatsFuncs[L2Norm.String()] = L2NormFunc
	StatsFuncs[VarPop.String()] = VarPopFunc
	StatsFuncs[StdPop.String()] = StdPopFunc
	StatsFuncs[SemPop.String()] = SemPopFunc
	StatsFuncs[Median.String()] = MedianFunc
	StatsFuncs[Q1.String()] = Q1Func
	StatsFuncs[Q3.String()] = Q3Func
}

// Standard calls a standard Stats enum function on given tensors.
// Output results are in the out tensor.
func Standard(stat Stats, in, out *tensor.Indexed) {
	StatsFuncs[stat.String()](in, out)
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
