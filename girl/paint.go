// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package girl

import (
	"errors"
	"image"
	"image/color"
	"math"

	"github.com/goki/gi/gist"
	"github.com/goki/ki/sliceclone"
	"github.com/goki/mat32"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
)

/*
This borrows heavily from: https://github.com/fogleman/gg

Copyright (C) 2016 Michael Fogleman

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// Paint provides the styling parameters and methods for rendering on to an
// RGBA image -- all dynamic rendering state is maintained in the State.
// Text rendering is handled separately in TextRender, but it depends
// minimally on styling parameters in FontStyle
type Paint struct {
	gist.Paint
}

func NewPaint() Paint {
	p := Paint{}
	p.Defaults()
	return p
}

// convenience for final draw for shapes when done
func (pc *Paint) FillStrokeClear(rs *State) {
	if pc.HasFill() {
		pc.FillPreserve(rs)
	}
	if pc.HasStroke() {
		pc.StrokePreserve(rs)
	}
	pc.ClearPath(rs)
}

//////////////////////////////////////////////////////////////////////////////////
// Path Manipulation

// TransformPoint multiplies the specified point by the current transform matrix,
// returning a transformed position.
func (pc *Paint) TransformPoint(rs *State, x, y float32) mat32.Vec2 {
	return rs.XForm.MulVec2AsPt(mat32.Vec2{x, y})
}

// BoundingBox computes the bounding box for an element in pixel int
// coordinates, applying current transform
func (pc *Paint) BoundingBox(rs *State, minX, minY, maxX, maxY float32) image.Rectangle {
	sw := float32(0.0)
	if pc.HasStroke() {
		sw = 0.5 * pc.StrokeWidth(rs)
	}
	tmin := rs.XForm.MulVec2AsPt(mat32.Vec2{minX, minY})
	tmax := rs.XForm.MulVec2AsPt(mat32.Vec2{maxX, maxY})
	tp1 := mat32.NewVec2(tmin.X-sw, tmin.Y-sw).ToPointFloor()
	tp2 := mat32.NewVec2(tmax.X+sw, tmax.Y+sw).ToPointCeil()
	return image.Rect(tp1.X, tp1.Y, tp2.X, tp2.Y)
}

// BoundingBoxFromPoints computes the bounding box for a slice of points
func (pc *Paint) BoundingBoxFromPoints(rs *State, points []mat32.Vec2) image.Rectangle {
	sz := len(points)
	if sz == 0 {
		return image.Rectangle{}
	}
	min := points[0]
	max := points[1]
	for i := 1; i < sz; i++ {
		min.SetMin(points[i])
		max.SetMax(points[i])
	}
	return pc.BoundingBox(rs, min.X, min.Y, max.X, max.Y)
}

// MoveTo starts a new subpath within the current path starting at the
// specified point.
func (pc *Paint) MoveTo(rs *State, x, y float32) {
	if rs.HasCurrent {
		rs.Path.Stop(false) // note: used to add a point to separate FillPath..
	}
	p := pc.TransformPoint(rs, x, y)
	rs.Path.Start(p.Fixed())
	rs.Start = p
	rs.Current = p
	rs.HasCurrent = true
}

// LineTo adds a line segment to the current path starting at the current
// point. If there is no current point, it is equivalent to MoveTo(x, y)
func (pc *Paint) LineTo(rs *State, x, y float32) {
	if !rs.HasCurrent {
		pc.MoveTo(rs, x, y)
	} else {
		p := pc.TransformPoint(rs, x, y)
		rs.Path.Line(p.Fixed())
		rs.Current = p
	}
}

// QuadraticTo adds a quadratic bezier curve to the current path starting at
// the current point. If there is no current point, it first performs
// MoveTo(x1, y1)
func (pc *Paint) QuadraticTo(rs *State, x1, y1, x2, y2 float32) {
	if !rs.HasCurrent {
		pc.MoveTo(rs, x1, y1)
	}
	p1 := pc.TransformPoint(rs, x1, y1)
	p2 := pc.TransformPoint(rs, x2, y2)
	rs.Path.QuadBezier(p1.Fixed(), p2.Fixed())
	rs.Current = p2
}

// CubicTo adds a cubic bezier curve to the current path starting at the
// current point. If there is no current point, it first performs
// MoveTo(x1, y1).
func (pc *Paint) CubicTo(rs *State, x1, y1, x2, y2, x3, y3 float32) {
	if !rs.HasCurrent {
		pc.MoveTo(rs, x1, y1)
	}
	// x0, y0 := rs.Current.X, rs.Current.Y
	b := pc.TransformPoint(rs, x1, y1)
	c := pc.TransformPoint(rs, x2, y2)
	d := pc.TransformPoint(rs, x3, y3)

	rs.Path.CubeBezier(b.Fixed(), c.Fixed(), d.Fixed())
	rs.Current = d
}

// ClosePath adds a line segment from the current point to the beginning
// of the current subpath. If there is no current point, this is a no-op.
func (pc *Paint) ClosePath(rs *State) {
	if rs.HasCurrent {
		rs.Path.Stop(true)
		rs.Current = rs.Start
	}
}

// ClearPath clears the current path. There is no current point after this
// operation.
func (pc *Paint) ClearPath(rs *State) {
	rs.Path.Clear()
	rs.HasCurrent = false
}

// NewSubPath starts a new subpath within the current path. There is no current
// point after this operation.
func (pc *Paint) NewSubPath(rs *State) {
	// if rs.HasCurrent {
	// 	rs.FillPath.Add1(rs.Start.Fixed())
	// }
	rs.HasCurrent = false
}

// Path Drawing

func (pc *Paint) capfunc() rasterx.CapFunc {
	switch pc.StrokeStyle.Cap {
	case gist.LineCapButt:
		return rasterx.ButtCap
	case gist.LineCapRound:
		return rasterx.RoundCap
	case gist.LineCapSquare:
		return rasterx.SquareCap
	case gist.LineCapCubic:
		return rasterx.CubicCap
	case gist.LineCapQuadratic:
		return rasterx.QuadraticCap
	}
	return nil
}

func (pc *Paint) joinmode() rasterx.JoinMode {
	switch pc.StrokeStyle.Join {
	case gist.LineJoinMiter:
		return rasterx.Miter
	case gist.LineJoinMiterClip:
		return rasterx.MiterClip
	case gist.LineJoinRound:
		return rasterx.Round
	case gist.LineJoinBevel:
		return rasterx.Bevel
	case gist.LineJoinArcs:
		return rasterx.Arc
	case gist.LineJoinArcsClip:
		return rasterx.ArcClip
	}
	return rasterx.Arc
}

// StrokeWidth obtains the current stoke width subject to transform (or not
// depending on VecEffNonScalingStroke)
func (pc *Paint) StrokeWidth(rs *State) float32 {
	dw := pc.StrokeStyle.Width.Dots
	if dw == 0 {
		return dw
	}
	if pc.VecEff == gist.VecEffNonScalingStroke {
		return dw
	}
	scx, scy := rs.XForm.ExtractScale()
	sc := 0.5 * (mat32.Abs(scx) + mat32.Abs(scy))
	lw := mat32.Max(sc*dw, pc.StrokeStyle.MinWidth.Dots)
	return lw
}

func (pc *Paint) stroke(rs *State) {
	if rs.Raster == nil {
		return
	}
	// pr := prof.Start("Paint.stroke")
	// defer pr.End()

	rs.RasterMu.Lock()
	defer rs.RasterMu.Unlock()

	dash := sliceclone.Float64(pc.StrokeStyle.Dashes)
	if dash != nil {
		scx, scy := rs.XForm.ExtractScale()
		sc := 0.5 * (math.Abs(float64(scx)) + math.Abs(float64(scy)))
		hasZero := false
		for i := range dash {
			dash[i] *= sc
			if dash[i] < 1 {
				hasZero = true
				break
			}
		}
		if hasZero {
			dash = nil
		}
	}

	rs.Raster.SetStroke(
		mat32.ToFixed(pc.StrokeWidth(rs)),
		mat32.ToFixed(pc.StrokeStyle.MiterLimit),
		pc.capfunc(), nil, nil, pc.joinmode(), // todo: supports leading / trailing caps, and "gaps"
		dash, 0)
	rs.Scanner.SetClip(rs.Bounds)
	rs.Path.AddTo(rs.Raster)
	fbox := rs.Raster.Scanner.GetPathExtent()
	// fmt.Printf("node: %v fbox: %v\n", g.Nm, fbox)
	rs.LastRenderBBox = image.Rectangle{Min: image.Point{fbox.Min.X.Floor(), fbox.Min.Y.Floor()},
		Max: image.Point{fbox.Max.X.Ceil(), fbox.Max.Y.Ceil()}}
	rs.Raster.SetColor(pc.StrokeStyle.Color.RenderColor(pc.FontStyle.Opacity*pc.StrokeStyle.Opacity, rs.LastRenderBBox, rs.XForm))
	rs.Raster.Draw()
	rs.Raster.Clear()

	/*
		rs.CompSpanner.DrawToImage(rs.Image)
		rs.CompSpanner.Clear()
	*/

}

