// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped_test

import (
	"os"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/runes"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/ptext"
	"cogentcore.org/core/paint/renderers/rasterx"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	. "cogentcore.org/core/text/runs"
)

func TestMain(m *testing.M) {
	ptext.FontLibrary.InitFontPaths(ptext.FontPaths...)
	paint.NewDefaultImageRenderer = rasterx.New
	os.Exit(m.Run())
}

// RunTest makes a rendering state, paint, and image with the given size, calls the given
// function, and then asserts the image using [imagex.Assert] with the given name.
func RunTest(t *testing.T, nm string, width int, height int, f func(pc *paint.Painter)) {
	pc := paint.NewPainter(width, height)
	f(pc)
	pc.RenderDone()
	imagex.Assert(t, pc.RenderImage(), nm)
}

func TestSpans(t *testing.T) {
	src := "The lazy fox typed in some familiar text"
	sr := []rune(src)
	sp := rich.Spans{}
	plain := rich.NewStyle()
	ital := rich.NewStyle().SetSlant(rich.Italic)
	ital.SetStrokeColor(colors.Red)
	boldBig := rich.NewStyle().SetWeight(rich.Bold).SetSize(1.5)
	sp.Add(plain, sr[:4])
	sp.Add(ital, sr[4:8])
	fam := []rune("familiar")
	ix := runes.Index(sr, fam)
	sp.Add(plain, sr[8:ix])
	sp.Add(boldBig, sr[ix:ix+8])
	sp.Add(plain, sr[ix+8:])

	ctx := &rich.Context{}
	ctx.Defaults()
	uc := units.Context{}
	uc.Defaults()
	ctx.ToDots(&uc)
	sh := NewShaper()
	runs := sh.Shape(sp, ctx)
	// fmt.Println(runs)

	RunTest(t, "fox_render", 300, 300, func(pc *paint.Painter) {
		pc.FillBox(math32.Vector2{}, math32.Vec2(300, 300), colors.Uniform(colors.White))
		pc.RenderDone()
		rnd := pc.Renderers[0].(*rasterx.Renderer)
		rnd.TextRuns(runs, ctx, math32.Vec2(20, 60))
	})

}
