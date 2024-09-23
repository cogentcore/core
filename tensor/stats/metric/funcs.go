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
// which is computed over the outermost row dimension and the
// output is the shape of the remaining inner cells (a scalar for 1D inputs).
// Use [tensor.As1D], [tensor.NewRowCellsView], [tensor.Cells1D] etc
// to reshape and reslice the data as needed.
// All metric functions skip over NaN's, as a missing value,
// and use the min of the length of the two tensors.
// Metric functions cannot be computed in parallel,
// e.g., using VectorizeThreaded or GPU, due to shared writing
// to the same output values.  Special implementations are required
// if that is needed.
type MetricFunc = func(a, b, out tensor.Tensor) error

// SumSquaresScaleOut64 computes the sum of squares differences between tensor values,
// returning scale and ss factors aggregated separately for better numerical stability, per BLAS.
func SumSquaresScaleOut64(a, b, out tensor.Tensor) (scale64, ss64 tensor.Tensor, err error) {
	if err = tensor.MustBeSameShape(a, b); err != nil {
		return
	}
	scale64, ss64, err = stats.Vectorize2Out64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecSSFunc(idx, tsr[0], tsr[1], tsr[2], tsr[3], 0, 1, func(a, b float64) float64 {
			return a - b
		})
	}, a, b, out)
	return
}

// SumSquaresOut64 computes the sum of squares differences between tensor values,
// and returns the Float64 output values for use in subsequent computations.
func SumSquaresOut64(a, b, out tensor.Tensor) (tensor.Tensor, error) {
	scale64, ss64, err := SumSquaresScaleOut64(a, b, out)
	if err != nil {
		return nil, err
	}
	nsub := out.Len()
	for i := range nsub {
		scale := scale64.Float1D(i)
		ss := ss64.Float1D(i)
		v := 0.0
		if math.IsInf(scale, 1) {
			v = math.Inf(1)
		} else {
			v = scale * scale * ss
		}
		scale64.SetFloat1D(v, i)
		out.SetFloat1D(v, i)
	}
	return scale64, err
}

// SumSquares computes the sum of squares differences between tensor values,
// See [MetricFunc] for general information.
func SumSquares(a, b, out tensor.Tensor) error {
	_, err := SumSquaresOut64(a, b, out)
	return err
}

// Euclidean computes the Euclidean square root of the sum of squares
// differences between tensor values, aka the L2 Norm.
func Euclidean(a, b, out tensor.Tensor) error {
	scale64, ss64, err := SumSquaresScaleOut64(a, b, out)
	if err != nil {
		return err
	}
	nsub := out.Len()
	for i := range nsub {
		scale := scale64.Float1D(i)
		ss := ss64.Float1D(i)
		v := 0.0
		if math.IsInf(scale, 1) {
			v = math.Inf(1)
		} else {
			v = scale * math.Sqrt(ss)
		}
		scale64.SetFloat1D(v, i)
		out.SetFloat1D(v, i)
	}
	return nil
}

// Abs computes the sum of the absolute value of differences between the
// tensor values, aka the L1 Norm.
// See [MetricFunc] for general information.
func Abs(a, b, out tensor.Tensor) error {
	if err := tensor.MustBeSameShape(a, b); err != nil {
		return err
	}
	_, err := stats.VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], tsr[2], 0, func(a, b, agg float64) float64 {
			return agg + math.Abs(a-b)
		})
	}, a, b, out)
	return err
}

// Hamming computes the sum of 1s for every element that is different,
// i.e., "city block" distance.
// See [MetricFunc] for general information.
func Hamming(a, b, out tensor.Tensor) error {
	if err := tensor.MustBeSameShape(a, b); err != nil {
		return err
	}
	_, err := stats.VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], tsr[2], 0, func(a, b, agg float64) float64 {
			if a != b {
				agg += 1
			}
			return agg
		})
	}, a, b, out)
	return err
}

