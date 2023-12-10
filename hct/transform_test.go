// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hct

import (
	"fmt"
	"image/color"
	"testing"
)

func TestBlend(t *testing.T) {
	// yellow and blue
	c := Blend(50, color.RGBA{255, 255, 0, 255}, color.RGBA{0, 0, 255, 255})
	fmt.Println("blend", c)
}
