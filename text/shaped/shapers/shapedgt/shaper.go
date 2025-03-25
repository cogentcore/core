// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"fmt"
	"os"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/fonts"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/tex"
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

	// outBuff is the output buffer to avoid excessive memory consumption.
	outBuff []shaping.Output

	sync.Mutex
}

type nilLogger struct{}

func (nl *nilLogger) Printf(format string, args ...any) {}

var didDebug = true

// NewShaper returns a new text shaper.
func NewShaper() shaped.Shaper {
	sh := &Shaper{}
	sh.fontMap = fontscan.NewFontMap(&nilLogger{})
	// TODO(text): figure out cache dir situation (especially on mobile and web)
	str, err := os.UserCacheDir()
	if errors.Log(err) != nil {
		// slog.Printf("failed resolving font cache dir: %v", err)
		// shaper.logger.Printf("skipping system font load")
	}
	// fmt.Println("cache dir:", str)
	if err := sh.fontMap.UseSystemFonts(str); err != nil {
		// note: we expect this error on js platform -- could do something exclusive here
		// under a separate build tag file..
		// errors.Log(err)
		// shaper.logger.Printf("failed loading system fonts: %v", err)
	}
	errors.Log(fonts.UseEmbeddedFonts(sh.fontMap))
	sh.shaper.SetFontCacheSize(32)

	if !didDebug {
		sh.FontDebug()
		didDebug = true
	}
	return sh
}

// Shape turns given input spans into [Runs] of rendered text,
// using given context needed for complete styling.
// The results are only valid until the next call to Shape or WrapParagraph:
// use slices.Clone if needed longer than that.
func (sh *Shaper) Shape(tx rich.Text, tsty *text.Style, rts *rich.Settings) []shaped.Run {
	sh.Lock()
	defer sh.Unlock()
	return sh.ShapeText(tx, tsty, rts, tx.Join())
}

// ShapeText shapes the spans in the given text using given style and settings,
// returning [shaped.Run] results.
func (sh *Shaper) ShapeText(tx rich.Text, tsty *text.Style, rts *rich.Settings, txt []rune) []shaped.Run {
	outs := sh.ShapeTextOutput(tx, tsty, rts, txt)
	runs := make([]shaped.Run, len(outs))
	for i := range outs {
		run := &Run{Output: outs[i]}
		si, _, _ := tx.Index(run.Runes().Start)
		sty, stx := tx.Span(si)
		run.SetFromStyle(sty, tsty)
		if sty.Special == rich.Math {
			sh.ShapeMathRun(run, sty, tsty, stx)
		}
		runs[i] = run
	}
	return runs
}

// ShapeTextOutput shapes the spans in the given text using given style and settings,
// returning raw go-text [shaping.Output].
func (sh *Shaper) ShapeTextOutput(tx rich.Text, tsty *text.Style, rts *rich.Settings, txt []rune) []shaping.Output {
	if tx.Len() == 0 {
		return nil
	}
	sty := rich.NewStyle()
	sh.outBuff = sh.outBuff[:0]
	for si, s := range tx {
		in := shaping.Input{}
		start, end := tx.Range(si)
		stx := sty.FromRunes(s) // sets sty, returns runes for span
		if len(stx) == 0 {
			continue
		}
		if sty.Special == rich.Math {
			o := shaping.Output{}
			o.Runes.Offset = start
			o.Runes.Count = end - start
			// todo: need to render this first and get bb, but we don't have
			// any place to stick it yet, b/c just have output and not run.
			// it is possible go-text may include an Output interface so we
			// could stick it there.
			// o.Advance = math32.ToFixed(bb.Size().X)
			sh.outBuff = append(sh.outBuff, o)
			si++ // skip the end special
			continue
		}
		q := StyleToQuery(sty, tsty, rts)
		sh.fontMap.SetQuery(q)

		in.Text = txt
		in.RunStart = start
		in.RunEnd = end
		in.Direction = shaped.GoTextDirection(sty.Direction, tsty)
		fsz := tsty.FontSize.Dots * sty.Size
		in.Size = math32.ToFixed(fsz)
		in.Script = rts.Script
		in.Language = rts.Language

		ins := sh.splitter.Split(in, sh.fontMap) // this is essential
		for _, in := range ins {
			if in.Face == nil {
				fmt.Println("nil face in input", len(stx), string(stx))
				// fmt.Printf("nil face for in: %#v\n", in)
				continue
			}
			o := sh.shaper.Shape(in)
			sh.outBuff = append(sh.outBuff, o)
		}
	}
	return sh.outBuff
}

// ShapeMathRun runs tex math to get path for math special
func (sh *Shaper) ShapeMathRun(run *Run, sty *rich.Style, tsty *text.Style, stx []rune) {
	p := errors.Log1(tex.ParseLaTeX(string(stx), tsty.FontSize.Dots*sty.Size))
	run.Path = p
	bb := p.FastBounds()
	run.MaxBounds = bb
	run.Output.Advance = math32.ToFixed(bb.Size().X)
}

// todo: do the paragraph splitting!  write fun in rich.Text

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
func StyleToQuery(sty *rich.Style, tsty *text.Style, rts *rich.Settings) fontscan.Query {
	q := fontscan.Query{}
	fam := tsty.FontFamily(sty)
	q.Families = rich.FamiliesToList(fam)
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
