// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import (
	"fmt"
	"os"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
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

// //go:embed fonts/*.ttf
// var efonts embed.FS // TODO

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
	// sh.fontMap.AddFont(errors.Log1(efonts.Open("fonts/Roboto-Regular.ttf")).(opentype.Resource), "Roboto", "Roboto") // TODO
	// for _, f := range collection {
	// 	shaper.Load(f)
	// 	shaper.defaultFaces = append(shaper.defaultFaces, string(f.Font.Typeface))
	// }
	sh.shaper.SetFontCacheSize(32)
	return sh
}

// FontSize returns the font shape sizing information for given font and text style,
// using given rune (often the letter 'm'). The GlyphBounds field of the [Run] result
// has the font ascent and descent information, and the BoundsBox() method returns a full
// bounding box for the given font, centered at the baseline.
func (sh *Shaper) FontSize(r rune, sty *rich.Style, tsty *text.Style, rts *rich.Settings) *Run {
	sp := rich.NewSpans(sty, r)
	out := sh.shapeText(sp, tsty, rts, []rune{r})
	return &Run{Output: out[0]}
}

// LineHeight returns the line height for given font and text style.
// For vertical text directions, this is actually the line width.
// It includes the [text.Style] LineSpacing multiplier on the natural
// font-derived line height, which is not generally the same as the font size.
func (sh *Shaper) LineHeight(sty *rich.Style, tsty *text.Style, rts *rich.Settings) float32 {
	run := sh.FontSize('m', sty, tsty, rts)
	bb := run.BoundsBox()
	dir := goTextDirection(rich.Default, tsty)
	if dir.IsVertical() {
		return tsty.LineSpacing * bb.Size().X
	}
	return tsty.LineSpacing * bb.Size().Y
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
		in.Direction = goTextDirection(sty.Direction, tsty)
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

// goTextDirection gets the proper go-text direction value from styles.
func goTextDirection(rdir rich.Directions, tsty *text.Style) di.Direction {
	dir := tsty.Direction
	if rdir != rich.Default {
		dir = rdir
	}
	return dir.ToGoText()
}

// todo: do the paragraph splitting!  write fun in rich.Spans

// WrapLines performs line wrapping and shaping on the given rich text source,
// using the given style information, where the [rich.Style] provides the default
// style information reflecting the contents of the source (e.g., the default family,
// weight, etc), for use in computing the default line height. Paragraphs are extracted
// first using standard newline markers, assumed to coincide with separate spans in the
// source text, and wrapped separately.
func (sh *Shaper) WrapLines(sp rich.Spans, defSty *rich.Style, tsty *text.Style, rts *rich.Settings, size math32.Vector2) *Lines {
	if tsty.FontSize.Dots == 0 {
		tsty.FontSize.Dots = 24
	}
	fsz := tsty.FontSize.Dots
	dir := goTextDirection(rich.Default, tsty)

	lht := sh.LineHeight(defSty, tsty, rts)
	lns := &Lines{Source: sp, Color: tsty.Color, SelectionColor: colors.Scheme.Select.Container, HighlightColor: colors.Scheme.Warn.Container, LineHeight: lht}

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
	// todo: WrapParagraph does NOT handle vertical text! file issue.
	lines, truncate := sh.wrapper.WrapParagraph(cfg, maxSize, txt, shaping.NewSliceIterator(outs))
	lns.Truncated = truncate > 0
	cspi := 0
	cspSt, cspEd := sp.Range(cspi)
	var off math32.Vector2
	for _, lno := range lines {
		// fmt.Println("line:", li, off)
		ln := Line{}
		var lsp rich.Spans
		var pos fixed.Point26_6
		setFirst := false
		for oi := range lno {
			out := &lno[oi]
			run := Run{Output: *out}
			rns := run.Runes()
			if !setFirst {
				ln.SourceRange.Start = rns.Start
				setFirst = true
			}
			ln.SourceRange.End = rns.End
			for rns.Start >= cspEd {
				cspi++
				cspSt, cspEd = sp.Range(cspi)
			}
			sty, cr := rich.NewStyleFromRunes(sp[cspi])
			if lns.FontSize == 0 {
				lns.FontSize = sty.Size * fsz
			}
			nsp := sty.ToRunes()
			coff := rns.Start - cspSt
			cend := coff + rns.Len()
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
			ln.Bounds.ExpandByBox(bb)
			pos = DirectionAdvance(run.Direction, pos, run.Advance)
			ln.Runs = append(ln.Runs, run)
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

// DirectionAdvance advances given position based on given direction.
func DirectionAdvance(dir di.Direction, pos fixed.Point26_6, adv fixed.Int26_6) fixed.Point26_6 {
	if dir.IsVertical() {
		pos.Y += -adv
	} else {
		pos.X += adv
	}
	return pos
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