// SumSquaresBinTolScaleOut64 computes the sum of squares differences between tensor values,
// with binary tolerance: differences < 0.5 are thresholded to 0.
// returning scale and ss factors aggregated separately for better numerical stability, per BLAS.
func SumSquaresBinTolScaleOut64(a, b, out tensor.Tensor) (scale64, ss64 tensor.Tensor, err error) {
	if err = tensor.MustBeSameShape(a, b); err != nil {
		return
	}
	scale64, ss64, err = stats.Vectorize2Out64(NFunc, func(idx int, tsr ...tensor.Tensor) {
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

// EuclideanBinTol computes the Euclidean square root of the sum of squares
// differences between tensor values, with binary tolerance:
// differences < 0.5 are thresholded to 0.
func EuclideanBinTol(a, b, out tensor.Tensor) error {
	scale64, ss64, err := SumSquaresBinTolScaleOut64(a, b, out)
	if err != nil {
		return err
	}
	nsub := out.Len()
	for i := range nsub {
		scale := scale64.Float1D(i)
		ss := ss64.Float1D(i)
		v := 0.0
		if math.IsInf(scale, 1) {
			v = math.Inf(1)
		} else {
			v = scale * math.Sqrt(ss)
		}
		scale64.SetFloat1D(v, i)
		out.SetFloat1D(v, i)
	}
	return nil
}

// SumSquaresBinTol computes the sum of squares differences between tensor values,
// with binary tolerance: differences < 0.5 are thresholded to 0.
func SumSquaresBinTol(a, b, out tensor.Tensor) error {
	scale64, ss64, err := SumSquaresBinTolScaleOut64(a, b, out)
	if err != nil {
		return err
	}
	nsub := out.Len()
	for i := range nsub {
		scale := scale64.Float1D(i)
		ss := ss64.Float1D(i)
		v := 0.0
		if math.IsInf(scale, 1) {
			v = math.Inf(1)
		} else {
			v = scale * scale * ss
		}
		scale64.SetFloat1D(v, i)
		out.SetFloat1D(v, i)
	}
	return nil
}

// CrossEntropy is a standard measure of the difference between two
// probabilty distributions, reflecting the additional entropy (uncertainty) associated
// with measuring probabilities under distribution b when in fact they come from
// distribution a.  It is also the entropy of a plus the divergence between a from b,
// using Kullback-Leibler (KL) divergence.  It is computed as:
// a * log(a/b) + (1-a) * log(1-a/1-b).
// See [MetricFunc] for general information.
func CrossEntropy(a, b, out tensor.Tensor) error {
	if err := tensor.MustBeSameShape(a, b); err != nil {
		return err
	}
	_, err := stats.VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], tsr[2], 0, func(a, b, agg float64) float64 {
			b = math32.Clamp(b, 0.000001, 0.999999)
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
	return err
}

// InnerProduct computes the sum of the co-products of the two on-NaN tensor values.
// See [MetricFunc] for general information.
func InnerProduct(a, b, out tensor.Tensor) error {
	if err := tensor.MustBeSameShape(a, b); err != nil {
		return err
	}
	_, err := stats.VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], tsr[2], 0, func(a, b, agg float64) float64 {
			return agg + a*b
		})
	}, a, b, out)
	return err
}

// Covariance computes the co-variance between two vectors,
// i.e., the mean of the co-product of each vector element minus
// the mean of that vector: cov(A,B) = E[(A - E(A))(B - E(B))].
func Covariance(a, b, out tensor.Tensor) error {
	if err := tensor.MustBeSameShape(a, b); err != nil {
		return err
	}
	amean, acount, err := stats.MeanOut64(a, out)
	if err != nil {
		return err
	}
	bmean, _, _ := stats.MeanOut64(b, out)
	cov64, _ := stats.VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		Vec2inFunc(idx, tsr[0], tsr[1], tsr[2], tsr[3], tsr[4], 0, func(a, b, am, bm, agg float64) float64 {
			return agg + (a-am)*(b-bm)
		})
	}, a, b, amean, bmean, out)
	nsub := out.Len()
	for i := range nsub {
		c := acount.Float1D(i)
		if c == 0 {
			continue
		}
		cov64.SetFloat1D(cov64.Float1D(i)/c, i)
		out.SetFloat1D(cov64.Float1D(i), i)
	}
	return nil
}

