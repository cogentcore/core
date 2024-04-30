// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

//go:generate core generate

// Stats is a list of different standard aggregation functions, which can be used
// to choose an aggregation function
type Stats int32 //enums:enum

const (
	// count of number of elements
	Count Stats = iota

	// sum of elements
	Sum

	// product of elements
	Prod

	// minimum value
	Min

	// max maximum value
	Max

	// minimum absolute value
	MinAbs

	// maximum absolute value
	MaxAbs

	// mean mean value
	Mean

	// sample variance (squared diffs from mean, divided by n-1)
	Var

	// sample standard deviation (sqrt of Var)
	Std

	// sample standard error of the mean (Std divided by sqrt(n))
	Sem

	// L1 Norm: sum of absolute values
	L1Norm

	// sum of squared values
	SumSq

	// L2 Norm: square-root of sum-of-squares
	L2Norm

	// population variance (squared diffs from mean, divided by n)
	VarPop

	// population standard deviation (sqrt of VarPop)
	StdPop

	// population standard error of the mean (StdPop divided by sqrt(n))
	SemPop

	// middle value in sorted ordering
	Median

	// Q1 first quartile = 25%ile value = .25 quantile value
	Q1

	// Q3 third quartile = 75%ile value = .75 quantile value
	Q3
)
