// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image/color"

	"goki.dev/colors/matcolor"
)

// Palette contains the main, global [MatPalette]. It can
// be used by end-user code for accessing tonal palette values,
// although [Scheme] is a more typical way to access the color
// scheme values. It defaults to a palette based around a
// primary color of Google Blue (#4285f4)
var Palette = matcolor.NewPalette(matcolor.KeyFromPrimary(color.RGBA{66, 133, 244, 255}))
