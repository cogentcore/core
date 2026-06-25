// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tex_test

import (
	"fmt"
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
	sz := math32.Vec2(float32(width), float32(height))
	pc := paint.NewPainter(sz)
	pc.FillBox(math32.Vector2{}, sz, colors.Uniform(colors.White))
	f(pc)
	img := paint.RenderToImage(pc)
	imagex.Assert(t, img, nm)
}

func TestTex(t *testing.T) {
	tests := []struct {
		name string
		tex  string
	}{
		{`abs-text`, `|x|`},
		{`dot-text`, `\dot x`},
		{`ddot-text`, `\ddot x`},
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
		{`partial-text`, `y = \partial x`},
		{`partial-disp`, `$\frac{\partial^2f}{\partial x^2}$`},
		{`array-disp`, `$\Phi = \left(\begin{array}{c}
\Phi^+ \\
\Phi^0 \\
\end{array}\right)$`},
		{`neq-text`, `a \neq b`},
		{`hat-text`, `\hat{p}`},
		{`hbar-text`, `\hat{p} = -i \hbar \vec{\nabla}`},
		{`binom-disp`, `$\binom{n}{k} = \frac{n!}{k!(n-k)!}$`},
		{`bigfrac-disp`, `$\frac{1+\frac{a}{b}} {\displaystyle 1+\frac{1}{1+\frac{1}{a}}}$`},
		{`cfrac-disp`, `$a_0+\cfrac{1}{a_1+\cfrac{1}{a_2+\cfrac{1}{a_3+\cdots}}}$`},
		{`lim-text`, `\lim_{h \to 0} (x-h)`},
		{`lim-disp`, `$\lim_{h \to 0 } \frac{f(x+h)-f(x)}{h}$`},
		{`forall-disp`, `$\forall x \in \mathbf{R}: \qquad x^{2} \geq 0$`},
		{`forall2-disp`, `$x^{2} \geq 0\qquad \text{for all }x\in\mathbf{R}$`},
		{`expers-disp`, `$p^3_{ij} \; m_\text{Knuth} \; \sum_{k=1}^3 k \\[5pt] a^x+y \neq a^{x+y} \; e^{x^2} \neq {e^x}^2$`},
		{`sqrt-disp`, `$\sqrt{p^2}$`},
		{`roots-disp`, `$\sqrt{x} \Leftrightarrow x^{1/2} \; \sqrt[3]{2} \; \sqrt{x^{2} + \sqrt{y}} \; \surd[x^2 + y^2]$`},
		{`lines-disp`, `$0.\overline{3} = \underline{\underline{1/3}}$`},
		{`underbrace-disp`, `$\underbrace{\overbrace{a+b+c}^6 \cdot \overbrace{d+e+f}^7}_\text{meaning of life} = 42$`},
		{`widehat-disp`, `$f''(x) = 2 \hat{XY} \quad \widehat{XY} \quad \bar{x_0} \quad \bar{x}_0$`},
		{`prime-text`, `$f''(x)$`},
		{`vecs-disp`, `$\vec{a} \qquad \vec{AB} \qquad \overrightarrow{AB}$`},
		{`stackrel-text`, `f_n(x) \stackrel{*}{\approx} 1`},
		{`sum-big-disp`, `$\sum^n_{\substack{0<i<n \\ j \subseteq i}}P(i,j) = Q(i,j)$`},
		{`left-right-disp`, `$1 + \left(\frac{1}{1-x^{2}} \right)^3 \qquad \left. \ddagger \frac{.}{.} \right)$`},
		{`bigg-disp`, `$\Big( \bigg( \Bigg( \quad \big\} \Big\} \bigg\} \Bigg\} \quad
\bigg\Downarrow \Bigg\Downarrow$`},
		{`matrix-disp`, `$\mathbf{X} = \left(
\begin{array}{ccc}
x_1 & x_2 & \ldots \\
x_3 & x_4 & \ldots \\
\vdots & \vdots & \ddots
\end{array} \right)$`},
		{`conds-disp`, `$|x| = \left\{
\begin{array}{rl}
-x & \text{if } x < 0,\\
0 & \text{if } x = 0,\\
x & \text{if } x > 0.
\end{array} \right.$`},
		{`cases-disp`, `$|x| =
\begin{cases}
-x & \text{if } x < 0,\\
0 & \text{if } x = 0,\\
x & \text{if } x > 0.
\end{cases}$`},
		{`matrix-ams-disp`, `$\begin{matrix}
1 & 2 \\
3 & 4
\end{matrix} \qquad
\begin{bmatrix}
p_{11} & p_{12} & \ldots
& p_{1n} \\
p_{21} & p_{22} & \ldots
& p_{2n} \\
\vdots & \vdots & \ddots
& \vdots \\
p_{m1} & p_{m2} & \ldots
& p_{mn}
\end{bmatrix}$`},
		{`phantom-disp`, `${}^{14}_{6}\text{C}
\qquad \text{versus} \qquad
{}^{14}_{\phantom{1}6}\text{C}$`},
		{`real-disp`, `$\Re \qquad \mathcal{R}$`},
		// {``, `$$`},
		// {``, `$$`},
		// {``, `$$`},
		// {``, `$$`},
		// {``, `$$`},
	}

	for _, test := range tests {
		// Debug = true
		// if test.name != "sqrt-disp" {
		// 	continue
		// }
		RunTest(t, test.name, 400, 150, func(pc *paint.Painter) {
			fmt.Println("\n\n#### ", test.name)
			pc.Fill.Color = colors.Uniform(color.Black)
			// fmt.Println("font size dots:", pc.Text.FontSize.Dots)
			pp, err := LaTeXMath(test.tex, pc.Text.FontSize.Dots)
			assert.NoError(t, err)
			assert.NotNil(t, pp)
			pp = pp.Translate(0, 40)
			pc.State.Path = pp
			pc.Draw()
			// reference text
			// sh := shaped.NewShaper()
			// tx := rich.NewText(&pc.Font, []rune("a=x"))
			// lns := sh.WrapLines(tx, &pc.Font, &pc.Text, &rich.Settings, math32.Vec2(1000, 50))
			// pc.DrawText(lns, math32.Vec2(0, 70))
		})
		// break
	}
}

