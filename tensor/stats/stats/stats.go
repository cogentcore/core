// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
)

//go:generate core generate

func init() {
	tensor.AddFunc(Count.FuncName(), CountFunc, 1)
	tensor.AddFunc(Sum.FuncName(), SumFunc, 1)
	tensor.AddFunc(SumAbs.FuncName(), SumAbsFunc, 1)
	tensor.AddFunc(L1Norm.FuncName(), SumAbsFunc, 1)
	tensor.AddFunc(Prod.FuncName(), ProdFunc, 1)
	tensor.AddFunc(Min.FuncName(), MinFunc, 1)
	tensor.AddFunc(Max.FuncName(), MaxFunc, 1)
	tensor.AddFunc(MinAbs.FuncName(), MinAbsFunc, 1)
	tensor.AddFunc(MaxAbs.FuncName(), MaxAbsFunc, 1)
	tensor.AddFunc(Mean.FuncName(), MeanFunc, 1)
	tensor.AddFunc(Var.FuncName(), VarFunc, 1)
	tensor.AddFunc(Std.FuncName(), StdFunc, 1)
	tensor.AddFunc(Sem.FuncName(), SemFunc, 1)
	tensor.AddFunc(SumSq.FuncName(), SumSqFunc, 1)
	tensor.AddFunc(L2Norm.FuncName(), L2NormFunc, 1)
	tensor.AddFunc(VarPop.FuncName(), VarPopFunc, 1)
	tensor.AddFunc(StdPop.FuncName(), StdPopFunc, 1)
	tensor.AddFunc(SemPop.FuncName(), SemPopFunc, 1)
	tensor.AddFunc(Median.FuncName(), MedianFunc, 1)
	tensor.AddFunc(Q1.FuncName(), Q1Func, 1)
	tensor.AddFunc(Q3.FuncName(), Q3Func, 1)
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

// FuncName returns the package-qualified function name to use
// in tensor.Call to call this function.
func (s Stats) FuncName() string {
	return "stats." + s.String()
}

// StripPackage removes any package name from given string,
// used for naming based on FuncName() which could be custom
// or have a package prefix.
func StripPackage(name string) string {
	spl := strings.Split(name, ".")
	if len(spl) > 1 {
		return spl[len(spl)-1]
	}
	return name
}

// Stat calls a standard Stats enum function on given tensors.
// Output results are in the out tensor.
func Stat(stat Stats, in, out *tensor.Indexed) {
	tensor.Call(stat.FuncName(), in, out)
}

// StatOut calls a standard Stats enum function on given tensor,
// returning output as a newly created tensor.
func StatOut(stat Stats, in *tensor.Indexed) *tensor.Indexed {
	return errors.Log1(tensor.CallOut(stat.FuncName(), in))[0] // note: error should never happen
}
