// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sltype

import (
	"fmt"
	"math"
	"testing"
)

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
		r := Uint32ToFloat32(tv.base + tv.add)
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
		r := Uint32ToFloat32Range11(tv.base + tv.add)
		if r != tv.res {
			fmt.Printf("base: %x  add: %x != result: %g -- got: %g\n", tv.base, tv.add, tv.res, r)
		}
	}
}

func TestCounter(t *testing.T) {
	counter := Uint64FromLoHi(Uint32Vec2{X: 0xfffffffe, Y: 0})
	ctr := counter
	ctr = Uint64Add32(ctr, 4)
	ctrlh := Uint64ToLoHi(ctr)
	if ctrlh.X != 2 && ctrlh.Y != 1 {
		t.Errorf("Should be 2, 1: %v\n", ctrlh)
	}
	ctr = counter
	ctr = Uint64Add32(ctr, 1)
	ctrlh = Uint64ToLoHi(ctr)
	if ctrlh.X != 0xffffffff && ctrlh.Y == 0 {
		t.Errorf("Should be 0, 0xfffffffe: %v\n", ctrlh)
	}
	ctr = counter
	ctr = Uint64Add32(ctr, 1)
	ctrlh = Uint64ToLoHi(ctr)
	if ctrlh.X != 0 && ctrlh.Y == 1 {
		t.Errorf("Should be 0, 1: %v\n", ctrlh)
	}
	ctr = counter
	ctr = Uint64Incr(ctr)
	ctr = Uint64Incr(ctr)
	ctrlh = Uint64ToLoHi(ctr)
	if ctrlh.X != 0 && ctrlh.Y != 1 {
		t.Errorf("Should be 0, 1: %v\n", ctrlh)
	}
}