func TestRelations(t *testing.T) {
	tests := []string{
		`<`, `>`, `=`, `\leq`, `\le`, `\geq`, `\ge`, `\equiv`,
		`\ll`, `\gg`, `\doteq`, `\prec`, `\succ`, `\sim`,
		`\preceq`, `\succeq`, `\simeq`, `\subset`, `\supset`, `\approx`,
		`\subseteq`, `\supseteq`, `\cong`, `\sqsubseteq`, `\sqsupseteq`,
		`\bowtie`, `\in`, `\ni`, `\owns`, `\propto`, `\vdash`, `\dashv`,
		`\models`, `\mid`, `\parallel`, `\perp`, `\smile`, `\frown`,
		`\asymp`, `:`, `\notin`, `\neq`, `\ne`,
	}
	// `\sqsubset`, `\sqsupset`, `\Join`, // latexsym
	width := 600
	RunTest(t, "all-relations", width, 300, func(pc *paint.Painter) {
		pc.Fill.Color = colors.Uniform(color.Black)
		fsize := pc.Text.FontSize.Dots
		y := fsize
		x := 0.5 * fsize
		for _, test := range tests {
			// Debug = true
			// if test != `\mid` {
			// 	continue
			// }
			pp, err := LaTeXMath(test, fsize)
			assert.NoError(t, err)
			assert.NotNil(t, pp)
			pp = pp.Translate(x, y)
			pc.State.Path = pp
			pc.Draw()

			x += fsize * 1.5
			if len(test) > 1 {
				pp, err = LaTeXMath(`\backslash \text{`+test[1:]+`}`, fsize)
				assert.NoError(t, err)
				assert.NotNil(t, pp)
				pp = pp.Translate(x, y)
				pc.State.Path = pp
				pc.Draw()
			}
			x += 8 * fsize
			if x > float32(width) {
				y += fsize * 1.5
				x = 0.5 * fsize
			}
		}
	})
}

func TestOperators(t *testing.T) {
	tests := []string{
		`+`, `-`, `\pm`, `\mp`, `\triangleleft`, `\cdot`, `\div`,
		`\triangleright`, `\times`, `\setminus`, `\star`, `\cup`,
		`\cap`, `\ast`, `\sqcup`, `\sqcap`, `\circ`, `\vee`, `\lor`,
		`\wedge`, `\land`, `\bullet`, `\oplus`, `\ominus`, `\diamond`,
		`\odot`, `\oslash`, `\uplus`, `\otimes`, `\bigcirc`,
		`\amalg`, `\bigtriangleup`, `\dagger`, `\ddagger`, `\wr`,
		`\bigtriangledown`,
	}
	width := 600
	RunTest(t, "all-operators", width, 300, func(pc *paint.Painter) {
		pc.Fill.Color = colors.Uniform(color.Black)
		fsize := pc.Text.FontSize.Dots
		y := fsize
		x := 0.5 * fsize
		for _, test := range tests {
			// Debug = true
			// if test != `\ast` {
			// 	continue
			// }
			pp, err := LaTeXMath(test, fsize)
			assert.NoError(t, err)
			assert.NotNil(t, pp)
			pp = pp.Translate(x, y)
			pc.State.Path = pp
			pc.Draw()

			x += fsize * 1.5
			if len(test) > 1 {
				pp, err = LaTeXMath(`\backslash \text{`+test[1:]+`}`, fsize)
				assert.NoError(t, err)
				assert.NotNil(t, pp)
				pp = pp.Translate(x, y)
				pc.State.Path = pp
				pc.Draw()
			}
			x += 8 * fsize
			if x > float32(width) {
				y += fsize * 1.5
				x = 0.5 * fsize
			}
		}
	})
}

