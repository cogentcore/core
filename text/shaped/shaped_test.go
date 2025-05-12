// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped_test

import (
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	_ "cogentcore.org/core/paint/renderers"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/htmltext"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/runes"
	"cogentcore.org/core/text/shaped"
	. "cogentcore.org/core/text/shaped"
	_ "cogentcore.org/core/text/shaped/shapers"
	_ "cogentcore.org/core/text/tex"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
	"github.com/go-text/typesetting/language"
	"github.com/stretchr/testify/assert"
)

// RunTest makes a rendering state, paint, and image with the given size, calls the given
// function, and then asserts the image using [imagex.Assert] with the given name.
func RunTest(t *testing.T, nm string, width int, height int, f func(pc *paint.Painter, sh Shaper, tsty *text.Style, rts *rich.Settings)) {
	rts := &rich.Settings{}
	rts.Defaults()
	uc := units.Context{}
	uc.Defaults()
	tsty := text.NewStyle()
	tsty.ToDots(&uc)
	// fmt.Println("fsz:", tsty.FontSize.Dots)
	pc := paint.NewPainter(width, height)
	pc.FillBox(math32.Vector2{}, math32.Vec2(float32(width), float32(height)), colors.Uniform(colors.White))
	sh := shaped.NewShaper()
	f(pc, sh, tsty, rts)
	pc.RenderToImage()
	imagex.Assert(t, pc.RenderImage(), nm)
}

func TestBasic(t *testing.T) {
	RunTest(t, "basic", 300, 300, func(pc *paint.Painter, sh Shaper, tsty *text.Style, rts *rich.Settings) {

		src := "The lazy fox typed in some familiar text"
		sr := []rune(src)

		plain := rich.NewStyle()
		ital := rich.NewStyle().SetSlant(rich.Italic).SetFillColor(colors.Red)
		boldBig := rich.NewStyle().SetWeight(rich.Bold).SetSize(1.5)
		ul := rich.NewStyle()
		ul.Decoration.SetFlag(true, rich.Underline)

		tx := rich.NewText(boldBig, sr[:4])
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
		// pc.FillBox(pos, math32.Vec2(200, 50), colors.Uniform(color.RGBA{0, 128, 0, 128}))
		pc.TextLines(lns, pos)
		pc.RenderToImage()

		assert.Equal(t, len(src), lns.RuneFromLinePos(textpos.Pos{3, 30}))

		for ri, _ := range src {
			lp := lns.RuneToLinePos(ri)
			assert.Equal(t, ri, lns.RuneFromLinePos(lp))

			// fmt.Println("\n####", ri, string(r))
			gb := lns.RuneBounds(ri)
			assert.NotEqual(t, gb, (math32.Box2{}))
			if gb == (math32.Box2{}) {
				break
			}
			gb = gb.Translate(pos)
			cp := gb.Center()
			cp.X += 1
			si := lns.RuneAtPoint(cp, pos)
			// fmt.Println(cp, si)
			// if ri != si {
			// 	fmt.Println(ri, si, gb, cp, lns.RuneBounds(si))
			// }
			assert.Equal(t, ri, si)
		}
	})
}

func TestHebrew(t *testing.T) {
	RunTest(t, "hebrew", 300, 300, func(pc *paint.Painter, sh Shaper, tsty *text.Style, rts *rich.Settings) {

		tsty.Direction = rich.RTL
		tsty.FontSize.Dots *= 1.5

		src := "אָהַבְתָּ אֵת יְיָ | אֱלֹהֶיךָ, בְּכָל-לְבָֽבְךָ, Let there be light וּבְכָל-נַפְשְׁךָ,"
		sr := []rune(src)
		plain := rich.NewStyle()
		tx := rich.NewText(plain, sr)

		lns := sh.WrapLines(tx, plain, tsty, rts, math32.Vec2(250, 250))
		pc.TextLines(lns, math32.Vec2(20, 60))
		pc.RenderToImage()
	})
}

func TestVertical(t *testing.T) {
	RunTest(t, "nihongo_ttb", 300, 300, func(pc *paint.Painter, sh Shaper, tsty *text.Style, rts *rich.Settings) {
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
		pc.RenderToImage()
	})

	RunTest(t, "nihongo_ltr", 300, 300, func(pc *paint.Painter, sh Shaper, tsty *text.Style, rts *rich.Settings) {
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
		pc.RenderToImage()
	})
}

func TestColors(t *testing.T) {
	RunTest(t, "colors", 300, 300, func(pc *paint.Painter, sh Shaper, tsty *text.Style, rts *rich.Settings) {
		tsty.FontSize.Dots *= 4

		stroke := rich.NewStyle().SetStrokeColor(colors.Red).SetBackground(colors.ToUniform(colors.Scheme.Select.Container))
		big := *stroke
		big.SetSize(1.5)

		src := "The lazy fox"
		sr := []rune(src)
		tx := rich.NewText(stroke, sr[:4])
		tx.AddSpan(&big, sr[4:8]).AddSpan(stroke, sr[8:])

		lns := sh.WrapLines(tx, stroke, tsty, rts, math32.Vec2(250, 250))
		pc.TextLines(lns, math32.Vec2(20, 10))
		pc.RenderToImage()
	})
}