func (pc *Paint) fill(rs *State) {
	if rs.Raster == nil {
		return
	}
	// pr := prof.Start("Paint.fill")
	// pr.End()

	rs.RasterMu.Lock()
	defer rs.RasterMu.Unlock()

	rf := &rs.Raster.Filler
	rf.SetWinding(pc.FillStyle.Rule == gist.FillRuleNonZero)
	rs.Scanner.SetClip(rs.Bounds)
	rs.Path.AddTo(rf)
	fbox := rs.Scanner.GetPathExtent()
	// fmt.Printf("node: %v fbox: %v\n", g.Nm, fbox)
	rs.LastRenderBBox = image.Rectangle{Min: image.Point{fbox.Min.X.Floor(), fbox.Min.Y.Floor()},
		Max: image.Point{fbox.Max.X.Ceil(), fbox.Max.Y.Ceil()}}
	if pc.FillStyle.Color.Source == gist.RadialGradient {
		rf.SetColor(pc.FillStyle.Color.RenderColor(pc.FontStyle.Opacity*pc.FillStyle.Opacity, rs.LastRenderBBox, rs.XForm))
	} else {
		rf.SetColor(pc.FillStyle.Color.RenderColor(pc.FontStyle.Opacity*pc.FillStyle.Opacity, rs.LastRenderBBox, rs.XForm))
	}
	rf.Draw()
	rf.Clear()

	/*
		rs.CompSpanner.DrawToImage(rs.Image)
		rs.CompSpanner.Clear()
	*/

}

