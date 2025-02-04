// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import (
	"strings"

	"github.com/go-text/typesetting/language"
)

// Settings holds the global settings for rich text styling,
// including language, script, and preferred font faces for
// each category of font.
type Settings struct {

	// Language is the preferred language used for rendering text.
	Language language.Language

	// Script is the specific writing system used for rendering text.
	Script language.Script

	// SansSerif is a font without serifs, where glyphs have plain stroke endings,
	// without ornamentation. Example sans-serif fonts include Arial, Helvetica,
	// Open Sans, Fira Sans, Lucida Sans, Lucida Sans Unicode, Trebuchet MS,
	// Liberation Sans, and Nimbus Sans L.
	// This can be a list of comma-separated names, tried in order.
	// "sans-serif" will be added automatically as a final backup.
	SansSerif string

	// Serif is a small line or stroke attached to the end of a larger stroke
	// in a letter. In serif fonts, glyphs have finishing strokes, flared or
	// tapering ends. Examples include Times New Roman, Lucida Bright,
	// Lucida Fax, Palatino, Palatino Linotype, Palladio, and URW Palladio.
	// This can be a list of comma-separated names, tried in order.
	// "serif" will be added automatically as a final backup.
	Serif string

	// Monospace fonts have all glyphs with he same fixed width.
	// Example monospace fonts include Fira Mono, DejaVu Sans Mono,
	// Menlo, Consolas, Liberation Mono, Monaco, and Lucida Console.
	// This can be a list of comma-separated names. serif will be added
	// automatically as a final backup.
	// This can be a list of comma-separated names, tried in order.
	// "monospace" will be added automatically as a final backup.
	Monospace string

	// Cursive glyphs generally have either joining strokes or other cursive
	// characteristics beyond those of italic typefaces. The glyphs are partially
	// or completely connected, and the result looks more like handwritten pen or
	// brush writing than printed letter work. Example cursive fonts include
	// Brush Script MT, Brush Script Std, Lucida Calligraphy, Lucida Handwriting,
	// and Apple Chancery.
	// This can be a list of comma-separated names, tried in order.
	// "cursive" will be added automatically as a final backup.
	Cursive string

	// Fantasy fonts are primarily decorative fonts that contain playful
	// representations of characters. Example fantasy fonts include Papyrus,
	// Herculanum, Party LET, Curlz MT, and Harrington.
	// This can be a list of comma-separated names, tried in order.
	// "fantasy" will be added automatically as a final backup.
	Fantasy string

	//	Math fonts are for displaying mathematical expressions, for example
	// superscript and subscript, brackets that cross several lines, nesting
	// expressions, and double-struck glyphs with distinct meanings.
	// This can be a list of comma-separated names, tried in order.
	// "math" will be added automatically as a final backup.
	Math string

	// Emoji fonts are specifically designed to render emoji.
	// This can be a list of comma-separated names, tried in order.
	// "emoji" will be added automatically as a final backup.
	Emoji string

	// Fangsong are a particular style of Chinese characters that are between
	// serif-style Song and cursive-style Kai forms. This style is often used
	// for government documents.
	// This can be a list of comma-separated names, tried in order.
	// "fangsong" will be added automatically as a final backup.
	Fangsong string

	// Custom is a custom font name.
	Custom string
}

func (rts *Settings) Defaults() {
	rts.Language = "en"
	rts.Script = language.Latin
	rts.SansSerif = "Arial"
	rts.Serif = "Times New Roman"
}

// AddFamily adds a family specifier to the given font string,
// handling the comma properly.
func AddFamily(rts, fam string) string {
	if rts == "" {
		return fam
	}
	return rts + ", " + fam
}

// FamiliesToList returns a list of the families, split by comma and space removed.
func FamiliesToList(fam string) []string {
	fs := strings.Split(fam, ",")
	os := make([]string, 0, len(fs))
	for _, f := range fs {
		rts := strings.TrimSpace(f)
		if rts == "" {
			continue
		}
		os = append(os, rts)
	}
	return os
}

// Family returns the font family specified by the given [Family] enum.
func (rts *Settings) Family(fam Family) string {
	switch fam {
	case SansSerif:
		return AddFamily(rts.SansSerif, "sans-serif")
	case Serif:
		return AddFamily(rts.Serif, "serif")
	case Monospace:
		return AddFamily(rts.Monospace, "monospace")
	case Cursive:
		return AddFamily(rts.Cursive, "cursive")
	case Fantasy:
		return AddFamily(rts.Fantasy, "fantasy")
	case Maths:
		return AddFamily(rts.Math, "math")
	case Emoji:
		return AddFamily(rts.Emoji, "emoji")
	case Fangsong:
		return AddFamily(rts.Fangsong, "fangsong")
	case Custom:
		return rts.Custom
	}
	return "sans-serif"
}
