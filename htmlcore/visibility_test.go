// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"image/color"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// TestHiddenOverContent tests that elements with visibility: hidden or
// opacity: 0 do not paint over the content under them (as the dropdown
// menus of the header of github.com, which are positioned over it).
func TestHiddenOverContent(t *testing.T) {
	for _, hide := range []string{"visibility: hidden", "opacity: 0"} {
		b := core.NewBody()
		ctx := NewContext()
		assert.NoError(t, ReadHTMLString(ctx, b, `<div style="position: relative; width: 200px; height: 100px">`+
			`<div style="background: blue; width: 200px; height: 100px"></div>`+
			`<div style="position: absolute; top: 0; left: 0; width: 200px; height: 100px; background: red; `+hide+`">hidden</div></div>`))
		img := b.RenderImage()
		// the body has a margin of 8px
		c := color.RGBAModel.Convert(img.At(int(50*b.Styles.UnitContext.Dots(1)), int(50*b.Styles.UnitContext.Dots(1)))).(color.RGBA)
		assert.Equal(t, color.RGBA{0, 0, 255, 255}, c, hide)
	}
}

// TestTranslate tests the translate functions of the transform property:
// lengths, and percentages of the size of the element itself (as the
// drawers of pages that are moved out of view, and the labels of
// github.com that are centered vertically), which do not change the layout.
func TestTranslate(t *testing.T) {
	assert.Equal(t, [2]string{"10px", "0"}, func() [2]string { x, y, _ := cssTranslate("transform", "translatex(10px)"); return [2]string{x, y} }())
	assert.Equal(t, [2]string{"0", "-50%"}, func() [2]string { x, y, _ := cssTranslate("transform", "translatey(-50%) rotate(45deg)"); return [2]string{x, y} }())
	assert.Equal(t, [2]string{"1px", "2px"}, func() [2]string { x, y, _ := cssTranslate("transform", "translate3d(1px, 2px, 0)"); return [2]string{x, y} }())
	b := core.NewBody()
	ctx := NewContext()
	assert.NoError(t, ReadHTMLString(ctx, b, `<div class="a" style="width: 100px; height: 40px; transform: translateY(-50%) translateX(10px)">a</div>`+
		`<div class="b" style="width: 100px; height: 40px; translate: 20px 5px">b</div><div class="c" style="width: 100px; height: 40px">c</div>`))
	pos := map[string][2]float32{}
	b.RenderImage(func() {
		b.WalkDown(func(n tree.Node) bool {
			if wb := core.AsWidget(n); wb != nil {
				if cl, ok := wb.Property("class").(string); ok {
					dp := wb.Styles.UnitContext.Dots(units.UnitDp)
					pos[cl] = [2]float32{wb.Geom.Pos.Total.X / dp, wb.Geom.Pos.Total.Y / dp}
				}
			}
			return tree.Continue
		})
	})
	assert.InDelta(t, pos["c"][0]+10, pos["a"][0], 0.5)
	assert.InDelta(t, pos["c"][1]-80-20, pos["a"][1], 1.5, "up by half of its height, without changing the layout")
	assert.InDelta(t, pos["c"][0]+20, pos["b"][0], 0.5)
	assert.InDelta(t, pos["c"][1]-40+5, pos["b"][1], 0.5)
}