// StrokePreserve strokes the current path with the current color, line width,
// line cap, line join and dash settings. The path is preserved after this
// operation.
func (pc *Paint) StrokePreserve(rs *State) {
	pc.stroke(rs)
}

// Stroke strokes the current path with the current color, line width,
// line cap, line join and dash settings. The path is cleared after this
// operation.
func (pc *Paint) Stroke(rs *State) {
	pc.StrokePreserve(rs)
	pc.ClearPath(rs)
}

// FillPreserve fills the current path with the current color. Open subpaths
// are implicitly closed. The path is preserved after this operation.
func (pc *Paint) FillPreserve(rs *State) {
	pc.fill(rs)
}

// Fill fills the current path with the current color. Open subpaths
// are implicitly closed. The path is cleared after this operation.
func (pc *Paint) Fill(rs *State) {
	pc.FillPreserve(rs)
	pc.ClearPath(rs)
}

// FillBox is an optimized fill of a square region with a uniform color if
// the given color spec is a solid color
func (pc *Paint) FillBox(rs *State, pos, size mat32.Vec2, clr *gist.ColorSpec) {
	if clr.Source == gist.SolidColor {
		b := rs.Bounds.Intersect(mat32.RectFromPosSizeMax(pos, size))
		draw.Draw(rs.Image, b, &image.Uniform{clr.Color}, image.ZP, draw.Src)
	} else {
		pc.FillStyle.SetColorSpec(clr)
		pc.DrawRectangle(rs, pos.X, pos.Y, size.X, size.Y)
		pc.Fill(rs)
	}
}

