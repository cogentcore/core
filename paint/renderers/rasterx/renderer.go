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
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/paint/renderers/rasterx/scan"
	"cogentcore.org/core/styles/units"
)

// Renderer is the rasterx renderer.
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

func New(size math32.Vector2) render.Renderer {
	rs := &Renderer{}
	rs.SetSize(units.UnitDot, size)
	return rs
}

func (rs *Renderer) Image() image.Image { return rs.image }
func (rs *Renderer) Source() []byte     { return nil }

func (rs *Renderer) Size() (units.Units, math32.Vector2) {
	return units.UnitDot, rs.size
}

func (rs *Renderer) SetSize(un units.Units, size math32.Vector2) {
	if rs.size == size {
		return
	}
	rs.size = size
	psz := size.ToPointCeil()
	rs.image = image.NewRGBA(image.Rectangle{Max: psz})
	rs.ImgSpanner = scan.NewImgSpanner(rs.image)
	rs.Scanner = scan.NewScanner(rs.ImgSpanner, psz.X, psz.Y)
	rs.Raster = NewDasher(psz.X, psz.Y, rs.Scanner)
}

// Render is the main rendering function.
func (rs *Renderer) Render(r render.Render) render.Renderer {
	for _, ri := range r {
		switch x := ri.(type) {
		case *render.Path:
			rs.RenderPath(x)
		case *pimage.Params:
			x.Render(rs.image)
		case *render.Text:
			rs.RenderText(x)
		}
	}
	return rs
}

func (rs *Renderer) RenderPath(pt *render.Path) {
	p := pt.Path
	if !ppath.ArcToCubeImmediate {
		p = p.ReplaceArcs()
	}
	pc := &pt.Context
	rs.Scanner.SetClip(pc.Bounds.Rect.ToRect())
	PathToRasterx(&rs.Path, p, pt.Context.Transform, math32.Vector2{})
	rs.Fill(pt)
	rs.Stroke(pt)
	rs.Path.Clear()
	rs.Raster.Clear()
}

func PathToRasterx(rs Adder, p ppath.Path, m math32.Matrix2, off math32.Vector2) {
	for s := p.Scanner(); s.Scan(); {
		cmd := s.Cmd()
		end := m.MulVector2AsPoint(s.End()).Add(off)
		switch cmd {
		case ppath.MoveTo:
			rs.Start(end.ToFixed())
		case ppath.LineTo:
			rs.Line(end.ToFixed())
		case ppath.QuadTo:
			cp1 := m.MulVector2AsPoint(s.CP1()).Add(off)
			rs.QuadBezier(cp1.ToFixed(), end.ToFixed())
		case ppath.CubeTo:
			cp1 := m.MulVector2AsPoint(s.CP1()).Add(off)
			cp2 := m.MulVector2AsPoint(s.CP2()).Add(off)
			rs.CubeBezier(cp1.ToFixed(), cp2.ToFixed(), end.ToFixed())
		case ppath.Close:
			rs.Stop(true)
		}
	}
}

func (rs *Renderer) Stroke(pt *render.Path) {
	pc := &pt.Context
	sty := &pc.Style
	if !sty.HasStroke() {
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
	rs.Path.AddTo(rs.Raster)
	rs.SetColor(rs.Raster, pc, sty.Stroke.Color, sty.Stroke.Opacity)
	rs.Raster.Draw()
}

func (rs *Renderer) SetColor(sc Scanner, pc *render.Context, clr image.Image, opacity float32) {
	if g, ok := clr.(gradient.Gradient); ok {
		fbox := sc.GetPathExtent()
		lastRenderBBox := image.Rectangle{Min: image.Point{fbox.Min.X.Floor(), fbox.Min.Y.Floor()},
			Max: image.Point{fbox.Max.X.Ceil(), fbox.Max.Y.Ceil()}}
		g.Update(opacity, math32.B2FromRect(lastRenderBBox), pc.Transform)
		sc.SetColor(clr)
	} else {
		if opacity < 1 {
			sc.SetColor(gradient.ApplyOpacity(clr, opacity))
		} else {
			sc.SetColor(clr)
		}
	}
}

// Fill fills the current path with the current color. Open subpaths
// are implicitly closed. The path is preserved after this operation.
func (rs *Renderer) Fill(pt *render.Path) {
	pc := &pt.Context
	sty := &pc.Style
	if !sty.HasFill() {
		return
	}
	rf := &rs.Raster.Filler
	rf.SetWinding(sty.Fill.Rule == ppath.NonZero)
	rs.Path.AddTo(rf)
	rs.SetColor(rf, pc, sty.Fill.Color, sty.Fill.Opacity)
	rf.Draw()
	rf.Clear()
}

func MeanScale(m math32.Matrix2) float32 {
	scx, scy := m.ExtractScale()
	return 0.5 * (math32.Abs(scx) + math32.Abs(scy))
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
	sc := MeanScale(pt.Context.Transform)
	return sc * dw
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