// CorrelationOut64 computes the correlation between two vectors,
// in range (-1..1) as the mean of the co-product of each vector
// element minus the mean of that vector, normalized by the product of their
// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
// (i.e., the standardized covariance).
// Equivalent to the cosine of mean-normalized vectors.
// Returns the Float64 output values for subsequent use.
func CorrelationOut64(a, b, out tensor.Tensor) (tensor.Tensor, error) {
	if err := tensor.MustBeSameShape(a, b); err != nil {
		return nil, err
	}
	amean, _, err := stats.MeanOut64(a, out)
	if err != nil {
		return nil, err
	}
	bmean, _, _ := stats.MeanOut64(b, out)
	ss64, avar64, bvar64, err := Vectorize3Out64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		Vec2in3outFunc(idx, tsr[0], tsr[1], tsr[2], tsr[3], tsr[4], tsr[5], tsr[6], 0, func(a, b, am, bm, ss, avar, bvar float64) (float64, float64, float64) {
			ad := a - am
			bd := b - bm
			ss += ad * bd   // between
			avar += ad * ad // within
			bvar += bd * bd
			return ss, avar, bvar
		})
	}, a, b, amean, bmean, out)
	if err != nil {
		return nil, err
	}

	nsub := out.Len()
	for i := range nsub {
		ss := ss64.Float1D(i)
		vp := math.Sqrt(avar64.Float1D(i) * bvar64.Float1D(i))
		if vp > 0 {
			ss /= vp
		}
		ss64.SetFloat1D(ss, i)
		out.SetFloat1D(ss, i)
	}
	return ss64, nil
}

// Correlation computes the correlation between two vectors,
// in range (-1..1) as the mean of the co-product of each vector
// element minus the mean of that vector, normalized by the product of their
// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
// (i.e., the standardized [CovarianceFunc]).
// Equivalent to the [CosineFunc] of mean-normalized vectors.
func Correlation(a, b, out tensor.Tensor) error {
	_, err := CorrelationOut64(a, b, out)
	return err
}

// InvCorrelation computes 1 minus the correlation between two vectors,
// in range (-1..1) as the mean of the co-product of each vector
// element minus the mean of that vector, normalized by the product of their
// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
// (i.e., the standardized covariance).
// Equivalent to the [CosineFunc] of mean-normalized vectors.
// This is useful for a difference measure instead of similarity,
// where more different vectors have larger metric values.
func InvCorrelation(a, b, out tensor.Tensor) error {
	cor64, err := CorrelationOut64(a, b, out)
	if err != nil {
		return err
	}
	nsub := out.Len()
	for i := range nsub {
		cor := cor64.Float1D(i)
		out.SetFloat1D(1-cor, i)
	}
	return nil
}

// CosineOut64 computes the high-dimensional angle between two vectors,
// in range (-1..1) as the normalized [InnerProductFunc]:
// inner product / sqrt(ssA * ssB).  See also [CorrelationFunc].
func CosineOut64(a, b, out tensor.Tensor) (tensor.Tensor, error) {
	ss64, avar64, bvar64, err := Vectorize3Out64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		Vec3outFunc(idx, tsr[0], tsr[1], tsr[2], tsr[3], tsr[4], 0, func(a, b, ss, avar, bvar float64) (float64, float64, float64) {
			ss += a * b
			avar += a * a
			bvar += b * b
			return ss, avar, bvar
		})
	}, a, b, out)
	if err != nil {
		return nil, err
	}
	nsub := out.Len()
	for i := range nsub {
		ss := ss64.Float1D(i)
		vp := math.Sqrt(avar64.Float1D(i) * bvar64.Float1D(i))
		if vp > 0 {
			ss /= vp
		}
		ss64.SetFloat1D(ss, i)
		out.SetFloat1D(ss, i)
	}
	return ss64, nil
}

// Cosine computes the high-dimensional angle between two vectors,
// in range (-1..1) as the normalized inner product:
// inner product / sqrt(ssA * ssB).  See also [CorrelationFunc]
func Cosine(a, b, out tensor.Tensor) error {
	_, err := CosineOut64(a, b, out)
	return err
}

// InvCosine computes 1 minus the cosine between two vectors,
// in range (-1..1) as the normalized inner product:
// inner product / sqrt(ssA * ssB).
// This is useful for a difference measure instead of similarity,
// where more different vectors have larger metric values.
func InvCosine(a, b, out tensor.Tensor) error {
	cos64, err := CosineOut64(a, b, out)
	if err != nil {
		return err
	}
	nsub := out.Len()
	for i := range nsub {
		cos := cos64.Float1D(i)
		out.SetFloat1D(1-cos, i)
	}
	return nil
}