// FillBoxColor is an optimized fill of a square region with given uniform color
func (pc *Paint) FillBoxColor(rs *State, pos, size mat32.Vec2, clr color.Color) {
	b := rs.Bounds.Intersect(mat32.RectFromPosSizeMax(pos, size))
	draw.Draw(rs.Image, b, &image.Uniform{clr}, image.ZP, draw.Src)
}

// ClipPreserve updates the clipping region by intersecting the current
// clipping region with the current path as it would be filled by pc.Fill().
// The path is preserved after this operation.
func (pc *Paint) ClipPreserve(rs *State) {
	clip := image.NewAlpha(rs.Image.Bounds())
	// painter := raster.NewAlphaOverPainter(clip) // todo!
	pc.fill(rs)
	if rs.Mask == nil {
		rs.Mask = clip
	} else { // todo: this one operation MASSIVELY slows down clip usage -- unclear why
		mask := image.NewAlpha(rs.Image.Bounds())
		draw.DrawMask(mask, mask.Bounds(), clip, image.ZP, rs.Mask, image.ZP, draw.Over)
		rs.Mask = mask
	}
}

// SetMask allows you to directly set the *image.Alpha to be used as a clipping
// mask. It must be the same size as the context, else an error is returned
// and the mask is unchanged.
func (pc *Paint) SetMask(rs *State, mask *image.Alpha) error {
	if mask.Bounds() != rs.Image.Bounds() {
		return errors.New("mask size must match context size")
	}
	rs.Mask = mask
	return nil
}

// AsMask returns an *image.Alpha representing the alpha channel of this
// context. This can be useful for advanced clipping operations where you first
// render the mask geometry and then use it as a mask.
func (pc *Paint) AsMask(rs *State) *image.Alpha {
	b := rs.Image.Bounds()
	mask := image.NewAlpha(b)
	draw.Draw(mask, b, rs.Image, image.ZP, draw.Src)
	return mask
}

// Clip updates the clipping region by intersecting the current
// clipping region with the current path as it would be filled by pc.Fill().
// The path is cleared after this operation.
func (pc *Paint) Clip(rs *State) {
	pc.ClipPreserve(rs)
	pc.ClearPath(rs)
}

// ResetClip clears the clipping region.
func (pc *Paint) ResetClip(rs *State) {
	rs.Mask = nil
}

//////////////////////////////////////////////////////////////////////////////////
// Convenient Drawing Functions

// Clear fills the entire image with the current fill color.
func (pc *Paint) Clear(rs *State) {
	src := image.NewUniform(&pc.FillStyle.Color.Color)
	draw.Draw(rs.Image, rs.Image.Bounds(), src, image.ZP, draw.Src)
}

// SetPixel sets the color of the specified pixel using the current stroke color.
func (pc *Paint) SetPixel(rs *State, x, y int) {
	rs.Image.Set(x, y, &pc.StrokeStyle.Color.Color)
}

func (pc *Paint) DrawLine(rs *State, x1, y1, x2, y2 float32) {
	pc.MoveTo(rs, x1, y1)
	pc.LineTo(rs, x2, y2)
}

func (pc *Paint) DrawPolyline(rs *State, points []mat32.Vec2) {
	sz := len(points)
	if sz < 2 {
		return
	}
	pc.MoveTo(rs, points[0].X, points[0].Y)
	for i := 1; i < sz; i++ {
		pc.LineTo(rs, points[i].X, points[i].Y)
	}
}

