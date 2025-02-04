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
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/shaping"
)

// Shaper is the text shaper and wrapper, from go-text/shaping.
type Shaper struct {
	shaper  shaping.HarfbuzzShaper
	wrapper shaping.LineWrapper
	FontMap *fontscan.FontMap

	//	outBuff is the output buffer to avoid excessive memory consumption.
	outBuff []shaping.Output
}

// todo: per gio: systemFonts bool, collection []FontFace
func NewShaper() *Shaper {
	sh := &Shaper{}
	sh.FontMap = fontscan.NewFontMap(nil)
	str, err := os.UserCacheDir()
	if errors.Log(err) != nil {
		// slog.Printf("failed resolving font cache dir: %v", err)
		// shaper.logger.Printf("skipping system font load")
	}
	if err := sh.FontMap.UseSystemFonts(str); err != nil {
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
func (sh *Shaper) Shape(sp rich.Spans, ctx *rich.Context) []shaping.Output {
	return sh.shapeText(sp, ctx, sp.Join())
}

// shapeText implements Shape using the full text generated from the source spans
func (sh *Shaper) shapeText(sp rich.Spans, ctx *rich.Context, txt []rune) []shaping.Output {
	sty := rich.NewStyle()
	sh.outBuff = sh.outBuff[:0]
	for si, s := range sp {
		in := shaping.Input{}
		start, end := sp.Range(si)
		sty.FromRunes(s)

		sh.FontMap.SetQuery(StyleToQuery(sty, ctx))

		in.Text = txt
		in.RunStart = start
		in.RunEnd = end
		in.Direction = sty.Direction.ToGoText()
		fsz := sty.FontSize(ctx)
		in.Size = math32.ToFixed(fsz)
		in.Script = ctx.Script
		in.Language = ctx.Language

		// todo: per gio:
		// inputs := s.splitBidi(input)
		// inputs = s.splitByFaces(inputs, s.splitScratch1[:0])
		// inputs = splitByScript(inputs, lcfg.Direction, s.splitScratch2[:0])
		ins := shaping.SplitByFace(in, sh.FontMap)
		// fmt.Println("nin:", len(ins))
		for _, i := range ins {
			o := sh.shaper.Shape(i)
			sh.outBuff = append(sh.outBuff, o)
		}
	}
	return sh.outBuff
}

func (sh *Shaper) WrapParagraph(sp rich.Spans, ctx *rich.Context, tstyle *text.Style, size math32.Vector2) *Lines {
	nctx := *ctx
	nctx.Direction = tstyle.Direction
	nctx.StandardSize = tstyle.FontSize
	lht := tstyle.LineHeight()
	nlines := int(math32.Floor(size.Y / lht))
	brk := shaping.WhenNecessary
	if !tstyle.WhiteSpace.HasWordWrap() {
		brk = shaping.Never
	} else if tstyle.WhiteSpace == text.WrapAlways {
		brk = shaping.Always
	}
	cfg := shaping.WrapConfig{
		Direction:                     tstyle.Direction.ToGoText(),
		TruncateAfterLines:            nlines,
		TextContinues:                 false, // todo! no effect if TruncateAfterLines is 0
		BreakPolicy:                   brk,   // or Never, Always
		DisableTrailingWhitespaceTrim: tstyle.WhiteSpace.KeepWhiteSpace(),
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
	outs := sh.shapeText(sp, ctx, txt)
	lines, truncate := sh.wrapper.WrapParagraph(cfg, int(size.X), txt, shaping.NewSliceIterator(outs))
	lns := &Lines{Color: ctx.Color}
	lns.Truncated = truncate > 0
	cspi := 0
	cspSt, cspEd := sp.Range(cspi)
	for _, lno := range lines {
		ln := Line{}
		var lsp rich.Spans
		for oi := range lno {
			out := &lno[oi]
			for out.Runes.Offset >= cspEd {
				cspi++
				cspSt, cspEd = sp.Range(cspi)
			}
			sty, cr := rich.NewStyleFromRunes(sp[cspi])
			if lns.FontSize == 0 {
				lns.FontSize = sty.FontSize(ctx)
			}
			nsp := sty.ToRunes()
			coff := out.Runes.Offset - cspSt
			cend := coff + out.Runes.Count
			nr := cr[coff:cend] // note: not a copy!
			nsp = append(nsp, nr...)
			lsp = append(lsp, nsp)
			if cend < (cspEd - cspSt) { // shouldn't happen, to combine multiple original spans
				fmt.Println("combined original span:", cend, cspEd-cspSt, cspi, string(cr), "prev:", string(nr), "next:", string(cr[cend:]))
			}
		}
		ln.Source = lsp
		ln.Runs = slices.Clone(lno)
		// todo: rest of it
		lns.Lines = append(lns.Lines, ln)
	}
	fmt.Println(lns)
	return lns
}

// StyleToQuery translates the rich.Style to go-text fontscan.Query parameters.
func StyleToQuery(sty *rich.Style, ctx *rich.Context) fontscan.Query {
	q := fontscan.Query{}
	q.Families = rich.FamiliesToList(sty.FontFamily(ctx))
	q.Aspect = StyleToAspect(sty)
	return q
}

// StyleToAspect translates the rich.Style to go-text font.Aspect parameters.
func StyleToAspect(sty *rich.Style) font.Aspect {
	as := font.Aspect{}
	as.Style = font.Style(sty.Slant)
	as.Weight = font.Weight(sty.Weight)
	as.Stretch = font.Stretch(sty.Stretch)
	return as
}
