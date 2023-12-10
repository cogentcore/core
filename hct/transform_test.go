// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hct

import (
	"fmt"
	"image/color"
	"math/rand"
	"testing"
)

func TestBlend(t *testing.T) {
	// yellow and blue
	c := Blend(50, color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 255})
	fmt.Println("blend", c)
}

func TestMinHueDistance(t *testing.T) {
	t.Skip("informational confirmation")
	for i := 0; i < 50; i++ {
		a := rand.Intn(360)
		b := rand.Intn(360)
		d := MinHueDistance(float32(a), float32(b))
		fmt.Println(a, b, d)
	}
}