func (pc *Paint) DrawPolylinePxToDots(rs *State, points []mat32.Vec2) {
	pu := &pc.UnContext
	sz := len(points)
	if sz < 2 {
		return
	}
	pc.MoveTo(rs, pu.PxToDots(points[0].X), pu.PxToDots(points[0].Y))
	for i := 1; i < sz; i++ {
		pc.LineTo(rs, pu.PxToDots(points[i].X), pu.PxToDots(points[i].Y))
	}
}

func (pc *Paint) DrawPolygon(rs *State, points []mat32.Vec2) {
	pc.DrawPolyline(rs, points)
	pc.ClosePath(rs)
}

func (pc *Paint) DrawPolygonPxToDots(rs *State, points []mat32.Vec2) {
	pc.DrawPolylinePxToDots(rs, points)
	pc.ClosePath(rs)
}

func (pc *Paint) DrawRectangle(rs *State, x, y, w, h float32) {
	pc.NewSubPath(rs)
	pc.MoveTo(rs, x, y)
	pc.LineTo(rs, x+w, y)
	pc.LineTo(rs, x+w, y+h)
	pc.LineTo(rs, x, y+h)
	pc.ClosePath(rs)
}

func (pc *Paint) DrawRoundedRectangle(rs *State, x, y, w, h, r float32) {
	x0, x1, x2, x3 := x, x+r, x+w-r, x+w
	y0, y1, y2, y3 := y, y+r, y+h-r, y+h
	pc.NewSubPath(rs)
	pc.MoveTo(rs, x1, y0)
	pc.LineTo(rs, x2, y0)
	pc.DrawArc(rs, x2, y1, r, mat32.DegToRad(270), mat32.DegToRad(360))
	pc.LineTo(rs, x3, y2)
	pc.DrawArc(rs, x2, y2, r, mat32.DegToRad(0), mat32.DegToRad(90))
	pc.LineTo(rs, x1, y3)
	pc.DrawArc(rs, x1, y2, r, mat32.DegToRad(90), mat32.DegToRad(180))
	pc.LineTo(rs, x0, y1)
	pc.DrawArc(rs, x1, y1, r, mat32.DegToRad(180), mat32.DegToRad(270))
	pc.ClosePath(rs)
}

// DrawEllipticalArc draws arc between angle1 and angle2 along an ellipse,
// using quadratic bezier curves -- centers of ellipse are at cx, cy with
// radii rx, ry -- see DrawEllipticalArcPath for a version compatible with SVG
// A/a path drawing, which uses previous position instead of two angles
func (pc *Paint) DrawEllipticalArc(rs *State, cx, cy, rx, ry, angle1, angle2 float32) {
	const n = 16
	for i := 0; i < n; i++ {
		p1 := float32(i+0) / n
		p2 := float32(i+1) / n
		a1 := angle1 + (angle2-angle1)*p1
		a2 := angle1 + (angle2-angle1)*p2
		x0 := cx + rx*mat32.Cos(a1)
		y0 := cy + ry*mat32.Sin(a1)
		x1 := cx + rx*mat32.Cos((a1+a2)/2)
		y1 := cy + ry*mat32.Sin((a1+a2)/2)
		x2 := cx + rx*mat32.Cos(a2)
		y2 := cy + ry*mat32.Sin(a2)
		ncx := 2*x1 - x0/2 - x2/2
		ncy := 2*y1 - y0/2 - y2/2
		if i == 0 && !rs.HasCurrent {
			pc.MoveTo(rs, x0, y0)
		}
		pc.QuadraticTo(rs, ncx, ncy, x2, y2)
	}
}

// following ellipse path code is all directly from srwiley/oksvg

// MaxDx is the Maximum radians a cubic splice is allowed to span
// in ellipse parametric when approximating an off-axis ellipse.
const MaxDx float32 = math.Pi / 8

