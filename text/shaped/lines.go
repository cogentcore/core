// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
	"fmt"
	"image"
	"image/color"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
	"golang.org/x/image/math/fixed"
)

// todo: split source at para boundaries and use wrap para on those.

// Lines is a list of Lines of shaped text, with an overall bounding
// box and position for the entire collection. This is the renderable
// unit of text, although it is not a [render.Item] because it lacks
// a position, and it can potentially be re-used in different positions.
type Lines struct {

	// Source is the original input source that generated this set of lines.
	// Each Line has its own set of spans that describes the Line contents.
	Source rich.Text

	// Lines are the shaped lines.
	Lines []Line

	// Offset is an optional offset to add to the position given when rendering.
	Offset math32.Vector2

	// Bounds is the bounding box for the entire set of rendered text,
	// relative to a rendering Position (and excluding any contribution
	// of Offset). Use Size() method to get the size and ToRect() to get
	// an [image.Rectangle].
	Bounds math32.Box2

	// FontSize is the [rich.Context] StandardSize from the Context used
	// at the time of shaping. Actual lines can be larger depending on font
	// styling parameters.
	FontSize float32

	// LineHeight is the line height used at the time of shaping.
	LineHeight float32

	// Truncated indicates whether any lines were truncated.
	Truncated bool

	// Direction is the default text rendering direction from the Context.
	Direction rich.Directions

	// Links holds any hyperlinks within shaped text.
	Links []rich.Hyperlink

	// Color is the default fill color to use for inking text.
	Color color.Color

	// SelectionColor is the color to use for rendering selected regions.
	SelectionColor image.Image

	// HighlightColor is the color to use for rendering highlighted regions.
	HighlightColor image.Image
}

// Line is one line of shaped text, containing multiple Runs.
// This is not an independent render target: see [Lines] (can always
// use one Line per Lines as needed).
type Line struct {

	// Source is the input source corresponding to the line contents,
	// derived from the original Lines Source. The style information for
	// each Run is embedded here.
	Source rich.Text

	// SourceRange is the range of runes in the original [Lines.Source] that
	// are represented in this line.
	SourceRange textpos.Range

	// Runs are the shaped [Run] elements.
	Runs []Run

	// Offset specifies the relative offset from the Lines Position
	// determining where to render the line in a target render image.
	// This is the baseline position (not the upper left: see Bounds for that).
	Offset math32.Vector2

	// Bounds is the bounding box for the Line of rendered text,
	// relative to the baseline rendering position (excluding any contribution
	// of Offset). This is centered at the baseline and the upper left
	// typically has a negative Y. Use Size() method to get the size
	// and ToRect() to get an [image.Rectangle]. This is based on the output
	// LineBounds, not the actual GlyphBounds.
	Bounds math32.Box2

	// Selections specifies region(s) of runes within this line that are selected,
	// and will be rendered with the [Lines.SelectionColor] background,
	// replacing any other background color that might have been specified.
	Selections []textpos.Range

	// Highlights specifies region(s) of runes within this line that are highlighted,
	// and will be rendered with the [Lines.HighlightColor] background,
	// replacing any other background color that might have been specified.
	Highlights []textpos.Range
}

func (ln *Line) String() string {
	return ln.Source.String() + fmt.Sprintf(" runs: %d\n", len(ln.Runs))
}

func (ls *Lines) String() string {
	str := ""
	for li := range ls.Lines {
		ln := &ls.Lines[li]
		str += fmt.Sprintf("#### Line: %d\n", li)
		str += ln.String()
	}
	return str
}

// StartAtBaseline removes the offset from the first line that causes
// the lines to be rendered starting at the upper left corner, so they
// will instead be rendered starting at the baseline position.
func (ls *Lines) StartAtBaseline() {
	if len(ls.Lines) == 0 {
		return
	}
	ls.Lines[0].Offset = math32.Vector2{}
}

// SetGlyphXAdvance sets the x advance on all glyphs to given value:
// for monospaced case.
func (ls *Lines) SetGlyphXAdvance(adv fixed.Int26_6) {
	for li := range ls.Lines {
		ln := &ls.Lines[li]
		for ri := range ln.Runs {
			rn := ln.Runs[ri]
			rn.SetGlyphXAdvance(adv)
		}
	}
}

// GetLinks gets the links for these lines, which are cached in Links.
func (ls *Lines) GetLinks() []rich.Hyperlink {
	if ls.Links != nil {
		return ls.Links
	}
	ls.Links = ls.Source.GetLinks()
	return ls.Links
}

// AlignXFactor aligns the lines along X axis according to alignment factor,
// as a proportion of size difference to add to offset (0.5 = center,
// 1 = right)
func (ls *Lines) AlignXFactor(fact float32) {
	wd := ls.Bounds.Size().X
	for li := range ls.Lines {
		ln := &ls.Lines[li]
		lwd := ln.Bounds.Size().X
		if lwd < wd {
			ln.Offset.X += fact * (wd - lwd)
		}
	}
}

// AlignX aligns the lines along X axis according to text style.
func (ls *Lines) AlignX(tsty *text.Style) {
	fact, _ := tsty.AlignFactors()
	if fact > 0 {
		ls.AlignXFactor(fact)
	}
}

// Clone returns a Clone copy of the Lines, with new Lines elements
// that still point to the same underlying Runs.
func (ls *Lines) Clone() *Lines {
	nls := &Lines{}
	*nls = *ls
	nln := len(ls.Lines)
	if nln > 0 {
		nln := make([]Line, nln)
		for i := range ls.Lines {
			nln[i] = ls.Lines[i]
		}
		nls.Lines = nln
	}
	return nls
}

// UpdateStyle updates the Decoration, Fill and Stroke colors from the given
// rich.Text Styles for each line and given text style.
// This rich.Text must match the content of the shaped one, and differ only
// in these non-layout styles.
func (ls *Lines) UpdateStyle(tx rich.Text, tsty *text.Style) {
	ls.Source = tx
	ls.Color = tsty.Color
	for i := range ls.Lines {
		ln := &ls.Lines[i]
		ln.UpdateStyle(tx, tsty)
	}
}

// UpdateStyle updates the Decoration, Fill and Stroke colors from the current
// rich.Text Style for each run  and given text style.
// This rich.Text must match the content of the shaped one, and differ only
// in these non-layout styles.
func (ln *Line) UpdateStyle(tx rich.Text, tsty *text.Style) {
	ln.Source = tx
	for ri, rn := range ln.Runs {
		fs := ln.RunStyle(ln.Source, ri)
		rb := rn.AsBase()
		rb.SetFromStyle(fs, tsty)
	}
}

// RunStyle returns the rich text style for given run index.
func (ln *Line) RunStyle(tx rich.Text, ri int) *rich.Style {
	rn := ln.Runs[ri]
	rs := rn.Runes().Start
	si, _, _ := tx.Index(rs)
	sty, _ := tx.Span(si)
	return sty
}
