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
