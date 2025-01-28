// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rasterx

import (
	"image"
	"slices"

	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/path"
	"cogentcore.org/core/paint/renderers/rasterx/scan"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// Renderer is the overall renderer for rasterx.
type Renderer struct {
	// Size is the size of the render target.
	Size math32.Vector2

	// Image is the image we are rendering to.
	Image *image.RGBA

	// Path is the current path.
	Path Path

	// rasterizer -- stroke / fill rendering engine from raster
	Raster *Dasher

	// scan scanner
	Scanner *scan.Scanner

	// scan spanner
	ImgSpanner *scan.ImgSpanner
}

// New returns a new rasterx Renderer, rendering to given image.
func New(size math32.Vector2, img *image.RGBA) *Renderer {
	rs := &Renderer{Size: size, Image: img}
	psz := size.ToPointCeil()
	rs.ImgSpanner = scan.NewImgSpanner(img)
	rs.Scanner = scan.NewScanner(rs.ImgSpanner, psz.X, psz.Y)
	rs.Raster = NewDasher(psz.X, psz.Y, rs.Scanner)
	return rs
}

// RenderSize returns the size of the render target, in dots (pixels).
func (rs *Renderer) RenderSize() (units.Units, math32.Vector2) {
	return units.UnitDot, rs.Size
}

// Render is the main rendering function.
func (rs *Renderer) Render(r paint.Render) {
	for _, ri := range r {
		switch x := ri.(type) {
		case *paint.Path:
			rs.RenderPath(x)
		}
	}
}

func (rs *Renderer) RenderPath(pt *paint.Path) {
	rs.Raster.Clear()
	p := pt.Path.ReplaceArcs()
	for s := p; s.Scan(); {
		cmd := s.Cmd()
		end := s.End()
		switch cmd {
		case path.MoveTo:
			rs.Path.Start(end.ToFixed())
		case path.LineTo:
			rs.Path.Line(end.ToFixed())
		case path.QuadTo:
			cp1 := s.CP1()
			rs.Path.QuadBezier(cp1.ToFixed(), end.ToFixed())
		case path.CubeTo:
			cp1 := s.CP1()
			cp2 := s.CP2()
			rs.Path.CubeBezier(cp1.ToFixed(), cp2.ToFixed(), end.ToFixed())
		case path.Close:
			rs.Path.Stop(true)
		}
	}
	rs.Fill(&pt.Style)
	rs.Stroke(&pt.Style)
	rs.Path.Clear()
}

func (rs *Renderer) Stroke(sty *styles.Path) {
	if sty.Off || sty.Stroke.Color == nil {
		return
	}

	dash := slices.Clone(sty.Dashes)
	if dash != nil {
		scx, scy := pc.Transform.ExtractScale()
		sc := 0.5 * (math32.Abs(scx) + math32.Abs(scy))
		for i := range dash {
			dash[i] *= sc
		}
	}

	pc.Raster.SetStroke(
		math32.ToFixed(pc.StrokeWidth()),
		math32.ToFixed(sty.MiterLimit),
		pc.capfunc(), nil, nil, pc.joinmode(), // todo: supports leading / trailing caps, and "gaps"
		dash, 0)
	pc.Scanner.SetClip(pc.Bounds)
	pc.Path.AddTo(pc.Raster)
	fbox := pc.Raster.Scanner.GetPathExtent()
	pc.LastRenderBBox = image.Rectangle{Min: image.Point{fbox.Min.X.Floor(), fbox.Min.Y.Floor()},
		Max: image.Point{fbox.Max.X.Ceil(), fbox.Max.Y.Ceil()}}
	if g, ok := sty.Color.(gradient.Gradient); ok {
		g.Update(sty.Opacity, math32.B2FromRect(pc.LastRenderBBox), pc.Transform)
		pc.Raster.SetColor(sty.Color)
	} else {
		if sty.Opacity < 1 {
			pc.Raster.SetColor(gradient.ApplyOpacity(sty.Color, sty.Opacity))
		} else {
			pc.Raster.SetColor(sty.Color)
		}
	}

	pc.Raster.Draw()
	pc.Raster.Clear()

}

// Fill fills the current path with the current color. Open subpaths
// are implicitly closed. The path is preserved after this operation.
func (rs *Renderer) Fill() {
	if pc.Fill.Color == nil {
		return
	}
	rf := &pc.Raster.Filler
	rf.SetWinding(pc.Fill.Rule == styles.FillRuleNonZero)
	pc.Scanner.SetClip(pc.Bounds)
	pc.Path.AddTo(rf)
	fbox := pc.Scanner.GetPathExtent()
	pc.LastRenderBBox = image.Rectangle{Min: image.Point{fbox.Min.X.Floor(), fbox.Min.Y.Floor()},
		Max: image.Point{fbox.Max.X.Ceil(), fbox.Max.Y.Ceil()}}
	if g, ok := pc.Fill.Color.(gradient.Gradient); ok {
		g.Update(pc.Fill.Opacity, math32.B2FromRect(pc.LastRenderBBox), pc.Transform)
		rf.SetColor(pc.Fill.Color)
	} else {
		if pc.Fill.Opacity < 1 {
			rf.SetColor(gradient.ApplyOpacity(pc.Fill.Color, pc.Fill.Opacity))
		} else {
			rf.SetColor(pc.Fill.Color)
		}
	}
	rf.Draw()
	rf.Clear()
}

// StrokeWidth obtains the current stoke width subject to transform (or not
// depending on VecEffNonScalingStroke)
func (rs *Renderer) StrokeWidth() float32 {
	dw := sty.Width.Dots
	if dw == 0 {
		return dw
	}
	if pc.VectorEffect == styles.VectorEffectNonScalingStroke {
		return dw
	}
	scx, scy := pc.Transform.ExtractScale()
	sc := 0.5 * (math32.Abs(scx) + math32.Abs(scy))
	lw := math32.Max(sc*dw, sty.MinWidth.Dots)
	return lw
}