// ellipsePrime gives tangent vectors for parameterized ellipse; a, b, radii,
// eta parameter, center cx, cy
func ellipsePrime(a, b, sinTheta, cosTheta, eta, cx, cy float32) (px, py float32) {
	bCosEta := b * mat32.Cos(eta)
	aSinEta := a * mat32.Sin(eta)
	px = -aSinEta*cosTheta - bCosEta*sinTheta
	py = -aSinEta*sinTheta + bCosEta*cosTheta
	return
}

// ellipsePointAt gives points for parameterized ellipse; a, b, radii, eta
// parameter, center cx, cy
func ellipsePointAt(a, b, sinTheta, cosTheta, eta, cx, cy float32) (px, py float32) {
	aCosEta := a * mat32.Cos(eta)
	bSinEta := b * mat32.Sin(eta)
	px = cx + aCosEta*cosTheta - bSinEta*sinTheta
	py = cy + aCosEta*sinTheta + bSinEta*cosTheta
	return
}

// FindEllipseCenter locates the center of the Ellipse if it exists. If it
// does not exist, the radius values will be increased minimally for a
// solution to be possible while preserving the rx to rb ratio.  rx and rb
// arguments are pointers that can be checked after the call to see if the
// values changed. This method uses coordinate transformations to reduce the
// problem to finding the center of a circle that includes the origin and an
// arbitrary point. The center of the circle is then transformed back to the
// original coordinates and returned.
func FindEllipseCenter(rx, ry *float32, rotX, startX, startY, endX, endY float32, sweep, largeArc bool) (cx, cy float32) {
	cos, sin := mat32.Cos(rotX), mat32.Sin(rotX)

	// Move origin to start point
	nx, ny := endX-startX, endY-startY

	// Rotate ellipse x-axis to coordinate x-axis
	nx, ny = nx*cos+ny*sin, -nx*sin+ny*cos
	// Scale X dimension so that rx = ry
	nx *= *ry / *rx // Now the ellipse is a circle radius ry; therefore foci and center coincide

	midX, midY := nx/2, ny/2
	midlenSq := midX*midX + midY*midY

	var hr float32 = 0.0
	if *ry**ry < midlenSq {
		// Requested ellipse does not exist; scale rx, ry to fit. Length of
		// span is greater than max width of ellipse, must scale *rx, *ry
		nry := mat32.Sqrt(midlenSq)
		if *rx == *ry {
			*rx = nry // prevents roundoff
		} else {
			*rx = *rx * nry / *ry
		}
		*ry = nry
	} else {
		hr = mat32.Sqrt(*ry**ry-midlenSq) / mat32.Sqrt(midlenSq)
	}
	// Notice that if hr is zero, both answers are the same.
	if (!sweep && !largeArc) || (sweep && largeArc) {
		cx = midX + midY*hr
		cy = midY - midX*hr
	} else {
		cx = midX - midY*hr
		cy = midY + midX*hr
	}

	// reverse scale
	cx *= *rx / *ry
	//Reverse rotate and translate back to original coordinates
	return cx*cos - cy*sin + startX, cx*sin + cy*cos + startY
}

