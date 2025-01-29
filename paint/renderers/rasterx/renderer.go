// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rasterx

import (
	"image"
	"slices"

	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/pimage"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/paint/ptext"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/paint/renderers/rasterx/scan"
	"cogentcore.org/core/styles/units"
)

type Renderer struct {
	size  math32.Vector2
	image *image.RGBA

	// Path is the current path.
	Path Path

	// rasterizer -- stroke / fill rendering engine from raster
	Raster *Dasher

	// scan scanner
	Scanner *scan.Scanner

	// scan spanner
	ImgSpanner *scan.ImgSpanner
}

func New(size math32.Vector2, img *image.RGBA) render.Renderer {
	psz := size.ToPointCeil()
	if img == nil {
		img = image.NewRGBA(image.Rectangle{Max: psz})
	}
	rs := &Renderer{size: size, image: img}
	rs.ImgSpanner = scan.NewImgSpanner(img)
	rs.Scanner = scan.NewScanner(rs.ImgSpanner, psz.X, psz.Y)
	rs.Raster = NewDasher(psz.X, psz.Y, rs.Scanner)
	return rs
}

func (rs *Renderer) IsImage() bool      { return true }
func (rs *Renderer) Image() *image.RGBA { return rs.image }
func (rs *Renderer) Code() []byte       { return nil }

func (rs *Renderer) Size() (units.Units, math32.Vector2) {
	return units.UnitDot, rs.size
}

func (rs *Renderer) SetSize(un units.Units, size math32.Vector2, img *image.RGBA) {
	if rs.size == size {
		return
	}
	rs.size = size
	psz := size.ToPointCeil()
	if img != nil {
		rs.image = img
	} else {
		rs.image = image.NewRGBA(image.Rectangle{Max: psz})
	}
	rs.ImgSpanner = scan.NewImgSpanner(rs.image)
	rs.Scanner = scan.NewScanner(rs.ImgSpanner, psz.X, psz.Y)
	rs.Raster = NewDasher(psz.X, psz.Y, rs.Scanner)
}

// Render is the main rendering function.
func (rs *Renderer) Render(r render.Render) {
	for _, ri := range r {
		switch x := ri.(type) {
		case *render.Path:
			rs.RenderPath(x)
		case *pimage.Params:
			x.Render(rs.image)
		case *ptext.Text:
			x.Render(rs.image, rs)
		}
	}
}

func (rs *Renderer) RenderPath(pt *render.Path) {
	rs.Raster.Clear()
	p := pt.Path.ReplaceArcs()
	m := pt.Context.Transform
	for s := p.Scanner(); s.Scan(); {
		cmd := s.Cmd()
		end := m.MulVector2AsPoint(s.End())
		switch cmd {
		case ppath.MoveTo:
			rs.Path.Start(end.ToFixed())
		case ppath.LineTo:
			rs.Path.Line(end.ToFixed())
		case ppath.QuadTo:
			cp1 := m.MulVector2AsPoint(s.CP1())
			rs.Path.QuadBezier(cp1.ToFixed(), end.ToFixed())
		case ppath.CubeTo:
			cp1 := m.MulVector2AsPoint(s.CP1())
			cp2 := m.MulVector2AsPoint(s.CP2())
			rs.Path.CubeBezier(cp1.ToFixed(), cp2.ToFixed(), end.ToFixed())
		case ppath.Close:
			rs.Path.Stop(true)
		}
	}
	rs.Fill(pt)
	rs.Stroke(pt)
	rs.Path.Clear()
}

