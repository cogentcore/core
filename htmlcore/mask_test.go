// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"image"
	"image/color"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// TestMaskImage tests that an element with a mask-image shows its
// background color through the mask (as the icons of Wikipedia), with
// mask-size, mask-position, and mask-repeat, not a box of the color.
func TestMaskImage(t *testing.T) {
	assert.Equal(t, "a b.svg", maskURL(`url("a b.svg")`))
	assert.Equal(t, "x.svg", maskURL(`url(x.svg) no-repeat center`))
	assert.Equal(t, `data:image/svg+xml;utf8,<svg a="1">`, maskURL(`url("data:image/svg+xml;utf8,<svg a=\"1\">")`), "escaped quotes")
	x, y := maskPosition("left top")
	assert.Equal(t, [2]float32{0, 0}, [2]float32{x, y})
	x, y = maskPosition("50% 50%")
	assert.Equal(t, [2]float32{0.5, 0.5}, [2]float32{x, y})
	assert.Equal(t, math32.Vec2(20, 10), maskSize("20px auto", math32.Vec2(40, 40), math32.Vec2(100, 50), 1))
	assert.Equal(t, math32.Vec2(40, 20), maskSize("contain", math32.Vec2(40, 40), math32.Vec2(100, 50), 1))
	assert.Equal(t, "20px auto", resolveMaskSize(defaultLengthEnv(), "calc(max(calc(1rem + 4px),10px)) auto"))

	// a circle in a 10x10 view box
	circle := `data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10" width="10" height="10"%3E%3Ccircle cx="5" cy="5" r="5"/%3E%3C/svg%3E`
	b := core.NewBody()
	ctx := NewContext()
	assert.NoError(t, ReadHTMLString(ctx, b, `<style>.icon { display: block; width: 40px; height: 40px; background-color: #ff0000; `+
		`mask-image: url('`+circle+`'); mask-size: 20px auto; mask-position: center; mask-repeat: no-repeat }`+
		`.tiled { display: block; width: 40px; height: 20px; background: #0000ff; -webkit-mask: url('`+circle+`') 0 0 / 20px }</style>`+
		`<span class="icon"></span><span class="tiled"></span><span class="icon faded" style="opacity: 0.5"></span>`))
	boxes := map[string]image.Rectangle{}
	img := b.RenderImage(func() {
		b.WalkDown(func(n tree.Node) bool {
			if wb := core.AsWidget(n); wb != nil {
				if cl, ok := wb.Property("class").(string); ok {
					if wb.Styles.Opacity < 1 {
						cl = "faded"
					}
					boxes[cl] = wb.Geom.TotalBBox
				}
			}
			return tree.Continue
		})
	})
	at := func(cl string, fx, fy float32) color.RGBA {
		bb := boxes[cl]
		return color.RGBAModel.Convert(img.At(bb.Min.X+int(fx*float32(bb.Dx())), bb.Min.Y+int(fy*float32(bb.Dy())))).(color.RGBA)
	}
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, at("icon", 0.5, 0.5), "the center of the circle")
	assert.Equal(t, color.RGBA{255, 255, 255, 255}, at("icon", 0.05, 0.05), "outside of the circle: not a box of the color")
	assert.Equal(t, color.RGBA{255, 255, 255, 255}, at("icon", 0.5, 0.15), "above the circle, which is 20px in the middle")
	assert.Equal(t, color.RGBA{0, 0, 255, 255}, at("tiled", 0.25, 0.5), "the first circle of the repeated mask")
	assert.Equal(t, color.RGBA{0, 0, 255, 255}, at("tiled", 0.75, 0.5), "the second circle of the repeated mask")
	assert.Equal(t, color.RGBA{255, 255, 255, 255}, at("tiled", 0.5, 0.05), "between the circles")
	faded := at("faded", 0.5, 0.5)
	assert.True(t, faded.R == 255 && faded.G > 100 && faded.G < 160, "the circle with an opacity of 0.5: %v", faded)
	assert.Equal(t, color.RGBA{255, 255, 255, 255}, at("faded", 0.05, 0.05), "outside of the faded circle")
}

// TestMinWidthBelowWidth tests that a min-width smaller than the width does
// not make an element smaller than its width (as the icons of Wikipedia,
// with a width of 20px and a min-width of 10px).
func TestMinWidthBelowWidth(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	assert.NoError(t, ReadHTMLString(ctx, b, `<span class="a" style="display: inline-block; width: 20px; height: 20px; min-width: 10px; min-height: 10px; background: red"></span>`+
		`<span class="b" style="display: inline-block; width: 20px; height: 20px; min-width: 30px; background: red"></span>`))
	sizes := map[string]math32.Vector2{}
	b.RenderImage(func() {
		b.WalkDown(func(n tree.Node) bool {
			if wb := core.AsWidget(n); wb != nil {
				if cl, ok := wb.Property("class").(string); ok {
					sizes[cl] = wb.Geom.Size.Actual.Total.DivScalar(wb.Styles.UnitContext.Dots(units.UnitDp))
				}
			}
			return tree.Continue
		})
	})
	assert.InDelta(t, 20, sizes["a"].X, 1)
	assert.InDelta(t, 20, sizes["a"].Y, 1)
	assert.InDelta(t, 30, sizes["b"].X, 1, "a larger min-width wins")
}

// TestRotatedMask tests that the rotate of a transform rotates the
// rendering of an element (as the carets of the contents of Wikipedia,
// down-pointing icons rotated by -90deg to point right), including its
// masked background.
func TestRotatedMask(t *testing.T) {
	assert.Equal(t, float32(-90), cssRotate("transform", "rotate(-90deg)"))
	assert.InDelta(t, 90, cssAngle("0.25turn"), 0.01)
	// a triangle in the top half of the box, pointing down
	tri := `data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10" width="10" height="10"%3E%3Cpath d="M0 0 H10 L5 5 Z"/%3E%3C/svg%3E`
	b := core.NewBody()
	ctx := NewContext()
	assert.NoError(t, ReadHTMLString(ctx, b, `<style>.caret { display: block; width: 40px; height: 40px; background-color: #000000; mask-image: url('`+tri+`'); mask-size: 40px; mask-repeat: no-repeat }</style>`+
		`<div class="down"><span class="caret"></span></div><div class="right" style="transform: rotate(-90deg)"><span class="caret"></span></div>`))
	boxes := map[string]image.Rectangle{}
	img := b.RenderImage(func() {
		b.WalkDown(func(n tree.Node) bool {
			if wb := core.AsWidget(n); wb != nil {
				if cl, ok := wb.Property("class").(string); ok && cl != "caret" {
					boxes[cl] = wb.Children[0].(core.Widget).AsWidget().Geom.TotalBBox
				}
			}
			return tree.Continue
		})
	})
	dark := func(cl string, fx, fy float32) bool {
		bb := boxes[cl]
		c := color.RGBAModel.Convert(img.At(bb.Min.X+int(fx*float32(bb.Dx())), bb.Min.Y+int(fy*float32(bb.Dy())))).(color.RGBA)
		return c.R < 100
	}
	assert.True(t, dark("down", 0.5, 0.15), "the top half")
	assert.False(t, dark("down", 0.5, 0.85), "not the bottom half")
	assert.True(t, dark("right", 0.15, 0.5), "rotated -90deg: the left half")
	assert.False(t, dark("right", 0.85, 0.5), "rotated: not the right half")
	assert.False(t, dark("right", 0.5, 0.15), "rotated: not the top")
}
