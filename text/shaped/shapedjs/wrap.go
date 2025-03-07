// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"fmt"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
	"github.com/go-text/typesetting/di"
	"golang.org/x/image/math/fixed"
)

// WrapLines performs line wrapping and shaping on the given rich text source,
// using the given style information, where the [rich.Style] provides the default
// style information reflecting the contents of the source (e.g., the default family,
// weight, etc), for use in computing the default line height. Paragraphs are extracted
// first using standard newline markers, assumed to coincide with separate spans in the
// source text, and wrapped separately. For horizontal text, the Lines will render with
// a position offset at the upper left corner of the overall bounding box of the text.
func (sh *Shaper) WrapLines(tx rich.Text, defSty *rich.Style, tsty *text.Style, rts *rich.Settings, size math32.Vector2) *shaped.Lines {
	sh.Lock()
	defer sh.Unlock()

	if tsty.FontSize.Dots == 0 {
		tsty.FontSize.Dots = 16
	}
	fsz := tsty.FontSize.Dots
	dir := goTextDirection(rich.Default, tsty)

	lht := sh.lineHeight(defSty, tsty, rts)
	lns := &shaped.Lines{Source: tx, Color: tsty.Color, SelectionColor: tsty.SelectColor, HighlightColor: tsty.HighlightColor, LineHeight: lht}

	lgap := lns.LineHeight - (lns.LineHeight / tsty.LineSpacing) // extra added for spacing
	nlines := int(math32.Floor(size.Y/lns.LineHeight)) * 2
	maxSize := int(size.X)
	if dir.IsVertical() {
		nlines = int(math32.Floor(size.X / lns.LineHeight))
		maxSize = int(size.Y)
		// fmt.Println(lht, nlines, maxSize)
	}
	// fmt.Println("lht:", lns.LineHeight, lgap, nlines)
	brk := true
	if !tsty.WhiteSpace.HasWordWrap() {
		brk = false
	}
	if !brk {
		maxSize = 100000
		nlines = 1
	}
	txt := tx.Join()
	outs := sh.shapeText(tx, tsty, rts, txt)
	lines, truncate := sh.WrapParagraph(brk, nlines, maxSize, txt, outs)
	lns.Truncated = truncate > 0
	cspi := 0
	cspSt, cspEd := tx.Range(cspi)
	var off math32.Vector2
	for li, lno := range lines {
		// fmt.Println("line:", li, off)
		ln := shaped.Line{}
		var lsp rich.Text
		var pos fixed.Point26_6
		setFirst := false
		var maxAsc fixed.Int26_6
		for oi := range lno {
			out := &lno[oi]
			if !dir.IsVertical() { // todo: vertical
				maxAsc = max(out.LineBounds.Ascent, maxAsc)
			}
			run := Run{Output: *out}
			rns := run.Runes()
			if !setFirst {
				ln.SourceRange.Start = rns.Start
				setFirst = true
			}
			ln.SourceRange.End = rns.End
			for rns.Start >= cspEd {
				cspi++
				cspSt, cspEd = tx.Range(cspi)
			}
			sty, cr := rich.NewStyleFromRunes(tx[cspi])
			if lns.FontSize == 0 {
				lns.FontSize = sty.Size * fsz
			}
			nsp := sty.ToRunes()
			coff := rns.Start - cspSt
			cend := coff + rns.Len()
			crsz := len(cr)
			if coff >= crsz || cend > crsz {
				fmt.Println("out of bounds:", string(cr), crsz, coff, cend)
				cend = min(crsz, cend)
				coff = min(crsz, coff)
			}
			if cend-coff == 0 {
				continue
			}
			nr := cr[coff:cend] // note: not a copy!
			nsp = append(nsp, nr...)
			lsp = append(lsp, nsp)
			// fmt.Println(sty, string(nr))
			if cend > (cspEd - cspSt) { // shouldn't happen, to combine multiple original spans
				fmt.Println("combined original span:", cend, cspEd-cspSt, cspi, string(cr), "prev:", string(nr), "next:", string(cr[cend:]))
			}
			run.Decoration = sty.Decoration
			if sty.Decoration.HasFlag(rich.FillColor) {
				run.FillColor = colors.Uniform(sty.FillColor)
			}
			if sty.Decoration.HasFlag(rich.StrokeColor) {
				run.StrokeColor = colors.Uniform(sty.StrokeColor)
			}
			if sty.Decoration.HasFlag(rich.Background) {
				run.Background = colors.Uniform(sty.Background)
			}
			bb := math32.B2FromFixed(run.Bounds().Add(pos))
			// fmt.Println(bb.Size().Y, lht)
			ln.Bounds.ExpandByBox(bb)
			pos = DirectionAdvance(run.Direction, pos, run.Output.Advance)
			ln.Runs = append(ln.Runs, &run)
		}
		if li == 0 { // set offset for first line based on max ascent
			if !dir.IsVertical() { // todo: vertical!
				off.Y = math32.FromFixed(maxAsc)
			}
		}
		// go back through and give every run the expanded line-level box
		for ri := range ln.Runs {
			run := ln.Runs[ri]
			rb := run.LineBounds()
			if dir.IsVertical() {
				rb.Min.X, rb.Max.X = ln.Bounds.Min.X, ln.Bounds.Max.X
				rb.Min.Y -= 2 // ensure some overlap along direction of rendering adjacent
				rb.Max.Y += 2
			} else {
				rb.Min.Y, rb.Max.Y = ln.Bounds.Min.Y, ln.Bounds.Max.Y
				rb.Min.X -= 2
				rb.Max.Y += 2
			}
			run.AsBase().MaxBounds = rb
		}
		ln.Source = lsp
		// offset has prior line's size built into it, but we need to also accommodate
		// any extra size in _our_ line beyond what is expected.
		ourOff := off
		// fmt.Println(ln.Bounds)
		// advance offset:
		if dir.IsVertical() {
			lwd := ln.Bounds.Size().X
			extra := max(lwd-lns.LineHeight, 0)
			if dir.Progression() == di.FromTopLeft {
				// fmt.Println("ftl lwd:", lwd, off.X)
				off.X += lwd + lgap
				ourOff.X += extra
			} else {
				// fmt.Println("!ftl lwd:", lwd, off.X)
				off.X -= lwd + lgap
				ourOff.X -= extra
			}
		} else { // always top-down, no progression issues
			lht := ln.Bounds.Size().Y
			extra := max(lht-lns.LineHeight, 0)
			// fmt.Println("extra:", extra)
			off.Y += lht + lgap
			if lht < lns.LineHeight {
				ln.Bounds.Max.Y += lns.LineHeight - lht
			}
			ourOff.Y += extra
		}
		ln.Offset = ourOff
		lns.Bounds.ExpandByBox(ln.Bounds.Translate(ln.Offset))
		lns.Lines = append(lns.Lines, ln)
	}
	if lns.Bounds.Size().Y < lht {
		lns.Bounds.Max.Y = lns.Bounds.Min.Y + lht
	}
	// fmt.Println(lns.Bounds)
	lns.AlignX(tsty)
	return lns
}
