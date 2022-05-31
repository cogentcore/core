// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package szalloc

import (
	"image"
	"math/rand"
	"testing"

	"github.com/goki/mat32"
)

func TestSzAlloc(t *testing.T) {
	var sa SzAlloc
	nsz := 300
	szs := make([]image.Point, nsz)
	for i := range szs {
		szs[i] = image.Point{X: rand.Intn(1024), Y: rand.Intn(1024)}
	}
	sa.SetSizes(image.Point{4, 4}, 20, szs)
	sa.Alloc()
	sa.PrintGps()
}

func TestPctWin(t *testing.T) {
	pct := float32(.7)
	for u := float32(0); u < 3; u += .1 {
		pu := mat32.Mod(u*pct, pct)
		_ = pu
		// fmt.Printf("u: %g   pu: %g\n", u, pu)
	}
}
