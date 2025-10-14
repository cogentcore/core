// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rasterx

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped/shapers/shapedgt"
	"github.com/go-text/typesetting/shaping"
)

var svgGlyphCache map[glyphKey]image.Image

func (rs *Renderer) GlyphSVG(ctx *render.Context, run *shapedgt.Run, g *shaping.Glyph, svgCmds []byte, bb math32.Box2, pos math32.Vector2, identity bool) {
	if svgGlyphCache == nil {
		svgGlyphCache = make(map[glyphKey]image.Image)
	}
	size := float32(run.Size.Floor())
	fam := run.Font.Style(&ctx.Style.Text).Family
	desc := run.Output.LineBounds.Descent
	fsize := math32.Vec2(size, size)
	gk := glyphKey{gid: g.GlyphID, sx: uint8(int(size) / 256), sy: uint8(int(size) % 256), ox: uint8(fam)}
	img, ok := svgGlyphCache[gk]
	if !ok {
		hadv := run.Face.HorizontalAdvance(g.GlyphID)
		scale := size / hadv
		if fam == rich.Monospace {
			scale *= 0.8
		}
		sv := svg.NewSVG(fsize)
		sv.GroupFilter = fmt.Sprintf("glyph%d", g.GlyphID) // critical: for filtering items with many glyphs
		b := bytes.NewBuffer(svgCmds)
		err := sv.ReadXML(b)
		errors.Log(err)
		sv.Translate.Y = size + math32.FromFixed(desc)
		sv.Scale = scale
		sv.Root.ViewBox.Size.SetScalar(size)
		img = sv.RenderImage()
		svgGlyphCache[gk] = img
	}
	left := int(math32.Round(pos.X + math32.FromFixed(g.XBearing)))
	top := int(math32.Round(pos.Y - math32.FromFixed(g.YBearing+desc) - fsize.Y))
	dbb := img.Bounds().Add(image.Point{left, top})
	ibb := dbb.Intersect(ctx.Bounds.Rect.ToRect())
	if ibb == (image.Rectangle{}) {
		return
	}
	sp := ibb.Min.Sub(dbb.Min)
	draw.Draw(rs.image, ibb, img, sp, draw.Over)
}
