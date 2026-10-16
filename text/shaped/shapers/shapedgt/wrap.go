// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"fmt"
	"math"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
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
// source text, and wrapped separately. For horizontal text, the Lines will render with
// a position offset at the upper left corner of the overall bounding box of the text.
// This is called under a mutex lock, so it is safe for parallel use.
func (sh *Shaper) WrapLines(tx rich.Text, defSty *rich.Style, tsty *text.Style, size math32.Vector2) *shaped.Lines {
	sh.Lock()
	defer sh.Unlock()
	if tsty.FontSize.Dots == 0 {
		tsty.FontSize.Dots = 16
	}

	return sh.WrapLinesBoundsCached(tx, defSty, tsty, size, nil)
}

// This should already have the mutex lock, and is used by shapedjs but is
// not an end-user call. Returns new lines and number of truncations.
func (sh *Shaper) WrapLinesOutput(outs []shaping.Output, txt []rune, tx rich.Text, defSty *rich.Style, tsty *text.Style, size math32.Vector2) ([]shaping.Line, int) {
	cfg, maxSize := wrapConfig(defSty, tsty, size)
	if lines, ok := singleLine(outs, txt, cfg, maxSize, !cfg.DisableTrailingWhitespaceTrim); ok {
		return lines, 0
	}
	return sh.wrapParagraph(cfg, maxSize, txt, outs)
}

// singleLine returns the outputs as one line when they fit on one line, as
// the line wrapper returns them, with the trailing whitespace of more than one
// run trimmed if trim, or false when the line wrapper is needed. The wrapper
// returns a single run that fits as it is, when the text has no mandatory line
// break before its end. For more than one run, the wrapper is needed when the
// text has a line break character, may break at every opportunity, is vertical
// or has runs in more than one direction, or does not fit.
func singleLine(outs []shaping.Output, txt []rune, cfg shaping.WrapConfig, maxSize int, trim bool) ([]shaping.Line, bool) {
	if len(outs) == 0 {
		return nil, false
	}
	maxWidth := fixed.I(min(maxSize, math.MaxInt32>>6))

	// A single run is checked as the wrapper checks it, in any direction.
	if len(outs) == 1 {
		if outs[0].Advance > maxWidth || hasMandatoryBreak(txt) {
			return nil, false
		}
		return []shaping.Line{outs}, true
	}

	// More runs must all be in the horizontal direction of the paragraph.
	dir := cfg.Direction
	if cfg.BreakPolicy == shaping.Always || dir.IsVertical() || dir.Progression() != di.FromTopLeft {
		return nil, false
	}
	var adv fixed.Int26_6
	for i := range outs {
		if outs[i].Direction != dir {
			return nil, false
		}
		adv += outs[i].Advance
	}
	if adv > maxWidth {
		return nil, false
	}

	// Any line break character (line break classes BK, CR, LF and NL) needs the wrapper.
	for _, r := range txt {
		if isLineBreak(r) {
			return nil, false
		}
	}

	// The wrapper sets the visual order of the runs (logical here)
	// and trims the trailing whitespace.
	for i := range outs {
		outs[i].VisualIndex = int32(i)
	}
	if !trim {
		return []shaping.Line{outs}, true
	}
	last := &outs[len(outs)-1]
	if n := len(last.Glyphs); n > 0 {
		if g := &last.Glyphs[n-1]; g.Width == 0 {
			g.XAdvance = 0
			g.Advance = 0
		}
		last.RecomputeAdvance()
	}
	return []shaping.Line{outs}, true
}

// isLineBreak returns whether the rune is a line break character
// (line break classes BK, CR, LF and NL).
func isLineBreak(r rune) bool {
	switch r {
	case '\n', '\r', '\v', '\f', 0x85, 0x2028, 0x2029:
		return true
	}
	return false
}

// hasMandatoryBreak returns whether the text has a mandatory line break before
// its end, as the line wrapper finds them: after a line break character other
// than the last rune, where CR followed by LF breaks after the LF.
func hasMandatoryBreak(txt []rune) bool {
	for i := 0; i < len(txt)-1; i++ {
		if r := txt[i]; isLineBreak(r) && !(r == '\r' && txt[i+1] == '\n') {
			return true
		}
	}
	return false
}

// wrapParagraph wraps the outputs as the line wrapper's WrapParagraph does,
// for outputs that [singleLine] has found need wrapping. WrapParagraph first
// checks for the case of singleLine, finding the break opportunities of the
// whole text, which it then finds again to wrap it.
func (sh *Shaper) wrapParagraph(cfg shaping.WrapConfig, maxSize int, txt []rune, outs []shaping.Output) ([]shaping.Line, int) {
	sh.wrapper.Prepare(cfg, txt, shaping.NewSliceIterator(outs))
	var lines []shaping.Line
	for {
		line, done := sh.wrapper.WrapNextLine(maxSize)
		if line.Line != nil {
			lines = append(lines, line.Line)
		}
		if done {
			return lines, line.Truncated
		}
	}
}

