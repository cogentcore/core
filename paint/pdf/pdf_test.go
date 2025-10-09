// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package pdf

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/htmltext"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	_ "cogentcore.org/core/text/shaped/shapers"
	_ "cogentcore.org/core/text/tex"
	"github.com/alecthomas/assert/v2"
)

// RunTest runs a test for given test case.
func RunTest(t *testing.T, nm string, width, height float32, f func(pd *PDF, sty *styles.Paint)) {
	ctx := units.NewContext()
	ctx.DPI = 72
	var b bytes.Buffer
	pd := New(&b, width, height, ctx)
	sty := styles.NewPaint()
	sty.UnitContext = *ctx
	f(pd, sty)
	pd.Close()
	os.Mkdir("testdata", 0777)
	os.WriteFile(filepath.Join("testdata", nm)+".pdf", b.Bytes(), 0666)
}

func TestPath(t *testing.T) {
	RunTest(t, "path", 50, 50, func(pd *PDF, sty *styles.Paint) {
		p := ppath.New().Rectangle(0, 0, 30, 20)

		sty.Stroke.Color = colors.Uniform(colors.Blue)
		sty.Fill.Color = colors.Uniform(colors.Red)
		sty.Stroke.Width.Px(2)
		sty.ToDots()

		pd.Path(*p, sty, math32.Translate2D(10, 20))
	})
}

func TestText(t *testing.T) {
	RunTest(t, "text", 300, 300, func(pd *PDF, sty *styles.Paint) {
		prv := UseStandardFonts()
		sh := shaped.NewShaper()

		src := "PDF can put <b>HTML</b> <br>formatted Text where you <i>want</i>"
		rsty := &sty.Font
		tsty := &sty.Text

		tx, err := htmltext.HTMLToRich([]byte(src), rsty, nil)
		// fmt.Println(tx)
		assert.NoError(t, err)
		lns := sh.WrapLines(tx, rsty, tsty, math32.Vec2(250, 250))

		// m := math32.Identity2()
		m := math32.Rotate2D(math32.DegToRad(15))

		pd.Text(sty, m, math32.Vec2(20, 20), lns)
		RestorePreviousFonts(prv)
	})
}

func TestMathInline(t *testing.T) {
	RunTest(t, "math-inline", 300, 300, func(pd *PDF, sty *styles.Paint) {
		prv := UseStandardFonts()
		sh := shaped.NewShaper()

		src := `y = \frac{1}{N} \left( \sum_{i=0}^{100} \frac{f(x^2)}{\sum x^2} \right)`
		rsty := &sty.Font
		tsty := &sty.Text

		tx := rich.NewText(rsty, []rune("math: "))
		tx.AddMathInline(rsty, src)
		tx.AddSpan(rsty, []rune(" and we should check line wrapping too"))
		lns := sh.WrapLines(tx, rsty, tsty, math32.Vec2(250, 250))

		m := math32.Identity2()
		pd.Text(sty, m, math32.Vec2(20, 20), lns)
		RestorePreviousFonts(prv)
	})
}

func TestMathDisplay(t *testing.T) {
	RunTest(t, "math-display", 300, 300, func(pd *PDF, sty *styles.Paint) {
		prv := UseStandardFonts()
		sh := shaped.NewShaper()

		src := `y = \frac{1}{N} \left( \sum_{i=0}^{100} \frac{f(x^2)}{\sum x^2} \right)`
		rsty := &sty.Font
		tsty := &sty.Text

		var tx rich.Text
		tx.AddMathDisplay(rsty, src)
		lns := sh.WrapLines(tx, rsty, tsty, math32.Vec2(250, 250))

		m := math32.Identity2()
		pd.Text(sty, m, math32.Vec2(20, 20), lns)
		RestorePreviousFonts(prv)
	})
}

func TestLinks(t *testing.T) {
	RunTest(t, "links", 300, 300, func(pd *PDF, sty *styles.Paint) {
		prv := UseStandardFonts()
		sh := shaped.NewShaper()
		m := math32.Identity2()
		rsty := &sty.Font
		tsty := &sty.Text
		sz := math32.Vec2(250, 250)

		txt := func(src string, pos math32.Vector2) {
			tx, err := htmltext.HTMLToRich([]byte(src), rsty, nil)
			assert.NoError(t, err)
			lns := sh.WrapLines(tx, rsty, tsty, sz)
			pd.Text(sty, m, pos, lns)
		}
		txt("Some random text here", math32.Vec2(10, 10))
		pd.w.AddAnchor("test", math32.Vec2(10, 10))
		txt(`A <a href="#test">link</a> to that text`, math32.Vec2(10, 30))

		RestorePreviousFonts(prv)
	})
}
