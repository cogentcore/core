// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matcolor

import (
	"image"
	"image/color"
)

//go:generate core generate

// Scheme contains the colors for one color scheme (ex: light or dark).
// To generate a scheme, use [NewScheme].
type Scheme struct {

	// Primary is the primary color applied to important elements
	Primary Accent

	// Secondary is the secondary color applied to less important elements
	Secondary Accent

	// Tertiary is the tertiary color applied as an accent to highlight elements and create contrast between other colors
	Tertiary Accent

	// Select is the selection color applied to selected or highlighted elements and text
	Select Accent

	// Error is the error color applied to elements that indicate an error or danger
	Error Accent

	// Success is the color applied to elements that indicate success
	Success Accent

	// Warn is the color applied to elements that indicate a warning
	Warn Accent

	// an optional map of custom accent colors
	Custom map[string]Accent

	// SurfaceDim is the color applied to elements that will always have the dimmest surface color (see Surface for more information)
	SurfaceDim image.Image

	// Surface is the color applied to contained areas, like the background of an app
	Surface image.Image

	// SurfaceBright is the color applied to elements that will always have the brightest surface color (see Surface for more information)
	SurfaceBright image.Image

	// SurfaceContainerLowest is the color applied to surface container elements that have the lowest emphasis (see SurfaceContainer for more information)
	SurfaceContainerLowest image.Image

	// SurfaceContainerLow is the color applied to surface container elements that have lower emphasis (see SurfaceContainer for more information)
	SurfaceContainerLow image.Image

	// SurfaceContainer is the color applied to container elements that contrast elements with the surface color
	SurfaceContainer image.Image

	// SurfaceContainerHigh is the color applied to surface container elements that have higher emphasis (see SurfaceContainer for more information)
	SurfaceContainerHigh image.Image

	// SurfaceContainerHighest is the color applied to surface container elements that have the highest emphasis (see SurfaceContainer for more information)
	SurfaceContainerHighest image.Image

	// SurfaceVariant is the color applied to contained areas that contrast standard Surface elements
	SurfaceVariant image.Image

	// OnSurface is the color applied to content on top of Surface elements
	OnSurface image.Image

	// OnSurfaceVariant is the color applied to content on top of SurfaceVariant elements
	OnSurfaceVariant image.Image

	// InverseSurface is the color applied to elements to make them the reverse color of the surrounding elements and create a contrasting effect
	InverseSurface image.Image

	// InverseOnSurface is the color applied to content on top of InverseSurface
	InverseOnSurface image.Image

	// InversePrimary is the color applied to interactive elements on top of InverseSurface
	InversePrimary image.Image

	// Background is the color applied to the background of the app and other low-emphasis areas
	Background image.Image

	// OnBackground is the color applied to content on top of Background
	OnBackground image.Image

	// Outline is the color applied to borders to create emphasized boundaries that need to have sufficient contrast
	Outline image.Image

	// OutlineVariant is the color applied to create decorative boundaries
	OutlineVariant image.Image

	// Shadow is the color applied to shadows
	Shadow image.Image

	// SurfaceTint is the color applied to tint surfaces
	SurfaceTint image.Image

	// Scrim is the color applied to scrims (semi-transparent overlays)
	Scrim image.Image
}

