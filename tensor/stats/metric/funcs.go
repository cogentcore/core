// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"math"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/stats"
)

// MetricFunc is the function signature for a metric function,
// where the output has the same shape as the inputs but with
// the outermost row dimension size of 1, and contains
// the metric value(s) for the "cells" in higher-dimensional tensors,
// and a single scalar value for a 1D input tensor.
// Critically, the metric is always computed over the outer row dimension,
// so each cell in a higher-dimensional output reflects the _row-wise_
// metric for that cell across the different rows.  To compute a metric
// on the [tensor.SubSpace] cells themselves, must call on a
// [tensor.New1DViewOf] the sub space.  See [simat] package.
// All metric functions skip over NaN's, as a missing value,
// and use the min of the length of the two tensors.
// Metric functions cannot be computed in parallel,
// e.g., using VectorizeThreaded or GPU, due to shared writing
// to the same output values.  Special implementations are required
// if that is needed.
type MetricFunc func(a, b, out *tensor.Indexed)

// SumSquaresScaleFuncOut64 computes the sum of squares differences between tensor values,
// returning scale and ss factors aggregated separately for better numerical stability, per BLAS.
func SumSquaresScaleFuncOut64(a, b, out *tensor.Indexed) (scale64, ss64 *tensor.Indexed) {
	scale64, ss64 = stats.Vectorize2Out64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		VecSSFunc(idx, tsr[0], tsr[1], tsr[2], tsr[3], 0, 1, func(a, b float64) float64 {
			return a - b
		})
	}, a, b, out)
	return
}

// SumSquaresFuncOut64 computes the sum of squares differences between tensor values,
// and returns the Float64 output values for use in subsequent computations.
func SumSquaresFuncOut64(a, b, out *tensor.Indexed) *tensor.Indexed {
	scale64, ss64 := SumSquaresScaleFuncOut64(a, b, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		scale := scale64.Tensor.Float1D(i)
		ss := ss64.Tensor.Float1D(i)
		v := 0.0
		if math.IsInf(scale, 1) {
			v = math.Inf(1)
		} else {
			v = scale * scale * ss
		}
		scale64.Tensor.SetFloat1D(v, i)
		out.Tensor.SetFloat1D(v, i)
	}
	return scale64
}

// SumSquaresFunc computes the sum of squares differences between tensor values,
// See [MetricFunc] for general information.
func SumSquaresFunc(a, b, out *tensor.Indexed) {
	SumSquaresFuncOut64(a, b, out)
}

// EuclideanFunc computes the Euclidean square root of the sum of squares
// differences between tensor values, aka the L2 Norm.
func EuclideanFunc(a, b, out *tensor.Indexed) {
	scale64, ss64 := SumSquaresScaleFuncOut64(a, b, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		scale := scale64.Tensor.Float1D(i)
		ss := ss64.Tensor.Float1D(i)
		v := 0.0
		if math.IsInf(scale, 1) {
			v = math.Inf(1)
		} else {
			v = scale * math.Sqrt(ss)
		}
		scale64.Tensor.SetFloat1D(v, i)
		out.Tensor.SetFloat1D(v, i)
	}
}

// AbsFunc computes the sum of the absolute value of differences between the
// tensor values, aka the L1 Norm.
// See [MetricFunc] for general information.
func AbsFunc(a, b, out *tensor.Indexed) {
	stats.VectorizeOut64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		VecFunc(idx, tsr[0], tsr[1], tsr[2], 0, func(a, b, agg float64) float64 {
			return agg + math.Abs(a-b)
		})
	}, a, b, out)
}

// HammingFunc computes the sum of 1s for every element that is different,
// i.e., "city block" distance.
// See [MetricFunc] for general information.
func HammingFunc(a, b, out *tensor.Indexed) {
	stats.VectorizeOut64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		VecFunc(idx, tsr[0], tsr[1], tsr[2], 0, func(a, b, agg float64) float64 {
			if a != b {
				agg += 1
			}
			return agg
		})
	}, a, b, out)
}