func TestLink(t *testing.T) {
	RunTest(t, "link", 300, 300, func(pc *paint.Painter, sh Shaper, tsty *text.Style, rts *rich.Settings) {
		src := `The <a href="https://example.com">link <b>and <i>it</i> is cool</b></a> and`
		sty := rich.NewStyle()
		tx, err := htmltext.HTMLToRich([]byte(src), sty, nil)
		assert.NoError(t, err)
		lns := sh.WrapLines(tx, sty, tsty, rts, math32.Vec2(250, 250))
		pc.TextLines(lns, math32.Vec2(10, 10))
		pc.RenderToImage()
	})
}

func TestSpacePos(t *testing.T) {
	RunTest(t, "space-pos", 300, 300, func(pc *paint.Painter, sh Shaper, tsty *text.Style, rts *rich.Settings) {
		src := `The and`
		sty := rich.NewStyle()
		tx := rich.NewText(sty, []rune(src))
		lns := sh.WrapLines(tx, sty, tsty, rts, math32.Vec2(250, 250))
		pos := math32.Vec2(10, 10)
		pc.TextLines(lns, pos)
		pc.RenderToImage()

		sb := lns.RuneBounds(3)
		// fmt.Println("sb:", sb)

		cp := sb.Center().Add(pos)
		si := lns.RuneAtPoint(cp, pos)
		// fmt.Println(si)
		assert.Equal(t, 3, si)
	})
}

func TestLinefeed(t *testing.T) {
	RunTest(t, "linefeed", 300, 300, func(pc *paint.Painter, sh Shaper, tsty *text.Style, rts *rich.Settings) {
		src := "Text2D can put <b>HTML</b> <br>formatted Text anywhere you might <i>want</i>"
		sty := rich.NewStyle()
		tx, err := htmltext.HTMLToRich([]byte(src), sty, nil)
		// fmt.Println(tx)
		assert.NoError(t, err)
		lns := sh.WrapLines(tx, sty, tsty, rts, math32.Vec2(250, 250))
		pos := math32.Vec2(10, 10)
		pc.TextLines(lns, pos)
		pc.RenderToImage()

		// sb := lns.RuneBounds(3)
		// // fmt.Println("sb:", sb)
		// cp := sb.Center().Add(pos)
		// si := lns.RuneAtPoint(cp, pos)
		// fmt.Println(si)
		// assert.Equal(t, 3, si)
	})
}

func TestLineCentering(t *testing.T) {
	RunTest(t, "linecentering", 300, 300, func(pc *paint.Painter, sh Shaper, tsty *text.Style, rts *rich.Settings) {
		src := "This is Line Centering"
		// src := "aceg"
		sty := rich.NewStyle()
		tsty.LineHeight = 3
		tx := rich.NewText(sty, []rune(src))
		lns := sh.WrapLines(tx, sty, tsty, rts, math32.Vec2(250, 250))
		pos := math32.Vec2(10, 10)
		pc.TextLines(lns, pos)
	})
}

func TestEmoji(t *testing.T) {
	RunTest(t, "emoji", 300, 300, func(pc *paint.Painter, sh Shaper, tsty *text.Style, rts *rich.Settings) {
		// src := "the " + "🧁" + "cake ✌️"
		// src := "the " + "🧁"
		src := "🧁"
		// src :=
		sty := rich.NewStyle()
		sty.Family = rich.SansSerif
		rts.SansSerif = "Noto Color Emoji"
		sty.Size = 3
		tx := rich.NewText(sty, []rune(src))
		lns := sh.WrapLines(tx, sty, tsty, rts, math32.Vec2(250, 250))
		pos := math32.Vec2(10, 10)
		pc.TextLines(lns, pos)
	})
}

func TestMathInline(t *testing.T) {
	tests := []struct {
		name string
		math string
	}{
		{`simple`, `y = f(x^2)`},
		{`frac`, `y = \frac{1}{N} \left( \sum_{i=0}^{100} \frac{f(x^2)}{\sum x^2} \right)`},
	}
	for _, test := range tests {
		// if test.name != "sqrt-all" {
		// 	continue
		// }
		fnm := "math-inline-" + test.name
		RunTest(t, fnm, 300, 300, func(pc *paint.Painter, sh Shaper, tsty *text.Style, rts *rich.Settings) {
			src := test.math
			sty := rich.NewStyle()
			tx := rich.NewText(sty, []rune("math: "))
			tx.AddMathInline(sty, src)
			tx.AddSpan(sty, []rune(" and we should check line wrapping too"))
			lns := sh.WrapLines(tx, sty, tsty, rts, math32.Vec2(250, 250))
			pos := math32.Vec2(10, 10)
			pc.TextLines(lns, pos)
		})
	}
}

func TestMathDisplay(t *testing.T) {
	tests := []struct {
		name string
		math string
	}{
		{`simple`, `y = f(x^2)`},
		{`frac`, `y = \frac{1}{N} \left( \sum_{i=0}^{100} \frac{f(x^2)}{\sum x^2} \right)`},
	}
	for _, test := range tests {
		// if test.name != "sqrt-all" {
		// 	continue
		// }
		fnm := "math-display-" + test.name
		RunTest(t, fnm, 300, 300, func(pc *paint.Painter, sh Shaper, tsty *text.Style, rts *rich.Settings) {
			src := test.math
			sty := rich.NewStyle()
			var tx rich.Text
			tx.AddMathDisplay(sty, src)
			lns := sh.WrapLines(tx, sty, tsty, rts, math32.Vec2(250, 250))
			pos := math32.Vec2(10, 10)
			pc.TextLines(lns, pos)
		})
	}
}
