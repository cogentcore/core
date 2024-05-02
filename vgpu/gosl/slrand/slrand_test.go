// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slrand

import (
	"fmt"
	"math"
	"testing"

	"github.com/emer/gosl/v2/sltype"
)

// Known Answer Test for values from the DEShawREsearch reference impl
func TestKAT(t *testing.T) {
	kats := []struct {
		ctr sltype.Uint2
		key uint32
		res sltype.Uint2
	}{{sltype.Uint2{0, 0}, 0, sltype.Uint2{0xff1dae59, 0x6cd10df2}},
		{sltype.Uint2{0xffffffff, 0xffffffff}, 0xffffffff, sltype.Uint2{0x2c3f628b, 0xab4fd7ad}},
		{sltype.Uint2{0x243f6a88, 0x85a308d3}, 0x13198a2e, sltype.Uint2{0xdd7ce038, 0xf62a4c12}}}

	for _, tv := range kats {
		r := Philox2x32(tv.ctr, tv.key)
		if r != tv.res {
			fmt.Printf("ctr: %v  key: %d != result: %v -- got: %v\n", tv.ctr, tv.key, tv.res, r)
		}
	}
}

// Float01 Known Answer Test for float conversion values from the DEShawREsearch reference impl
func TestFloat01KAT(t *testing.T) {
	minint := math.MinInt32
	kats := []struct {
		base uint32
		add  uint32
		res  float32
	}{{0, 0, 1.16415321826934814453e-10},
		{uint32(minint), 0, 0.5},
		{math.MaxInt32, 0, 0.5},
		{math.MaxUint32, 0, 0.99999994},
	}

	for _, tv := range kats {
		r := Uint32ToFloat(tv.base + tv.add)
		if r != tv.res {
			fmt.Printf("base: %x  add: %x != result: %g -- got: %g\n", tv.base, tv.add, tv.res, r)
		}
	}
}

// Float11 Known Answer Test for float conversion values from the DEShawREsearch reference impl
func TestFloat11KAT(t *testing.T) {
	minint := math.MinInt32
	kats := []struct {
		base uint32
		add  uint32
		res  float32
	}{{0, 0, 2.32830643653869628906e-10},
		{uint32(minint), 0, -1.0},
		{math.MaxInt32, 0, 1.0},
		{math.MaxUint32, 0, -2.32830643653869628906e-10},
	}

	for _, tv := range kats {
		r := Uint32ToFloat11(tv.base + tv.add)
		if r != tv.res {
			fmt.Printf("base: %x  add: %x != result: %g -- got: %g\n", tv.base, tv.add, tv.res, r)
		}
	}
}

func TestRand(t *testing.T) {
	var counter sltype.Uint2
	for i := 0; i < 10; i++ {
		fmt.Printf("%g\t%g\t%g\n", Float(&counter, 0), Float11(&counter, 1), NormFloat(&counter, 2))
	}
}

func TestCounter(t *testing.T) {
	counter := sltype.Uint2{X: 0xfffffffe, Y: 0}
	ctr := counter
	CounterAdd(&ctr, 4)
	if ctr.X != 2 && ctr.Y != 1 {
		t.Errorf("Should be 2, 1: %v\n", ctr)
	}
	ctr = counter
	CounterAdd(&ctr, 1)
	if ctr.X != 0xffffffff && ctr.Y == 0 {
		t.Errorf("Should be 0, 0xfffffffe: %v\n", ctr)
	}
	ctr = counter
	CounterAdd(&ctr, 2)
	if ctr.X != 0 && ctr.Y == 1 {
		t.Errorf("Should be 0, 1: %v\n", ctr)
	}
	ctr = counter
	CounterIncr(&ctr)
	CounterIncr(&ctr)
	if ctr.X != 0 && ctr.Y != 1 {
		t.Errorf("Should be 0, 1: %v\n", ctr)
	}
}

func TestIntn(t *testing.T) {
	var counter sltype.Uint2
	n := uint32(20)
	for i := 0; i < 1000; i++ {
		r := Uintn(&counter, 0, n)
		if r >= n {
			t.Errorf("r >= n: %d\n", r)
		}
		// fmt.Printf("%d\t%d\n", i, r)
	}
}
