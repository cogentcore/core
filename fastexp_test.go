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

func BenchmarkFastExp(b *testing.B) {
	for n := 0; n < b.N; n++ {
		FastExp(float32(n%40 - 20))
	}
}

func BenchmarkExpStd64(b *testing.B) {
	for n := 0; n < b.N; n++ {
		math.Exp(float64(n%40 - 20))
	}
}

/*
func BenchmarkExp32(b *testing.B) {
	for n := 0; n < b.N; n++ {
		math32.Exp(float32(n%40 - 20))
	}
}
*/
