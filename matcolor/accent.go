// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matcolor

import "image/color"

// Accent contains the four standard variations of a base accent color.
type Accent struct { //gti:add

	// Base is the base color
	Base color.RGBA

	// On is the color applied to content on top of
	On color.RGBA

	// Container is the color applied to elements with less emphasis than
	Container color.RGBA

	// OnContainer is the color applied to content on top of
	OnContainer color.RGBA
}

// NewAccentLight returns a new light theme [Accent] from the given [Tones]
func NewAccentLight(tones Tones) Accent {
	return Accent{
		Base:        tones.AbsTone(40),
		On:          tones.AbsTone(100),
		Container:   tones.AbsTone(90),
		OnContainer: tones.AbsTone(10),
	}
}

// NewAccentDark returns a new dark theme [Accent] from the given [Tones]
func NewAccentDark(tones Tones) Accent {
	return Accent{
		Base:        tones.AbsTone(80),
		On:          tones.AbsTone(20),
		Container:   tones.AbsTone(30),
		OnContainer: tones.AbsTone(90),
	}
}
