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
	scale := math32.FromFixed(run.Size) / float32(run.Face.Upem())
	top := pos.Y - math32.FromFixed(g.YBearing)
	sv, ok := svgGlyphs[g.GlyphID]
	if !ok {
		sv = svg.NewSVG(size, size)
		b := bytes.NewBufferString(svgCmds)
		err := sv.ReadXML(b)
		errors.Log(err)
		sv.Scale = scale
		sv.Render()
		svgGlyphs[g.GlyphID] = sv
	}
	if sv.Geom.Size.X != size || sv.Scale != scale {
		sv.Resize(image.Point{size, size})
		sv.Scale = scale
		sv.Render()
	}
	img := sv.RenderImage()
	draw.Draw(rs.image, img.Bounds(), img, image.Point{int(pos.X), int(top)}, draw.Over)
}