// wrapConfig returns the line wrapping configuration and the maximum line
// size for wrapping text with the given styles to the given size.
func wrapConfig(defSty *rich.Style, tsty *text.Style, size math32.Vector2) (shaping.WrapConfig, int) {
	lht := tsty.LineHeightDots(defSty)
	dir := shaped.GoTextDirection(rich.Default, tsty)
	nlines := int(math32.Floor(size.Y/lht)) * 2
	maxSize := int(size.X)
	if dir.IsVertical() {
		nlines = int(math32.Floor(size.X / lht))
		maxSize = int(size.Y)
		// fmt.Println(lht, nlines, maxSize)
	}
	// fmt.Println("lht:", lns.LineHeight, lgap, nlines)
	brk := shaping.WhenNecessary
	switch tsty.WhiteSpace {
	case text.WrapNever:
		brk = shaping.Never
	case text.WhiteSpacePre:
		maxSize = 100000
	case text.WrapAlways:
		brk = shaping.Always
	}
	if brk == shaping.Never {
		maxSize = 100000
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
	// 	wc.Truncator = s.ShapeText(params.PxPerEm, params.Locale, []rune(params.Truncator))[0]
	// }
	// todo: WrapParagraph does NOT handle vertical text! file issue.
	return cfg, maxSize
}

// This should already have the mutex lock, and is used by shapedjs but is
// not an end-user call.
func (sh *Shaper) LinesBounds(lines []shaping.Line, truncated int, tx rich.Text, defSty *rich.Style, tsty *text.Style, size math32.Vector2) *shaped.Lines {

	lht := tsty.LineHeightDots(defSty)
	lns := &shaped.Lines{Source: tx, Color: tsty.Color, SelectionColor: tsty.SelectColor, HighlightColor: tsty.HighlightColor, LineHeight: lht}
	lns.Truncated = truncated > 0

	fsz := tsty.FontHeight(defSty)
	dir := shaped.GoTextDirection(rich.Default, tsty)
	// fmt.Println(fsz, lht, lht/fsz, tsty.LineHeight)

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
		maxLHt := lht
		for oi := range lno {
			out := &lno[oi]
			FixOutputZeros(out)
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
				// fmt.Println("out of bounds:", string(cr), crsz, coff, cend)
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
			run.SetFromStyle(sty, tsty)
			if sty.IsMath() {
				mt := sh.maths[cspi]
				if mt != nil {
					run.Math = *mt
					run.MaxBounds = mt.BBox
					bb := run.MaxBounds.Translate(math32.Vector2FromFixed(pos))
					ln.Bounds.ExpandByBox(bb)
					pos.X += math32.ToFixed(run.MaxBounds.Size().X)
					ysz := bb.Size().Y
					// fmt.Println("math ysz:", ysz, "maxAsc:", maxAsc)
					maxAsc = max(maxAsc, math32.ToFixed(-bb.Min.Y))
					maxLHt = max(maxLHt, ysz)
				}
			} else {
				llht := tsty.LineHeightDots(sty)
				maxLHt = max(maxLHt, llht)
				bb := math32.B2FromFixed(run.RunBounds().Add(pos))
				ln.Bounds.ExpandByBox(bb)
				// fmt.Println("adv:", pos, run.Output.Advance, bb.Size().X)
				pos = DirectionAdvance(run.Direction, pos, run.Output.Advance)
			}
			ln.Runs = append(ln.Runs, &run)
		}
		if li == 0 { // set offset for first line based on max ascent
			if !dir.IsVertical() { // todo: vertical!
				off.Y = math32.FromFixed(maxAsc)
			}
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
				off.X += lwd // ?
				ourOff.X += extra
			} else {
				// fmt.Println("!ftl lwd:", lwd, off.X)
				off.X -= lwd // ?
				ourOff.X -= extra
			}
		} else { // always top-down, no progression issues
			lby := ln.Bounds.Size().Y // the result at this point is centered with this height
			// which includes the natural line height property of the font itself.
			lpd := 0.5 * (maxLHt - lby) // half of diff
			if li > 0 {
				ourOff.Y += (lpd + (maxLHt - lns.LineHeight))
			}
			ln.Bounds.Min.Y -= lpd
			ln.Bounds.Max.Y += lpd
			off.Y += maxLHt
			// fmt.Println("lby:", lby, fsz, maxLHt, lpd, ourOff.Y)
		}
		// go back through and give every run the expanded line-level box
		for ri := range ln.Runs {
			run := ln.Runs[ri]
			rb := run.LineBounds()
			if dir.IsVertical() {
				rb.Min.X, rb.Max.X = ln.Bounds.Min.X, ln.Bounds.Max.X
				rb.Min.Y -= 1 // ensure some overlap along direction of rendering adjacent
				rb.Max.Y += 1
			} else {
				rb.Min.Y, rb.Max.Y = ln.Bounds.Min.Y, ln.Bounds.Max.Y
				rb.Min.X -= 1
				rb.Max.Y += 1
			}
			run.AsBase().MaxBounds = rb
		}
		ln.Offset = ourOff
		if tsty.WhiteSpace.HasWordWrap() && size.X > 0 && ln.Bounds.Size().X > size.X {
			// fmt.Println("size exceeded:", ln.Bounds.Size().X, size.X)
			ln.Bounds.Max.X -= ln.Bounds.Size().X - size.X
		}
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
