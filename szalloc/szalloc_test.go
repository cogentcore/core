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

func TestRandSzAlloc(t *testing.T) {
	var sa SzAlloc
	nsz := 300
	szs := make([]image.Point, nsz)
	for i := range szs {
		szs[i] = image.Point{X: rand.Intn(1024), Y: rand.Intn(1024)}
	}
	sa.SetSizes(image.Point{4, 4}, 20, szs)
	sa.Alloc()
	if len(sa.GpAllocs) != 16 {
		t.Error("failed, N gpallocs != 16\n")
	}
	// sa.PrintGps()
}

func TestUniqSzAlloc(t *testing.T) {
	var sa SzAlloc
	nsz := 20
	szs := make([]image.Point, nsz)
	for i := range szs {
		if i%2 == 0 {
			szs[i] = image.Point{X: 9, Y: 9}
		} else {
			szs[i] = image.Point{X: rand.Intn(1024), Y: rand.Intn(1024)}
		}
	}
	sa.SetSizes(image.Point{4, 4}, 20, szs)
	sa.Alloc()
	if len(sa.GpAllocs) != 11 {
		t.Error("failed, N gpallocs != 11\n")
	}
	// sa.PrintGps()
}

func TestPctWin(t *testing.T) {
	pct := float32(.7)
	for u := float32(0); u < 3; u += .1 {
		pu := mat32.Mod(u*pct, pct)
		_ = pu
		// fmt.Printf("u: %g   pu: %g\n", u, pu)
	}
}
