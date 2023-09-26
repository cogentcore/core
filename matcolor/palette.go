// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matcolor

// Palette contains a tonal palette with tonal values
// for each of the standard colors and any custom colors.
// Use [NewPalette] to create a new palette.
type Palette struct {

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

// NewPalette creates a new [Palette] from the given key colors.
func NewPalette(key *Key) *Palette {
	p := &Palette{
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
