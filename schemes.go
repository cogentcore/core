// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import "image/color"

// Schemes contains multiple color schemes
// (light, dark, and any custom ones).
type Schemes struct {
	Light Scheme
	Dark  Scheme
	// TODO: maybe custom schemes?
}

// TheSchemes are the global color schemes.
var TheSchemes = NewSchemes(NewPalette(KeyFromPrimary(color.RGBA{66, 133, 244, 255}))) // primary: #4285f4 (Google Blue)

// NewSchemes returns new [Schemes] for the given
// [Palette] containing both light and dark schemes.
func NewSchemes(p *MatPalette) *Schemes {
	return &Schemes{
		Light: NewLightScheme(p),
		Dark:  NewDarkScheme(p),
	}
}
