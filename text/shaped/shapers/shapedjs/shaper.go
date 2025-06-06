// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"syscall/js"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/shaped/shapers/shapedgt"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
	"github.com/go-text/typesetting/shaping"
)

var (
	canvasInited bool
	canvas       js.Value
	ctx          js.Value
)

// initCanvas ensures that the shared text measuring canvas is available.
func initCanvas() {
	if canvasInited {
		return
	}
	canvas = js.Global().Get("OffscreenCanvas").New(1000, 100)
	ctx = canvas.Call("getContext", "2d")
	canvasInited = true
}

// Shaper is the html canvas version of text shaping,
// which bootstraps off of the go-text version and corrects
// the results using the results of measuring the text.
type Shaper struct {
	shapedgt.Shaper
}

// NewShaper returns a new text shaper.
func NewShaper() shaped.Shaper {
	initCanvas()
	sh := shapedgt.NewShaper()
	if sh == nil {
		panic("nil gt shaper!")
	}
	return &Shaper{Shaper: *sh.(*shapedgt.Shaper)}
}

// Shape turns given input spans into [Runs] of rendered text,
// using given context needed for complete styling.
// The results are only valid until the next call to Shape or WrapParagraph:
// use slices.Clone if needed longer than that.
// This is called under a mutex lock, so it is safe for parallel use.
func (sh *Shaper) Shape(tx rich.Text, tsty *text.Style, rts *rich.Settings) []shaped.Run {
	sh.Lock()
	defer sh.Unlock()
	return sh.shapeAdjust(tx, tsty, rts, tx.Join())
}

// shapeAdjust turns given input spans into [Runs] of rendered text,
// using given context needed for complete styling.
// The results are only valid until the next call to Shape or WrapParagraph:
// use slices.Clone if needed longer than that.
func (sh *Shaper) shapeAdjust(tx rich.Text, tsty *text.Style, rts *rich.Settings, txt []rune) []shaped.Run {
	return sh.adjustRuns(sh.ShapeText(tx, tsty, rts, txt), tx, tsty, rts)
}

// adjustRuns adjusts the given run metrics based on the html measureText results.
// This should already have the mutex lock, and is used by shapedjs but is
// not an end-user call.
func (sh *Shaper) adjustRuns(runs []shaped.Run, tx rich.Text, tsty *text.Style, rts *rich.Settings) []shaped.Run {
	for _, run := range runs {
		grun := run.(*shapedgt.Run)
		out := &grun.Output
		fnt := &grun.Font
		sh.adjustOutput(out, fnt, tx, tsty, rts)
	}
	return runs
}

// WrapLines performs line wrapping and shaping on the given rich text source,
// using the given style information, where the [rich.Style] provides the default
// style information reflecting the contents of the source (e.g., the default family,
// weight, etc), for use in computing the default line height. Paragraphs are extracted
// first using standard newline markers, assumed to coincide with separate spans in the
// source text, and wrapped separately. For horizontal text, the Lines will render with
// a position offset at the upper left corner of the overall bounding box of the text.
// This is called under a mutex lock, so it is safe for parallel use.
func (sh *Shaper) WrapLines(tx rich.Text, defSty *rich.Style, tsty *text.Style, rts *rich.Settings, size math32.Vector2) *shaped.Lines {
	sh.Lock()
	defer sh.Unlock()
	if tsty.FontSize.Dots == 0 {
		tsty.FontSize.Dots = 16
	}

	txt := tx.Join()
	// sptx := tx.Clone()
	// sptx.SplitSpaces() // no advantage to doing this
	outs := sh.ShapeTextOutput(tx, tsty, rts, txt)
	for oi := range outs {
		out := &outs[oi]
		si, _, _ := tx.Index(out.Runes.Offset)
		sty, _ := tx.Span(si)
		fnt := text.NewFont(sty, tsty)
		sh.adjustOutput(out, fnt, tx, tsty, rts)
	}
	lines, truncated := sh.WrapLinesOutput(outs, txt, tx, defSty, tsty, rts, size)
	for _, lno := range lines {
		for oi := range lno {
			out := &lno[oi]
			si, _, _ := tx.Index(out.Runes.Offset)
			sty, _ := tx.Span(si)
			fnt := text.NewFont(sty, tsty)
			sh.adjustOutput(out, fnt, tx, tsty, rts)
		}
	}
	return sh.LinesBounds(lines, truncated, tx, defSty, tsty, size)
}

// adjustOutput adjusts the given run metrics based on the html measureText results.
// This should already have the mutex lock, and is used by shapedjs but is
// not an end-user call.
func (sh *Shaper) adjustOutput(out *shaping.Output, fnt *text.Font, tx rich.Text, tsty *text.Style, rts *rich.Settings) {
	rng := textpos.Range{out.Runes.Offset, out.Runes.Offset + out.Runes.Count}
	si, sn, ri := tx.Index(rng.Start)
	sty, stx := tx.Span(si)
	if sty.IsMath() {
		return
	}
	ri -= sn
	if ri+rng.Len() > len(stx) {
		rng.End -= (ri + rng.Len()) - len(stx)
		// fmt.Println("shape range err:", tx, string(stx), len(stx), si, sn, ri, rng)
		// return
	}
	rtx := stx[ri : ri+rng.Len()]
	SetFontStyle(ctx, fnt, tsty, 0)

	spm := MeasureText(ctx, string(rtx))
	msz := spm.FontBoundingBoxAscent + spm.FontBoundingBoxDescent
	out.Advance = math32.ToFixed(spm.Width)
	out.Size = math32.ToFixed(msz)
	out.LineBounds.Ascent = math32.ToFixed(spm.FontBoundingBoxAscent)
	out.LineBounds.Descent = -math32.ToFixed(spm.FontBoundingBoxDescent)
	out.GlyphBounds.Ascent = math32.ToFixed(spm.ActualBoundingBoxAscent)
	out.GlyphBounds.Descent = -math32.ToFixed(spm.ActualBoundingBoxDescent)
	ng := len(out.Glyphs)
	for gi := 0; gi < ng; gi++ {
		g := &out.Glyphs[gi]
		gri := g.ClusterIndex - rng.Start
		// nrtx := len(rtx)
		ed := gri + g.GlyphCount
		gtx := rtx[gri:ed]
		gm := theGlyphCache.Glyph(ctx, fnt, tsty, gtx, g.GlyphID)
		if g.GlyphCount > 1 {
			gi += g.GlyphCount - 1
		}
		msz := gm.ActualBoundingBoxAscent + gm.ActualBoundingBoxDescent
		mwd := -gm.ActualBoundingBoxLeft + gm.ActualBoundingBoxRight
		// todo: conditional on vertical / horiz
		g.XAdvance = math32.ToFixed(gm.Width)
		g.Width = math32.ToFixed(mwd)
		g.Height = -math32.ToFixed(msz)
		g.XBearing = -math32.ToFixed(gm.ActualBoundingBoxLeft)
		g.YBearing = math32.ToFixed(gm.ActualBoundingBoxAscent)
	}
}
