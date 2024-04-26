// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package norm

import (
	"math"

	"cogentcore.org/core/math32"
)

///////////////////////////////////////////
//  N

// N32 computes the number of non-NaN vector values.
// Skips NaN's
func N32(a []float32) float32 {
	n := 0
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		n++
	}
	return float32(n)
}

// N64 computes the number of non-NaN vector values.
// Skips NaN's
func N64(a []float64) float64 {
	n := 0
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		n++
	}
	return float64(n)
}

///////////////////////////////////////////
//  Sum

// Sum32 computes the sum of vector values.
// Skips NaN's
func Sum32(a []float32) float32 {
	s := float32(0)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		s += av
	}
	return s
}

// Sum64 computes the sum of vector values.
// Skips NaN's
func Sum64(a []float64) float64 {
	s := float64(0)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		s += av
	}
	return s
}

///////////////////////////////////////////
//  Mean

// Mean32 computes the mean of the vector (sum / N).
// Skips NaN's
func Mean32(a []float32) float32 {
	s := float32(0)
	n := 0
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		s += av
		n++
	}
	if n > 0 {
		s /= float32(n)
	}
	return s
}

// Mean64 computes the mean of the vector (sum / N).
// Skips NaN's
func Mean64(a []float64) float64 {
	s := float64(0)
	n := 0
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		s += av
		n++
	}
	if n > 0 {
		s /= float64(n)
	}
	return s
}

///////////////////////////////////////////
//  Var

// Var32 returns the sample variance of non-NaN elements.
func Var32(a []float32) float32 {
	mean := Mean32(a)
	n := 0
	s := float32(0)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		dv := av - mean
		s += dv * dv
		n++
	}
	if n > 1 {
		s /= float32(n - 1)
	}
	return s
}

// Var64 returns the sample variance of non-NaN elements.
func Var64(a []float64) float64 {
	mean := Mean64(a)
	n := 0
	s := float64(0)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		dv := av - mean
		s += dv * dv
		n++
	}
	if n > 1 {
		s /= float64(n - 1)
	}
	return s
}

///////////////////////////////////////////
//  Std

// Std32 returns the sample standard deviation of non-NaN elements in vector.
func Std32(a []float32) float32 {
	return math32.Sqrt(Var32(a))
}

// Std64 returns the sample standard deviation of non-NaN elements in vector.
func Std64(a []float64) float64 {
	return math.Sqrt(Var64(a))
}

///////////////////////////////////////////
//  Max

// Max32 computes the max over vector values.
// Skips NaN's
func Max32(a []float32) float32 {
	m := float32(-math.MaxFloat32)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		m = math32.Max(m, av)
	}
	return m
}

// MaxIndex32 computes the max over vector values, and returns index of max as well
// Skips NaN's
func MaxIndex32(a []float32) (float32, int) {
	m := float32(-math.MaxFloat32)
	mi := -1
	for i, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		if av > m {
			m = av
			mi = i
		}
	}
	return m, mi
}

// Max64 computes the max over vector values.
// Skips NaN's
func Max64(a []float64) float64 {
	m := float64(-math.MaxFloat64)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		m = math.Max(m, av)
	}
	return m
}

// MaxIndex64 computes the max over vector values, and returns index of max as well
// Skips NaN's
func MaxIndex64(a []float64) (float64, int) {
	m := float64(-math.MaxFloat64)
	mi := -1
	for i, av := range a {
		if math.IsNaN(av) {
			continue
		}
		if av > m {
			m = av
			mi = i
		}
	}
	return m, mi
}

///////////////////////////////////////////
//  MaxAbs

// MaxAbs32 computes the max of absolute value over vector values.
// Skips NaN's
func MaxAbs32(a []float32) float32 {
	m := float32(0)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		m = math32.Max(m, math32.Abs(av))
	}
	return m
}

// MaxAbs64 computes the max over vector values.
// Skips NaN's
func MaxAbs64(a []float64) float64 {
	m := float64(0)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		m = math.Max(m, math.Abs(av))
	}
	return m
}

///////////////////////////////////////////
//  Min

// Min32 computes the max over vector values.
// Skips NaN's
func Min32(a []float32) float32 {
	m := float32(math.MaxFloat32)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		m = math32.Min(m, av)
	}
	return m
}

// MinIndex32 computes the min over vector values, and returns index of min as well
// Skips NaN's
func MinIndex32(a []float32) (float32, int) {
	m := float32(math.MaxFloat32)
	mi := -1
	for i, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		if av < m {
			m = av
			mi = i
		}
	}
	return m, mi
}

// Min64 computes the max over vector values.
// Skips NaN's
func Min64(a []float64) float64 {
	m := float64(math.MaxFloat64)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		m = math.Min(m, av)
	}
	return m
}

// MinIndex64 computes the min over vector values, and returns index of min as well
// Skips NaN's
func MinIndex64(a []float64) (float64, int) {
	m := float64(math.MaxFloat64)
	mi := -1
	for i, av := range a {
		if math.IsNaN(av) {
			continue
		}
		if av < m {
			m = av
			mi = i
		}
	}
	return m, mi
}

///////////////////////////////////////////
//  MinAbs

// MinAbs32 computes the max of absolute value over vector values.
// Skips NaN's
func MinAbs32(a []float32) float32 {
	m := float32(math.MaxFloat32)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		m = math32.Min(m, math32.Abs(av))
	}
	return m
}

// MinAbs64 computes the max over vector values.
// Skips NaN's
func MinAbs64(a []float64) float64 {
	m := float64(math.MaxFloat64)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		m = math.Min(m, math.Abs(av))
	}
	return m
}
