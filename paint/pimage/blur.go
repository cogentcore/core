// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pimage

import (
	"image"
	"math"

	"github.com/anthonynsimon/bild/clone"
	"github.com/anthonynsimon/bild/convolution"
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
