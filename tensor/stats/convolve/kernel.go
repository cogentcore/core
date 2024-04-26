// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package convolve

import (
	"math"

	"cogentcore.org/core/math32"
)

// GaussianKernel32 returns a normalized gaussian kernel for smoothing
// with given half-width and normalized sigma (actual sigma = khalf * sigma).
// A sigma value of .5 is typical for smaller half-widths for containing
// most of the gaussian efficiently -- anything lower than .33 is inefficient --
// generally just use a lower half-width instead.
func GaussianKernel32(khalf int, sigma float32) []float32 {
	ksz := khalf*2 + 1
	kern := make([]float32, ksz)
	sigdiv := 1 / (sigma * float32(khalf))
	var sum float32
	for i := 0; i < ksz; i++ {
		x := sigdiv * float32(i-khalf)
		kv := math32.Exp(-0.5 * x * x)
		kern[i] = kv
		sum += kv
	}
	nfac := 1 / sum
	for i := 0; i < ksz; i++ {
		kern[i] *= nfac
	}
	return kern
}

// GaussianKernel64 returns a normalized gaussian kernel
// with given half-width and normalized sigma (actual sigma = khalf * sigma)
// A sigma value of .5 is typical for smaller half-widths for containing
// most of the gaussian efficiently -- anything lower than .33 is inefficient --
// generally just use a lower half-width instead.
func GaussianKernel64(khalf int, sigma float64) []float64 {
	ksz := khalf*2 + 1
	kern := make([]float64, ksz)
	sigdiv := 1 / (sigma * float64(khalf))
	var sum float64
	for i := 0; i < ksz; i++ {
		x := sigdiv * float64(i-khalf)
		kv := math.Exp(-0.5 * x * x)
		kern[i] = kv
		sum += kv
	}
	nfac := 1 / sum
	for i := 0; i < ksz; i++ {
		kern[i] *= nfac
	}
	return kern
}