func TestBigOperators(t *testing.T) {
	tests := []string{
		`\sum`, `\bigcup`, `\bigvee`, `\prod`, `\bigcap`, `\bigwedge`, `\coprod`,
		`\bigsqcup`, `\biguplus`, `\int`, `\oint`, `\bigodot`,
		`\bigoplus`, `\bigotimes`,
	}
	width := 600
	RunTest(t, "big-operators", width, 300, func(pc *paint.Painter) {
		pc.Fill.Color = colors.Uniform(color.Black)
		fsize := pc.Text.FontSize.Dots
		y := fsize
		x := 0.5 * fsize
		for _, test := range tests {
			// Debug = true
			// if test != `\biguplus` {
			// 	continue
			// }
			pp, err := LaTeXMath(`$`+test+`$`, fsize)
			assert.NoError(t, err)
			assert.NotNil(t, pp)
			pp = pp.Translate(x, y)
			pc.State.Path = pp
			pc.Draw()

			x += fsize * 2
			if len(test) > 1 {
				pp, err = LaTeXMath(`\backslash \text{`+test[1:]+`}`, fsize)
				assert.NoError(t, err)
				assert.NotNil(t, pp)
				pp = pp.Translate(x, y+fsize)
				pc.State.Path = pp
				pc.Draw()
			}
			x += 8 * fsize
			if x > float32(width) {
				y += fsize * 2.5
				x = 0.5 * fsize
			}
		}
	})
}

func TestArrows(t *testing.T) {
	tests := []string{
		`\leftarrow`, `\gets`, `\longleftarrow`, `\rightarrow`, `\to`, `\longrightarrow`,
		`\leftrightarrow`, `\longleftrightarrow`, `\Leftarrow`, `\Longleftarrow`,
		`\Rightarrow`, `\Longrightarrow`, `\Leftrightarrow`, `\Longleftrightarrow`,
		`\mapsto`, `\longmapsto`, `\hookleftarrow`, `\hookrightarrow`, `\leftharpoonup`,
		`\rightharpoonup`, `\leftharpoondown`, `\rightharpoondown`, `\rightleftharpoons`,
		`\iff`, `\uparrow`, `\downarrow`, `\updownarrow`, `\Uparrow`, `\Downarrow`,
		`\Updownarrow`, `\nearrow`, `\searrow`, `\swarrow`, `\nwarrow`,
	}
	// todo: mapsto and hook are sent as double chars, rendering both
	width := 600
	RunTest(t, "all-arrows", width, 500, func(pc *paint.Painter) {
		pc.Fill.Color = colors.Uniform(color.Black)
		fsize := pc.Text.FontSize.Dots
		y := fsize
		x := 0.5 * fsize
		for _, test := range tests {
			// Debug = true
			// if test != `\mapsto` {
			// 	continue
			// }
			pp, err := LaTeXMath(test, fsize)
			assert.NoError(t, err)
			assert.NotNil(t, pp)
			pp = pp.Translate(x, y)
			pc.State.Path = pp
			pc.Draw()

			x += fsize * 3
			if len(test) > 1 {
				pp, err = LaTeXMath(`\backslash \text{`+test[1:]+`}`, fsize)
				assert.NoError(t, err)
				assert.NotNil(t, pp)
				pp = pp.Translate(x, y)
				pc.State.Path = pp
				pc.Draw()
			}
			x += 18 * fsize
			if x > float32(width) {
				y += fsize * 1.5
				x = 0.5 * fsize
			}
		}
	})
}

func TestSymbols(t *testing.T) {
	tests := []string{
		`\dots`, `\cdots`, `\vdots`, `\ddots`, `\hbar`, `\imath`, `\jmath`, `\ell`,
		`\Re`, `\Im`, `\aleph`, `\wp`, `\forall`, `\exists`,
		`\partial`, `'`, `\prime`, `\emptyset`, `\infty`,
		`\nabla`, `\triangle`, `\bot`, `\top`, `\angle`,
		`\surd`, `\diamondsuit`, `\heartsuit`, `\clubsuit`, `\spadesuit`, `\neg`, `\lnot`,
		`\flat`, `\natural`, `\sharp`,
	}
	width := 600
	RunTest(t, "all-symbols", width, 300, func(pc *paint.Painter) {
		pc.Fill.Color = colors.Uniform(color.Black)
		fsize := pc.Text.FontSize.Dots
		y := fsize
		x := 0.5 * fsize
		for _, test := range tests {
			// Debug = true
			// if test != `\mid` {
			// 	continue
			// }
			pp, err := LaTeXMath(test, fsize)
			assert.NoError(t, err)
			assert.NotNil(t, pp)
			pp = pp.Translate(x, y)
			pc.State.Path = pp
			pc.Draw()

			x += fsize * 1.5
			if len(test) > 1 {
				pp, err = LaTeXMath(`\backslash \text{`+test[1:]+`}`, fsize)
				assert.NoError(t, err)
				assert.NotNil(t, pp)
				pp = pp.Translate(x, y)
				pc.State.Path = pp
				pc.Draw()
			}
			x += 8 * fsize
			if x > float32(width) {
				y += fsize * 2
				x = 0.5 * fsize
			}
		}
	})
}
