// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image"
	"image/color"

	"github.com/anthonynsimon/bild/adjust"
	"github.com/anthonynsimon/bild/clone"
)

// Full represents a fully specified color that can either be a solid color or an
// [image.Image]. If Image is nil, it is a solid color; otherwise, it is an image.
// Solid should typically be set using the [Full.SetSolid] method to ensure that
// Image is nil and thus Solid will be taken into account. Image is frequently set
// to a gradient type (eg: [gradient.Linear], [gradient.Radial], or [gradient.Conic]),
// which all implement the [image.Image] interface.
type Full struct {
	Image image.Image
	Solid color.RGBA
}

// SolidFull returns a new [Full] from the given solid color.
func SolidFull(solid color.Color) Full {
	return Full{Solid: AsRGBA(solid)}
}

// ImageFull returns a new [Full] from the given image.
func ImageFull(img image.Image) Full {
	return Full{Image: img}
}

// IsNil returns whether the color is nil, checking both the image
// and the solid color.
func (f *Full) IsNil() bool {
	return f.Image == nil && IsNil(f.Solid)
}

// TODO(kai): does SolidOrNil really make sense?

// SolidOrNil returns the solid color if it is non-nil, or nil otherwise.
// It is should be used by consumers that explicitly handle nil colors.
func (f *Full) SolidOrNil() color.Color {
	if IsNil(f.Solid) {
		return nil
	}
	return f.Solid
}

// SetSolid sets the color to the given solid [color.Color],
// also setting the image to nil.
func (f *Full) SetSolid(solid color.Color) {
	f.Solid = AsRGBA(solid)
	f.Image = nil
}

// SetName sets the color to the solid color with the given name,
// also setting the image to nil.
func (f *Full) SetName(name string) error {
	s, err := FromName(name)
	if err != nil {
		return err
	}
	f.SetSolid(s)
	return nil
}

// CopyFrom copies from the given full color, making a copy of the image
// if it is non-nil.
func (f *Full) CopyFrom(cp Full) {
	f.Solid = cp.Solid
	if cp.Image != nil {
		f.Image = clone.AsRGBA(cp.Image)
	}
}

// ApplyOpacity applies the given opacity to the color and the image if it is non-nil.
func (f *Full) ApplyOpacity(opacity float32) {
	f.Solid = ApplyOpacity(f.Solid, opacity)
	if f.Image != nil {
		f.Image = adjust.Apply(f.Image, func(r color.RGBA) color.RGBA {
			return ApplyOpacity(r, opacity)
		})
	}
}

// SetAny sets the color from the given value of any type in the given Context.
// It handles values of types [Full], [color.Color], [image.Image], and string. If no Context
// is provided, it uses [BaseContext] with [Transparent].
func (f *Full) SetAny(val any, ctx ...Context) error {
	switch v := val.(type) {
	case *Full:
		*f = *v
	case Full:
		*f = v
	case color.Color:
		f.SetSolid(v)
	case image.Image:
		f.Image = v
	case string:
		f.SetString(v, ctx...)
	}
	return nil
}