// SumSquaresBinTolScaleFuncOut64 computes the sum of squares differences between tensor values,
// with binary tolerance: differences < 0.5 are thresholded to 0.
// returning scale and ss factors aggregated separately for better numerical stability, per BLAS.
func SumSquaresBinTolScaleFuncOut64(a, b, out *tensor.Indexed) (scale64, ss64 *tensor.Indexed) {
	scale64, ss64 = stats.Vectorize2Out64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		VecSSFunc(idx, tsr[0], tsr[1], tsr[2], tsr[3], 0, 1, func(a, b float64) float64 {
			d := a - b
			if math.Abs(d) < 0.5 {
				d = 0
			}
			return d
		})
	}, a, b, out)
	return
}

// EuclideanBinTolFunc computes the Euclidean square root of the sum of squares
// differences between tensor values, with binary tolerance:
// differences < 0.5 are thresholded to 0.
func EuclideanBinTolFunc(a, b, out *tensor.Indexed) {
	scale64, ss64 := SumSquaresBinTolScaleFuncOut64(a, b, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		scale := scale64.Tensor.Float1D(i)
		ss := ss64.Tensor.Float1D(i)
		v := 0.0
		if math.IsInf(scale, 1) {
			v = math.Inf(1)
		} else {
			v = scale * math.Sqrt(ss)
		}
		scale64.Tensor.SetFloat1D(v, i)
		out.Tensor.SetFloat1D(v, i)
	}
}

// SumSquaresBinTolFunc computes the sum of squares differences between tensor values,
// with binary tolerance: differences < 0.5 are thresholded to 0.
func SumSquaresBinTolFunc(a, b, out *tensor.Indexed) {
	scale64, ss64 := SumSquaresBinTolScaleFuncOut64(a, b, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		scale := scale64.Tensor.Float1D(i)
		ss := ss64.Tensor.Float1D(i)
		v := 0.0
		if math.IsInf(scale, 1) {
			v = math.Inf(1)
		} else {
			v = scale * scale * ss
		}
		scale64.Tensor.SetFloat1D(v, i)
		out.Tensor.SetFloat1D(v, i)
	}
}

// CrossEntropyFunc is a standard measure of the difference between two
// probabilty distributions, reflecting the additional entropy (uncertainty) associated
// with measuring probabilities under distribution b when in fact they come from
// distribution a.  It is also the entropy of a plus the divergence between a from b,
// using Kullback-Leibler (KL) divergence.  It is computed as:
// a * log(a/b) + (1-a) * log(1-a/1-b).
// See [MetricFunc] for general information.
func CrossEntropyFunc(a, b, out *tensor.Indexed) {
	stats.VectorizeOut64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		VecFunc(idx, tsr[0], tsr[1], tsr[2], 0, func(a, b, agg float64) float64 {
			b = math32.Clamp64(b, 0.000001, 0.999999)
			if a >= 1.0 {
				agg += -math.Log(b)
			} else if a <= 0.0 {
				agg += -math.Log(1.0 - b)
			} else {
				agg += a*math.Log(a/b) + (1-a)*math.Log((1-a)/(1-b))
			}
			return agg
		})
	}, a, b, out)
}

// InnerProductFunc computes the sum of the co-products of the two on-NaN tensor values.
// See [MetricFunc] for general information.
func InnerProductFunc(a, b, out *tensor.Indexed) {
	stats.VectorizeOut64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		VecFunc(idx, tsr[0], tsr[1], tsr[2], 0, func(a, b, agg float64) float64 {
			return agg + a*b
		})
	}, a, b, out)
}

// CovarianceFunc computes the co-variance between two vectors,
// i.e., the mean of the co-product of each vector element minus
// the mean of that vector: cov(A,B) = E[(A - E(A))(B - E(B))].
func CovarianceFunc(a, b, out *tensor.Indexed) {
	amean, acount := stats.MeanFuncOut64(a, out)
	bmean, _ := stats.MeanFuncOut64(b, out)
	cov64 := stats.VectorizeOut64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		Vec2inFunc(idx, tsr[0], tsr[1], tsr[2], tsr[3], tsr[4], 0, func(a, b, am, bm, agg float64) float64 {
			return agg + (a-am)*(b-bm)
		})
	}, a, b, amean, bmean, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		c := acount.Tensor.Float1D(i)
		if c == 0 {
			continue
		}
		cov64.Tensor.SetFloat1D(cov64.Tensor.Float1D(i)/c, i)
		out.Tensor.SetFloat1D(cov64.Tensor.Float1D(i), i)
	}
}

