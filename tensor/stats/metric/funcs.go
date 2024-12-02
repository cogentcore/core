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
type MetricFunc = func(a, b tensor.Tensor) tensor.Values

// MetricOutFunc is the function signature for a metric function,
// that takes output values as the final argument. See [MetricFunc].
// This version is for computationally demanding cases and saves
// reallocation of output.
type MetricOutFunc = func(a, b tensor.Tensor, out tensor.Values) error

// SumSquaresScaleOut64 computes the sum of squares differences between tensor values,
// returning scale and ss factors aggregated separately for better numerical stability, per BLAS.
func SumSquaresScaleOut64(a, b tensor.Tensor) (scale64, ss64 *tensor.Float64, err error) {
	if err = tensor.MustBeSameShape(a, b); err != nil {
		return
	}
	scale64, ss64 = Vectorize2Out64(a, b, 0, 1, func(a, b, scale, ss float64) (float64, float64) {
		if math.IsNaN(a) || math.IsNaN(b) {
			return scale, ss
		}
		d := a - b
		if d == 0 {
			return scale, ss
		}
		absxi := math.Abs(d)
		if scale < absxi {
			ss = 1 + ss*(scale/absxi)*(scale/absxi)
			scale = absxi
		} else {
			ss = ss + (absxi/scale)*(absxi/scale)
		}
		return scale, ss
	})
	return
}