// DrawEllipticalArcPath is draws an arc centered at cx,cy with radii rx, ry, through
// given angle, either via the smaller or larger arc, depending on largeArc --
// returns in lx, ly the last points which are then set to the current cx, cy
// for the path drawer
func (pc *Paint) DrawEllipticalArcPath(rs *State, cx, cy, ocx, ocy, pcx, pcy, rx, ry, angle float32, largeArc, sweep bool) (lx, ly float32) {
	rotX := angle * math.Pi / 180 // Convert degrees to radians
	startAngle := mat32.Atan2(pcy-cy, pcx-cx) - rotX
	endAngle := mat32.Atan2(ocy-cy, ocx-cx) - rotX
	deltaTheta := endAngle - startAngle
	arcBig := mat32.Abs(deltaTheta) > math.Pi

	// Approximate ellipse using cubic bezier splines
	etaStart := mat32.Atan2(mat32.Sin(startAngle)/ry, mat32.Cos(startAngle)/rx)
	etaEnd := mat32.Atan2(mat32.Sin(endAngle)/ry, mat32.Cos(endAngle)/rx)
	deltaEta := etaEnd - etaStart
	if (arcBig && !largeArc) || (!arcBig && largeArc) { // Go has no boolean XOR
		if deltaEta < 0 {
			deltaEta += math.Pi * 2
		} else {
			deltaEta -= math.Pi * 2
		}
	}
	// This check might be needed if the center point of the ellipse is
	// at the midpoint of the start and end lines.
	if deltaEta < 0 && sweep {
		deltaEta += math.Pi * 2
	} else if deltaEta >= 0 && !sweep {
		deltaEta -= math.Pi * 2
	}

	// Round up to determine number of cubic splines to approximate bezier curve
	segs := int(mat32.Abs(deltaEta)/MaxDx) + 1
	dEta := deltaEta / float32(segs) // span of each segment
	// Approximate the ellipse using a set of cubic bezier curves by the method of
	// L. Maisonobe, "Drawing an elliptical arc using polylines, quadratic
	// or cubic Bezier curves", 2003
	// https://www.spaceroots.org/documents/ellipse/elliptical-arc.pdf
	tde := mat32.Tan(dEta / 2)
	alpha := mat32.Sin(dEta) * (mat32.Sqrt(4+3*tde*tde) - 1) / 3 // Math is fun!
	lx, ly = pcx, pcy
	sinTheta, cosTheta := mat32.Sin(rotX), mat32.Cos(rotX)
	ldx, ldy := ellipsePrime(rx, ry, sinTheta, cosTheta, etaStart, cx, cy)

	for i := 1; i <= segs; i++ {
		eta := etaStart + dEta*float32(i)
		var px, py float32
		if i == segs {
			px, py = ocx, ocy // Just makes the end point exact; no roundoff error
		} else {
			px, py = ellipsePointAt(rx, ry, sinTheta, cosTheta, eta, cx, cy)
		}
		dx, dy := ellipsePrime(rx, ry, sinTheta, cosTheta, eta, cx, cy)
		pc.CubicTo(rs, lx+alpha*ldx, ly+alpha*ldy, px-alpha*dx, py-alpha*dy, px, py)
		lx, ly, ldx, ldy = px, py, dx, dy
	}
	return lx, ly
}

func (pc *Paint) DrawEllipse(rs *State, x, y, rx, ry float32) {
	pc.NewSubPath(rs)
	pc.DrawEllipticalArc(rs, x, y, rx, ry, 0, 2*mat32.Pi)
	pc.ClosePath(rs)
}

func (pc *Paint) DrawArc(rs *State, x, y, r, angle1, angle2 float32) {
	pc.DrawEllipticalArc(rs, x, y, r, r, angle1, angle2)
}

func (pc *Paint) DrawCircle(rs *State, x, y, r float32) {
	pc.NewSubPath(rs)
	pc.DrawEllipticalArc(rs, x, y, r, r, 0, 2*mat32.Pi)
	pc.ClosePath(rs)
}

func (pc *Paint) DrawRegularPolygon(rs *State, n int, x, y, r, rotation float32) {
	angle := 2 * mat32.Pi / float32(n)
	rotation -= mat32.Pi / 2
	if n%2 == 0 {
		rotation += angle / 2
	}
	pc.NewSubPath(rs)
	for i := 0; i < n; i++ {
		a := rotation + angle*float32(i)
		pc.LineTo(rs, x+r*mat32.Cos(a), y+r*mat32.Sin(a))
	}
	pc.ClosePath(rs)
}

// DrawImage draws the specified image at the specified point.
func (pc *Paint) DrawImage(rs *State, fmIm image.Image, x, y float32) {
	pc.DrawImageAnchored(rs, fmIm, x, y, 0, 0)
}

