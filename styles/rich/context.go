// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import (
	"log/slog"

	"cogentcore.org/core/styles/units"
)

// Context holds the global context for rich text styling,
// holding properties that apply to a collection of [rich.Text] elements,
// so it does not need to be redundantly encoded in each such element.
type Context struct {

	// Standard is the standard font size. The Style provides a multiplier
	// on this value.
	Standard units.Value

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

// AddFamily adds a family specifier to the given font string,
// handling the comma properly.
func AddFamily(s, fam string) string {
	if s == "" {
		return fam
	}
	return s + ", " + fam
}

// Family returns the font family specified by the given [Family] enum.
func (ctx *Context) Family(fam Family) string {
	switch fam {
	case SansSerif:
		return AddFamily(ctx.SansSerif, "sans-serif")
	case Serif:
		return AddFamily(ctx.Serif, "serif")
	case Monospace:
		return AddFamily(ctx.Monospace, "monospace")
	case Cursive:
		return AddFamily(ctx.Cursive, "cursive")
	case Fantasy:
		return AddFamily(ctx.Fantasy, "fantasy")
	case Maths:
		return AddFamily(ctx.Math, "math")
	case Emoji:
		return AddFamily(ctx.Emoji, "emoji")
	case Fangsong:
		return AddFamily(ctx.Fangsong, "fangsong")
	case Custom:
		return ctx.Custom
	}
	return "sans-serif"
}

// ToDots runs ToDots on unit values, to compile down to raw Dots pixels.
func (ctx *Context) ToDots(uc *units.Context) {
	if ctx.Standard.Unit == units.UnitEm || ctx.Standard.Unit == units.UnitEx || ctx.Standard.Unit == units.UnitCh {
		slog.Error("girl/styles.Font.Size was set to Em, Ex, or Ch; that is recursive and unstable!", "unit", ctx.Standard.Unit)
		ctx.Standard.Dp(16)
	}
	ctx.Standard.ToDots(uc)
}

// SizeDots returns the font size based on given multiplier * Standard.Dots
func (ctx *Context) SizeDots(multiplier float32) float32 {
	return ctx.Standard.Dots * multiplier
}
