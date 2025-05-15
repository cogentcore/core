// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
)

// FontSize returns the font shape sizing information for given font and text style,
// using given rune (often the letter 'm'). The GlyphBounds field of the [Run] result
// has the font ascent and descent information, and the BoundsBox() method returns a full
// bounding box for the given font, centered at the baseline.
// This is called under a mutex lock, so it is safe for parallel use.
func (sh *Shaper) FontSize(r rune, sty *rich.Style, tsty *text.Style, rts *rich.Settings) shaped.Run {
	sh.Lock()
	defer sh.Unlock()
	return sh.fontSize(r, sty, tsty, rts)
}

// LineHeight returns the line height for given font and text style.
// For vertical text directions, this is actually the line width.
// It includes the [text.Style] LineHeight multiplier on the natural
// font-derived line height, which is not generally the same as the font size.
// This is called under a mutex lock, so it is safe for parallel use.
func (sh *Shaper) LineHeight(sty *rich.Style, tsty *text.Style, rts *rich.Settings) float32 {
	sh.Lock()
	defer sh.Unlock()
	return sh.lineHeight(sty, tsty, rts)
}

// fontSize returns the font shape sizing information for given font and text style,
// using given rune (often the letter 'm'). The GlyphBounds field of the [Run] result
// has the font ascent and descent information, and the BoundsBox() method returns a full
// bounding box for the given font, centered at the baseline.
func (sh *Shaper) fontSize(r rune, sty *rich.Style, tsty *text.Style, rts *rich.Settings) shaped.Run {
	tx := rich.NewText(sty, []rune{r})
	return sh.shapeAdjust(tx, tsty, rts, []rune{r})[0]
}

// lineHeight returns the line height for given font and text style.
// For vertical text directions, this is actually the line width.
// It includes the [text.Style] LineHeight multiplier on the natural
// font-derived line height, which is not generally the same as the font size.
func (sh *Shaper) lineHeight(sty *rich.Style, tsty *text.Style, rts *rich.Settings) float32 {
	run := sh.fontSize('M', sty, tsty, rts)
	bb := run.LineBounds()
	dir := shaped.GoTextDirection(rich.Default, tsty)
	if dir.IsVertical() {
		return math32.Round(tsty.LineHeight * bb.Size().X)
	}
	lht := math32.Round(tsty.LineHeight * bb.Size().Y)
	// fmt.Println("lht:", tsty.LineHeight, bb.Size().Y, lht)
	return lht
}
