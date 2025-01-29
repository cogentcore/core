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

func New(size math32.Vector2, img *image.RGBA) paint.Renderer {
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
	// todo: update sizes of other scanner etc?
	if img != nil {
		rs.image = img
		return
	}
	psz := size.ToPointCeil()
	rs.image = image.NewRGBA(image.Rectangle{Max: psz})
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
	// todo: transform!
	p := pt.Path.ReplaceArcs()
	m := pt.Context.Transform
	for s := p.Scanner(); s.Scan(); {
		cmd := s.Cmd()
		end := m.MulVector2AsPoint(s.End())
		switch cmd {
		case path.MoveTo:
			rs.Path.Start(end.ToFixed())
		case path.LineTo:
			rs.Path.Line(end.ToFixed())
		case path.QuadTo:
			cp1 := m.MulVector2AsPoint(s.CP1())
			rs.Path.QuadBezier(cp1.ToFixed(), end.ToFixed())
		case path.CubeTo:
			cp1 := m.MulVector2AsPoint(s.CP1())
			cp2 := m.MulVector2AsPoint(s.CP2())
			rs.Path.CubeBezier(cp1.ToFixed(), cp2.ToFixed(), end.ToFixed())
		case path.Close:
			rs.Path.Stop(true)
		}
	}
	rs.Fill(pt)
	rs.Stroke(pt)
	rs.Path.Clear()
}

func (rs *Renderer) Stroke(pt *paint.Path) {
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
func (rs *Renderer) Fill(pt *paint.Path) {
	pc := &pt.Context
	sty := &pc.Style
	if sty.Fill.Color == nil {
		return
	}
	rf := &rs.Raster.Filler
	rf.SetWinding(sty.Fill.Rule == path.NonZero)
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
func (rs *Renderer) StrokeWidth(pt *paint.Path) float32 {
	pc := &pt.Context
	sty := &pc.Style
	dw := sty.Stroke.Width.Dots
	if dw == 0 {
		return dw
	}
	if sty.VectorEffect == path.VectorEffectNonScalingStroke {
		return dw
	}
	scx, scy := pc.Transform.ExtractScale()
	sc := 0.5 * (math32.Abs(scx) + math32.Abs(scy))
	lw := math32.Max(sc*dw, sty.Stroke.MinWidth.Dots)
	return lw
}

func capfunc(st path.Caps) CapFunc {
	switch st {
	case path.CapButt:
		return ButtCap
	case path.CapRound:
		return RoundCap
	case path.CapSquare:
		return SquareCap
	}
	return nil
}

func joinmode(st path.Joins) JoinMode {
	switch st {
	case path.JoinMiter:
		return Miter
	case path.JoinMiterClip:
		return MiterClip
	case path.JoinRound:
		return Round
	case path.JoinBevel:
		return Bevel
	case path.JoinArcs:
		return Arc
	case path.JoinArcsClip:
		return ArcClip
	}
	return Arc
}
