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
	size := run.Size.Floor()
	fsize := image.Point{X: size, Y: size}
	scale := 82.0 / float32(run.Face.Upem())
	fam := run.Font.Style(&ctx.Style.Text).Family
	if fam == rich.Monospace {
		scale *= 0.8
	}
	gk := glyphKey{gid: g.GlyphID, sx: uint8(size / 256), sy: uint8(size % 256), ox: uint8(fam)}
	img, ok := svgGlyphCache[gk]
	if !ok {
		sv := svg.NewSVG(math32.FromPoint(fsize))
		sv.GroupFilter = fmt.Sprintf("glyph%d", g.GlyphID) // critical: for filtering items with many glyphs
		b := bytes.NewBuffer(svgCmds)
		err := sv.ReadXML(b)
		errors.Log(err)
		sv.Translate.Y = float32(run.Face.Upem())
		sv.Scale = scale
		img = sv.RenderImage()
		svgGlyphCache[gk] = img
	}
	left := int(math32.Round(pos.X + math32.FromFixed(g.XBearing)))
	desc := run.Output.LineBounds.Descent
	top := int(math32.Round(pos.Y - math32.FromFixed(g.YBearing+desc) - float32(fsize.Y)))
	dbb := img.Bounds().Add(image.Point{left, top})
	ibb := dbb.Intersect(ctx.Bounds.Rect.ToRect())
	if ibb == (image.Rectangle{}) {
		return
	}
	sp := ibb.Min.Sub(dbb.Min)
	draw.Draw(rs.image, ibb, img, sp, draw.Over)
}
