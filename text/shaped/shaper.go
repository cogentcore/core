// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
	"os"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
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
	outBuff []shaper.Output
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
	sh.SetFontCacheSize(32)
	return sh
}

// Shape turns given input spans into [Runs] of rendered text,
// using given context needed for complete styling.
func (sh *Shaper) Shape(sp rich.Spans, ctx *rich.Context) *Runs {
	return sh.shapeText(sp, ctx, sp.Join())
}

// shapeText implements Shape using the full text generated from the source spans
func (sh *Shaper) shapeText(sp rich.Spans, ctx *rich.Context, txt []rune) *Runs {
	txt := sp.Join() // full text
	sty := rich.NewStyle()
	for si, s := range sp {
		run := Run{}
		in := shaping.Input{}
		start, end := sp.Range(si)
		sty.FromRunes(s)

		sh.FontMap.SetQuery(StyleToQuery(sty, ctx))

		in.Text = txt
		in.RunStart = start
		in.RunEnd = end
		in.Direction = sty.Direction.ToGoText()
		fsz := sty.FontSize(ctx)
		run.FontSize = fsz
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
			o := sh.HarfbuzzShaper.Shape(i)
			run.Subs = append(run.Subs, o)
			run.Index = si
		}
		runs.Runs = append(runs.Runs, run)
	}
	return runs
}

func (sh *Shaper) WrapParagraph(sp rich.Spans, ctx *rich.Context, maxWidth float32) *Runs {
	cfg := shaping.WrapConfig{
		Direction:                     ctx.Direction.ToGoText(),
		TruncateAfterLines:            0,
		TextContinues:                 false,                 // no effect if TruncateAfterLines is 0
		BreakPolicy:                   shaping.WhenNecessary, // or Never, Always
		DisableTrailingWhitespaceTrim: false,                 // true for editor lines context, false for text display context
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
	runs := sh.shapeText(sp, ctx, txt)
	outs := sh.outputs(runs)
	lines, truncate := LineWrapper.WrapParagraph(wc, int(maxWidth), txt, shaping.NewSliceIterator(outs))
	// now go through and remake spans and runs based on lines
	lruns := sh.lineRuns(runs, txt, outs, lines)
	return lruns
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

// outputs returns all of the outputs from given text runs, using outBuff backing store.
func (sh *Shaper) outputs(runs *Runs) []shaper.Output {
	nouts := 0
	for ri := range runs.Runs {
		run := &runs.Runs[ri]
		nouts += len(run.Subs)
	}
	slicesx.SetLength(sh.outBuff, nouts)
	idx := 0
	for ri := range runs.Runs {
		run := &runs.Runs[ri]
		for si := range run.Subs {
			sh.outBuff[idx] = run.Subs[si]
			idx++
		}
	}
	return sh.outBuff
}

// lineRuns returns a new Runs based on original Runs and outs, and given wrapped lines.
// The Spans will be regenerated based on the actual lines made.
func (sh *Shaper) lineRuns(src *Runs, txt []rune, outs []shaper.Output, lines []shaping.Line) *Runs {
	for li, ln := range lines {

	}
}
