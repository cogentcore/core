// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"image/color"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// TestSelectColorScheme tests that the selection of text that starts with
// tabs (whose shaped render is cached) has the selection color of the color
// scheme after it changes, as from light to dark mode, in the tabs as well
// as in the text, where the tabs kept the color of the old scheme.
func TestSelectColorScheme(t *testing.T) {
	palette, schemes, scheme, dark := colors.Palette, colors.Schemes, colors.Scheme, matcolor.SchemeIsDark
	defer func() {
		colors.Palette, colors.Schemes, colors.Scheme, matcolor.SchemeIsDark = palette, schemes, scheme, dark
	}()
	colors.SetScheme(false)

	b := core.NewBody()
	ed := NewBase(b)
	ed.Lines.SetString("\t\tfunc a() {}\n\t\tb := 1\n")
	var tabPos, textPos math32.Vector2
	var light color.RGBA
	img := core.RenderImage(b, func() {
		ed.AsyncLock()
		ed.SelectRegion = textpos.Region{Start: textpos.Pos{}, End: textpos.Pos{Line: 1, Char: 6}}
		ed.NeedsRender()
		ed.AsyncUnlock()
	}, func() {
		ed.AsyncLock()
		defer ed.AsyncUnlock()
		sc := math32.FromPoint(ed.Scene.SceneGeom.Pos)
		half := ed.charSize.Mul(math32.Vec2(0.5, 0.5))
		tabPos = ed.charStartPos(textpos.Pos{Line: 0, Char: 0}).Sub(sc).Add(half)
		textPos = ed.charStartPos(textpos.Pos{Line: 0, Char: 7}).Sub(sc).Add(math32.Vec2(half.X, 2)) // above the glyphs
		light = colors.ToUniform(colors.Scheme.Select.Container)
		// the dark mode
		colors.SetScheme(true)
		ed.Restyle()
	})
	at := func(p math32.Vector2) color.RGBA { return img.RGBAAt(int(p.X), int(p.Y)) }
	darkSel := colors.ToUniform(colors.Scheme.Select.Container)
	assert.NotEqual(t, light, darkSel)
	assert.Equal(t, darkSel, at(textPos), "the selection of the text")
	assert.Equal(t, darkSel, at(tabPos), "the selection of the tabs")
}
