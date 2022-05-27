// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package szalloc

import (
	"image"
	"math/rand"
	"testing"
)

func TestSzAlloc(t *testing.T) {
	var sa SzAlloc
	nsz := 200
	szs := make([]image.Point, nsz)
	for i := range szs {
		szs[i] = image.Point{X: rand.Intn(1024), Y: rand.Intn(1024)}
	}
	sa.SetSizes(16, szs)
	sa.Alloc()

}
