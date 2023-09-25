// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import "image/color"

// MatPalette contains a Material Design 3 tonal palette
// with tonal values for each of the standard colors and
// any custom colors. The main palette is stored in [Palette].
type MatPalette struct {

	// the tones for the primary key color
	Primary Tones `desc:"the tones for the primary key color"`

	// the tones for the secondary key color
	Secondary Tones `desc:"the tones for the secondary key color"`

	// the tones for the tertiary key color
	Tertiary Tones `desc:"the tones for the tertiary key color"`

	// the tones for the error key color
	Error Tones `desc:"the tones for the error key color"`

	// the tones for the neutral key color
	Neutral Tones `desc:"the tones for the neutral key color"`

	// the tones for the neutral variant key color
	NeutralVariant Tones `desc:"the tones for the neutral variant key color"`

	// an optional map of tones for custom accent key colors
	Custom map[string]Tones `desc:"an optional map of tones for custom accent key colors"`
}

// Palette contains the main, global [MatPalette]. It can
// be used by end-user code for accessing tonal palette values,
// although [Scheme] is a more typical way to access the color
// scheme values. It defaults to a palette based around a
// primary color of Google Blue (#4285f4)
var Palette = NewPalette(KeyFromPrimary(color.RGBA{66, 133, 244, 255})) // primary: #4285f4 (Google Blue)

// NewPalette creates a new [MatPalette] from the given key colors.
func NewPalette(key *Key) *MatPalette {
	p := &MatPalette{
		Primary:        NewTones(key.Primary),
		Secondary:      NewTones(key.Secondary),
		Tertiary:       NewTones(key.Tertiary),
		Error:          NewTones(key.Error),
		Neutral:        NewTones(key.Neutral),
		NeutralVariant: NewTones(key.NeutralVariant),
	}
	for name, c := range key.Custom {
		p.Custom[name] = NewTones(c)
	}
	return p
}
