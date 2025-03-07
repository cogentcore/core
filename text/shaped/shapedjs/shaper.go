// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"sync"
	"syscall/js"

	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
	"github.com/go-text/typesetting/di"
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
	// todo: replace this with an offscreen canvas!
	document := js.Global().Get("document")
	app := document.Call("getElementById", "app")
	canvas = app // document.Call("createElement", "canvas")
	// document.Get("body").Call("appendChild", elem)
	ctx = canvas.Call("getContext", "2d")
	canvasInited = true
}

type Shaper struct {
	// outBuff is the output buffer to avoid excessive memory consumption.
	outBuff []Output
	sync.Mutex
}

// NewShaper returns a new text shaper.
func NewShaper() shaped.Shaper {
	initCanvas()
	sh := &Shaper{}
	return sh
}

// Shape turns given input spans into [Runs] of rendered text,
// using given context needed for complete styling.
// The results are only valid until the next call to Shape or WrapParagraph:
// use slices.Clone if needed longer than that.
func (sh *Shaper) Shape(tx rich.Text, tsty *text.Style, rts *rich.Settings) []shaped.Run {
	sh.Lock()
	defer sh.Unlock()

	outs := sh.shapeText(tx, tsty, rts, tx.Join())
	runs := make([]shaped.Run, len(outs))
	for i := range outs {
		run := &Run{Output: outs[i]}
		runs[i] = run
	}
	return runs
}

// shapeText implements Shape using the full text generated from the source spans.
func (sh *Shaper) shapeText(tx rich.Text, tsty *text.Style, rts *rich.Settings, txt []rune) []Output {
	if tx.Len() == 0 {
		return nil
	}
	sty := rich.NewStyle()
	sh.outBuff = sh.outBuff[:0]
	for si, s := range tx {
		start, end := tx.Range(si)
		rs := sty.FromRunes(s)
		if len(rs) == 0 {
			continue
		}
		fn := NewFont(sty, tsty, rts)
		SetFontStyle(ctx, fn, tsty, 0)

		spm := MeasureText(ctx, string(rs))
		out := Output{Advance: spm.Width, Size: fn.Size, Direction: sty.Direction, Runes: textpos.Range{Start: start, End: end}}
		out.LineBounds.Ascent = spm.FontBoundingBoxAscent
		out.LineBounds.Descent = spm.FontBoundingBoxDescent
		// todo: gap
		out.GlyphBounds.Ascent = spm.ActualBoundingBoxAscent
		out.GlyphBounds.Descent = spm.ActualBoundingBoxDescent
		// actual gap = 0 always
		gs := make([]Glyph, len(rs))
		for ri, rn := range rs {
			g := theGlyphCache.Glyph(ctx, fn, tsty, rn)
			gs[ri] = *g
		}
		out.Glyphs = gs
		sh.outBuff = append(sh.outBuff, out)
	}
	return sh.outBuff
}

// goTextDirection gets the proper go-text direction value from styles.
func goTextDirection(rdir rich.Directions, tsty *text.Style) di.Direction {
	dir := tsty.Direction
	if rdir != rich.Default {
		dir = rdir
	}
	return dir.ToGoText()
}
