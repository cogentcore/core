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
	"cogentcore.org/core/text/shaped/shapedgt"
	"cogentcore.org/core/text/text"
)

var (
	canvasInited bool
	canvas       js.Value
	ctx          js.Value
	debug        = true
)

// initCanvas ensures that the shared text measuring canvas is available.
func initCanvas() {
	if canvasInited {
		return
	}
	// todo: replace this with an offscreen canvas!
	document := js.Global().Get("document")
	app := document.Call("getElementById", "app")
	canvas = app // document.Call("createElement", "canvas")
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
	sh.Lock()
	defer sh.Unlock()
	return sh.AdjustRuns(sh.ShapeText(tx, tsty, rts, txt), tx, tsty, rts)
}

// AdjustRuns adjusts the given run metrics based on the html measureText results.
func (sh *Shaper) AdjustRuns(runs []shaped.Run, tx rich.Text, tsty *text.Style, rts *rich.Settings) []shaped.Run {
	for _, run := range runs {
		sh.AdjustRun(run, tx, tsty, rts)
	}
	return runs
}

// AdjustRun adjusts the given run metrics based on the html measureText results.
func (sh *Shaper) AdjustRun(run shaped.Run, tx rich.Text, tsty *text.Style, rts *rich.Settings) {
	grun := run.(*shapedgt.Run)
	gout := &grun.Output
	rng := run.Runes()
	si, _, ri := tx.Index(rng.Start)
	sty, stx := tx.Span(si)
	grun.SetFromStyle(sty, tsty)
	rtx := stx[ri : ri+rng.Len()]
	SetFontStyle(ctx, &grun.Font, tsty, 0)
	fmt.Println("si:", si, ri, sty, string(rtx))

	spm := MeasureText(ctx, string(rtx))
	if debug {
		fmt.Println("\nrun:", string(rtx))
		fmt.Println("lba:\t", math32.FromFixed(gout.LineBounds.Ascent), "\t=\t", spm.FontBoundingBoxAscent)
		fmt.Println("lbd:\t", math32.FromFixed(gout.LineBounds.Descent), "\t=\t", spm.FontBoundingBoxDescent)
	}
	gout.LineBounds.Ascent = math32.ToFixed(spm.FontBoundingBoxAscent)
	gout.LineBounds.Descent = math32.ToFixed(spm.FontBoundingBoxDescent)
	gout.GlyphBounds.Ascent = math32.ToFixed(spm.ActualBoundingBoxAscent)
	gout.GlyphBounds.Descent = math32.ToFixed(spm.ActualBoundingBoxDescent)
	gout.Advance = math32.ToFixed(spm.Width)
	gout.Size = math32.ToFixed(spm.ActualBoundingBoxAscent + spm.ActualBoundingBoxDescent)
	ng := len(gout.Glyphs)
	for gi := 0; gi < ng; gi++ {
		g := &gout.Glyphs[gi]
		gri := g.ClusterIndex - rng.Start
		gtx := rtx[ri+gri : ri+gri+g.GlyphCount]
		gm := theGlyphCache.Glyph(ctx, &grun.Font, tsty, gtx, g.GlyphID)
		if g.GlyphCount > 1 {
			gi += g.GlyphCount - 1
		}
		if debug {
			fmt.Println("\ngi:", gi, string(gtx))
			fmt.Println("adv:\t", math32.FromFixed(g.XAdvance), "\t=\t", gm.Width)
		}
		// todo: conditional on vertical / horiz
		g.XAdvance = math32.ToFixed(gm.Width)
		// g.Height = -(m.ActualBoundingBoxAscent + m.ActualBoundingBoxDescent)
		// g.XBearing = m.ActualBoundingBoxLeft
		// g.YBearing = m.HangingBaseline
	}
}
