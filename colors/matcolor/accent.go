// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matcolor

import (
	"image"
)

// Accent contains the four standard variations of a base accent color.
type Accent struct {

	// Base is the base color for typically high-emphasis content.
	Base image.Image

	// On is the color applied to content on top of [Accent.Base].
	On image.Image

	// Container is the color applied to elements with less emphasis than [Accent.Base].
	Container image.Image

	// OnContainer is the color applied to content on top of [Accent.Container].
	OnContainer image.Image
}

// NewAccentLight returns a new light theme [Accent] from the given [Tones].
func NewAccentLight(tones Tones) Accent {
	return Accent{
		Base:        tones.AbsToneUniform(40),
		On:          tones.AbsToneUniform(100),
		Container:   tones.AbsToneUniform(90),
		OnContainer: tones.AbsToneUniform(10),
	}
}

// NewAccentDark returns a new dark theme [Accent] from the given [Tones].
func NewAccentDark(tones Tones) Accent {
	return Accent{
		Base:        tones.AbsToneUniform(80),
		On:          tones.AbsToneUniform(20),
		Container:   tones.AbsToneUniform(30),
		OnContainer: tones.AbsToneUniform(90),
	}
}
