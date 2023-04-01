// Copyright 2021 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mat32

import (
	"fmt"
	"math"
	"testing"
)

func TestFastExp(t *testing.T) {
	for x := float32(-87); x <= 88.43114; x += 1.0e-01 {
		fx := FastExp(x)
		sx := float32(math.Exp(float64(x)))
		if Abs((fx-sx)/sx) > 1.0e-5 {
			fmt.Printf("Exp4 at: %g  err from cx: %g  vs  %g\n", x, fx, sx)
		}
	}
}

func TestFastExpNaN(t *testing.T) {
	// found this in boa function EncodeVal of PopCodeParams
	x := float32(9.398623)
	x2 := -(x * x) // should be 88.33411429612902
	fx := FastExp(x2)
	if math.IsNaN(float64(fx)) {
		t.Fatalf("Found NaN at x=%g fx=%g", x2, fx)
	}
}

func TestFastExpNaNRange(t *testing.T) {
	cnt := 0
	for x := float32(-89); x <= -87; x += 1.0e-03 {
		cnt++
		if cnt%1000 == 0 {
			fmt.Printf("Trying x: %f\n", x)
		}
		fx := FastExp(x)
		if math.IsNaN(float64(fx)) {
			t.Fatalf("Found NaN at x=%f", x)
		}
	}
}

func TestFastExpFindNaN(t *testing.T) {
	// use binary search to find the NaN cutoff point
	left := float32(-89)
	right := float32(-87)
	x := float32(0)
	for cnt := 0; cnt < 100000; cnt++ {
		x = (left + right) / 2
		if cnt%10000 == 0 {
			fmt.Printf("Trying x: %f\n", x)
		}
		// make sure you comment out the cutoff point in FastExp first
		fx := FastExp(x)
		if math.IsNaN(float64(fx)) {
			t.Errorf("Found NaN at x=%f", x)
			left = x
		} else {
			right = x
		}
	}
}

var result32 float32

func BenchmarkFastExp(b *testing.B) {
	// pre-convert the input, such that we're not measuring the speed of
	// the mod and sub operations.
	input := make([]float32, b.N)
	for i := range input {
		input[i] = float32(i%40 - 20)
	}

	b.ResetTimer()

	var x float32
	for n := 0; n < b.N; n++ {
		x += FastExp(input[n])
	}
	result32 = x
}

var result64 float64

func BenchmarkExpStd64(b *testing.B) {
	input := make([]float64, b.N)
	for i := range input {
		input[i] = float64(i%40 - 20)
	}

	b.ResetTimer()

	var x float64
	for n := 0; n < b.N; n++ {
		x += math.Exp(input[n])
	}
	result64 = x
}

func BenchmarkExp32(b *testing.B) {
	var x float32

	input := make([]float32, b.N)
	for i := range input {
		input[i] += float32(i%40 - 20)
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		x = float32(math.Exp(float64(input[n])))
	}
	result32 = x
}