func (rs *Renderer) Stroke(pt *render.Path) {
	pc := &pt.Context
	sty := &pc.Style
	if sty.Off || sty.Stroke.Color == nil {
		return
	}

	dash := slices.Clone(sty.Stroke.Dashes)
	if dash != nil {
		scx, scy := pc.Transform.ExtractScale()
		sc := 0.5 * (math32.Abs(scx) + math32.Abs(scy))
		for i := range dash {
			dash[i] *= sc
		}
	}

	sw := rs.StrokeWidth(pt)

	rs.Raster.SetStroke(
		math32.ToFixed(sw),
		math32.ToFixed(sty.Stroke.MiterLimit),
		capfunc(sty.Stroke.Cap), nil, nil, joinmode(sty.Stroke.Join),
		dash, 0)
	rs.Scanner.SetClip(pc.Bounds.Rect.ToRect())
	rs.Path.AddTo(rs.Raster)
	fbox := rs.Raster.Scanner.GetPathExtent()
	lastRenderBBox := image.Rectangle{Min: image.Point{fbox.Min.X.Floor(), fbox.Min.Y.Floor()},
		Max: image.Point{fbox.Max.X.Ceil(), fbox.Max.Y.Ceil()}}
	if g, ok := sty.Stroke.Color.(gradient.Gradient); ok {
		g.Update(sty.Stroke.Opacity, math32.B2FromRect(lastRenderBBox), pc.Transform)
		rs.Raster.SetColor(sty.Stroke.Color)
	} else {
		if sty.Stroke.Opacity < 1 {
			rs.Raster.SetColor(gradient.ApplyOpacity(sty.Stroke.Color, sty.Stroke.Opacity))
		} else {
			rs.Raster.SetColor(sty.Stroke.Color)
		}
	}

	rs.Raster.Draw()
	rs.Raster.Clear()
}

// Fill fills the current path with the current color. Open subpaths
// are implicitly closed. The path is preserved after this operation.
func (rs *Renderer) Fill(pt *render.Path) {
	pc := &pt.Context
	sty := &pc.Style
	if sty.Fill.Color == nil {
		return
	}
	rf := &rs.Raster.Filler
	rf.SetWinding(sty.Fill.Rule == ppath.NonZero)
	rs.Scanner.SetClip(pc.Bounds.Rect.ToRect())
	rs.Path.AddTo(rf)
	fbox := rs.Scanner.GetPathExtent()
	lastRenderBBox := image.Rectangle{Min: image.Point{fbox.Min.X.Floor(), fbox.Min.Y.Floor()},
		Max: image.Point{fbox.Max.X.Ceil(), fbox.Max.Y.Ceil()}}
	if g, ok := sty.Fill.Color.(gradient.Gradient); ok {
		g.Update(sty.Fill.Opacity, math32.B2FromRect(lastRenderBBox), pc.Transform)
		rf.SetColor(sty.Fill.Color)
	} else {
		if sty.Fill.Opacity < 1 {
			rf.SetColor(gradient.ApplyOpacity(sty.Fill.Color, sty.Fill.Opacity))
		} else {
			rf.SetColor(sty.Fill.Color)
		}
	}
	rf.Draw()
	rf.Clear()
}

// StrokeWidth obtains the current stoke width subject to transform (or not
// depending on VecEffNonScalingStroke)
func (rs *Renderer) StrokeWidth(pt *render.Path) float32 {
	pc := &pt.Context
	sty := &pc.Style
	dw := sty.Stroke.Width.Dots
	if dw == 0 {
		return dw
	}
	if sty.VectorEffect == ppath.VectorEffectNonScalingStroke {
		return dw
	}
	scx, scy := pc.Transform.ExtractScale()
	sc := 0.5 * (math32.Abs(scx) + math32.Abs(scy))
	lw := math32.Max(sc*dw, sty.Stroke.MinWidth.Dots)
	return lw
}

func capfunc(st ppath.Caps) CapFunc {
	switch st {
	case ppath.CapButt:
		return ButtCap
	case ppath.CapRound:
		return RoundCap
	case ppath.CapSquare:
		return SquareCap
	}
	return nil
}

func joinmode(st ppath.Joins) JoinMode {
	switch st {
	case ppath.JoinMiter:
		return Miter
	case ppath.JoinMiterClip:
		return MiterClip
	case ppath.JoinRound:
		return Round
	case ppath.JoinBevel:
		return Bevel
	case ppath.JoinArcs:
		return Arc
	case ppath.JoinArcsClip:
		return ArcClip
	}
	return Arc
}
