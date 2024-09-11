// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate core generate

package metric

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
)

func init() {
	tensor.AddFunc(Euclidean.String(), EuclideanFunc, 1)
	tensor.AddFunc(SumSquares.String(), SumSquaresFunc, 1)
	tensor.AddFunc(Abs.String(), AbsFunc, 1)
	tensor.AddFunc(Hamming.String(), HammingFunc, 1)
	tensor.AddFunc(EuclideanBinTol.String(), EuclideanBinTolFunc, 1)
	tensor.AddFunc(SumSquaresBinTol.String(), SumSquaresBinTolFunc, 1)
	tensor.AddFunc(InvCosine.String(), InvCosineFunc, 1)
	tensor.AddFunc(InvCorrelation.String(), InvCorrelationFunc, 1)
	tensor.AddFunc(InnerProduct.String(), InnerProductFunc, 1)
	tensor.AddFunc(CrossEntropy.String(), CrossEntropyFunc, 1)
	tensor.AddFunc(Covariance.String(), CovarianceFunc, 1)
	tensor.AddFunc(Correlation.String(), CorrelationFunc, 1)
	tensor.AddFunc(Cosine.String(), CosineFunc, 1)
}

// Standard calls a standard Metrics enum function on given tensors.
// Output results are in the out tensor.
func Standard(metric Metrics, a, b, out *tensor.Indexed) {
	tensor.Call(metric.String(), a, b, out)
}

// StandardOut calls a standard Metrics enum function on given tensors,
// returning output as a newly created tensor.
func StandardOut(metric Metrics, a, b *tensor.Indexed) *tensor.Indexed {
	return errors.Log1(tensor.CallOut(metric.String(), a, b))[0] // note: error should never happen
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
