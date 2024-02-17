// Copyright (c) 2023, Cogent Core. All rights reserved.
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
	c := Blend(50, color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBA{A: 255})
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