// NewLightScheme returns a new light-themed [Scheme]
// based on the given [Palette].
func NewLightScheme(p *Palette) Scheme {
	s := Scheme{
		Primary:   NewAccentLight(p.Primary),
		Secondary: NewAccentLight(p.Secondary),
		Tertiary:  NewAccentLight(p.Tertiary),
		Select:    NewAccentLight(p.Select),
		Error:     NewAccentLight(p.Error),
		Success:   NewAccentLight(p.Success),
		Warn:      NewAccentLight(p.Warn),
		Custom:    map[string]Accent{},

		SurfaceDim:    p.Neutral.AbsToneUniform(87),
		Surface:       p.Neutral.AbsToneUniform(98),
		SurfaceBright: p.Neutral.AbsToneUniform(98),

		SurfaceContainerLowest:  p.Neutral.AbsToneUniform(100),
		SurfaceContainerLow:     p.Neutral.AbsToneUniform(96),
		SurfaceContainer:        p.Neutral.AbsToneUniform(94),
		SurfaceContainerHigh:    p.Neutral.AbsToneUniform(92),
		SurfaceContainerHighest: p.Neutral.AbsToneUniform(90),

		SurfaceVariant:   p.NeutralVariant.AbsToneUniform(90),
		OnSurface:        p.NeutralVariant.AbsToneUniform(10),
		OnSurfaceVariant: p.NeutralVariant.AbsToneUniform(30),

		InverseSurface:   p.Neutral.AbsToneUniform(20),
		InverseOnSurface: p.Neutral.AbsToneUniform(95),
		InversePrimary:   p.Primary.AbsToneUniform(80),

		Background:   p.Neutral.AbsToneUniform(98),
		OnBackground: p.Neutral.AbsToneUniform(10),

		Outline:        p.NeutralVariant.AbsToneUniform(50),
		OutlineVariant: p.NeutralVariant.AbsToneUniform(80),

		Shadow:      p.Neutral.AbsToneUniform(0),
		SurfaceTint: p.Primary.AbsToneUniform(40),
		Scrim:       p.Neutral.AbsToneUniform(0),
	}
	for nm, c := range p.Custom {
		s.Custom[nm] = NewAccentLight(c)
	}
	return s
	// TODO: maybe fixed colors
}

// NewDarkScheme returns a new dark-themed [Scheme]
// based on the given [Palette].
func NewDarkScheme(p *Palette) Scheme {
	s := Scheme{
		Primary:   NewAccentDark(p.Primary),
		Secondary: NewAccentDark(p.Secondary),
		Tertiary:  NewAccentDark(p.Tertiary),
		Select:    NewAccentDark(p.Select),
		Error:     NewAccentDark(p.Error),
		Success:   NewAccentDark(p.Success),
		Warn:      NewAccentDark(p.Warn),
		Custom:    map[string]Accent{},

		SurfaceDim:    p.Neutral.AbsToneUniform(6),
		Surface:       p.Neutral.AbsToneUniform(6),
		SurfaceBright: p.Neutral.AbsToneUniform(24),

		SurfaceContainerLowest:  p.Neutral.AbsToneUniform(4),
		SurfaceContainerLow:     p.Neutral.AbsToneUniform(10),
		SurfaceContainer:        p.Neutral.AbsToneUniform(12),
		SurfaceContainerHigh:    p.Neutral.AbsToneUniform(17),
		SurfaceContainerHighest: p.Neutral.AbsToneUniform(22),

		SurfaceVariant:   p.NeutralVariant.AbsToneUniform(30),
		OnSurface:        p.NeutralVariant.AbsToneUniform(90),
		OnSurfaceVariant: p.NeutralVariant.AbsToneUniform(80),

		InverseSurface:   p.Neutral.AbsToneUniform(90),
		InverseOnSurface: p.Neutral.AbsToneUniform(20),
		InversePrimary:   p.Primary.AbsToneUniform(40),

		Background:   p.Neutral.AbsToneUniform(6),
		OnBackground: p.Neutral.AbsToneUniform(90),

		Outline:        p.NeutralVariant.AbsToneUniform(60),
		OutlineVariant: p.NeutralVariant.AbsToneUniform(30),

		// We want some visible "glow" shadow, but not too much
		Shadow:      image.NewUniform(color.RGBA{127, 127, 127, 127}),
		SurfaceTint: p.Primary.AbsToneUniform(80),
		Scrim:       p.Neutral.AbsToneUniform(0),
	}
	for nm, c := range p.Custom {
		s.Custom[nm] = NewAccentDark(c)
	}
	return s
	// TODO: custom and fixed colors?
}

// SchemeIsDark is whether the currently active color scheme
// is a dark-themed or light-themed color scheme. In almost
// all cases, it should be set via [cogentcore.org/core/colors.SetScheme],
// not directly.
var SchemeIsDark = false
