// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/anthonynsimon/bild/blur"
)

// This mostly replicates the first test from this reference:
// https://docs.scipy.org/doc/scipy/reference/generated/scipy.ndimage.gaussian_filter.html
func TestGaussianBlur(t *testing.T) {
	sigma := 1.0
	k := GaussianBlurKernel1D(sigma)
	fmt.Println(k.Matrix)

	testIn := []uint8{}
	for n := uint8(0); n < 50; n += 2 {
		testIn = append(testIn, n)
	}
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	for i, v := range testIn {
		img.Set(i%5, i/5, color.RGBA{v, v, v, v})
	}
	blr := GaussianBlur(img, sigma)
	for i := range testIn {
		fmt.Print(blr.At(i%5, i/5).(color.RGBA).R, " ")
		if i%5 == 4 {
			fmt.Println("")
		}
	}
	fmt.Println("bild:")

	bildRad := sigma // 0.5 * sigma * sigma
	blrBild := blur.Gaussian(img, bildRad)
	for i := range testIn {
		fmt.Print(blrBild.At(i%5, i/5).(color.RGBA).R, " ")
		if i%5 == 4 {
			fmt.Println("")
		}
	}

	// our results -- these could be rounding errors
	// 3 5 7 8 10
	// 10 12 14 15 17  <- correct
	// 20 22 24 25 27  <- correct
	// 29 31 33 34 36  <- correct
	// 36 38 40 41 43

	// scipy says:
	// 4  6  8  9 11
	// 10 12 14 15 17
	// 20 22 24 25 27
	// 29 31 33 34 36
	// 35 37 39 40 42
}

func TestEdgeBlurFactors(t *testing.T) {
	fmt.Println(EdgeBlurFactors(4))
}
