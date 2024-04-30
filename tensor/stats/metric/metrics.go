// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate core generate

package metric

// Func32 is a distance / similarity metric operating on slices of float32 numbers
type Func32 func(a, b []float32) float32

// Func64 is a distance / similarity metric operating on slices of float64 numbers
type Func64 func(a, b []float64) float64

// StdMetrics are standard metric functions
type StdMetrics int32 //enums:enum

const (
	Euclidean StdMetrics = iota
	SumSquares
	Abs
	Hamming

	EuclideanBinTol
	SumSquaresBinTol

	// InvCosine is 1-Cosine -- useful to convert into an Increasing metric
	InvCosine

	// InvCorrelation is 1-Correlation -- useful to convert into an Increasing metric
	InvCorrelation

	CrossEntropy

	// Everything below here is !Increasing -- larger = closer, not farther
	InnerProduct
	Covariance
	Correlation
	Cosine
)

// Increasing returns true if the distance metric is such that metric
// values increase as a function of distance (e.g., Euclidean)
// and false if metric values decrease as a function of distance
// (e.g., Cosine, Correlation)
func Increasing(std StdMetrics) bool {
	if std >= InnerProduct {
		return false
	}
	return true
}

// StdFunc32 returns a standard metric function as specified
func StdFunc32(std StdMetrics) Func32 {
	switch std {
	case Euclidean:
		return Euclidean32
	case SumSquares:
		return SumSquares32
	case Abs:
		return Abs32
	case Hamming:
		return Hamming32
	case EuclideanBinTol:
		return EuclideanBinTol32
	case SumSquaresBinTol:
		return SumSquaresBinTol32
	case InvCorrelation:
		return InvCorrelation32
	case InvCosine:
		return InvCosine32
	case CrossEntropy:
		return CrossEntropy32
	case InnerProduct:
		return InnerProduct32
	case Covariance:
		return Covariance32
	case Correlation:
		return Correlation32
	case Cosine:
		return Cosine32
	}
	return nil
}

// StdFunc64 returns a standard metric function as specified
func StdFunc64(std StdMetrics) Func64 {
	switch std {
	case Euclidean:
		return Euclidean64
	case SumSquares:
		return SumSquares64
	case Abs:
		return Abs64
	case Hamming:
		return Hamming64
	case EuclideanBinTol:
		return EuclideanBinTol64
	case SumSquaresBinTol:
		return SumSquaresBinTol64
	case InvCorrelation:
		return InvCorrelation64
	case InvCosine:
		return InvCosine64
	case CrossEntropy:
		return CrossEntropy64
	case InnerProduct:
		return InnerProduct64
	case Covariance:
		return Covariance64
	case Correlation:
		return Correlation64
	case Cosine:
		return Cosine64
	}
	return nil
}
