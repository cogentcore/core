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
// the outer-most row dimension size of 1, and contains
// the metric value(s) for the "cells" in higher-dimensional tensors,
// and a single scalar value for a 1D input tensor.
// All metric functions skip over NaN's, as a missing value.
// Metric functions cannot be computed in parallel,
// e.g., using VectorizeThreaded or GPU, due to shared writing
// to the same output values.  Special implementations are required
// if that is needed.
type MetricFunc func(a, b, out *tensor.Indexed)

// SumSquaresScaleFuncOut64 computes the sum of squares differences between tensor values,
// returning scale and ss factors aggregated separately for better numerical stability, per BLAS.
func SumSquaresScaleFuncOut64(a, b, out *tensor.Indexed) (scale64, ss64 *tensor.Indexed) {
	scale64, ss64 = stats.Vectorize2Out64(func(idx int, tsr ...*tensor.Indexed) {
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
		scale64.Tensor.SetFloat1D(i, v)
		out.Tensor.SetFloat1D(i, v)
	}
	return scale64
}

// SumSquaresFunc computes the sum of squares differences between tensor values,
// See [MetricFunc] for general information.
func SumSquaresFunc(a, b, out *tensor.Indexed) {
	SumSquaresFuncOut64(a, b, out)
}

// EuclideanFunc computes the Euclidean square root of the sum of squares
// differences between tensor values.
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
		scale64.Tensor.SetFloat1D(i, v)
		out.Tensor.SetFloat1D(i, v)
	}
}

// SumSquaresBinTolScaleFuncOut64 computes the sum of squares differences between tensor values,
// with binary tolerance: differences < 0.5 are thresholded to 0.
// returning scale and ss factors aggregated separately for better numerical stability, per BLAS.
func SumSquaresBinTolScaleFuncOut64(a, b, out *tensor.Indexed) (scale64, ss64 *tensor.Indexed) {
	scale64, ss64 = stats.Vectorize2Out64(func(idx int, tsr ...*tensor.Indexed) {
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
		scale64.Tensor.SetFloat1D(i, v)
		out.Tensor.SetFloat1D(i, v)
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
		scale64.Tensor.SetFloat1D(i, v)
		out.Tensor.SetFloat1D(i, v)
	}
}

// CrossEntropyFunc computes the sum of the co-products of the two on-NaN tensor values.
// See [MetricFunc] for general information.
func CrossEntropyFunc(a, b, out *tensor.Indexed) {
	stats.VectorizeOut64(func(idx int, tsr ...*tensor.Indexed) {
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
	stats.VectorizeOut64(func(idx int, tsr ...*tensor.Indexed) {
		VecFunc(idx, tsr[0], tsr[1], tsr[2], 0, func(a, b, agg float64) float64 {
			return agg + a*b
		})
	}, a, b, out)
}

// CovarianceFunc computes the covariance between two vectors,
// i.e., the mean of the co-product of each vector element minus
// the mean of that vector: cov(A,B) = E[(A - E(A))(B - E(B))].
func CovarianceFunc(a, b, out *tensor.Indexed) {
	amean, acount := stats.MeanFuncOut64(a, out)
	bmean, _ := stats.MeanFuncOut64(b, out)
	cov64 := stats.VectorizeOut64(func(idx int, tsr ...*tensor.Indexed) {
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
		cov64.Tensor.SetFloat1D(i, cov64.Tensor.Float1D(i)/c)
		out.Tensor.SetFloat1D(i, cov64.Tensor.Float1D(i))
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
	ss64, avar64, bvar64 := Vectorize3Out64(func(idx int, tsr ...*tensor.Indexed) {
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
		ss64.Tensor.SetFloat1D(i, ss)
		out.Tensor.SetFloat1D(i, ss)
	}
	return ss64
}

// CorrelationFunc computes the correlation between two vectors,
// in range (-1..1) as the mean of the co-product of each vector
// element minus the mean of that vector, normalized by the product of their
// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
// (i.e., the standardized covariance).
// Equivalent to the cosine of mean-normalized vectors.
func CorrelationFunc(a, b, out *tensor.Indexed) {
	CorrelationFuncOut64(a, b, out)
}

// InvCorrelationFunc computes 1 minus the correlation between two vectors,
// in range (-1..1) as the mean of the co-product of each vector
// element minus the mean of that vector, normalized by the product of their
// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
// (i.e., the standardized covariance).
// Equivalent to the cosine of mean-normalized vectors.
// This is useful for a difference measure instead of similarity,
// where more different vectors have larger metric values.
func InvCorrelationFunc(a, b, out *tensor.Indexed) {
	cor64 := CorrelationFuncOut64(a, b, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		cor := cor64.Tensor.Float1D(i)
		out.Tensor.SetFloat1D(i, 1-cor)
	}
}

// CosineFuncOut64 computes the correlation between two vectors,
// in range (-1..1) as the normalized inner product:
// inner product / sqrt(ssA * ssB).
func CosineFuncOut64(a, b, out *tensor.Indexed) *tensor.Indexed {
	ss64, avar64, bvar64 := Vectorize3Out64(func(idx int, tsr ...*tensor.Indexed) {
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
		ss64.Tensor.SetFloat1D(i, ss)
		out.Tensor.SetFloat1D(i, ss)
	}
	return ss64
}

// CosineFunc computes the correlation between two vectors,
// in range (-1..1) as the normalized inner product:
// inner product / sqrt(ssA * ssB).
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
		out.Tensor.SetFloat1D(i, 1-cos)
	}
}
