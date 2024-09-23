// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate core generate

package metric

import (
	"cogentcore.org/core/tensor"
)

func init() {
	tensor.AddFunc(Euclidean.FuncName(), EuclideanFunc, 1)
	tensor.AddFunc(SumSquares.FuncName(), SumSquaresFunc, 1)
	tensor.AddFunc(Abs.FuncName(), AbsFunc, 1)
	tensor.AddFunc(Hamming.FuncName(), HammingFunc, 1)
	tensor.AddFunc(EuclideanBinTol.FuncName(), EuclideanBinTolFunc, 1)
	tensor.AddFunc(SumSquaresBinTol.FuncName(), SumSquaresBinTolFunc, 1)
	tensor.AddFunc(InvCosine.FuncName(), InvCosineFunc, 1)
	tensor.AddFunc(InvCorrelation.FuncName(), InvCorrelationFunc, 1)
	tensor.AddFunc(InnerProduct.FuncName(), InnerProductFunc, 1)
	tensor.AddFunc(CrossEntropy.FuncName(), CrossEntropyFunc, 1)
	tensor.AddFunc(Covariance.FuncName(), CovarianceFunc, 1)
	tensor.AddFunc(Correlation.FuncName(), CorrelationFunc, 1)
	tensor.AddFunc(Cosine.FuncName(), CosineFunc, 1)
}

// Metrics are standard metric functions
type Metrics int32 //enums:enum

const (
	// Euclidean is the square root of the sum of squares differences
	// between tensor values, aka the L2Norm.
	Euclidean Metrics = iota

	// SumSquares is the sum of squares differences between tensor values.
	SumSquares

	// Abs is the sum of the absolute value of differences
	// between tensor values, aka the L1Norm.
	Abs

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

// FuncName returns the package-qualified function name to use
// in tensor.Call to call this function.
func (m Metrics) FuncName() string {
	return "metric." + m.String()
}

// Call calls a standard Metrics enum function on given tensors.
// Output results are in the out tensor.
func (m Metrics) Call(a, b, out tensor.Tensor) {
	tensor.Call(m.FuncName(), a, b, out)
}

// CallOut calls a standard Metrics enum function on given tensors,
// returning output as a newly created tensor.
func (m Metrics) CallOut(a, b tensor.Tensor) tensor.Tensor {
	return tensor.CallOut(m.FuncName(), a, b)
}

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
