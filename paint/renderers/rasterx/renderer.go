// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rasterx

import (
	"image"
	"slices"

	"cogentcore.org/core/base/profile"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/pimage"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/paint/ptext"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/paint/renderers/rasterx/scan"
	"cogentcore.org/core/styles/units"
	gvrx "github.com/srwiley/rasterx"
	"github.com/srwiley/scanFT"
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

	ScanGV *gvrx.ScannerGV
	ScanFT *scanFT.ScannerFT
	Ptr    *scanFT.RGBAPainter
}

func New(size math32.Vector2) render.Renderer {
	rs := &Renderer{}
	rs.SetSize(units.UnitDot, size)
	return rs
}

func (rs *Renderer) IsImage() bool      { return true }
func (rs *Renderer) Image() *image.RGBA { return rs.image }
func (rs *Renderer) Code() []byte       { return nil }

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
	rs.ScanGV = gvrx.NewScannerGV(psz.X, psz.Y, rs.image, rs.image.Bounds())
	rs.Ptr = scanFT.NewRGBAPainter(rs.image)
	rs.ScanFT = scanFT.NewScannerFT(psz.X, psz.Y, rs.Ptr)
	rs.Raster = NewDasher(psz.X, psz.Y, rs.Scanner)
	// rs.Raster = NewDasher(psz.X, psz.Y, rs.ScanGV)
	// rs.Raster = NewDasher(psz.X, psz.Y, rs.ScanFT)
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
	// pr := profile.Start("rasterx-replace-arcs")
	// p := pt.Path.ReplaceArcs()
	// pr.End()
	p := pt.Path
	pr := profile.Start("rasterx-path")
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
	pr.End()
	pr = profile.Start("rasterx-fill")
	rs.Fill(pt)
	pr.End()
	pr = profile.Start("rasterx-stroke")
	rs.Stroke(pt)
	pr.End()
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
	rs.SetColor(rs.Raster, pc, sty.Stroke.Color, sty.Stroke.Opacity)
	pr := profile.Start("rasterx-draw")
	rs.Raster.Draw()
	rs.Raster.Clear()
	pr.End()
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
	if sty.Fill.Color == nil {
		return
	}
	rf := &rs.Raster.Filler
	rf.SetWinding(sty.Fill.Rule == ppath.NonZero)
	rs.Scanner.SetClip(pc.Bounds.Rect.ToRect())
	rs.Path.AddTo(rf)
	rs.SetColor(rf, pc, sty.Fill.Color, sty.Fill.Opacity)
	pr := profile.Start("rasterx-draw")
	rf.Draw()
	rf.Clear()
	pr.End()
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