// CorrelationFuncOut64 computes the correlation between two vectors,
// in range (-1..1) as the mean of the co-product of each vector
// element minus the mean of that vector, normalized by the product of their
// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
// (i.e., the standardized covariance).
// Equivalent to the cosine of mean-normalized vectors.
// Returns the Float64 output values for subsequent use.
func CorrelationFuncOut64(a, b, out *tensor.Indexed) *tensor.Indexed {
	amean, _ := stats.MeanFuncOut64(a, out)
	bmean, _ := stats.MeanFuncOut64(b, out)
	ss64, avar64, bvar64 := Vectorize3Out64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		Vec2in3outFunc(idx, tsr[0], tsr[1], tsr[2], tsr[3], tsr[4], tsr[5], tsr[6], 0, func(a, b, am, bm, ss, avar, bvar float64) (float64, float64, float64) {
			ad := a - am
			bd := b - bm
			ss += ad * bd   // between
			avar += ad * ad // within
			bvar += bd * bd
			return ss, avar, bvar
		})
	}, a, b, amean, bmean, out)

	nsub := out.Tensor.Len()
	for i := range nsub {
		ss := ss64.Tensor.Float1D(i)
		vp := math.Sqrt(avar64.Tensor.Float1D(i) * bvar64.Tensor.Float1D(i))
		if vp > 0 {
			ss /= vp
		}
		ss64.Tensor.SetFloat1D(ss, i)
		out.Tensor.SetFloat1D(ss, i)
	}
	return ss64
}

// CorrelationFunc computes the correlation between two vectors,
// in range (-1..1) as the mean of the co-product of each vector
// element minus the mean of that vector, normalized by the product of their
// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
// (i.e., the standardized [CovarianceFunc]).
// Equivalent to the [CosineFunc] of mean-normalized vectors.
func CorrelationFunc(a, b, out *tensor.Indexed) {
	CorrelationFuncOut64(a, b, out)
}

// InvCorrelationFunc computes 1 minus the correlation between two vectors,
// in range (-1..1) as the mean of the co-product of each vector
// element minus the mean of that vector, normalized by the product of their
// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
// (i.e., the standardized covariance).
// Equivalent to the [CosineFunc] of mean-normalized vectors.
// This is useful for a difference measure instead of similarity,
// where more different vectors have larger metric values.
func InvCorrelationFunc(a, b, out *tensor.Indexed) {
	cor64 := CorrelationFuncOut64(a, b, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		cor := cor64.Tensor.Float1D(i)
		out.Tensor.SetFloat1D(1-cor, i)
	}
}

// CosineFuncOut64 computes the high-dimensional angle between two vectors,
// in range (-1..1) as the normalized [InnerProductFunc]:
// inner product / sqrt(ssA * ssB).  See also [CorrelationFunc].
func CosineFuncOut64(a, b, out *tensor.Indexed) *tensor.Indexed {
	ss64, avar64, bvar64 := Vectorize3Out64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		Vec3outFunc(idx, tsr[0], tsr[1], tsr[2], tsr[3], tsr[4], 0, func(a, b, ss, avar, bvar float64) (float64, float64, float64) {
			ss += a * b
			avar += a * a
			bvar += b * b
			return ss, avar, bvar
		})
	}, a, b, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		ss := ss64.Tensor.Float1D(i)
		vp := math.Sqrt(avar64.Tensor.Float1D(i) * bvar64.Tensor.Float1D(i))
		if vp > 0 {
			ss /= vp
		}
		ss64.Tensor.SetFloat1D(ss, i)
		out.Tensor.SetFloat1D(ss, i)
	}
	return ss64
}

// CosineFunc computes the high-dimensional angle between two vectors,
// in range (-1..1) as the normalized inner product:
// inner product / sqrt(ssA * ssB).  See also [CorrelationFunc]
func CosineFunc(a, b, out *tensor.Indexed) {
	CosineFuncOut64(a, b, out)
}

// InvCosineFunc computes 1 minus the cosine between two vectors,
// in range (-1..1) as the normalized inner product:
// inner product / sqrt(ssA * ssB).
// This is useful for a difference measure instead of similarity,
// where more different vectors have larger metric values.
func InvCosineFunc(a, b, out *tensor.Indexed) {
	cos64 := CosineFuncOut64(a, b, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		cos := cos64.Tensor.Float1D(i)
		out.Tensor.SetFloat1D(1-cos, i)
	}
}
