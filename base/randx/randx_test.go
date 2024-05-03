// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package randx

import (
	"math"
	"testing"

	"cogentcore.org/core/base/num"
)

func TestPoisson(t *testing.T) {
	// t.Skip()
	vr := 8.0
	mi := 30
	rnd := NewGlobalRand()
	pd := make([]int, mi)
	for i := 0; i < 100000; i++ {
		kv := int(PoissonGen(vr, rnd))
		// fmt.Printf("poisson: %d\n", kv)
		if kv < mi {
			pd[kv]++
		}
	}

	ed := make([]int, 30)
	li := 0
	ep := math.Exp(-vr)
	p := 1.0
	for i := 0; i < 1000000; i++ {
		p *= rnd.Float64()
		if p <= ep {
			d := i - li
			if d < mi {
				ed[d]++
			}
			li = i
			p = 1
		}
	}

	mxi := 0
	mxe := 0
	im := 0
	em := 0
	for i := 0; i < mi; i++ {
		v := pd[i]
		if v > mxi {
			mxi = v
			im = i
		}
		v = ed[i]
		if v > mxe {
			mxe = v
			em = i
		}
	}
	// fmt.Printf("pd: %v\n", pd)
	// fmt.Printf("max idx: %d\n", im)
	// fmt.Printf("ed: %v\n", ed)
	// fmt.Printf("max idx: %d\n", em)
	if num.Abs(im-int(vr)) > 1 {
		t.Errorf("mode != lambda: %d != %d (tol 1)\n", im, int(vr))
	}
	if num.Abs(em-int(vr)) > 1 {
		t.Errorf("empirical mode != lambda: %d != %d (tol 1)\n", em, int(vr))
	}
}
