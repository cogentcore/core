// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import "image/color"

// Full represents a fully specified color that can either be a solid color or
// a gradient. If Gradient is nil, it is a solid color; otherwise, it is a gradient.
// Solid should typically be set using the [Full.SetSolid] method to
// ensure that Gradient is nil and thus Solid will be taken into account.
type Full struct {
	Solid    color.RGBA
	Gradient *Gradient
}
