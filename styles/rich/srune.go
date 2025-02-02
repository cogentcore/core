// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import (
	"image/color"
	"math"
)

// srune is a uint32 rune value that encodes the font styles.
// There is no attempt to pack these values into the Private Use Areas
// of unicode, because they are never encoded into the unicode directly.
// Because we have the room, we use at least 4 bits = 1 hex F for each
// element of the style property. Size and Color values are added after
// the main style rune element.

// RuneFromStyle returns the style rune that encodes the given style values.
func RuneFromStyle(s *Style) rune {
	return RuneFromDecoration(s.Decoration) | RuneFromSpecial(s.Special) | RuneFromStretch(s.Stretch) | RuneFromWeight(s.Weight) | RuneFromSlant(s.Slant) | RuneFromFamily(s.Family)
}

// RuneToStyle sets all the style values decoded from given rune.
func RuneToStyle(s *Style, r rune) {
	s.Decoration = RuneToDecoration(r)
	s.Special = RuneToSpecial(r)
	s.Stretch = RuneToStretch(r)
	s.Weight = RuneToWeight(r)
	s.Slant = RuneToSlant(r)
	s.Family = RuneToFamily(r)
}

// NumColors returns the number of colors for decoration style encoded
// in given rune.
func NumColors(r rune) int {
	return RuneToDecoration(r).NumColors()
}

// ToRunes returns the rune(s) that encode the given style
// including any additional colors beyond the style and size runes.
func (s *Style) ToRunes() []rune {
	r := RuneFromStyle(s)
	rs := []rune{r, rune(math.Float32bits(s.Size))}
	if s.Decoration.NumColors() == 0 {
		return rs
	}
	if s.Decoration.HasFlag(FillColor) {
		rs = append(rs, ColorToRune(s.FillColor))
	}
	if s.Decoration.HasFlag(StrokeColor) {
		rs = append(rs, ColorToRune(s.StrokeColor))
	}
	if s.Decoration.HasFlag(Background) {
		rs = append(rs, ColorToRune(s.Background))
	}
	return rs
}

// FromRunes sets the Style properties from the given rune encodings
// which must be the proper length including colors.
func (s *Style) FromRunes(rs ...rune) {
	RuneToStyle(s, rs[0])
	s.Size = math.Float32frombits(uint32(rs[1]))
	ci := NStyleRunes
	if s.Decoration.HasFlag(FillColor) {
		s.FillColor = ColorFromRune(rs[ci])
		ci++
	}
	if s.Decoration.HasFlag(StrokeColor) {
		s.StrokeColor = ColorFromRune(rs[ci])
		ci++
	}
	if s.Decoration.HasFlag(Background) {
		s.Background = ColorFromRune(rs[ci])
		ci++
	}
}

// ColorToRune converts given color to a rune uint32 value.
func ColorToRune(c color.Color) rune {
	r, g, b, a := c.RGBA() // uint32
	r8 := r >> 8
	g8 := g >> 8
	b8 := b >> 8
	a8 := a >> 8
	return rune(r8<<24) + rune(g8<<16) + rune(b8<<8) + rune(a8)
}

// ColorFromRune converts given color from a rune uint32 value.
func ColorFromRune(r rune) color.RGBA {
	ru := uint32(r)
	r8 := uint8((ru & 0xFF000000) >> 24)
	g8 := uint8((ru & 0x00FF0000) >> 16)
	b8 := uint8((ru & 0x0000FF00) >> 8)
	a8 := uint8((ru & 0x000000FF))
	return color.RGBA{r8, g8, b8, a8}
}

const (
	DecorationStart = 0
	DecorationMask  = 0x000000FF
	SpecialStart    = 8
	SpecialMask     = 0x00000F00
	StretchStart    = 12
	StretchMask     = 0x0000F000
	WeightStart     = 16
	WeightMask      = 0x000F0000
	SlantStart      = 20
	SlantMask       = 0x00F00000
	FamilyStart     = 24
	FamilyMask      = 0x0F000000
)

// RuneFromDecoration returns the rune bit values for given decoration.
func RuneFromDecoration(d Decorations) rune {
	return rune(d)
}

// RuneToDecoration returns the Decoration bit values from given rune.
func RuneToDecoration(r rune) Decorations {
	return Decorations(uint32(r) & DecorationMask)
}

// RuneFromSpecial returns the rune bit values for given special.
func RuneFromSpecial(d Specials) rune {
	return rune(d + 1<<SpecialStart)
}

// RuneToSpecial returns the Specials value from given rune.
func RuneToSpecial(r rune) Specials {
	return Specials((uint32(r) & SpecialMask) >> SpecialStart)
}

// RuneFromStretch returns the rune bit values for given stretch.
func RuneFromStretch(d Stretch) rune {
	return rune(d + 1<<StretchStart)
}

// RuneToStretch returns the Stretch value from given rune.
func RuneToStretch(r rune) Stretch {
	return Stretch((uint32(r) & StretchMask) >> StretchStart)
}

// RuneFromWeight returns the rune bit values for given weight.
func RuneFromWeight(d Weights) rune {
	return rune(d + 1<<WeightStart)
}

// RuneToWeight returns the Weights value from given rune.
func RuneToWeight(r rune) Weights {
	return Weights((uint32(r) & WeightMask) >> WeightStart)
}

// RuneFromSlant returns the rune bit values for given slant.
func RuneFromSlant(d Slants) rune {
	return rune(d + 1<<SlantStart)
}

// RuneToSlant returns the Slants value from given rune.
func RuneToSlant(r rune) Slants {
	return Slants((uint32(r) & SlantMask) >> SlantStart)
}

// RuneFromFamily returns the rune bit values for given family.
func RuneFromFamily(d Family) rune {
	return rune(d + 1<<FamilyStart)
}

// RuneToFamily returns the Familys value from given rune.
func RuneToFamily(r rune) Family {
	return Family((uint32(r) & FamilyMask) >> FamilyStart)
}
