// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"math"
	"unicode"

	"github.com/cogentcore/typesetting/di"
	"github.com/cogentcore/typesetting/language"
	"github.com/cogentcore/typesetting/shaping"
	"golang.org/x/image/math/fixed"
)

// isSimple returns whether the rune is shaped as one glyph of its own,
// left to right, by itself: runes of the Latin, Greek and Cyrillic scripts
// and of common punctuation and symbols, without combining marks, control,
// format and line break characters, which need the go-text shaper (or its
// line wrapper treats them specially).
func isSimple(r rune) bool {
	switch {
	case r < 0x20:
		return false
	case r < 0x7f:
		return true
	case r < 0xa0: // DEL and C1 controls
		return false
	case r < 0x300:
		return r != 0xad // soft hyphen
	case r < 0x370: // combining diacritical marks
		return false
	case r < 0x483: // Greek, Cyrillic
		return true
	case r < 0x48a: // Cyrillic combining marks
		return false
	case r < 0x530: // Cyrillic, Cyrillic Supplement
		return true
	case r < 0x1e00:
		return false
	case r < 0x2000: // Latin Extended Additional, Greek Extended
		return true
	case r < 0x2010: // spaces and format characters
		return r <= 0x200a && r != 0x2007
	case r < 0x2028:
		return true
	case r < 0x2030: // separators, embedding controls, narrow no-break space
		return false
	case r < 0x205f:
		return true
	case r < 0x2070: // format characters
		return false
	case r < 0x20d0: // superscripts and subscripts, currency
		return true
	case r < 0x2100: // combining marks for symbols
		return false
	case r < 0x2400: // letterlike, number forms, arrows, math operators, technical
		return true
	case r < 0x2460:
		return false
	case r < 0x2600: // enclosed alphanumerics, box drawing, blocks, shapes
		return true
	}
	return false
}

// simpleOutput returns the output of the span of the input when it is simple
// text (see [isSimple]) in one face, with a glyph for each rune with the
// nominal glyph of the face (and the advance of the face for spaces), or
// false when it needs the go-text shaper. The shaper uses this for
// [Shaper.SimpleSpans], when the outputs are measured again after shaping
// (as in shapedjs), so the glyph extents, kerning and ligatures of the
// go-text shaper are not needed.
func (sh *Shaper) simpleOutput(in *shaping.Input) (shaping.Output, bool) {
	rs := in.Text[in.RunStart:in.RunEnd]
	if in.Direction != di.DirectionLTR || len(rs) == 0 {
		return shaping.Output{}, false
	}
	for _, r := range rs {
		if !isSimple(r) {
			return shaping.Output{}, false
		}
	}

	// The face is that of the first rune, which must have the glyphs of the
	// others. The font map resolves them to it as well, unless it is a
	// fallback face for the first rune and the face of the query has some of
	// them, which only changes the glyphs that other renderers draw, as the
	// browser measures and draws the text with its own fonts.
	sh.fontMap.SetScript(language.LookupScript(rs[0]))
	face := sh.fontMap.ResolveFace(rs[0])
	if face == nil {
		return shaping.Output{}, false
	}

	// Each rune is a glyph, whose advance is replaced by the measured one.
	// Empty glyphs keep the advance of the go-text shaper, scaled as it scales
	// it, as their width (see [FixOutputZeros]).
	out := shaping.Output{Size: in.Size, Direction: in.Direction, Face: face, Runes: shaping.Range{Offset: in.RunStart, Count: len(rs)}}
	out.Glyphs = make([]shaping.Glyph, len(rs))
	scale := float64(int32(in.Size.Ceil())<<6) / float64(face.Upem())
	for i, r := range rs {
		gid, ok := face.NominalGlyph(r)
		if !ok {
			return shaping.Output{}, false
		}
		g := &out.Glyphs[i]
		*g = shaping.Glyph{ClusterIndex: in.RunStart + i, RuneCount: 1, GlyphCount: 1, GlyphID: gid}
		if unicode.IsSpace(r) {
			g.XAdvance = fixed.Int26_6(math.Round(float64(face.HorizontalAdvance(gid)) * scale))
		}
	}
	return out, true
}
