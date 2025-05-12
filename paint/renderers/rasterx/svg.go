// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rasterx

import (
	"bytes"
	"image"
	"image/draw"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped/shapers/shapedgt"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/shaping"
)

var svgGlyphs map[font.GID]*svg.SVG

func (rs *Renderer) GlyphSVG(ctx *render.Context, run *shapedgt.Run, g *shaping.Glyph, svgCmds string, bb math32.Box2, pos math32.Vector2, identity bool) {
	if svgGlyphs == nil {
		svgGlyphs = make(map[font.GID]*svg.SVG)
	}
	size := run.Size.Floor()
	fsize := image.Point{X: size, Y: size}
	scale := 82.0 / float32(run.Face.Upem())
	if run.Font.Style(&ctx.Style.Text).Family == rich.Monospace {
		scale *= 0.8
	}
	sv, ok := svgGlyphs[g.GlyphID]
	if !ok {
		sv = svg.NewSVG(fsize.X, fsize.Y)
		b := bytes.NewBufferString(svgCmds)
		err := sv.ReadXML(b)
		errors.Log(err)
		sv.Translate.Y = float32(run.Face.Upem())
		sv.Scale = scale
		sv.Render()
		svgGlyphs[g.GlyphID] = sv
	}
	if sv.Geom.Size != fsize || sv.Scale != scale {
		// fmt.Println("re-render:", sv.Geom.Size, fsize, sv.Scale, scale)
		sv.Resize(fsize)
		sv.Translate.Y = float32(run.Face.Upem())
		sv.Scale = scale
		sv.Render()
	}
	img := sv.RenderImage()
	// fmt.Printf("%#v\n", g)
	// fmt.Printf("%#v\n", run)
	left := int(math32.Round(pos.X + math32.FromFixed(g.XBearing)))
	desc := run.Output.LineBounds.Descent
	top := int(math32.Round(pos.Y - math32.FromFixed(g.YBearing+desc) - float32(fsize.Y)))
	db := img.Bounds().Add(image.Point{left, top})
	draw.Draw(rs.image, db, img, image.Point{}, draw.Over)
}
