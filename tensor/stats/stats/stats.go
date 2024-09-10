// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"fmt"

	"cogentcore.org/core/tensor"
)

//go:generate core generate

// Funcs is a registry of named stats functions,
// which can then be called by standard enum or
// string name for custom functions.
var Funcs map[string]StatsFunc

func init() {
	Funcs = make(map[string]StatsFunc)
	Funcs[Count.String()] = CountFunc
	Funcs[Sum.String()] = SumFunc
	Funcs[SumAbs.String()] = SumAbsFunc
	Funcs[L1Norm.String()] = SumAbsFunc
	Funcs[Prod.String()] = ProdFunc
	Funcs[Min.String()] = MinFunc
	Funcs[Max.String()] = MaxFunc
	Funcs[MinAbs.String()] = MinAbsFunc
	Funcs[MaxAbs.String()] = MaxAbsFunc
	Funcs[Mean.String()] = MeanFunc
	Funcs[Var.String()] = VarFunc
	Funcs[Std.String()] = StdFunc
	Funcs[Sem.String()] = SemFunc
	Funcs[SumSq.String()] = SumSqFunc
	Funcs[L2Norm.String()] = L2NormFunc
	Funcs[VarPop.String()] = VarPopFunc
	Funcs[StdPop.String()] = StdPopFunc
	Funcs[SemPop.String()] = SemPopFunc
	Funcs[Median.String()] = MedianFunc
	Funcs[Q1.String()] = Q1Func
	Funcs[Q3.String()] = Q3Func
}

// Standard calls a standard Stats enum function on given tensors.
// Output results are in the out tensor.
func Standard(stat Stats, in, out *tensor.Indexed) {
	Funcs[stat.String()](in, out)
}

// Call calls a registered stats function on given tensors.
// Output results are in the out tensor.  Returns an
// error if name not found.
func Call(name string, in, out *tensor.Indexed) error {
	f, ok := Funcs[name]
	if !ok {
		return fmt.Errorf("stats.Call: function %q not registered", name)
	}
	f(in, out)
	return nil
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
