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
	tensor.AddFunc(MetricEuclidean.FuncName(), Euclidean, 1)
	tensor.AddFunc(MetricSumSquares.FuncName(), SumSquares, 1)
	tensor.AddFunc(MetricAbs.FuncName(), Abs, 1)
	tensor.AddFunc(MetricHamming.FuncName(), Hamming, 1)
	tensor.AddFunc(MetricEuclideanBinTol.FuncName(), EuclideanBinTol, 1)
	tensor.AddFunc(MetricSumSquaresBinTol.FuncName(), SumSquaresBinTol, 1)
	tensor.AddFunc(MetricInvCosine.FuncName(), InvCosine, 1)
	tensor.AddFunc(MetricInvCorrelation.FuncName(), InvCorrelation, 1)
	tensor.AddFunc(MetricInnerProduct.FuncName(), InnerProduct, 1)
	tensor.AddFunc(MetricCrossEntropy.FuncName(), CrossEntropy, 1)
	tensor.AddFunc(MetricCovariance.FuncName(), Covariance, 1)
	tensor.AddFunc(MetricCorrelation.FuncName(), Correlation, 1)
	tensor.AddFunc(MetricCosine.FuncName(), Cosine, 1)
}

// Metrics are standard metric functions
type Metrics int32 //enums:enum -trim-prefix Metric

const (
	// Euclidean is the square root of the sum of squares differences
	// between tensor values, aka the L2Norm.
	MetricEuclidean Metrics = iota

	// SumSquares is the sum of squares differences between tensor values.
	MetricSumSquares

	// Abs is the sum of the absolute value of differences
	// between tensor values, aka the L1Norm.
	MetricAbs

	// Hamming is the sum of 1s for every element that is different,
	// i.e., "city block" distance.
	MetricHamming

	// EuclideanBinTol is the [Euclidean] square root of the sum of squares
	// differences between tensor values, with binary tolerance:
	// differences < 0.5 are thresholded to 0.
	MetricEuclideanBinTol

	// SumSquaresBinTol is the [SumSquares] differences between tensor values,
	// with binary tolerance: differences < 0.5 are thresholded to 0.
	MetricSumSquaresBinTol

	// InvCosine is 1-[Cosine], which is useful to convert it
	// to an Increasing metric where more different vectors have larger metric values.
	MetricInvCosine

	// InvCorrelation is 1-[Correlation], which is useful to convert it
	// to an Increasing metric where more different vectors have larger metric values.
	MetricInvCorrelation

	// CrossEntropy is a standard measure of the difference between two
	// probabilty distributions, reflecting the additional entropy (uncertainty) associated
	// with measuring probabilities under distribution b when in fact they come from
	// distribution a.  It is also the entropy of a plus the divergence between a from b,
	// using Kullback-Leibler (KL) divergence.  It is computed as:
	// a * log(a/b) + (1-a) * log(1-a/1-b).
	MetricCrossEntropy

	/////////////////////////////////////////////////////////////////////////
	// Everything below here is !Increasing -- larger = closer, not farther

	// InnerProduct is the sum of the co-products of the tensor values.
	MetricInnerProduct

	// Covariance is co-variance between two vectors,
	// i.e., the mean of the co-product of each vector element minus
	// the mean of that vector: cov(A,B) = E[(A - E(A))(B - E(B))].
	MetricCovariance

	// Correlation is the standardized [Covariance] in the range (-1..1),
	// computed as the mean of the co-product of each vector
	// element minus the mean of that vector, normalized by the product of their
	// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
	// Equivalent to the [Cosine] of mean-normalized vectors.
	MetricCorrelation

	// Cosine is high-dimensional angle between two vectors,
	// in range (-1..1) as the normalized [InnerProduct]:
	// inner product / sqrt(ssA * ssB).  See also [Correlation].
	MetricCosine
)

// FuncName returns the package-qualified function name to use
// in tensor.Call to call this function.
func (m Metrics) FuncName() string {
	return "metric." + m.String()
}

// Func returns function for given metric.
func (m Metrics) Func() MetricFunc {
	fn := errors.Log1(tensor.FuncByName(m.FuncName()))
	return fn.Fun.(MetricFunc)
}

// Call calls a standard Metrics enum function on given tensors.
// Output results are in the out tensor.
func (m Metrics) Call(a, b, out tensor.Tensor) error {
	return tensor.Call(m.FuncName(), a, b, out)
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
	if m >= MetricInnerProduct {
		return false
	}
	return true
}

// AsMetricFunc returns given function as a [MetricFunc] function,
// or an error if it does not fit that signature.
func AsMetricFunc(fun any) (MetricFunc, error) {
	mfun, ok := fun.(MetricFunc)
	if !ok {
		return nil, errors.New("metric.AsMetricFunc: function does not fit the MetricFunc signature")
	}
	return mfun, nil
}