// DrawImageAnchored draws the specified image at the specified anchor point.
// The anchor point is x - w * ax, y - h * ay, where w, h is the size of the
// image. Use ax=0.5, ay=0.5 to center the image at the specified point.
func (pc *Paint) DrawImageAnchored(rs *State, fmIm image.Image, x, y, ax, ay float32) {
	s := fmIm.Bounds().Size()
	x -= ax * float32(s.X)
	y -= ay * float32(s.Y)
	transformer := draw.BiLinear
	m := rs.XForm.Translate(x, y)
	s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
	if rs.Mask == nil {
		transformer.Transform(rs.Image, s2d, fmIm, fmIm.Bounds(), draw.Over, nil)
	} else {
		transformer.Transform(rs.Image, s2d, fmIm, fmIm.Bounds(), draw.Over, &draw.Options{
			DstMask:  rs.Mask,
			DstMaskP: image.ZP,
		})
	}
}

// DrawImageScaled draws the specified image starting at given upper-left point,
// such that the size of the image is rendered as specified by w, h parameters
// (an additional scaling is applied to the transform matrix used in rendering)
func (pc *Paint) DrawImageScaled(rs *State, fmIm image.Image, x, y, w, h float32) {
	s := fmIm.Bounds().Size()
	isz := mat32.NewVec2FmPoint(s)
	isc := mat32.Vec2{w, h}.Div(isz)

	transformer := draw.BiLinear
	m := rs.XForm.Translate(x, y).Scale(isc.X, isc.Y)
	s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
	if rs.Mask == nil {
		transformer.Transform(rs.Image, s2d, fmIm, fmIm.Bounds(), draw.Over, nil)
	} else {
		transformer.Transform(rs.Image, s2d, fmIm, fmIm.Bounds(), draw.Over, &draw.Options{
			DstMask:  rs.Mask,
			DstMaskP: image.ZP,
		})
	}
}

//////////////////////////////////////////////////////////////////////////////////
// Transformation Matrix Operations

// Identity resets the current transformation matrix to the identity matrix.
// This results in no translating, scaling, rotating, or shearing.
func (pc *Paint) Identity() {
	pc.XForm = mat32.Identity2D()
}

// Translate updates the current matrix with a translation.
func (pc *Paint) Translate(x, y float32) {
	pc.XForm = pc.XForm.Translate(x, y)
}

// Scale updates the current matrix with a scaling factor.
// Scaling occurs about the origin.
func (pc *Paint) Scale(x, y float32) {
	pc.XForm = pc.XForm.Scale(x, y)
}

// ScaleAbout updates the current matrix with a scaling factor.
// Scaling occurs about the specified point.
func (pc *Paint) ScaleAbout(sx, sy, x, y float32) {
	pc.Translate(x, y)
	pc.Scale(sx, sy)
	pc.Translate(-x, -y)
}

// Rotate updates the current matrix with a clockwise rotation.
// Rotation occurs about the origin. Angle is specified in radians.
func (pc *Paint) Rotate(angle float32) {
	pc.XForm = pc.XForm.Rotate(angle)
}

// RotateAbout updates the current matrix with a clockwise rotation.
// Rotation occurs about the specified point. Angle is specified in radians.
func (pc *Paint) RotateAbout(angle, x, y float32) {
	pc.Translate(x, y)
	pc.Rotate(angle)
	pc.Translate(-x, -y)
}

// Shear updates the current matrix with a shearing angle.
// Shearing occurs about the origin.
func (pc *Paint) Shear(x, y float32) {
	pc.XForm = pc.XForm.Shear(x, y)
}

// ShearAbout updates the current matrix with a shearing angle.
// Shearing occurs about the specified point.
func (pc *Paint) ShearAbout(sx, sy, x, y float32) {
	pc.Translate(x, y)
	pc.Shear(sx, sy)
	pc.Translate(-x, -y)
}

// InvertY flips the Y axis so that Y grows from bottom to top and Y=0 is at
// the bottom of the image.
func (pc *Paint) InvertY(rs *State) {
	pc.Translate(0, float32(rs.Image.Bounds().Size().Y))
	pc.Scale(1, -1)
}
