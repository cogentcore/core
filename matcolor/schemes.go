// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matcolor

// Schemes contains multiple color schemes
// (light, dark, and any custom ones).
type Schemes struct {
	Light Scheme
	Dark  Scheme
	// TODO: maybe custom schemes?
}

// NewSchemes returns new [Schemes] for the given
// [Palette] containing both light and dark schemes.
func NewSchemes(p *Palette) *Schemes {
	return &Schemes{
		Light: NewLightScheme(p),
		Dark:  NewDarkScheme(p),
	}
}
