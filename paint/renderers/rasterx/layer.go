// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rasterx

import (
	"image"
	"image/color"
	"image/draw"

	"cogentcore.org/core/paint/render"
)

// layer is a context that is pushed: if it is composited as a whole (see
// [render.Context.IsLayer]), the rendering within it is to its own image
// (the layer), which is composited onto the image under it, through its
// mask, when it is popped.
type layer struct {
	// under is the image under the layer, nil if the context is not a layer.
	under *image.RGBA

	// image is the image of the layer.
	image *image.RGBA

	// region is the region of the images that the layer composites onto.
	region image.Rectangle

	// mask is the mask of the layer (its clip mask, mask, and opacity), nil
	// for none.
	mask *image.Alpha

	// opacity is the opacity of the layer, if it has no mask.
	opacity float32
}

// pushContext pushes the given context: if it is a layer, the rendering
// is to a new layer until it is popped.
func (rs *Renderer) pushContext(ctx *render.Context) {
	if !ctx.IsLayer() {
		rs.layers = append(rs.layers, layer{})
		return
	}
	region := ctx.Bounds.Rect.ToRect().Intersect(rs.image.Bounds())
	if ctx.Bounds.Rect.Size().X <= 0 || ctx.Bounds.Rect.Size().Y <= 0 {
		region = rs.image.Bounds()
	}
	ly := layer{under: rs.image, opacity: 1}
	if ctx.Opacity > 0 && ctx.Opacity < 1 {
		ly.opacity = ctx.Opacity
	}
	if len(ctx.ClipMask) > 0 || len(ctx.Mask) > 0 {
		ly.mask = image.NewAlpha(region)
		for i := range ly.mask.Pix {
			ly.mask.Pix[i] = uint8(255 * ly.opacity)
		}
		if len(ctx.ClipMask) > 0 {
			rs.applyMask(ly.mask, ctx.ClipMask, false)
		}
		if len(ctx.Mask) > 0 {
			rs.applyMask(ly.mask, ctx.Mask, true)
		}
		region = alphaBounds(ly.mask)
		ly.mask = ly.mask.SubImage(region).(*image.Alpha)
	}
	ly.region = region
	ly.image = rs.newLayerImage(region)
	rs.layers = append(rs.layers, ly)
	rs.setImage(ly.image)
}

// popContext pops the context that was last pushed: if it is a layer,
// it is composited onto the image under it, through its mask.
func (rs *Renderer) popContext() {
	n := len(rs.layers)
	if n == 0 {
		return
	}
	ly := rs.layers[n-1]
	rs.layers = rs.layers[:n-1]
	if ly.under == nil {
		return
	}
	rs.setImage(ly.under)
	if !ly.region.Empty() {
		var mask image.Image = ly.mask
		if ly.mask == nil {
			mask = image.NewUniform(color.Alpha{A: uint8(255*ly.opacity + 0.5)})
		}
		draw.DrawMask(ly.under, ly.region, ly.image, ly.region.Min, mask, ly.region.Min, draw.Over)
	}
	rs.freeLayers = append(rs.freeLayers, ly.image)
}

// setImage sets the image that is rendered to.
func (rs *Renderer) setImage(img *image.RGBA) {
	rs.image = img
	rs.ImgSpanner.SetImage(img)
}

// newLayerImage returns an image for a layer, of the size of the
// renderer, which is transparent in the given region (the rest of it is
// not used).
func (rs *Renderer) newLayerImage(region image.Rectangle) *image.RGBA {
	var img *image.RGBA
	if n := len(rs.freeLayers); n > 0 {
		img = rs.freeLayers[n-1]
		rs.freeLayers = rs.freeLayers[:n-1]
	} else {
		img = image.NewRGBA(rs.image.Bounds())
	}
	if region.Empty() {
		return img
	}
	for y := region.Min.Y; y < region.Max.Y; y++ {
		i := img.PixOffset(region.Min.X, y)
		clear(img.Pix[i : i+4*region.Dx()])
	}
	return img
}

// applyMask multiplies the given mask by the rendering of the given
// items: by its alpha, or, if luminance, by its luminance times its alpha
// (as for SVG masks, with the coefficients of browsers).
func (rs *Renderer) applyMask(mask *image.Alpha, items render.Render, luminance bool) {
	region := mask.Rect
	img := rs.newLayerImage(region)
	under := rs.image
	rs.setImage(img)
	depth := len(rs.layers)
	rs.renderItems(items)
	for len(rs.layers) > depth { // contexts of the items that are not popped
		rs.popContext()
	}
	rs.setImage(under)
	for y := region.Min.Y; y < region.Max.Y; y++ {
		mi := mask.PixOffset(region.Min.X, y)
		pi := img.PixOffset(region.Min.X, y)
		for x := 0; x < region.Dx(); x, mi, pi = x+1, mi+1, pi+4 {
			var a uint32
			if luminance { // the colors are premultiplied by alpha
				p := img.Pix[pi : pi+3]
				a = (2125*uint32(p[0]) + 7154*uint32(p[1]) + 721*uint32(p[2]) + 5000) / 10000
			} else {
				a = uint32(img.Pix[pi+3])
			}
			mask.Pix[mi] = uint8((uint32(mask.Pix[mi])*a + 127) / 255)
		}
	}
	rs.freeLayers = append(rs.freeLayers, img)
}

// alphaBounds returns the bounds of the pixels of the given mask that are
// not transparent (empty if none are).
func alphaBounds(m *image.Alpha) image.Rectangle {
	var b image.Rectangle
	r := m.Rect
	for y := r.Min.Y; y < r.Max.Y; y++ {
		i := m.PixOffset(r.Min.X, y)
		row := m.Pix[i : i+r.Dx()]
		x0 := -1
		for x, a := range row {
			if a != 0 {
				x0 = x
				break
			}
		}
		if x0 < 0 {
			continue
		}
		x1 := x0
		for x := len(row) - 1; x > x0; x-- {
			if row[x] != 0 {
				x1 = x
				break
			}
		}
		b = b.Union(image.Rect(r.Min.X+x0, y, r.Min.X+x1+1, y+1))
	}
	return b
}
