// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cam16

import (
	"fmt"
	"image/color"
	"testing"
)

func TestBlend(t *testing.T) {
	// yellow and blue
	c := Blend(50, color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBA{A: 255})
	fmt.Println("blend", c)
}
