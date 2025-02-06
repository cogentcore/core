// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// FontSize returns the font shape sizing information for given font and text style,
// using given rune (often the letter 'm'). The GlyphBounds field of the [Run] result
// has the font ascent and descent information, and the BoundsBox() method returns a full
// bounding box for the given font, centered at the baseline.
func (sh *Shaper) FontSize(r rune, sty *rich.Style, tsty *text.Style, rts *rich.Settings) *Run {
	tx := rich.NewText(sty, []rune{r})
	out := sh.shapeText(tx, tsty, rts, []rune{r})
	return &Run{Output: out[0]}
}

// LineHeight returns the line height for given font and text style.
// For vertical text directions, this is actually the line width.
// It includes the [text.Style] LineSpacing multiplier on the natural
// font-derived line height, which is not generally the same as the font size.
func (sh *Shaper) LineHeight(sty *rich.Style, tsty *text.Style, rts *rich.Settings) float32 {
	run := sh.FontSize('M', sty, tsty, rts)
	bb := run.BoundsBox()
	dir := goTextDirection(rich.Default, tsty)
	if dir.IsVertical() {
		return math32.Round(tsty.LineSpacing * bb.Size().X)
	}
	lht := math32.Round(tsty.LineSpacing * bb.Size().Y)
	// fmt.Println("lht:", tsty.LineSpacing, bb.Size().Y, lht)
	return lht
}
