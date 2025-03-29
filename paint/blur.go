// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"cogentcore.org/core/math32"
)

// EdgeBlurFactors returns multiplicative factors that replicate the effect
// of a Gaussian kernel applied to a sharp edge transition in the middle of
// a line segment, with a given Gaussian sigma, and radius = sigma * radiusFactor.
// The returned line factors go from -radius to +radius.
// For low-contrast (opacity) cases, radiusFactor = 1 works well,
// because values beyond 1 sigma are effectively invisible, but 2 looks
// better for greater contrast cases.
func EdgeBlurFactors(sigma, radiusFactor float32) []float32 {
	radius := math32.Ceil(sigma * radiusFactor)
	irad := int(radius)
	klen := irad*2 + 1
	sfactor := -0.5 / (sigma * sigma)

	if klen < 0 {
		return []float32{}
	}
	k := make([]float32, klen)
	sum := float32(0)
	rstart := -radius + 0.5
	for i, x := 0, rstart; i < klen; i, x = i+1, x+1 {
		v := math32.FastExp(sfactor * (x * x))
		sum += v
		k[i] = v
	}
	for i, v := range k {
		k[i] = v / sum
	}

	line := make([]float32, klen)
	for li := range line {
		sum := float32(0)
		for ki, v := range k {
			if ki >= (klen - li) {
				break
			}
			sum += v
		}
		line[li] = sum
	}
	return line
}
