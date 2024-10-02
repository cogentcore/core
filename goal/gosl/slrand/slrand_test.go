// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slrand

import (
	"fmt"
	"testing"

	"cogentcore.org/core/gpu/gosl/sltype"
	"github.com/stretchr/testify/assert"
)

// Known Answer Test for values from the DEShawREsearch reference impl
func TestKAT(t *testing.T) {
	kats := []struct {
		ctr sltype.Uint32Vec2
		key uint32
		res sltype.Uint32Vec2
	}{{sltype.Uint32Vec2{0, 0}, 0, sltype.Uint32Vec2{0xff1dae59, 0x6cd10df2}},
		{sltype.Uint32Vec2{0xffffffff, 0xffffffff}, 0xffffffff, sltype.Uint32Vec2{0x2c3f628b, 0xab4fd7ad}},
		{sltype.Uint32Vec2{0x243f6a88, 0x85a308d3}, 0x13198a2e, sltype.Uint32Vec2{0xdd7ce038, 0xf62a4c12}}}

	for _, tv := range kats {
		r := Philox2x32(sltype.Uint64FromLoHi(tv.ctr), tv.key)
		if r != tv.res {
			fmt.Printf("ctr: %v  key: %d != result: %v -- got: %v\n", tv.ctr, tv.key, tv.res, r)
		}
	}
}

func TestRand(t *testing.T) {
	trgs := [][]float32{
		{0.9965466, -0.84816515, 0.10381041},
		{0.86274576, -0.25368667, 0.7390654},
		{0.018057441, -0.5596625, 0.87044024},
		{0.010364834, 0.38940117, 0.4972646},
		{0.75196105, 0.57544005, 0.37224847},
		{0.23327535, 0.4237375, 0.19016998},
		{0.50003797, 0.82759297, 0.6614841},
		{0.6322405, -0.21457514, 0.17761084},
		{0.59605914, 0.9313429, 0.257},
		{0.57019144, -0.32633832, 0.9563069},
	}

	var counter uint64
	for i := uint32(0); i < 10; i++ {
		f, f11, fn := Float32(counter, i, 0), Float32Range11(counter, i, 1), Float32Norm(counter, i, 2)
		// fmt.Printf("{%g, %g, %g},\n", f, f11, fn)
		assert.Equal(t, trgs[i][0], f)
		assert.Equal(t, trgs[i][1], f11)
		assert.Equal(t, trgs[i][2], fn)
	}
}

func TestIntn(t *testing.T) {
	var counter uint64
	n := uint32(20)
	for i := uint32(0); i < 1000; i++ {
		r := Uint32N(counter, i, 0, n)
		if r >= n {
			t.Errorf("r >= n: %d\n", r)
		}
		// fmt.Printf("%d\t%d\n", i, r)
	}
}
