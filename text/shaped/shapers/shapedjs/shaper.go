// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"fmt"
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
	debug        = false
)

// initCanvas ensures that the shared text measuring canvas is available.
func initCanvas() {
	if canvasInited {
		return
	}
	canvas = js.Global().Get("OffscreenCanvas").New(1000, 100)
	// todo: replace this with an offscreen canvas!
	// document := js.Global().Get("document")
	// app := document.Call("getElementById", "app")
	// canvas = app // document.Call("createElement", "canvas")
	// document.Get("body").Call("appendChild", elem)
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
	return &Shaper{Shaper: *sh.(*shapedgt.Shaper)}
}

// Shape turns given input spans into [Runs] of rendered text,
// using given context needed for complete styling.
// The results are only valid until the next call to Shape or WrapParagraph:
// use slices.Clone if needed longer than that.
func (sh *Shaper) Shape(tx rich.Text, tsty *text.Style, rts *rich.Settings) []shaped.Run {
	sh.Lock()
	defer sh.Unlock()
	return sh.ShapeAdjust(tx, tsty, rts, tx.Join())
}

// ShapeAdjust turns given input spans into [Runs] of rendered text,
// using given context needed for complete styling.
// The results are only valid until the next call to Shape or WrapParagraph:
// use slices.Clone if needed longer than that.
func (sh *Shaper) ShapeAdjust(tx rich.Text, tsty *text.Style, rts *rich.Settings, txt []rune) []shaped.Run {
	return sh.AdjustRuns(sh.ShapeText(tx, tsty, rts, txt), tx, tsty, rts)
	// return sh.ShapeText(tx, tsty, rts, txt)
}

// AdjustRuns adjusts the given run metrics based on the html measureText results.
func (sh *Shaper) AdjustRuns(runs []shaped.Run, tx rich.Text, tsty *text.Style, rts *rich.Settings) []shaped.Run {
	for _, run := range runs {
		grun := run.(*shapedgt.Run)
		out := &grun.Output
		fnt := &grun.Font
		sh.AdjustOutput(out, fnt, tx, tsty, rts)
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
func (sh *Shaper) WrapLines(tx rich.Text, defSty *rich.Style, tsty *text.Style, rts *rich.Settings, size math32.Vector2) *shaped.Lines {
	sh.Lock()
	defer sh.Unlock()
	if tsty.FontSize.Dots == 0 {
		tsty.FontSize.Dots = 16
	}

	lht := sh.lineHeight(defSty, tsty, rts) // note: this overwrites output buffer so must do outs after!
	txt := tx.Join()
	outs := sh.ShapeTextOutput(tx, tsty, rts, txt)
	for oi := range outs {
		out := &outs[oi]
		si, _, _ := tx.Index(out.Runes.Offset)
		sty, _ := tx.Span(si)
		fnt := text.NewFont(sty, tsty)
		sh.AdjustOutput(out, fnt, tx, tsty, rts)
	}
	lines, truncated := sh.WrapLinesOutput(outs, txt, tx, defSty, tsty, lht, rts, size)
	for _, lno := range lines {
		for oi := range lno {
			out := &lno[oi]
			si, _, _ := tx.Index(out.Runes.Offset)
			sty, _ := tx.Span(si)
			fnt := text.NewFont(sty, tsty)
			sh.AdjustOutput(out, fnt, tx, tsty, rts)
		}
	}
	return sh.LinesBounds(lines, truncated, tx, tsty, lht)
}

// AdjustOutput adjusts the given run metrics based on the html measureText results.
func (sh *Shaper) AdjustOutput(out *shaping.Output, fnt *text.Font, tx rich.Text, tsty *text.Style, rts *rich.Settings) {
	rng := textpos.Range{out.Runes.Offset, out.Runes.Offset + out.Runes.Count}
	si, sn, ri := tx.Index(rng.Start)
	_, stx := tx.Span(si)
	ri -= sn
	// fmt.Println(string(stx))
	rtx := stx[ri : ri+rng.Len()]
	SetFontStyle(ctx, fnt, tsty, 0)
	// fmt.Println("si:", si, ri, sty, string(rtx))

	spm := MeasureText(ctx, string(rtx))
	msz := spm.FontBoundingBoxAscent + spm.FontBoundingBoxDescent
	if debug {
		fmt.Println("\nrun:", string(rtx))
		fmt.Println("adv:\t", math32.FromFixed(out.Advance), "\t=\t", spm.Width)
		fmt.Println("siz:\t", math32.FromFixed(out.Size), "\t=\t", msz)
		fmt.Println("lba:\t", math32.FromFixed(out.LineBounds.Ascent), "\t=\t", spm.FontBoundingBoxAscent)
		fmt.Println("lbd:\t", math32.FromFixed(out.LineBounds.Descent), "\t=\t", -spm.FontBoundingBoxDescent)
		fmt.Println("gba:\t", math32.FromFixed(out.GlyphBounds.Ascent), "\t=\t", spm.ActualBoundingBoxAscent)
		fmt.Println("gbd:\t", math32.FromFixed(out.GlyphBounds.Descent), "\t=\t", -spm.ActualBoundingBoxDescent)
	}
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
		if debug {
			fmt.Println("\ngi:", gi, string(gtx))
			fmt.Println("adv:\t", math32.FromFixed(g.XAdvance), "\t=\t", gm.Width)
			// yadv = 0
			fmt.Println("wdt:\t", math32.FromFixed(g.Width), "\t=\t", mwd)
			fmt.Println("hgt:\t", math32.FromFixed(g.Height), "\t=\t", -msz)
			fmt.Println("xbr:\t", math32.FromFixed(g.XBearing), "\t=\t", -gm.ActualBoundingBoxLeft)
			fmt.Println("ybr:\t", math32.FromFixed(g.YBearing), "\t=\t", gm.ActualBoundingBoxAscent)
		}
		// todo: conditional on vertical / horiz
		g.XAdvance = math32.ToFixed(gm.Width)
		g.Width = math32.ToFixed(mwd)
		g.Height = -math32.ToFixed(msz)
		g.XBearing = -math32.ToFixed(gm.ActualBoundingBoxLeft)
		g.YBearing = math32.ToFixed(gm.ActualBoundingBoxAscent)
	}
}
