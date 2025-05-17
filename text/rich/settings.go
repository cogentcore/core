// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import (
	"strings"

	"github.com/go-text/typesetting/language"
)

func init() {
	DefaultSettings.Defaults()
}

// DefaultSettings contains the default global text settings.
// This will be updated from rich.DefaultSettings.
var DefaultSettings Settings

// FontName is a special string that provides a font chooser.
// It is aliased to [core.FontName] as well.
type FontName string

// Settings holds the global settings for rich text styling,
// including language, script, and preferred font faces for
// each category of font.
type Settings struct {

	// Language is the preferred language used for rendering text.
	Language language.Language

	// Script is the specific writing system used for rendering text.
	// todo: no idea how to set this based on language or anything else.
	Script language.Script `display:"-"`

	// SansSerif is a font without serifs, where glyphs have plain stroke endings,
	// without ornamentation. Example sans-serif fonts include Arial, Helvetica,
	// Noto Sans, Open Sans, Fira Sans, Lucida Sans, Lucida Sans Unicode, Trebuchet MS,
	// Liberation Sans, Nimbus Sans L, Roboto.
	// This can be a list of comma-separated names, tried in order.
	// "sans-serif" will be added automatically as a final backup.
	SansSerif FontName `default:"Noto Sans"`

	// Serif is a small line or stroke attached to the end of a larger stroke
	// in a letter. In serif fonts, glyphs have finishing strokes, flared or
	// tapering ends. Examples include Times New Roman, Lucida Bright,
	// Lucida Fax, Palatino, Palatino Linotype, Palladio, and URW Palladio.
	// This can be a list of comma-separated names, tried in order.
	// "serif" will be added automatically as a final backup.
	Serif FontName

	// Monospace fonts have all glyphs with he same fixed width.
	// Example monospace fonts include Roboto Mono, Fira Mono, DejaVu Sans Mono,
	// Menlo, Consolas, Liberation Mono, Monaco, and Lucida Console.
	// This can be a list of comma-separated names. serif will be added
	// automatically as a final backup.
	// This can be a list of comma-separated names, tried in order.
	// "monospace" will be added automatically as a final backup.
	Monospace FontName `default:"Roboto Mono"`

	// Cursive glyphs generally have either joining strokes or other cursive
	// characteristics beyond those of italic typefaces. The glyphs are partially
	// or completely connected, and the result looks more like handwritten pen or
	// brush writing than printed letter work. Example cursive fonts include
	// Brush Script MT, Brush Script Std, Lucida Calligraphy, Lucida Handwriting,
	// and Apple Chancery.
	// This can be a list of comma-separated names, tried in order.
	// "cursive" will be added automatically as a final backup.
	Cursive FontName

	// Fantasy fonts are primarily decorative fonts that contain playful
	// representations of characters. Example fantasy fonts include Papyrus,
	// Herculanum, Party LET, Curlz MT, and Harrington.
	// This can be a list of comma-separated names, tried in order.
	// "fantasy" will be added automatically as a final backup.
	Fantasy FontName

	// Math fonts are for displaying mathematical expressions, for example
	// superscript and subscript, brackets that cross several lines, nesting
	// expressions, and double-struck glyphs with distinct meanings.
	// This can be a list of comma-separated names, tried in order.
	// "math" will be added automatically as a final backup.
	Math FontName

	// Emoji fonts are specifically designed to render emoji.
	// This can be a list of comma-separated names, tried in order.
	// "emoji" will be added automatically as a final backup.
	Emoji FontName

	// Fangsong are a particular style of Chinese characters that are between
	// serif-style Song and cursive-style Kai forms. This style is often used
	// for government documents.
	// This can be a list of comma-separated names, tried in order.
	// "fangsong" will be added automatically as a final backup.
	Fangsong FontName
}

func (rts *Settings) Defaults() {
	rts.Language = language.DefaultLanguage()
	rts.SansSerif = "Noto Sans"
	rts.Monospace = "Roboto Mono"
}

// AddFamily adds a family specifier to the given font string,
// handling the comma properly.
func AddFamily(rts FontName, fam string) string {
	if rts == "" {
		return fam
	}
	s := string(rts)
	// if strings.Contains(s, " ") { // no! this is bad
	// 	s = `"` + s + `"`
	// }
	return s + ", " + fam
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
		return AddFamily(rts.SansSerif, `-apple-system, BlinkMacSystemFont, "Segoe UI", Oxygen, Ubuntu, Cantarell, "Open Sans", "Helvetica Neue", sans-serif, emoji`)
	case Serif:
		return AddFamily(rts.Serif, `serif, emoji`)
	case Monospace:
		return AddFamily(rts.Monospace, `monospace, emoji`)
	case Cursive:
		return AddFamily(rts.Cursive, `cursive, emoji`)
	case Fantasy:
		return AddFamily(rts.Fantasy, `fantasy, emoji`)
	case Math:
		return AddFamily(rts.Math, "math")
	case Emoji:
		return AddFamily(rts.Emoji, "emoji")
	case Fangsong:
		return AddFamily(rts.Fangsong, "fangsong")
	}
	return "sans-serif"
}
