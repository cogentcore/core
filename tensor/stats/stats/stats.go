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
	tensor.AddFunc(StatCount.FuncName(), Count)
	tensor.AddFunc(StatSum.FuncName(), Sum)
	tensor.AddFunc(StatSumAbs.FuncName(), SumAbs)
	tensor.AddFunc(StatL1Norm.FuncName(), SumAbs)
	tensor.AddFunc(StatProd.FuncName(), Prod)
	tensor.AddFunc(StatMin.FuncName(), Min)
	tensor.AddFunc(StatMax.FuncName(), Max)
	tensor.AddFunc(StatMinAbs.FuncName(), MinAbs)
	tensor.AddFunc(StatMaxAbs.FuncName(), MaxAbs)
	tensor.AddFunc(StatMean.FuncName(), Mean)
	tensor.AddFunc(StatVar.FuncName(), Var)
	tensor.AddFunc(StatStd.FuncName(), Std)
	tensor.AddFunc(StatSem.FuncName(), Sem)
	tensor.AddFunc(StatSumSq.FuncName(), SumSq)
	tensor.AddFunc(StatL2Norm.FuncName(), L2Norm)
	tensor.AddFunc(StatVarPop.FuncName(), VarPop)
	tensor.AddFunc(StatStdPop.FuncName(), StdPop)
	tensor.AddFunc(StatSemPop.FuncName(), SemPop)
	tensor.AddFunc(StatMedian.FuncName(), Median)
	tensor.AddFunc(StatQ1.FuncName(), Q1)
	tensor.AddFunc(StatQ3.FuncName(), Q3)
}

// Stats is a list of different standard aggregation functions, which can be used
// to choose an aggregation function
type Stats int32 //enums:enum -trim-prefix Stat

const (
	// count of number of elements.
	StatCount Stats = iota

	// sum of elements.
	StatSum

	// sum of absolute-value-of elements (= L1Norm).
	StatSumAbs

	// L1 Norm: sum of absolute values (= SumAbs).
	StatL1Norm

	// product of elements.
	StatProd

	// minimum value.
	StatMin

	// maximum value.
	StatMax

	// minimum of absolute values.
	StatMinAbs

	// maximum of absolute values.
	StatMaxAbs

	// mean value = sum / count.
	StatMean

	// sample variance (squared deviations from mean, divided by n-1).
	StatVar

	// sample standard deviation (sqrt of Var).
	StatStd

	// sample standard error of the mean (Std divided by sqrt(n)).
	StatSem

	// sum of squared values.
	StatSumSq

	// L2 Norm: square-root of sum-of-squares.
	StatL2Norm

	// population variance (squared diffs from mean, divided by n).
	StatVarPop

	// population standard deviation (sqrt of VarPop).
	StatStdPop

	// population standard error of the mean (StdPop divided by sqrt(n)).
	StatSemPop

	// middle value in sorted ordering.
	StatMedian

	// Q1 first quartile = 25%ile value = .25 quantile value.
	StatQ1

	// Q3 third quartile = 75%ile value = .75 quantile value.
	StatQ3
)

// FuncName returns the package-qualified function name to use
// in tensor.Call to call this function.
func (s Stats) FuncName() string {
	return "stats." + s.String()
}

// Func returns function for given stat.
func (s Stats) Func() StatsFunc {
	fn := errors.Log1(tensor.FuncByName(s.FuncName()))
	return fn.Fun.(StatsFunc)
}

// Call calls this statistic function on given tensors.
// returning output as a newly created tensor.
func (s Stats) Call(in tensor.Tensor) tensor.Values {
	return s.Func()(in)
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

// AsStatsFunc returns given function as a [StatsFunc] function,
// or an error if it does not fit that signature.
func AsStatsFunc(fun any) (StatsFunc, error) {
	sfun, ok := fun.(StatsFunc)
	if !ok {
		return nil, errors.New("metric.AsStatsFunc: function does not fit the StatsFunc signature")
	}
	return sfun, nil
}
