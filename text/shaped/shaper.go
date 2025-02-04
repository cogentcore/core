// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
	"fmt"
	"os"
	"slices"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
)

// Shaper is the text shaper and wrapper, from go-text/shaping.
type Shaper struct {
	shaper   shaping.HarfbuzzShaper
	wrapper  shaping.LineWrapper
	fontMap  *fontscan.FontMap
	splitter shaping.Segmenter

	//	outBuff is the output buffer to avoid excessive memory consumption.
	outBuff []shaping.Output
}

// DirectionAdvance advances given position based on given direction.
func DirectionAdvance(dir di.Direction, pos fixed.Point26_6, adv fixed.Int26_6) fixed.Point26_6 {
	if dir.IsVertical() {
		pos.Y += adv
	} else {
		pos.X += adv
	}
	return pos
}

// todo: per gio: systemFonts bool, collection []FontFace
func NewShaper() *Shaper {
	sh := &Shaper{}
	sh.fontMap = fontscan.NewFontMap(nil)
	str, err := os.UserCacheDir()
	if errors.Log(err) != nil {
		// slog.Printf("failed resolving font cache dir: %v", err)
		// shaper.logger.Printf("skipping system font load")
	}
	// fmt.Println("cache dir:", str)
	if err := sh.fontMap.UseSystemFonts(str); err != nil {
		errors.Log(err)
		// shaper.logger.Printf("failed loading system fonts: %v", err)
	}
	// for _, f := range collection {
	// 	shaper.Load(f)
	// 	shaper.defaultFaces = append(shaper.defaultFaces, string(f.Font.Typeface))
	// }
	sh.shaper.SetFontCacheSize(32)
	return sh
}

// Shape turns given input spans into [Runs] of rendered text,
// using given context needed for complete styling.
// The results are only valid until the next call to Shape or WrapParagraph:
// use slices.Clone if needed longer than that.
func (sh *Shaper) Shape(sp rich.Spans, tsty *text.Style, rts *rich.Settings) []shaping.Output {
	return sh.shapeText(sp, tsty, rts, sp.Join())
}

// shapeText implements Shape using the full text generated from the source spans
func (sh *Shaper) shapeText(sp rich.Spans, tsty *text.Style, rts *rich.Settings, txt []rune) []shaping.Output {
	sty := rich.NewStyle()
	sh.outBuff = sh.outBuff[:0]
	for si, s := range sp {
		in := shaping.Input{}
		start, end := sp.Range(si)
		sty.FromRunes(s)
		q := StyleToQuery(sty, rts)
		sh.fontMap.SetQuery(q)

		in.Text = txt
		in.RunStart = start
		in.RunEnd = end
		in.Direction = sty.Direction.ToGoText()
		fsz := tsty.FontSize.Dots * sty.Size
		in.Size = math32.ToFixed(fsz)
		in.Script = rts.Script
		in.Language = rts.Language

		ins := sh.splitter.Split(in, sh.fontMap) // this is essential
		for _, in := range ins {
			o := sh.shaper.Shape(in)
			sh.outBuff = append(sh.outBuff, o)
		}
	}
	return sh.outBuff
}

func (sh *Shaper) WrapParagraph(sp rich.Spans, tsty *text.Style, rts *rich.Settings, size math32.Vector2) *Lines {
	if tsty.FontSize.Dots == 0 {
		tsty.FontSize.Dots = 24
	}
	fsz := tsty.FontSize.Dots
	dir := tsty.Direction.ToGoText()
	lht := tsty.LineHeight()
	lgap := lht - fsz
	fmt.Println("lgap:", lgap)
	nlines := int(math32.Floor(size.Y / lht))
	brk := shaping.WhenNecessary
	if !tsty.WhiteSpace.HasWordWrap() {
		brk = shaping.Never
	} else if tsty.WhiteSpace == text.WrapAlways {
		brk = shaping.Always
	}
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
	txt := sp.Join()
	outs := sh.shapeText(sp, tsty, rts, txt)
	lines, truncate := sh.wrapper.WrapParagraph(cfg, int(size.X), txt, shaping.NewSliceIterator(outs))
	lns := &Lines{Color: tsty.Color}
	lns.Truncated = truncate > 0
	cspi := 0
	cspSt, cspEd := sp.Range(cspi)
	fmt.Println("st:", cspSt, cspEd)
	var off math32.Vector2
	for _, lno := range lines {
		ln := Line{}
		var lsp rich.Spans
		var pos fixed.Point26_6
		for oi := range lno {
			out := &lno[oi]
			for out.Runes.Offset >= cspEd {
				cspi++
				cspSt, cspEd = sp.Range(cspi)
				fmt.Println("nxt:", cspi, cspSt, cspEd, out.Runes.Offset)
			}
			sty, cr := rich.NewStyleFromRunes(sp[cspi])
			if lns.FontSize == 0 {
				lns.FontSize = sty.Size * fsz
			}
			nsp := sty.ToRunes()
			coff := out.Runes.Offset - cspSt
			cend := coff + out.Runes.Count
			nr := cr[coff:cend] // note: not a copy!
			nsp = append(nsp, nr...)
			lsp = append(lsp, nsp)
			fmt.Println(sty, string(nr))
			if cend < (cspEd - cspSt) { // shouldn't happen, to combine multiple original spans
				fmt.Println("combined original span:", cend, cspEd-cspSt, cspi, string(cr), "prev:", string(nr), "next:", string(cr[cend:]))
			}
			bb := math32.B2FromFixed(OutputBounds(out).Add(pos))
			ln.Bounds.ExpandByBox(bb)
			pos = DirectionAdvance(out.Direction, pos, out.Advance)
		}
		ln.Source = lsp
		ln.Runs = slices.Clone(lno)
		ln.Offset = off
		fmt.Println(ln.Bounds)
		lns.Bounds.ExpandByBox(ln.Bounds.Translate(ln.Offset))
		// advance offset:
		if dir.IsVertical() {
			lwd := ln.Bounds.Size().X
			if dir.Progression() == di.FromTopLeft {
				off.X += lwd + lgap
			} else {
				off.X -= lwd + lgap
			}
		} else {
			lht := ln.Bounds.Size().Y
			if dir.Progression() == di.FromTopLeft {
				off.Y += lht + lgap
			} else {
				off.Y -= lht + lgap
			}
		}
		// todo: rest of it
		lns.Lines = append(lns.Lines, ln)
	}
	fmt.Println(lns.Bounds)
	return lns
}

// StyleToQuery translates the rich.Style to go-text fontscan.Query parameters.
func StyleToQuery(sty *rich.Style, rts *rich.Settings) fontscan.Query {
	q := fontscan.Query{}
	q.Families = rich.FamiliesToList(sty.FontFamily(rts))
	q.Aspect = StyleToAspect(sty)
	return q
}

// StyleToAspect translates the rich.Style to go-text font.Aspect parameters.
func StyleToAspect(sty *rich.Style) font.Aspect {
	as := font.Aspect{}
	as.Style = font.Style(1 + sty.Slant)
	as.Weight = font.Weight(sty.Weight.ToFloat32())
	as.Stretch = font.Stretch(sty.Stretch.ToFloat32())
	return as
}
