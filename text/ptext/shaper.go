// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ptext

import (
	"os"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/shaping"
)

// Shaper is the text shaper, from go-text/shaping.
type Shaper struct {
	shaping.HarfbuzzShaper

	FontMap *fontscan.FontMap
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
	runs := &Runs{Spans: sp}

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
		in.Size = math32.ToFixed(sty.FontSize(ctx))
		in.Script = ctx.Script
		in.Language = ctx.Language

		// todo: per gio:
		// inputs := s.splitBidi(input)
		// inputs = s.splitByFaces(inputs, s.splitScratch1[:0])
		// inputs = splitByScript(inputs, lcfg.Direction, s.splitScratch2[:0])
		ins := shaping.SplitByFace(in, sh.FontMap) // todo: can't pass buffer here
		for _, i := range ins {
			o := sh.HarfbuzzShaper.Shape(i)
			run.Subs = append(run.Subs, o)
		}
		runs.Runs = append(runs.Runs, run)
	}
	return runs
}

func StyleToQuery(sty *rich.Style, ctx *rich.Context) fontscan.Query {
	q := fontscan.Query{}
	q.Families = rich.FamiliesToList(sty.FontFamily(ctx))
	q.Aspect = StyleToAspect(sty)
	return q
}

func StyleToAspect(sty *rich.Style) font.Aspect {
	as := font.Aspect{}
	as.Style = font.Style(sty.Slant)
	as.Weight = font.Weight(sty.Weight)
	as.Stretch = font.Stretch(sty.Stretch)
	return as
}
