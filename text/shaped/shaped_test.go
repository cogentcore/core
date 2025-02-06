// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped_test

import (
	"image/color"
	"os"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/runes"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/renderers/rasterx"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	. "cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
	"github.com/go-text/typesetting/language"
)

func TestMain(m *testing.M) {
	// ptext.FontLibrary.InitFontPaths(ptext.FontPaths...)
	paint.NewDefaultImageRenderer = rasterx.New
	os.Exit(m.Run())
}

// RunTest makes a rendering state, paint, and image with the given size, calls the given
// function, and then asserts the image using [imagex.Assert] with the given name.
func RunTest(t *testing.T, nm string, width int, height int, f func(pc *paint.Painter, sh *Shaper, tsty *text.Style, rts *rich.Settings)) {
	rts := &rich.Settings{}
	rts.Defaults()
	uc := units.Context{}
	uc.Defaults()
	tsty := text.NewStyle()
	tsty.ToDots(&uc)
	// fmt.Println("fsz:", tsty.FontSize.Dots)
	pc := paint.NewPainter(width, height)
	pc.FillBox(math32.Vector2{}, math32.Vec2(float32(width), float32(height)), colors.Uniform(colors.White))
	sh := NewShaper()
	f(pc, sh, tsty, rts)
	pc.RenderDone()
	imagex.Assert(t, pc.RenderImage(), nm)
}

func TestBasic(t *testing.T) {
	RunTest(t, "basic", 300, 300, func(pc *paint.Painter, sh *Shaper, tsty *text.Style, rts *rich.Settings) {

		src := "The lazy fox typed in some familiar text"
		sr := []rune(src)

		plain := rich.NewStyle()
		ital := rich.NewStyle().SetSlant(rich.Italic).SetFillColor(colors.Red)
		boldBig := rich.NewStyle().SetWeight(rich.Bold).SetSize(1.5)
		ul := rich.NewStyle()
		ul.Decoration.SetFlag(true, rich.Underline)

		tx := rich.NewText(plain, sr[:4])
		tx.AddSpan(ital, sr[4:8])
		fam := []rune("familiar")
		ix := runes.Index(sr, fam)
		tx.AddSpan(ul, sr[8:ix])
		tx.AddSpan(boldBig, sr[ix:ix+8])
		tx.AddSpan(ul, sr[ix+8:])

		lns := sh.WrapLines(tx, plain, tsty, rts, math32.Vec2(250, 250))
		lns.SelectRegion(textpos.Range{7, 30})
		lns.SelectRegion(textpos.Range{34, 40})
		pos := math32.Vec2(20, 60)
		pc.FillBox(pos, math32.Vec2(200, 50), colors.Uniform(color.RGBA{0, 128, 0, 128}))
		pc.TextLines(lns, pos)
		pc.RenderDone()
	})
}

func TestHebrew(t *testing.T) {
	RunTest(t, "hebrew", 300, 300, func(pc *paint.Painter, sh *Shaper, tsty *text.Style, rts *rich.Settings) {

		tsty.Direction = rich.RTL
		tsty.FontSize.Dots *= 1.5

		src := "אָהַבְתָּ אֵת יְיָ | אֱלֹהֶיךָ, בְּכָל-לְבָֽבְךָ, Let there be light וּבְכָל-נַפְשְׁךָ,"
		sr := []rune(src)
		plain := rich.NewStyle()
		tx := rich.NewText(plain, sr)

		lns := sh.WrapLines(tx, plain, tsty, rts, math32.Vec2(250, 250))
		pc.TextLines(lns, math32.Vec2(20, 60))
		pc.RenderDone()
	})
}

func TestVertical(t *testing.T) {
	RunTest(t, "nihongo_ttb", 300, 300, func(pc *paint.Painter, sh *Shaper, tsty *text.Style, rts *rich.Settings) {
		rts.Language = "ja"
		rts.Script = language.Han
		tsty.Direction = rich.TTB // rich.BTT // note: apparently BTT is actually never used
		tsty.FontSize.Dots *= 1.5

		plain := rich.NewStyle()

		// todo: word wrapping and sideways rotation in vertical not currently working
		// src := "国際化活動 W3C ワールド・ワイド・Hello!"
		// src := "国際化活動 Hello!"
		src := "国際化活動"
		sr := []rune(src)
		tx := rich.NewText(plain, sr)

		lns := sh.WrapLines(tx, plain, tsty, rts, math32.Vec2(150, 50))
		// pc.TextLines(lns, math32.Vec2(100, 200))
		pc.TextLines(lns, math32.Vec2(60, 100))
		pc.RenderDone()
	})

	RunTest(t, "nihongo_ltr", 300, 300, func(pc *paint.Painter, sh *Shaper, tsty *text.Style, rts *rich.Settings) {
		rts.Language = "ja"
		rts.Script = language.Han
		tsty.FontSize.Dots *= 1.5

		// todo: word wrapping and sideways rotation in vertical not currently working
		src := "国際化活動 W3C ワールド・ワイド・Hello!"
		sr := []rune(src)
		plain := rich.NewStyle()
		tx := rich.NewText(plain, sr)

		lns := sh.WrapLines(tx, plain, tsty, rts, math32.Vec2(250, 250))
		pc.TextLines(lns, math32.Vec2(20, 60))
		pc.RenderDone()
	})
}

func TestColors(t *testing.T) {
	RunTest(t, "colors", 300, 300, func(pc *paint.Painter, sh *Shaper, tsty *text.Style, rts *rich.Settings) {
		tsty.FontSize.Dots *= 4

		stroke := rich.NewStyle().SetStrokeColor(colors.Red).SetBackground(colors.ToUniform(colors.Scheme.Select.Container))
		big := *stroke
		big.SetSize(1.5)

		src := "The lazy fox"
		sr := []rune(src)
		sp := rich.NewText(stroke, sr[:4])
		sp.AddSpan(&big, sr[4:8]).AddSpan(stroke, sr[8:])

		lns := sh.WrapLines(sp, stroke, tsty, rts, math32.Vec2(250, 250))
		pc.TextLines(lns, math32.Vec2(20, 80))
		pc.RenderDone()
	})
}
