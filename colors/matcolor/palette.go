// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matcolor

// Palette contains a tonal palette with tonal values
// for each of the standard colors and any custom colors.
// Use [NewPalette] to create a new palette.
type Palette struct {

	// the tones for the primary key color
	Primary Tones

	// the tones for the secondary key color
	Secondary Tones

	// the tones for the tertiary key color
	Tertiary Tones

	// the tones for the select key color
	Select Tones

	// the tones for the error key color
	Error Tones

	// the tones for the success key color
	Success Tones

	// the tones for the warn key color
	Warn Tones

	// the tones for the neutral key color
	Neutral Tones

	// the tones for the neutral variant key color
	NeutralVariant Tones

	// an optional map of tones for custom accent key colors
	Custom map[string]Tones
}

// NewPalette creates a new [Palette] from the given key colors.
func NewPalette(key *Key) *Palette {
	p := &Palette{
		Primary:        NewTones(key.Primary),
		Secondary:      NewTones(key.Secondary),
		Tertiary:       NewTones(key.Tertiary),
		Select:         NewTones(key.Select),
		Error:          NewTones(key.Error),
		Success:        NewTones(key.Success),
		Warn:           NewTones(key.Warn),
		Neutral:        NewTones(key.Neutral),
		NeutralVariant: NewTones(key.NeutralVariant),
	}
	for name, c := range key.Custom {
		p.Custom[name] = NewTones(c)
	}
	return p
}
