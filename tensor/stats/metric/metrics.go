// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate core generate

package metric

import (
	"fmt"

	"cogentcore.org/core/tensor"
)

// Funcs is a registry of named metric functions,
// which can then be called by standard enum or
// string name for custom functions.
var Funcs map[string]MetricFunc

func init() {
	Funcs = make(map[string]MetricFunc)
	Funcs[Euclidean.String()] = EuclideanFunc
	Funcs[L2Norm.String()] = EuclideanFunc
	Funcs[SumSquares.String()] = SumSquaresFunc
	Funcs[Abs.String()] = AbsFunc
	Funcs[L1Norm.String()] = AbsFunc
	Funcs[Hamming.String()] = HammingFunc
	Funcs[EuclideanBinTol.String()] = EuclideanBinTolFunc
	Funcs[SumSquaresBinTol.String()] = SumSquaresBinTolFunc
	Funcs[InvCosine.String()] = InvCosineFunc
	Funcs[InvCorrelation.String()] = InvCorrelationFunc
	Funcs[InnerProduct.String()] = InnerProductFunc
	Funcs[CrossEntropy.String()] = CrossEntropyFunc
	Funcs[Covariance.String()] = CovarianceFunc
	Funcs[Correlation.String()] = CorrelationFunc
	Funcs[Cosine.String()] = CosineFunc
}

// Standard calls a standard Metrics enum function on given tensors.
// Output results are in the out tensor.
func Standard(metric Metrics, a, b, out *tensor.Indexed) {
	Funcs[metric.String()](a, b, out)
}

// Call calls a registered stats function on given tensors.
// Output results are in the out tensor.  Returns an
// error if name not found.
func Call(name string, a, b, out *tensor.Indexed) error {
	f, ok := Funcs[name]
	if !ok {
		return fmt.Errorf("metric.Call: function %q not registered", name)
	}
	f(a, b, out)
	return nil
}

// Metrics are standard metric functions
type Metrics int32 //enums:enum

const (
	// Euclidean is the square root of the sum of squares differences
	// between tensor values, aka the [L2Norm].
	Euclidean Metrics = iota

	// L2Norm is the square root of the sum of squares differences
	// between tensor values, aka [Euclidean] distance.
	L2Norm

	// SumSquares is the sum of squares differences between tensor values.
	SumSquares

	// Abs is the sum of the absolute value of differences
	// between tensor values, aka the [L1Norm].
	Abs

	// L1Norm is the sum of the absolute value of differences
	// between tensor values (same as [Abs]).
	L1Norm

	// Hamming is the sum of 1s for every element that is different,
	// i.e., "city block" distance.
	Hamming

	// EuclideanBinTol is the [Euclidean] square root of the sum of squares
	// differences between tensor values, with binary tolerance:
	// differences < 0.5 are thresholded to 0.
	EuclideanBinTol

	// SumSquaresBinTol is the [SumSquares] differences between tensor values,
	// with binary tolerance: differences < 0.5 are thresholded to 0.
	SumSquaresBinTol

	// InvCosine is 1-[Cosine], which is useful to convert it
	// to an Increasing metric where more different vectors have larger metric values.
	InvCosine

	// InvCorrelation is 1-[Correlation], which is useful to convert it
	// to an Increasing metric where more different vectors have larger metric values.
	InvCorrelation

	// CrossEntropy is a standard measure of the difference between two
	// probabilty distributions, reflecting the additional entropy (uncertainty) associated
	// with measuring probabilities under distribution b when in fact they come from
	// distribution a.  It is also the entropy of a plus the divergence between a from b,
	// using Kullback-Leibler (KL) divergence.  It is computed as:
	// a * log(a/b) + (1-a) * log(1-a/1-b).
	CrossEntropy

	/////////////////////////////////////////////////////////////////////////
	// Everything below here is !Increasing -- larger = closer, not farther

	// InnerProduct is the sum of the co-products of the tensor values.
	InnerProduct

	// Covariance is co-variance between two vectors,
	// i.e., the mean of the co-product of each vector element minus
	// the mean of that vector: cov(A,B) = E[(A - E(A))(B - E(B))].
	Covariance

	// Correlation is the standardized [Covariance] in the range (-1..1),
	// computed as the mean of the co-product of each vector
	// element minus the mean of that vector, normalized by the product of their
	// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
	// Equivalent to the [Cosine] of mean-normalized vectors.
	Correlation

	// Cosine is high-dimensional angle between two vectors,
	// in range (-1..1) as the normalized [InnerProduct]:
	// inner product / sqrt(ssA * ssB).  See also [Correlation].
	Cosine
)

// Increasing returns true if the distance metric is such that metric
// values increase as a function of distance (e.g., Euclidean)
// and false if metric values decrease as a function of distance
// (e.g., Cosine, Correlation)
func (m Metrics) Increasing() bool {
	if m >= InnerProduct {
		return false
	}
	return true
}
