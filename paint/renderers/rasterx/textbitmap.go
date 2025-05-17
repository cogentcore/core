// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rasterx

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/text/shaped/shapers/shapedgt"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/shaping"
	scale "golang.org/x/image/draw"
)

var bitmapGlyphCache map[glyphKey]*image.RGBA

func (rs *Renderer) GlyphBitmap(ctx *render.Context, run *shapedgt.Run, g *shaping.Glyph, bitmap font.GlyphBitmap, fill, stroke image.Image, bb math32.Box2, pos math32.Vector2, identity bool) error {
	if bitmapGlyphCache == nil {
		bitmapGlyphCache = make(map[glyphKey]*image.RGBA)
	}
	// todo: this needs serious work to function with transforms
	x := pos.X
	y := pos.Y
	top := y - math32.FromFixed(g.YBearing)
	bottom := top - math32.FromFixed(g.Height)
	right := x + math32.FromFixed(g.Width)
	dbb := image.Rect(int(x), int(top), int(right), int(bottom))
	ibb := dbb.Intersect(ctx.Bounds.Rect.ToRect())
	if ibb == (image.Rectangle{}) {
		return nil
	}

	fam := run.Font.Style(&ctx.Style.Text).Family
	size := dbb.Size()

	gk := glyphKey{gid: g.GlyphID, sx: uint8(size.Y / 256), sy: uint8(size.Y % 256), ox: uint8(fam)}
	img, ok := bitmapGlyphCache[gk]
	if !ok {
		img = image.NewRGBA(image.Rectangle{Max: size})
		switch bitmap.Format {
		case font.BlackAndWhite:
			rec := image.Rect(0, 0, bitmap.Width, bitmap.Height)
			sub := image.NewPaletted(rec, color.Palette{color.Transparent, colors.ToUniform(fill)})

			for i := range sub.Pix {
				sub.Pix[i] = bitAt(bitmap.Data, i)
			}
			// note: NearestNeighbor is better than bilinear
			scale.NearestNeighbor.Scale(img, img.Bounds(), sub, sub.Bounds(), draw.Src, nil)
		case font.JPG, font.PNG, font.TIFF:
			pix, _, err := image.Decode(bytes.NewReader(bitmap.Data))
			if err != nil {
				return err
			}
			scale.NearestNeighbor.Scale(img, img.Bounds(), pix, pix.Bounds(), draw.Src, nil)
		}
		bitmapGlyphCache[gk] = img
	}

	sp := ibb.Min.Sub(dbb.Min)
	draw.Draw(rs.image, ibb, img, sp, draw.Over)

	if bitmap.Outline != nil {
		rs.GlyphOutline(ctx, run, g, *bitmap.Outline, fill, stroke, bb, pos, identity)
	}
	return nil
}

// bitAt returns the bit at the given index in the byte slice.
func bitAt(b []byte, i int) byte {
	return (b[i/8] >> (7 - i%8)) & 1
}
