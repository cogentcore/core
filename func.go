// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import "image/color"

// Func is a function that returns a color for an (x, y) coordinate point.
type Func func(x, y int) color.Color

// SolidFunc returns a [Func] that always returns the same given solid color.
func SolidFunc(c color.Color) Func {
	return func(x, y int) color.Color {
		return c
	}
}
