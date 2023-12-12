// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"image"
	"log/slog"
	"math"

	"github.com/anthonynsimon/bild/clone"
	"github.com/anthonynsimon/bild/convolution"
	"goki.dev/mat32/v2"
)

// scipy impl:
// https://github.com/scipy/scipy/blob/4bfc152f6ee1ca48c73c06e27f7ef021d729f496/scipy/ndimage/filters.py#L136
// #L214 has the invocation: radius = Ceil(sigma)

// bild uses:
// math.Exp(-0.5 * (x * x / (2 * radius))
// so   sigma = sqrt(radius) / 2
// and radius = sigma * sigma * 2

// GaussianBlurKernel1D returns a 1D Gaussian kernel.
// Sigma is the standard deviation,
// and the radius of the kernel is 4 * sigma.
func GaussianBlurKernel1D(sigma float64) *convolution.Kernel {
	sigma2 := sigma * sigma
	sfactor := -0.5 / sigma2
	radius := math.Ceil(4 * sigma) // truncate = 4 in scipy
	length := 2*int(radius) + 1

	// Create the 1-d gaussian kernel
	k := convolution.NewKernel(length, 1)
	for i, x := 0, -radius; i < length; i, x = i+1, x+1 {
		k.Matrix[i] = math.Exp(sfactor * (x * x))
	}
	return k
}

// GaussianBlur returns a smoothly blurred version of the image using
// a Gaussian function. Sigma is the standard deviation of the Gaussian
// function, and a kernel of radius = 4 * Sigma is used.
func GaussianBlur(src image.Image, sigma float64) *image.RGBA {
	if sigma <= 0 {
		return clone.AsRGBA(src)
	}

	k := GaussianBlurKernel1D(sigma).Normalized()

	// Perform separable convolution
	options := convolution.Options{Bias: 0, Wrap: false, KeepAlpha: false}
	result := convolution.Convolve(src, k, &options)
	result = convolution.Convolve(result, k.Transposed(), &options)

	return result
}

// EdgeBlurFactors returns multiplicative factors that replicate the effect
// of a Gaussian kernel applied to a sharp edge transition in the middle of
// a line segment, with a given Gaussian sigma, and radius = sigma * radiusFactor.
// The returned line factors go from -radius to +radius.
// For low-contrast (opacity) cases, radiusFactor = 1 works well,
// because values beyond 1 sigma are effectively invisible, but 2 looks
// better for greater contrast cases.
func EdgeBlurFactors(sigma, radiusFactor float32) []float32 {
	radius := mat32.Ceil(sigma * radiusFactor)
	irad := int(radius)
	klen := irad*2 + 1
	sfactor := -0.5 / (sigma * sigma)

	if klen < 0 {
		slog.Error("unexpected error (need to fix): paint.EdgeBlurFactors: got out of range klen", "klen", klen, "radius", radius, "sigma", sigma, "radiusFactor", radiusFactor)
		return []float32{}
	}
	k := make([]float32, klen)
	sum := float32(0)
	rstart := -radius + 0.5
	for i, x := 0, rstart; i < klen; i, x = i+1, x+1 {
		v := mat32.FastExp(sfactor * (x * x))
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
