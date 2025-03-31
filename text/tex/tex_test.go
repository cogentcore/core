// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tex_test

import (
	"image/color"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	_ "cogentcore.org/core/paint/renderers"
	. "cogentcore.org/core/text/tex"
	"github.com/stretchr/testify/assert"
)

// RunTest makes a rendering state, paint, and image with the given size, calls the given
// function, and then asserts the image using [imagex.Assert] with the given name.
func RunTest(t *testing.T, nm string, width int, height int, f func(pc *paint.Painter)) {
	pc := paint.NewPainter(width, height)
	pc.FillBox(math32.Vector2{}, math32.Vec2(float32(width), float32(height)), colors.Uniform(colors.White))
	f(pc)
	pc.RenderToImage()
	imagex.Assert(t, pc.RenderImage(), nm)
}

func TestTex(t *testing.T) {
	tests := []struct {
		name string
		tex  string
	}{
		{`sum-text`, `y = \sum_{i=0}^{100} f(x_i)`},
		{`sum-disp`, `$y = \sum_{i=0}^{100} f(x_i)$`},
		{`int-text`, `y = \int_{i=0}^{100} f(x_i)`},
		{`int-disp`, `$y = \int_{i=0}^{100} f(x_i)$`},
		{`ops-text`, `y = \prod_i^j \coprod \int \oint \bigcap \bigcup`},
		{`ops-disp`, `$y = \prod_i^j \coprod \int \oint \bigcap \bigcup$`},
		{`ops2-text`, `y = \bigsqcup \bigvee \bigwedge \bigodot \bigotimes \bigoplus \biguplus`},
		{`ops2-disp`, `$y = \bigsqcup \bigvee \bigwedge \bigodot \bigotimes \bigoplus \biguplus$`},
		{`lb-sum-text`, `y = \left( \sum_{i=0}^{100} f(x_i) \right)`},
		{`lb-sum-disp`, `$y = \left( \sum_{i=0}^{100} f(x_i) \right)$`},
		{`parens-all`, `$\left(\vbox to 27pt{}\left(\vbox to 24pt{}\left(\vbox to 21pt{}
\Biggl(\biggl(\Bigl(\bigl(({\scriptstyle({\scriptscriptstyle(\hskip3pt
)})})\bigr)\Bigr)\biggr)\Biggr)\right)\right)\right)$`},
		{`brackets-all`, `$\left[\vbox to 27pt{}\left[\vbox to 24pt{}\left[\vbox to 21pt{}
\Biggl[\biggl[\Bigl[\bigl[{\scriptstyle[{\scriptscriptstyle[\hskip3pt
]}]}]\bigr]\Bigr]\biggr]\Biggr]\right]\right]\right]$`},
		{`braces-all`, `$\left\{\vbox to 27pt{}\left\{\vbox to 24pt{}\left\{\vbox to 21pt{}
\Biggl\{\biggl\{\Bigl\{\bigl\{\{{\scriptstyle\{{\scriptscriptstyle\{\hskip3pt
\}}\}}\}\bigr\}\Bigr\}\biggr\}\Biggr\}\right\}\right\}\right\}$`},
		{`sqrt-all`, `$\sqrt{1+\sqrt{1+\sqrt{1+\sqrt{1+\sqrt{1+\sqrt{1+\sqrt{1+x}}}}}}}$`},
		{`frac-text`, `a = \left( \frac{\overline{f}(x^2)}{\prod_i^j \sum_i^j f_i(x_j^2)} \right)`},
		{`frac-disp`, `$a = \left( \frac{\overline{f}(x^2)}{\prod_i^j \sum_i^j f_i(x_j^2)} \right)$`},
	}

	for _, test := range tests {
		// if test.name != "sqrt-all" {
		// 	continue
		// }
		RunTest(t, test.name, 400, 150, func(pc *paint.Painter) {
			pc.Fill.Color = colors.Uniform(color.Black)
			// fmt.Println("font size dots:", pc.Text.FontSize.Dots)
			pp, err := ParseLaTeX(test.tex, pc.Text.FontSize.Dots)
			assert.NoError(t, err)
			*pp = pp.Translate(0, 40)
			pc.State.Path = *pp
			pc.PathDone()
			// reference text
			// sh := shaped.NewShaper()
			// tx := rich.NewText(&pc.Font, []rune("a=x"))
			// lns := sh.WrapLines(tx, &pc.Font, &pc.Text, &rich.DefaultSettings, math32.Vec2(1000, 50))
			// pc.TextLines(lns, math32.Vec2(0, 70))
		})
	}
}
