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
	var x float32
	for n := 0; n < b.N; n++ {
		x = FastExp(float32(n%40 - 20))
	}
	result32 = x
}

var result64 float64
func BenchmarkExpStd64(b *testing.B) {
	var x float64
	for n := 0; n < b.N; n++ {
		x = math.Exp(float64(n%40 - 20))
	}
	result64 = x
}

/*
func BenchmarkExp32(b *testing.B) {
	var x float32
	for n := 0; n < b.N; n++ {
		x = math32.Exp(float32(n%40 - 20))
	}
	result32 = x
}
*/
