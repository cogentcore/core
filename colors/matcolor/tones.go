// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matcolor

import (
	"image"
	"image/color"

	"cogentcore.org/core/colors/cam/hct"
)

// Tones contains cached color values for each tone
// of a seed color. To get a tonal value, use [Tones.Tone].
type Tones struct {

	// the key color used to generate these tones
	Key color.RGBA

	// the cached map of tonal color values
	Tones map[int]color.RGBA
}

// NewTones returns a new set of [Tones]
// for the given color.
func NewTones(c color.RGBA) Tones {
	return Tones{
		Key:   c,
		Tones: map[int]color.RGBA{},
	}
}

// AbsTone returns the color at the given absolute
// tone on a scale of 0 to 100. It uses the cached
// value if it exists, and it caches the value if
// it is not already.
func (t *Tones) AbsTone(tone int) color.RGBA {
	if c, ok := t.Tones[tone]; ok {
		return c
	}
	c := hct.FromColor(t.Key)
	c.SetTone(float32(tone))
	r := c.AsRGBA()
	t.Tones[tone] = r
	return r
}

// AbsToneUniform returns [image.Uniform] of [Tones.AbsTone].
func (t *Tones) AbsToneUniform(tone int) *image.Uniform {
	return image.NewUniform(t.AbsTone(tone))
}

// Tone returns the color at the given tone, relative to the "0" tone
// for the current color scheme (0 for light-themed schemes and 100 for
// dark-themed schemes).
func (t *Tones) Tone(tone int) color.RGBA {
	if SchemeIsDark {
		return t.AbsTone(100 - tone)
	}
	return t.AbsTone(tone)
}

// ToneUniform returns [image.Uniform] of [Tones.Tone].
func (t *Tones) ToneUniform(tone int) *image.Uniform {
	return image.NewUniform(t.Tone(tone))
}