// SumSquaresOut64 computes the sum of squares differences between tensor values,
// and returns the Float64 output values for use in subsequent computations.
func SumSquaresOut64(a, b tensor.Tensor, out tensor.Values) (*tensor.Float64, error) {
	scale64, ss64, err := SumSquaresScaleOut64(a, b)
	if err != nil {
		return nil, err
	}
	osz := tensor.CellsSize(a.ShapeSizes())
	out.SetShapeSizes(osz...)
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

// SumSquaresOut computes the sum of squares differences between tensor values,
// See [MetricOutFunc] for general information.
func SumSquaresOut(a, b tensor.Tensor, out tensor.Values) error {
	_, err := SumSquaresOut64(a, b, out)
	return err
}

// SumSquares computes the sum of squares differences between tensor values,
// See [MetricFunc] for general information.
func SumSquares(a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2(SumSquaresOut, a, b)
}

// L2NormOut computes the L2 Norm: square root of the sum of squares
// differences between tensor values, aka the Euclidean distance.
func L2NormOut(a, b tensor.Tensor, out tensor.Values) error {
	scale64, ss64, err := SumSquaresScaleOut64(a, b)
	if err != nil {
		return err
	}
	osz := tensor.CellsSize(a.ShapeSizes())
	out.SetShapeSizes(osz...)
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

// L2Norm computes the L2 Norm: square root of the sum of squares
// differences between tensor values, aka the Euclidean distance.
// See [MetricFunc] for general information.
func L2Norm(a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2(L2NormOut, a, b)
}

// L1NormOut computes the sum of the absolute value of differences between the
// tensor values, the L1 Norm.
// See [MetricOutFunc] for general information.
func L1NormOut(a, b tensor.Tensor, out tensor.Values) error {
	if err := tensor.MustBeSameShape(a, b); err != nil {
		return err
	}
	VectorizeOut64(a, b, out, 0, func(a, b, agg float64) float64 {
		if math.IsNaN(a) || math.IsNaN(b) {
			return agg
		}
		return agg + math.Abs(a-b)
	})
	return nil
}

// L1Norm computes the sum of the absolute value of differences between the
// tensor values, the L1 Norm.
// See [MetricFunc] for general information.
func L1Norm(a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2(L1NormOut, a, b)
}

// HammingOut computes the sum of 1s for every element that is different,
// i.e., "city block" distance.
// See [MetricOutFunc] for general information.
func HammingOut(a, b tensor.Tensor, out tensor.Values) error {
	if err := tensor.MustBeSameShape(a, b); err != nil {
		return err
	}
	VectorizeOut64(a, b, out, 0, func(a, b, agg float64) float64 {
		if math.IsNaN(a) || math.IsNaN(b) {
			return agg
		}
		if a != b {
			agg += 1
		}
		return agg
	})
	return nil
}

// Hamming computes the sum of 1s for every element that is different,
// i.e., "city block" distance.
// See [MetricFunc] for general information.
func Hamming(a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2(HammingOut, a, b)
}

// SumSquaresBinTolScaleOut64 computes the sum of squares differences between tensor values,
// with binary tolerance: differences < 0.5 are thresholded to 0.
// returning scale and ss factors aggregated separately for better numerical stability, per BLAS.
func SumSquaresBinTolScaleOut64(a, b tensor.Tensor) (scale64, ss64 *tensor.Float64, err error) {
	if err = tensor.MustBeSameShape(a, b); err != nil {
		return
	}
	scale64, ss64 = Vectorize2Out64(a, b, 0, 1, func(a, b, scale, ss float64) (float64, float64) {
		if math.IsNaN(a) || math.IsNaN(b) {
			return scale, ss
		}
		d := a - b
		if math.Abs(d) < 0.5 {
			return scale, ss
		}
		absxi := math.Abs(d)
		if scale < absxi {
			ss = 1 + ss*(scale/absxi)*(scale/absxi)
			scale = absxi
		} else {
			ss = ss + (absxi/scale)*(absxi/scale)
		}
		return scale, ss
	})
	return
}

// L2NormBinTolOut computes the L2 Norm square root of the sum of squares
// differences between tensor values (aka Euclidean distance), with binary tolerance:
// differences < 0.5 are thresholded to 0.
func L2NormBinTolOut(a, b tensor.Tensor, out tensor.Values) error {
	scale64, ss64, err := SumSquaresBinTolScaleOut64(a, b)
	if err != nil {
		return err
	}
	osz := tensor.CellsSize(a.ShapeSizes())
	out.SetShapeSizes(osz...)
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

// L2NormBinTol computes the L2 Norm square root of the sum of squares
// differences between tensor values (aka Euclidean distance), with binary tolerance:
// differences < 0.5 are thresholded to 0.
// See [MetricFunc] for general information.
func L2NormBinTol(a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2(L2NormBinTolOut, a, b)
}

// SumSquaresBinTolOut computes the sum of squares differences between tensor values,
// with binary tolerance: differences < 0.5 are thresholded to 0.
func SumSquaresBinTolOut(a, b tensor.Tensor, out tensor.Values) error {
	scale64, ss64, err := SumSquaresBinTolScaleOut64(a, b)
	if err != nil {
		return err
	}
	osz := tensor.CellsSize(a.ShapeSizes())
	out.SetShapeSizes(osz...)
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

// SumSquaresBinTol computes the sum of squares differences between tensor values,
// with binary tolerance: differences < 0.5 are thresholded to 0.
// See [MetricFunc] for general information.
func SumSquaresBinTol(a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2(SumSquaresBinTolOut, a, b)
}

// CrossEntropyOut is a standard measure of the difference between two
// probabilty distributions, reflecting the additional entropy (uncertainty) associated
// with measuring probabilities under distribution b when in fact they come from
// distribution a.  It is also the entropy of a plus the divergence between a from b,
// using Kullback-Leibler (KL) divergence.  It is computed as:
// a * log(a/b) + (1-a) * log(1-a/1-b).
// See [MetricOutFunc] for general information.
func CrossEntropyOut(a, b tensor.Tensor, out tensor.Values) error {
	if err := tensor.MustBeSameShape(a, b); err != nil {
		return err
	}
	VectorizeOut64(a, b, out, 0, func(a, b, agg float64) float64 {
		if math.IsNaN(a) || math.IsNaN(b) {
			return agg
		}
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
	return nil
}

// CrossEntropy is a standard measure of the difference between two
// probabilty distributions, reflecting the additional entropy (uncertainty) associated
// with measuring probabilities under distribution b when in fact they come from
// distribution a.  It is also the entropy of a plus the divergence between a from b,
// using Kullback-Leibler (KL) divergence.  It is computed as:
// a * log(a/b) + (1-a) * log(1-a/1-b).
// See [MetricFunc] for general information.
func CrossEntropy(a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2(CrossEntropyOut, a, b)
}

// DotProductOut computes the sum of the element-wise products of the
// two tensors (aka the inner product).
// See [MetricOutFunc] for general information.
func DotProductOut(a, b tensor.Tensor, out tensor.Values) error {
	if err := tensor.MustBeSameShape(a, b); err != nil {
		return err
	}
	VectorizeOut64(a, b, out, 0, func(a, b, agg float64) float64 {
		if math.IsNaN(a) || math.IsNaN(b) {
			return agg
		}
		return agg + a*b
	})
	return nil
}

// DotProductOut computes the sum of the element-wise products of the
// two tensors (aka the inner product).
// See [MetricFunc] for general information.
func DotProduct(a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2(DotProductOut, a, b)
}

// CovarianceOut computes the co-variance between two vectors,
// i.e., the mean of the co-product of each vector element minus
// the mean of that vector: cov(A,B) = E[(A - E(A))(B - E(B))].
func CovarianceOut(a, b tensor.Tensor, out tensor.Values) error {
	if err := tensor.MustBeSameShape(a, b); err != nil {
		return err
	}
	amean, acount := stats.MeanOut64(a, out)
	bmean, _ := stats.MeanOut64(b, out)
	cov64 := VectorizePreOut64(a, b, out, 0, amean, bmean, func(a, b, am, bm, agg float64) float64 {
		if math.IsNaN(a) || math.IsNaN(b) {
			return agg
		}
		return agg + (a-am)*(b-bm)
	})
	osz := tensor.CellsSize(a.ShapeSizes())
	out.SetShapeSizes(osz...)
	nsub := out.Len()
	for i := range nsub {
		c := acount.Float1D(i)
		if c == 0 {
			continue
		}
		cov := cov64.Float1D(i) / c
		out.SetFloat1D(cov, i)
	}
	return nil
}

// Covariance computes the co-variance between two vectors,
// i.e., the mean of the co-product of each vector element minus
// the mean of that vector: cov(A,B) = E[(A - E(A))(B - E(B))].
// See [MetricFunc] for general information.
func Covariance(a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2(CovarianceOut, a, b)
}

// CorrelationOut64 computes the correlation between two vectors,
// in range (-1..1) as the mean of the co-product of each vector
// element minus the mean of that vector, normalized by the product of their
// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
// (i.e., the standardized covariance).
// Equivalent to the cosine of mean-normalized vectors.
// Returns the Float64 output values for subsequent use.
func CorrelationOut64(a, b tensor.Tensor, out tensor.Values) (*tensor.Float64, error) {
	if err := tensor.MustBeSameShape(a, b); err != nil {
		return nil, err
	}
	amean, _ := stats.MeanOut64(a, out)
	bmean, _ := stats.MeanOut64(b, out)
	ss64, avar64, bvar64 := VectorizePre3Out64(a, b, 0, 0, 0, amean, bmean, func(a, b, am, bm, ss, avar, bvar float64) (float64, float64, float64) {
		if math.IsNaN(a) || math.IsNaN(b) {
			return ss, avar, bvar
		}
		ad := a - am
		bd := b - bm
		ss += ad * bd   // between
		avar += ad * ad // within
		bvar += bd * bd
		return ss, avar, bvar
	})
	osz := tensor.CellsSize(a.ShapeSizes())
	out.SetShapeSizes(osz...)
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

// CorrelationOut computes the correlation between two vectors,
// in range (-1..1) as the mean of the co-product of each vector
// element minus the mean of that vector, normalized by the product of their
// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
// (i.e., the standardized [Covariance]).
// Equivalent to the [Cosine] of mean-normalized vectors.
func CorrelationOut(a, b tensor.Tensor, out tensor.Values) error {
	_, err := CorrelationOut64(a, b, out)
	return err
}

// Correlation computes the correlation between two vectors,
// in range (-1..1) as the mean of the co-product of each vector
// element minus the mean of that vector, normalized by the product of their
// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
// (i.e., the standardized [Covariance]).
// Equivalent to the [Cosine] of mean-normalized vectors.
// See [MetricFunc] for general information.
func Correlation(a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2(CorrelationOut, a, b)
}

// InvCorrelationOut computes 1 minus the correlation between two vectors,
// in range (-1..1) as the mean of the co-product of each vector
// element minus the mean of that vector, normalized by the product of their
// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
// (i.e., the standardized covariance).
// Equivalent to the [Cosine] of mean-normalized vectors.
// This is useful for a difference measure instead of similarity,
// where more different vectors have larger metric values.
func InvCorrelationOut(a, b tensor.Tensor, out tensor.Values) error {
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

// InvCorrelation computes 1 minus the correlation between two vectors,
// in range (-1..1) as the mean of the co-product of each vector
// element minus the mean of that vector, normalized by the product of their
// standard deviations: cor(A,B) = E[(A - E(A))(B - E(B))] / sigma(A) sigma(B).
// (i.e., the standardized covariance).
// Equivalent to the [Cosine] of mean-normalized vectors.
// This is useful for a difference measure instead of similarity,
// where more different vectors have larger metric values.
// See [MetricFunc] for general information.
func InvCorrelation(a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2(InvCorrelationOut, a, b)
}

// CosineOut64 computes the high-dimensional angle between two vectors,
// in range (-1..1) as the normalized [Dot]:
// dot product / sqrt(ssA * ssB).  See also [Correlation].
func CosineOut64(a, b tensor.Tensor, out tensor.Values) (*tensor.Float64, error) {
	if err := tensor.MustBeSameShape(a, b); err != nil {
		return nil, err
	}
	ss64, avar64, bvar64 := Vectorize3Out64(a, b, 0, 0, 0, func(a, b, ss, avar, bvar float64) (float64, float64, float64) {
		if math.IsNaN(a) || math.IsNaN(b) {
			return ss, avar, bvar
		}
		ss += a * b
		avar += a * a
		bvar += b * b
		return ss, avar, bvar
	})
	osz := tensor.CellsSize(a.ShapeSizes())
	out.SetShapeSizes(osz...)
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

// CosineOut computes the high-dimensional angle between two vectors,
// in range (-1..1) as the normalized dot product:
// dot product / sqrt(ssA * ssB).  See also [Correlation]
func CosineOut(a, b tensor.Tensor, out tensor.Values) error {
	_, err := CosineOut64(a, b, out)
	return err
}

// Cosine computes the high-dimensional angle between two vectors,
// in range (-1..1) as the normalized dot product:
// dot product / sqrt(ssA * ssB).  See also [Correlation]
// See [MetricFunc] for general information.
func Cosine(a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2(CosineOut, a, b)
}

// InvCosineOut computes 1 minus the cosine between two vectors,
// in range (-1..1) as the normalized dot product:
// dot product / sqrt(ssA * ssB).
// This is useful for a difference measure instead of similarity,
// where more different vectors have larger metric values.
func InvCosineOut(a, b tensor.Tensor, out tensor.Values) error {
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

// InvCosine computes 1 minus the cosine between two vectors,
// in range (-1..1) as the normalized dot product:
// dot product / sqrt(ssA * ssB).
// This is useful for a difference measure instead of similarity,
// where more different vectors have larger metric values.
// See [MetricFunc] for general information.
func InvCosine(a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2(InvCosineOut, a, b)
}
