// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import "testing"

var benchSink float32

func BenchmarkPow(b *testing.B) {
	for i := range b.N {
		benchSink = Pow(float32(i%100)*0.1+0.5, 2.4)
	}
}

func BenchmarkSqrt(b *testing.B) {
	for i := range b.N {
		benchSink = Sqrt(float32(i % 1000))
	}
}

func BenchmarkSinCos(b *testing.B) {
	for i := range b.N {
		s, c := Sincos(float32(i%628) * 0.01)
		benchSink = s + c
	}
}

func BenchmarkAbs(b *testing.B) {
	for i := range b.N {
		benchSink = Abs(float32(i%200) - 100)
	}
}

func BenchmarkVector2Length(b *testing.B) {
	v := Vec2(3, 4)
	for range b.N {
		benchSink = v.Length()
	}
}

func BenchmarkMatrix2Mul(b *testing.B) {
	m := Identity2().Rotate(0.3).Scale(2, 3)
	p := Vec2(1, 2)
	for range b.N {
		p = m.MulPoint(p)
	}
	benchSink = p.X
}
