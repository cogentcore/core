// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"image"
	"image/color"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestZIndexFlexItem tests that a flex item with a z-index, which is not
// positioned, is rendered over the items after it with lower ones, as in
// CSS (as the site header of pkg.go.dev, whose dropdown menus are over
// the sticky header of the page, a grid item with a lower z-index), and
// that the z-index of a static element that is not a flex or grid item
// has no effect.
func TestZIndexFlexItem(t *testing.T) {
	src := `<div style="display: flex; flex-direction: column; width: 300px">` +
		`<header id="header" style="z-index: 20; background: gray"><div style="position: relative; width: 100px; height: 40px">` +
		`<div id="menu" style="position: absolute; top: 40px; width: 100px; height: 100px; background: red"></div></div></header>` +
		`<main id="main" style="z-index: 10; height: 200px; background: blue"><div id="block" style="z-index: 5">text</div></main></div>`
	bd := core.NewBody()
	require.NoError(t, ReadHTMLString(NewContext(), bd, src))
	ws := map[string]*core.WidgetBase{}
	bd.WalkDown(func(k tree.Node) bool {
		if wb := core.AsWidget(k); wb != nil {
			if id, ok := wb.Property("id").(string); ok {
				ws[id] = wb
			}
		}
		return tree.Continue
	})
	var c image.Point
	img := bd.RenderImage(func() {}, func() {
		require.NotNil(t, ws["menu"])
		bb := ws["menu"].Geom.TotalBBox
		c = bb.Min.Add(bb.Size().Div(2))
		assert.Equal(t, 20, ws["header"].Styles.ZIndex)
		assert.Equal(t, 0, ws["block"].Styles.ZIndex, "the z-index of a block in a block has no effect")
	})
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, color.RGBAModel.Convert(img.At(c.X, c.Y)), "the menu is over the main")
}
