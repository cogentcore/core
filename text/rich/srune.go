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
	return RuneFromDecoration(s.Decoration) | RuneFromSpecial(s.Special) | RuneFromStretch(s.Stretch) | RuneFromWeight(s.Weight) | RuneFromSlant(s.Slant) | RuneFromFamily(s.Family) | RuneFromDirection(s.Direction)
}

// RuneToStyle sets all the style values decoded from given rune.
func RuneToStyle(s *Style, r rune) {
	s.Decoration = RuneToDecoration(r)
	s.Special = RuneToSpecial(r)
	s.Stretch = RuneToStretch(r)
	s.Weight = RuneToWeight(r)
	s.Slant = RuneToSlant(r)
	s.Family = RuneToFamily(r)
	s.Direction = RuneToDirection(r)
}

// SpanLen returns the length of the starting style runes and
// following content runes for given slice of span runes.
// Does not need to decode full style, so is very efficient.
func SpanLen(s []rune) (sn int, rn int) {
	r0 := s[0]
	nc := RuneToDecoration(r0).NumColors()
	sn = 2 + nc // style + size + nc
	isLink := RuneToSpecial(r0) == Link
	if !isLink {
		rn = max(0, len(s)-sn)
		return
	}
	ln := int(s[sn]) // link len
	sn += ln + 1
	rn = max(0, len(s)-sn)
	return
}

// FromRunes sets the Style properties from the given rune encodings
// which must be the proper length including colors. Any remaining
// runes after the style runes are returned: this is the source string.
func (s *Style) FromRunes(rs []rune) []rune {
	RuneToStyle(s, rs[0])
	s.Size = math.Float32frombits(uint32(rs[1]))
	ci := 2
	if s.Decoration.HasFlag(fillColor) {
		s.fillColor = ColorFromRune(rs[ci])
		ci++
	}
	if s.Decoration.HasFlag(strokeColor) {
		s.strokeColor = ColorFromRune(rs[ci])
		ci++
	}
	if s.Decoration.HasFlag(background) {
		s.background = ColorFromRune(rs[ci])
		ci++
	}
	if s.Special == Link {
		ln := int(rs[ci])
		ci++
		s.URL = string(rs[ci : ci+ln])
		ci += ln
	}
	if ci < len(rs) {
		return rs[ci:]
	}
	return nil
}

// ToRunes returns the rune(s) that encode the given style
// including any additional colors beyond the style and size runes,
// and the URL for a link.
func (s *Style) ToRunes() []rune {
	r := RuneFromStyle(s)
	rs := []rune{r, rune(math.Float32bits(s.Size))}
	if s.Decoration.NumColors() == 0 {
		return rs
	}
	if s.Decoration.HasFlag(fillColor) {
		rs = append(rs, ColorToRune(s.fillColor))
	}
	if s.Decoration.HasFlag(strokeColor) {
		rs = append(rs, ColorToRune(s.strokeColor))
	}
	if s.Decoration.HasFlag(background) {
		rs = append(rs, ColorToRune(s.background))
	}
	if s.Special == Link {
		rs = append(rs, rune(len(s.URL)))
		rs = append(rs, []rune(s.URL)...)
	}
	return rs
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
	DecorationMask  = 0x000007FF // 11 bits reserved for deco
	SlantStart      = 11
	SlantMask       = 0x00000800 // 1 bit for slant
	SpecialStart    = 12
	SpecialMask     = 0x0000F000
	StretchStart    = 16
	StretchMask     = 0x000F0000
	WeightStart     = 20
	WeightMask      = 0x00F00000
	FamilyStart     = 24
	FamilyMask      = 0x0F000000
	DirectionStart  = 28
	DirectionMask   = 0xF0000000
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
	return rune(d << SpecialStart)
}

// RuneToSpecial returns the Specials value from given rune.
func RuneToSpecial(r rune) Specials {
	return Specials((uint32(r) & SpecialMask) >> SpecialStart)
}

// RuneFromStretch returns the rune bit values for given stretch.
func RuneFromStretch(d Stretch) rune {
	return rune(d << StretchStart)
}

// RuneToStretch returns the Stretch value from given rune.
func RuneToStretch(r rune) Stretch {
	return Stretch((uint32(r) & StretchMask) >> StretchStart)
}

// RuneFromWeight returns the rune bit values for given weight.
func RuneFromWeight(d Weights) rune {
	return rune(d << WeightStart)
}

// RuneToWeight returns the Weights value from given rune.
func RuneToWeight(r rune) Weights {
	return Weights((uint32(r) & WeightMask) >> WeightStart)
}

// RuneFromSlant returns the rune bit values for given slant.
func RuneFromSlant(d Slants) rune {
	return rune(d << SlantStart)
}

// RuneToSlant returns the Slants value from given rune.
func RuneToSlant(r rune) Slants {
	return Slants((uint32(r) & SlantMask) >> SlantStart)
}

// RuneFromFamily returns the rune bit values for given family.
func RuneFromFamily(d Family) rune {
	return rune(d << FamilyStart)
}

// RuneToFamily returns the Familys value from given rune.
func RuneToFamily(r rune) Family {
	return Family((uint32(r) & FamilyMask) >> FamilyStart)
}

// RuneFromDirection returns the rune bit values for given direction.
func RuneFromDirection(d Directions) rune {
	return rune(d << DirectionStart)
}

// RuneToDirection returns the Directions value from given rune.
func RuneToDirection(r rune) Directions {
	return Directions((uint32(r) & DirectionMask) >> DirectionStart)
}
