// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matcolor

import "image/color"

// Accent contains the four standard variations of a base accent color.
type Accent struct {

	// Base is the base color
	Base color.RGBA `desc:"Base is the base color"`

	// On is the color applied to content on top of [Accent.Base]
	On color.RGBA `desc:"On is the color applied to content on top of [Accent.Base]"`

	// Container is the color applied to elements with less emphasis than [Accent.Base]
	Container color.RGBA `desc:"Container is the color applied to elements with less emphasis than [Accent.Base]"`

	// OnContainer is the color applied to content on top of [Accent.Container]
	OnContainer color.RGBA `desc:"OnContainer is the color applied to content on top of [Accent.Container]"`
}

// NewAccentLight returns a new light theme [Accent] from the given [Tones]
func NewAccentLight(tones Tones) Accent {
	return Accent{
		Base:        tones.Tone(40),
		On:          tones.Tone(100),
		Container:   tones.Tone(90),
		OnContainer: tones.Tone(10),
	}
}

// NewAccentDark returns a new dark theme [Accent] from the given [Tones]
func NewAccentDark(tones Tones) Accent {
	return Accent{
		Base:        tones.Tone(80),
		On:          tones.Tone(20),
		Container:   tones.Tone(30),
		OnContainer: tones.Tone(90),
	}
}
