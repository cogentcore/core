// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
	"fmt"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
)

// WrapLines performs line wrapping and shaping on the given rich text source,
// using the given style information, where the [rich.Style] provides the default
// style information reflecting the contents of the source (e.g., the default family,
// weight, etc), for use in computing the default line height. Paragraphs are extracted
// first using standard newline markers, assumed to coincide with separate spans in the
// source text, and wrapped separately.
func (sh *Shaper) WrapLines(tx rich.Text, defSty *rich.Style, tsty *text.Style, rts *rich.Settings, size math32.Vector2) *Lines {
	if tsty.FontSize.Dots == 0 {
		tsty.FontSize.Dots = 24
	}
	fsz := tsty.FontSize.Dots
	dir := goTextDirection(rich.Default, tsty)

	lht := sh.LineHeight(defSty, tsty, rts)
	lns := &Lines{Source: tx, Color: tsty.Color, SelectionColor: colors.Scheme.Select.Container, HighlightColor: colors.Scheme.Warn.Container, LineHeight: lht}

	lgap := lns.LineHeight - (lns.LineHeight / tsty.LineSpacing) // extra added for spacing
	nlines := int(math32.Floor(size.Y / lns.LineHeight))
	maxSize := int(size.X)
	if dir.IsVertical() {
		nlines = int(math32.Floor(size.X / lns.LineHeight))
		maxSize = int(size.Y)
		// fmt.Println(lht, nlines, maxSize)
	}
	// fmt.Println("lht:", lns.LineHeight, lgap, nlines)
	brk := shaping.WhenNecessary
	if !tsty.WhiteSpace.HasWordWrap() {
		brk = shaping.Never
	} else if tsty.WhiteSpace == text.WrapAlways {
		brk = shaping.Always
	}
	if brk == shaping.Never {
		maxSize = 1000
		nlines = 1
	}
	// fmt.Println(brk, nlines, maxSize)
	cfg := shaping.WrapConfig{
		Direction:                     dir,
		TruncateAfterLines:            nlines,
		TextContinues:                 false, // todo! no effect if TruncateAfterLines is 0
		BreakPolicy:                   brk,   // or Never, Always
		DisableTrailingWhitespaceTrim: tsty.WhiteSpace.KeepWhiteSpace(),
	}
	// from gio:
	// if wc.TruncateAfterLines > 0 {
	// 	if len(params.Truncator) == 0 {
	// 		params.Truncator = "…"
	// 	}
	// 	// We only permit a single run as the truncator, regardless of whether more were generated.
	// 	// Just use the first one.
	// 	wc.Truncator = s.shapeText(params.PxPerEm, params.Locale, []rune(params.Truncator))[0]
	// }
	txt := tx.Join()
	outs := sh.shapeText(tx, tsty, rts, txt)
	// todo: WrapParagraph does NOT handle vertical text! file issue.
	lines, truncate := sh.wrapper.WrapParagraph(cfg, maxSize, txt, shaping.NewSliceIterator(outs))
	lns.Truncated = truncate > 0
	cspi := 0
	cspSt, cspEd := tx.Range(cspi)
	var off math32.Vector2
	for li, lno := range lines {
		// fmt.Println("line:", li, off)
		ln := Line{}
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
			pos = DirectionAdvance(run.Direction, pos, run.Advance)
			ln.Runs = append(ln.Runs, run)
		}
		if li == 0 { // set offset for first line based on max ascent
			if !dir.IsVertical() { // todo: vertical!
				off.Y = math32.FromFixed(maxAsc)
			}
		}
		// go back through and give every run the expanded line-level box
		for ri := range ln.Runs {
			run := &ln.Runs[ri]
			rb := run.BoundsBox()
			if dir.IsVertical() {
				rb.Min.X, rb.Max.X = ln.Bounds.Min.X, ln.Bounds.Max.X
				rb.Min.Y -= 2 // ensure some overlap along direction of rendering adjacent
				rb.Max.Y += 2
			} else {
				rb.Min.Y, rb.Max.Y = ln.Bounds.Min.Y, ln.Bounds.Max.Y
				rb.Min.X -= 2
				rb.Max.Y += 2
			}
			run.MaxBounds = rb
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
		// todo: rest of it
		lns.Lines = append(lns.Lines, ln)
	}
	// fmt.Println(lns.Bounds)
	return lns
}

// WrapSizeEstimate is the size to use for layout during the SizeUp pass,
// for word wrap case, where the sizing actually matters,
// based on trying to fit the given number of characters into the given content size
// with given font height, and ratio of width to height.
// Ratio is used when csz is 0: 1.618 is golden, and smaller numbers to allow
// for narrower, taller text columns.
func WrapSizeEstimate(csz math32.Vector2, nChars int, ratio float32, sty *rich.Style, tsty *text.Style) math32.Vector2 {
	chars := float32(nChars)
	fht := tsty.FontHeight(sty)
	if fht == 0 {
		fht = 16
	}
	area := chars * fht * fht
	if csz.X > 0 && csz.Y > 0 {
		ratio = csz.X / csz.Y
		// fmt.Println(lb, "content size ratio:", ratio)
	}
	// w = ratio * h
	// w^2 + h^2 = a
	// (ratio*h)^2 + h^2 = a
	h := math32.Sqrt(area) / math32.Sqrt(ratio+1)
	w := ratio * h
	if w < csz.X { // must be at least this
		w = csz.X
		h = area / w
		h = max(h, csz.Y)
	}
	sz := math32.Vec2(w, h)
	return sz
}
