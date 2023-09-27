// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image/color"
)

// Full represents a fully specified color that can either be a solid color or
// a gradient. If Gradient is nil, it is a solid color; otherwise, it is a gradient.
// Solid should typically be set using the [Full.SetSolid] method to
// ensure that Gradient is nil and thus Solid will be taken into account.
type Full struct {
	Solid    color.RGBA
	Gradient *Gradient
}

// IsNil returns whether the color is nil, checking both the gradient
// and the solid color.
func (f *Full) IsNil() bool {
	return f.Gradient == nil && IsNil(f.Solid)
}

// SolidOrNil returns the solid color if it is not non-nil, or nil otherwise.
// It is should be used by consumers that explicitly handle nil colors.
func (f *Full) SolidOrNil() color.Color {
	if IsNil(f.Solid) {
		return nil
	}
	return f.Solid
}

// SetSolid sets the color to the given solid color,
// also setting the gradient to nil.
func (f *Full) SetSolid(solid color.RGBA) {
	f.Solid = solid
	f.Gradient = nil
}

// SetSolid sets the color to the given solid [color.Color],
// also setting the gradient to nil.
func (f *Full) SetColor(clr color.Color) {
	f.Solid = AsRGBA(clr)
	f.Gradient = nil
}

// SetSolid sets the color to the solid color with the given name,
// also setting the gradient to nil.
func (f *Full) SetName(name string) {
	f.Solid = LogFromName(name)
	f.Gradient = nil
}
