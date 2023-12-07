// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"errors"
	"image"
	"image/color"
	"math"
	"slices"

	"github.com/srwiley/rasterx"
	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/mat32/v2"
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

// Context provides the rendering state, styling parameters, and methods for
// painting. It is the main entry point to the paint API; most things are methods
// on Context, although Text rendering is handled separately in TextRender.
// A Context is typically constructed through [NewContext], [NewContextFromImage],
// or [NewContextFromRGBA], although it can also be constructed directly through
// a struct literal when an existing [State] and [styles.Paint] exist.
type Context struct {
	*State
	*styles.Paint
}

// NewContext returns a new [Context] associated with a new [image.RGBA]
// with the given width and height.
func NewContext(width, height int) *Context {
	pc := &Context{&State{}, &styles.Paint{}}

	sz := image.Pt(width, height)
	img := image.NewRGBA(image.Rectangle{Max: sz})
	pc.Init(width, height, img)

	pc.Defaults()
	pc.SetUnitContextExt(sz)

	return pc
}

// convenience for final draw for shapes when done
func (pc *Context) FillStrokeClear() {
	if pc.HasFill() {
		pc.FillPreserve()
	}
	if pc.HasStroke() {
		pc.StrokePreserve()
	}
	pc.ClearPath()
}

//////////////////////////////////////////////////////////////////////////////////
// Path Manipulation

// TransformPoint multiplies the specified point by the current transform matrix,
// returning a transformed position.
func (pc *Context) TransformPoint(x, y float32) mat32.Vec2 {
	return pc.CurXForm.MulVec2AsPt(mat32.Vec2{x, y})
}

// BoundingBox computes the bounding box for an element in pixel int
// coordinates, applying current transform
func (pc *Context) BoundingBox(minX, minY, maxX, maxY float32) image.Rectangle {
	sw := float32(0.0)
	if pc.HasStroke() {
		sw = 0.5 * pc.StrokeWidth()
	}
	tmin := pc.CurXForm.MulVec2AsPt(mat32.Vec2{minX, minY})
	tmax := pc.CurXForm.MulVec2AsPt(mat32.Vec2{maxX, maxY})
	tp1 := mat32.NewVec2(tmin.X-sw, tmin.Y-sw).ToPointFloor()
	tp2 := mat32.NewVec2(tmax.X+sw, tmax.Y+sw).ToPointCeil()
	return image.Rect(tp1.X, tp1.Y, tp2.X, tp2.Y)
}

// BoundingBoxFromPoints computes the bounding box for a slice of points
func (pc *Context) BoundingBoxFromPoints(points []mat32.Vec2) image.Rectangle {
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
	return pc.BoundingBox(min.X, min.Y, max.X, max.Y)
}

// MoveTo starts a new subpath within the current path starting at the
// specified point.
func (pc *Context) MoveTo(x, y float32) {
	if pc.HasCurrent {
		pc.Path.Stop(false) // note: used to add a point to separate FillPath..
	}
	p := pc.TransformPoint(x, y)
	pc.Path.Start(p.Fixed())
	pc.Start = p
	pc.Current = p
	pc.HasCurrent = true
}

// LineTo adds a line segment to the current path starting at the current
// point. If there is no current point, it is equivalent to MoveTo(x, y)
func (pc *Context) LineTo(x, y float32) {
	if !pc.HasCurrent {
		pc.MoveTo(x, y)
	} else {
		p := pc.TransformPoint(x, y)
		pc.Path.Line(p.Fixed())
		pc.Current = p
	}
}

// QuadraticTo adds a quadratic bezier curve to the current path starting at
// the current point. If there is no current point, it first performs
// MoveTo(x1, y1)
func (pc *Context) QuadraticTo(x1, y1, x2, y2 float32) {
	if !pc.HasCurrent {
		pc.MoveTo(x1, y1)
	}
	p1 := pc.TransformPoint(x1, y1)
	p2 := pc.TransformPoint(x2, y2)
	pc.Path.QuadBezier(p1.Fixed(), p2.Fixed())
	pc.Current = p2
}

// CubicTo adds a cubic bezier curve to the current path starting at the
// current point. If there is no current point, it first performs
// MoveTo(x1, y1).
func (pc *Context) CubicTo(x1, y1, x2, y2, x3, y3 float32) {
	if !pc.HasCurrent {
		pc.MoveTo(x1, y1)
	}
	// x0, y0 := pc.Current.X, pc.Current.Y
	b := pc.TransformPoint(x1, y1)
	c := pc.TransformPoint(x2, y2)
	d := pc.TransformPoint(x3, y3)

	pc.Path.CubeBezier(b.Fixed(), c.Fixed(), d.Fixed())
	pc.Current = d
}

// ClosePath adds a line segment from the current point to the beginning
// of the current subpath. If there is no current point, this is a no-op.
func (pc *Context) ClosePath() {
	if pc.HasCurrent {
		pc.Path.Stop(true)
		pc.Current = pc.Start
	}
}

// ClearPath clears the current path. There is no current point after this
// operation.
func (pc *Context) ClearPath() {
	pc.Path.Clear()
	pc.HasCurrent = false
}

// NewSubPath starts a new subpath within the current path. There is no current
// point after this operation.
func (pc *Context) NewSubPath() {
	// if pc.HasCurrent {
	// 	pc.FillPath.Add1(pc.Start.Fixed())
	// }
	pc.HasCurrent = false
}

// Path Drawing

func (pc *Context) capfunc() rasterx.CapFunc {
	switch pc.StrokeStyle.Cap {
	case styles.LineCapButt:
		return rasterx.ButtCap
	case styles.LineCapRound:
		return rasterx.RoundCap
	case styles.LineCapSquare:
		return rasterx.SquareCap
	case styles.LineCapCubic:
		return rasterx.CubicCap
	case styles.LineCapQuadratic:
		return rasterx.QuadraticCap
	}
	return nil
}

func (pc *Context) joinmode() rasterx.JoinMode {
	switch pc.StrokeStyle.Join {
	case styles.LineJoinMiter:
		return rasterx.Miter
	case styles.LineJoinMiterClip:
		return rasterx.MiterClip
	case styles.LineJoinRound:
		return rasterx.Round
	case styles.LineJoinBevel:
		return rasterx.Bevel
	case styles.LineJoinArcs:
		return rasterx.Arc
	case styles.LineJoinArcsClip:
		return rasterx.ArcClip
	}
	return rasterx.Arc
}

// StrokeWidth obtains the current stoke width subject to transform (or not
// depending on VecEffNonScalingStroke)
func (pc *Context) StrokeWidth() float32 {
	dw := pc.StrokeStyle.Width.Dots
	if dw == 0 {
		return dw
	}
	if pc.VecEff == styles.VecEffNonScalingStroke {
		return dw
	}
	scx, scy := pc.CurXForm.ExtractScale()
	sc := 0.5 * (mat32.Abs(scx) + mat32.Abs(scy))
	lw := mat32.Max(sc*dw, pc.StrokeStyle.MinWidth.Dots)
	return lw
}

func (pc *Context) stroke() {
	if pc.Raster == nil {
		return
	}
	// pr := prof.Start("Paint.stroke")
	// defer pr.End()

	pc.RasterMu.Lock()
	defer pc.RasterMu.Unlock()

	dash := slices.Clone(pc.StrokeStyle.Dashes)
	if dash != nil {
		scx, scy := pc.CurXForm.ExtractScale()
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

	pc.Raster.SetStroke(
		mat32.ToFixed(pc.StrokeWidth()),
		mat32.ToFixed(pc.StrokeStyle.MiterLimit),
		pc.capfunc(), nil, nil, pc.joinmode(), // todo: supports leading / trailing caps, and "gaps"
		dash, 0)
	pc.Scanner.SetClip(pc.Bounds)
	pc.Path.AddTo(pc.Raster)
	fbox := pc.Raster.Scanner.GetPathExtent()
	// fmt.Printf("node: %v fbox: %v\n", g.Nm, fbox)
	pc.LastRenderBBox = image.Rectangle{Min: image.Point{fbox.Min.X.Floor(), fbox.Min.Y.Floor()},
		Max: image.Point{fbox.Max.X.Ceil(), fbox.Max.Y.Ceil()}}
	pc.Raster.SetColor(pc.StrokeStyle.Color.RenderColor(pc.FontStyle.Opacity*pc.StrokeStyle.Opacity, pc.LastRenderBBox, pc.CurXForm))
	pc.Raster.Draw()
	pc.Raster.Clear()

	/*
		pc.CompSpanner.DrawToImage(pc.Image)
		pc.CompSpanner.Clear()
	*/

}

func (pc *Context) fill() {
	if pc.Raster == nil {
		return
	}
	// pr := prof.Start("Paint.fill")
	// pr.End()

	pc.RasterMu.Lock()
	defer pc.RasterMu.Unlock()

	rf := &pc.Raster.Filler
	rf.SetWinding(pc.FillStyle.Rule == styles.FillRuleNonZero)
	pc.Scanner.SetClip(pc.Bounds)
	pc.Path.AddTo(rf)
	fbox := pc.Scanner.GetPathExtent()
	// fmt.Printf("node: %v fbox: %v\n", g.Nm, fbox)
	pc.LastRenderBBox = image.Rectangle{Min: image.Point{fbox.Min.X.Floor(), fbox.Min.Y.Floor()},
		Max: image.Point{fbox.Max.X.Ceil(), fbox.Max.Y.Ceil()}}
	rf.SetColor(pc.FillStyle.Color.RenderColor(pc.FontStyle.Opacity*pc.FillStyle.Opacity, pc.LastRenderBBox, pc.CurXForm))
	rf.Draw()
	rf.Clear()

	/*
		pc.CompSpanner.DrawToImage(pc.Image)
		pc.CompSpanner.Clear()
	*/

}

// StrokePreserve strokes the current path with the current color, line width,
// line cap, line join and dash settings. The path is preserved after this
// operation.
func (pc *Context) StrokePreserve() {
	pc.stroke()
}

// Stroke strokes the current path with the current color, line width,
// line cap, line join and dash settings. The path is cleared after this
// operation.
func (pc *Context) Stroke() {
	pc.StrokePreserve()
	pc.ClearPath()
}

// FillPreserve fills the current path with the current color. Open subpaths
// are implicitly closed. The path is preserved after this operation.
func (pc *Context) FillPreserve() {
	pc.fill()
}

// Fill fills the current path with the current color. Open subpaths
// are implicitly closed. The path is cleared after this operation.
func (pc *Context) Fill() {
	pc.FillPreserve()
	pc.ClearPath()
}

// FillBox is an optimized fill of a square region with a uniform color if
// the given color spec is a solid color
func (pc *Context) FillBox(pos, size mat32.Vec2, clr *colors.Full) {
	if clr.Gradient == nil {
		b := pc.Bounds.Intersect(mat32.RectFromPosSizeMax(pos, size))
		draw.Draw(pc.Image, b, &image.Uniform{clr.Solid}, image.Point{}, draw.Src)
	} else {
		pc.FillStyle.SetFullColor(clr)
		pc.DrawRectangle(pos.X, pos.Y, size.X, size.Y)
		pc.Fill()
	}
}

// FillBoxColor is an optimized fill of a square region with given uniform color
func (pc *Context) FillBoxColor(pos, size mat32.Vec2, clr color.Color) {
	b := pc.Bounds.Intersect(mat32.RectFromPosSizeMax(pos, size))
	draw.Draw(pc.Image, b, &image.Uniform{clr}, image.Point{}, draw.Src)
}

// BlurBox blurs the given already drawn region with the given blur radius.
// The blur radius passed to this function is the actual Gaussian
// standard deviation (σ). This means that you need to divide a CSS-standard
// blur radius value by two before passing it this function
// (see https://stackoverflow.com/questions/65454183/how-does-blur-radius-value-in-box-shadow-property-affect-the-resulting-blur).
func (pc *Context) BlurBox(pos, size mat32.Vec2, blurRadius float32) {
	rect := mat32.RectFromPosSizeMax(pos, size)
	sub := pc.Image.SubImage(rect)
	sub = GaussianBlur(sub, float64(blurRadius))
	draw.Draw(pc.Image, rect, sub, rect.Min, draw.Src)
}

// ClipPreserve updates the clipping region by intersecting the current
// clipping region with the current path as it would be filled by pc.Fill().
// The path is preserved after this operation.
func (pc *Context) ClipPreserve() {
	clip := image.NewAlpha(pc.Image.Bounds())
	// painter := raster.NewAlphaOverPainter(clip) // todo!
	pc.fill()
	if pc.Mask == nil {
		pc.Mask = clip
	} else { // todo: this one operation MASSIVELY slows down clip usage -- unclear why
		mask := image.NewAlpha(pc.Image.Bounds())
		draw.DrawMask(mask, mask.Bounds(), clip, image.Point{}, pc.Mask, image.Point{}, draw.Over)
		pc.Mask = mask
	}
}

// SetMask allows you to directly set the *image.Alpha to be used as a clipping
// mask. It must be the same size as the context, else an error is returned
// and the mask is unchanged.
func (pc *Context) SetMask(mask *image.Alpha) error {
	if mask.Bounds() != pc.Image.Bounds() {
		return errors.New("mask size must match context size")
	}
	pc.Mask = mask
	return nil
}

// AsMask returns an *image.Alpha representing the alpha channel of this
// context. This can be useful for advanced clipping operations where you first
// render the mask geometry and then use it as a mask.
func (pc *Context) AsMask() *image.Alpha {
	b := pc.Image.Bounds()
	mask := image.NewAlpha(b)
	draw.Draw(mask, b, pc.Image, image.Point{}, draw.Src)
	return mask
}

// Clip updates the clipping region by intersecting the current
// clipping region with the current path as it would be filled by pc.Fill().
// The path is cleared after this operation.
func (pc *Context) Clip() {
	pc.ClipPreserve()
	pc.ClearPath()
}

// ResetClip clears the clipping region.
func (pc *Context) ResetClip() {
	pc.Mask = nil
}

//////////////////////////////////////////////////////////////////////////////////
// Convenient Drawing Functions

// Clear fills the entire image with the current fill color.
func (pc *Context) Clear() {
	src := image.NewUniform(&pc.FillStyle.Color.Solid)
	draw.Draw(pc.Image, pc.Image.Bounds(), src, image.Point{}, draw.Src)
}

// SetPixel sets the color of the specified pixel using the current stroke color.
func (pc *Context) SetPixel(x, y int) {
	pc.Image.Set(x, y, &pc.StrokeStyle.Color.Solid)
}

func (pc *Context) DrawLine(x1, y1, x2, y2 float32) {
	pc.MoveTo(x1, y1)
	pc.LineTo(x2, y2)
}

func (pc *Context) DrawPolyline(points []mat32.Vec2) {
	sz := len(points)
	if sz < 2 {
		return
	}
	pc.MoveTo(points[0].X, points[0].Y)
	for i := 1; i < sz; i++ {
		pc.LineTo(points[i].X, points[i].Y)
	}
}

func (pc *Context) DrawPolylinePxToDots(points []mat32.Vec2) {
	pu := &pc.UnContext
	sz := len(points)
	if sz < 2 {
		return
	}
	pc.MoveTo(pu.PxToDots(points[0].X), pu.PxToDots(points[0].Y))
	for i := 1; i < sz; i++ {
		pc.LineTo(pu.PxToDots(points[i].X), pu.PxToDots(points[i].Y))
	}
}

func (pc *Context) DrawPolygon(points []mat32.Vec2) {
	pc.DrawPolyline(points)
	pc.ClosePath()
}

func (pc *Context) DrawPolygonPxToDots(points []mat32.Vec2) {
	pc.DrawPolylinePxToDots(points)
	pc.ClosePath()
}

// // DrawRectangle draws a rectangle by setting the stroke style and width
// // and calling DrawConsistentRectangle if the given border width and
// // color styles for each side are the same. Otherwise, it calls DrawChangingRectangle.
// func (pc *Paint) DrawRectangle1(x, y, w, h float32, bs styles.Border) {
// 	if bs.Color.AllSame() && bs.Width.Dots().AllSame() {
// 		// set the color if it is not the same as the already set color
// 		if pc.StrokeStyle.Color.Source != styles.SolidColor || bs.Color.Top != pc.StrokeStyle.Color.Color {
// 			pc.StrokeStyle.SetColor(bs.Color.Top)
// 		}
// 		pc.StrokeStyle.Width = bs.Width.Top
// 		pc.DrawConsistentRectangle(x, y, w, h)
// 		return
// 	}
// 	pc.DrawChangingRectangle(x, y, w, h, bs)
// }

// DrawBorder is a higher-level function that draws, strokes, and fills
// an potentially rounded border box with the given position, size, and border styles.
func (pc *Context) DrawBorder(x, y, w, h float32, bs styles.Border) {
	r := bs.Radius.Dots()
	if bs.Color.AllSame() && bs.Width.Dots().AllSame() {
		// set the color if it is not nil and the stroke style is not on and set to the correct color
		if !colors.IsNil(bs.Color.Top) && (!pc.StrokeStyle.On || pc.StrokeStyle.Color.Gradient != nil || bs.Color.Top != pc.StrokeStyle.Color.Solid) {
			pc.StrokeStyle.SetColor(bs.Color.Top)
		}
		pc.StrokeStyle.Width = bs.Width.Top
		if r.IsZero() {
			pc.DrawRectangle(x, y, w, h)
		} else {
			pc.DrawRoundedRectangle(x, y, w, h, r)
		}
		pc.FillStrokeClear()
		return
	}

	// use consistent rounded rectangle for fill, and then draw borders side by side
	pc.DrawRoundedRectangle(x, y, w, h, r)
	pc.Fill()

	// clamp border radius values
	min := mat32.Min(w/2, h/2)
	r.Top = mat32.Clamp(r.Top, 0, min)
	r.Right = mat32.Clamp(r.Right, 0, min)
	r.Bottom = mat32.Clamp(r.Bottom, 0, min)
	r.Left = mat32.Clamp(r.Left, 0, min)

	// position values
	var (
		xtl, ytl   = x, y                 // top left
		xtli, ytli = x + r.Top, y + r.Top // top left inset

		xtr, ytr   = x + w, y                     // top right
		xtri, ytri = x + w - r.Right, y + r.Right // top right inset

		xbr, ybr   = x + w, y + h                       // bottom right
		xbri, ybri = x + w - r.Bottom, y + h - r.Bottom // bottom right inset

		xbl, ybl   = x, y + h                   // bottom left
		xbli, ybli = x + r.Left, y + h - r.Left // bottom left inset
	)

	// SidesTODO: need to figure out how to style rounded corners correctly
	// (in CSS they are split in the middle between different border side styles)

	pc.NewSubPath()
	pc.MoveTo(xtli, ytl)

	// set the color if it is not the same as the already set color
	if pc.StrokeStyle.Color.Gradient != nil || bs.Color.Top != pc.StrokeStyle.Color.Solid {
		pc.StrokeStyle.SetColor(bs.Color.Top)
	}
	pc.StrokeStyle.Width = bs.Width.Top
	pc.LineTo(xtri, ytr)
	if r.Right != 0 {
		pc.DrawArc(xtri, ytri, r.Right, mat32.DegToRad(270), mat32.DegToRad(360))
	}
	// if the color or width is changing for the next one, we have to stroke now
	if bs.Color.Top != bs.Color.Right || bs.Width.Top.Dots != bs.Width.Right.Dots {
		pc.Stroke()
		pc.NewSubPath()
		pc.MoveTo(xtr, ytri)
	}

	if bs.Color.Right != pc.StrokeStyle.Color.Solid {
		pc.StrokeStyle.SetColor(bs.Color.Right)
	}
	pc.StrokeStyle.Width = bs.Width.Right
	pc.LineTo(xbr, ybri)
	if r.Bottom != 0 {
		pc.DrawArc(xbri, ybri, r.Bottom, mat32.DegToRad(0), mat32.DegToRad(90))
	}
	if bs.Color.Right != bs.Color.Bottom || bs.Width.Right.Dots != bs.Width.Bottom.Dots {
		pc.Stroke()
		pc.NewSubPath()
		pc.MoveTo(xbri, ybr)
	}

	if bs.Color.Bottom != pc.StrokeStyle.Color.Solid {
		pc.StrokeStyle.SetColor(bs.Color.Bottom)
	}
	pc.StrokeStyle.Width = bs.Width.Bottom
	pc.LineTo(xbli, ybl)
	if r.Left != 0 {
		pc.DrawArc(xbli, ybli, r.Left, mat32.DegToRad(90), mat32.DegToRad(180))
	}
	if bs.Color.Bottom != bs.Color.Left || bs.Width.Bottom.Dots != bs.Width.Left.Dots {
		pc.Stroke()
		pc.NewSubPath()
		pc.MoveTo(xbl, ybli)
	}

	if bs.Color.Left != pc.StrokeStyle.Color.Solid {
		pc.StrokeStyle.SetColor(bs.Color.Left)
	}
	pc.StrokeStyle.Width = bs.Width.Left
	pc.LineTo(xtl, ytli)
	if r.Top != 0 {
		pc.DrawArc(xtli, ytli, r.Top, mat32.DegToRad(180), mat32.DegToRad(270))
	}
	pc.LineTo(xtli, ytl)
	pc.Stroke()
}

// DrawRectangle draws (but does not stroke or fill) a standard rectangle with a consistent border
func (pc *Context) DrawRectangle(x, y, w, h float32) {
	pc.NewSubPath()
	pc.MoveTo(x, y)
	pc.LineTo(x+w, y)
	pc.LineTo(x+w, y+h)
	pc.LineTo(x, y+h)
	pc.ClosePath()
}

// // DrawChangingRectangle draws a rectangle with changing border styles
// func (pc *Paint) DrawChangingRectangle(x, y, w, h float32, bs styles.Border) {
// 	// use consistent rectangle for fill, and then draw borders side by side
// 	pc.DrawConsistentRectangle(x, y, w, h)
// 	pc.Fill()

// 	pc.NewSubPath()
// 	pc.MoveTo(x, y)

// 	// set the color if it is not the same as the already set color
// 	if pc.StrokeStyle.Color.Source != styles.SolidColor || bs.Color.Top != pc.StrokeStyle.Color.Color {
// 		pc.StrokeStyle.SetColor(bs.Color.Top)
// 	}
// 	pc.StrokeStyle.Width = bs.Width.Top
// 	pc.LineTo(x+w, y)
// 	// if the color or width is changing for the next one, we have to stroke now
// 	if bs.Color.Top != bs.Color.Right || bs.Width.Top.Dots != bs.Width.Right.Dots {
// 		pc.Stroke()
// 		pc.NewSubPath()
// 		pc.MoveTo(x+w, y)
// 	}

// 	if bs.Color.Right != pc.StrokeStyle.Color.Color {
// 		pc.StrokeStyle.SetColor(bs.Color.Right)
// 	}
// 	pc.StrokeStyle.Width = bs.Width.Right
// 	pc.LineTo(x+w, y+h)
// 	if bs.Color.Right != bs.Color.Bottom || bs.Width.Right.Dots != bs.Width.Bottom.Dots {
// 		pc.Stroke()
// 		pc.NewSubPath()
// 		pc.MoveTo(x+w, y+h)
// 	}

// 	if bs.Color.Bottom != pc.StrokeStyle.Color.Color {
// 		pc.StrokeStyle.SetColor(bs.Color.Bottom)
// 	}
// 	pc.StrokeStyle.Width = bs.Width.Bottom
// 	pc.LineTo(x, y+h)
// 	if bs.Color.Bottom != bs.Color.Left || bs.Width.Bottom.Dots != bs.Width.Left.Dots {
// 		pc.Stroke()
// 		pc.NewSubPath()
// 		pc.MoveTo(x, y+h)
// 	}

// 	if bs.Color.Left != pc.StrokeStyle.Color.Color {
// 		pc.StrokeStyle.SetColor(bs.Color.Left)
// 	}
// 	pc.StrokeStyle.Width = bs.Width.Left
// 	pc.LineTo(x, y)
// }

// // DrawRectangle draws a rounded rectangle by setting the stroke style and width
// // and calling DrawConsistentRoundedRectangle if the given border width and
// // color styles for each side are the same. Otherwise, it calls DrawChangingRoundedRectangle.
// func (pc *Paint) DrawRoundedRectangle1(x, y, w, h float32, bs styles.Border) {
// 	if bs.Color.AllSame() && bs.Width.Dots().AllSame() {
// 		// set the color if it is not the same as the already set color
// 		if pc.StrokeStyle.Color.Source != styles.SolidColor || bs.Color.Top != pc.StrokeStyle.Color.Color {
// 			pc.StrokeStyle.SetColor(bs.Color.Top)
// 		}
// 		pc.StrokeStyle.Width = bs.Width.Top
// 		pc.DrawConsistentRoundedRectangle(x, y, w, h, bs.Radius.Dots())
// 		return
// 	}
// 	pc.DrawChangingRoundedRectangle(x, y, w, h, bs)
// }

// DrawRoundedRectangle draws a standard rounded rectangle
// with a consistent border and with the given x and y position,
// width and height, and border radius for each corner.
func (pc *Context) DrawRoundedRectangle(x, y, w, h float32, r styles.SideFloats) {
	// clamp border radius values
	min := mat32.Min(w/2, h/2)
	r.Top = mat32.Clamp(r.Top, 0, min)
	r.Right = mat32.Clamp(r.Right, 0, min)
	r.Bottom = mat32.Clamp(r.Bottom, 0, min)
	r.Left = mat32.Clamp(r.Left, 0, min)

	// position values; some variables are missing because they are unused
	var (
		xtl, ytl   = x, y                 // top left
		xtli, ytli = x + r.Top, y + r.Top // top left inset

		ytr        = y                            // top right
		xtri, ytri = x + w - r.Right, y + r.Right // top right inset

		xbr        = x + w                              // bottom right
		xbri, ybri = x + w - r.Bottom, y + h - r.Bottom // bottom right inset

		ybl        = y + h                      // bottom left
		xbli, ybli = x + r.Left, y + h - r.Left // bottom left inset
	)

	// SidesTODO: need to figure out how to style rounded corners correctly
	// (in CSS they are split in the middle between different border side styles)

	pc.NewSubPath()
	pc.MoveTo(xtli, ytl)

	pc.LineTo(xtri, ytr)
	if r.Right != 0 {
		pc.DrawArc(xtri, ytri, r.Right, mat32.DegToRad(270), mat32.DegToRad(360))
	}

	pc.LineTo(xbr, ybri)
	if r.Bottom != 0 {
		pc.DrawArc(xbri, ybri, r.Bottom, mat32.DegToRad(0), mat32.DegToRad(90))
	}

	pc.LineTo(xbli, ybl)
	if r.Left != 0 {
		pc.DrawArc(xbli, ybli, r.Left, mat32.DegToRad(90), mat32.DegToRad(180))
	}

	pc.LineTo(xtl, ytli)
	if r.Top != 0 {
		pc.DrawArc(xtli, ytli, r.Top, mat32.DegToRad(180), mat32.DegToRad(270))
	}
	pc.ClosePath()
}

// DrawRoundedShadowBlur draws a standard rounded rectangle
// with a consistent border and with the given x and y position,
// width and height, and border radius for each corner.
// The blurSigma and radiusFactor args add a blurred shadow with
// an effective Gaussian sigma = blurSigma, and radius = radiusFactor * sigma.
// This shadow is rendered around the given box size up to given radius.
// See EdgeBlurFactors for underlying blur factor code.
// Using radiusFactor = 1 works well for weak shadows, where the fringe beyond
// 1 sigma is essentially invisible.  To match the CSS standard, you then
// pass blurSigma = blur / 2, radiusFactor = 1.  For darker shadows,
// use blurSigma = blur / 2, radiusFactor = 2, and reserve extra space for the full shadow.
// The effective blurRadius is clamped to be <= w-2 and h-2.
func (pc *Context) DrawRoundedShadowBlur(blurSigma, radiusFactor, x, y, w, h float32, r styles.SideFloats) {
	if blurSigma <= 0 || radiusFactor <= 0 {
		pc.DrawRoundedRectangle(x, y, w, h, r)
		return
	}
	x = mat32.Floor(x)
	y = mat32.Floor(y)
	w = mat32.Ceil(w)
	h = mat32.Ceil(h)
	br := mat32.Ceil(radiusFactor * blurSigma)
	br = mat32.Clamp(br, 1, w/2-2)
	br = mat32.Clamp(br, 1, h/2-2)
	// radiusFactor = mat32.Ceil(br / blurSigma)
	radiusFactor = br / blurSigma
	blurs := EdgeBlurFactors(blurSigma, radiusFactor)

	origStroke := pc.StrokeStyle
	origFill := pc.FillStyle
	origOpacity := pc.FillStyle.Opacity

	pc.StrokeStyle.On = false
	pc.DrawRoundedRectangle(x+br, y+br, w-2*br, h-2*br, r)
	pc.FillStrokeClear()
	pc.StrokeStyle.On = true
	pc.FillStyle.On = false
	pc.StrokeStyle.Color.SetSolid(pc.FillStyle.Color.Solid)
	pc.StrokeStyle.Width.Dots = 1.5 // is the key number: 1 makes lines very transparent overall
	for i, b := range blurs {
		bo := br - float32(i)
		pc.StrokeStyle.Opacity = b * origOpacity
		pc.DrawRoundedRectangle(x+bo, y+bo, w-2*bo, h-2*bo, r)
		pc.Stroke()

	}
	pc.StrokeStyle = origStroke
	pc.FillStyle = origFill
}

// DrawEllipticalArc draws arc between angle1 and angle2 along an ellipse,
// using quadratic bezier curves -- centers of ellipse are at cx, cy with
// radii rx, ry -- see DrawEllipticalArcPath for a version compatible with SVG
// A/a path drawing, which uses previous position instead of two angles
func (pc *Context) DrawEllipticalArc(cx, cy, rx, ry, angle1, angle2 float32) {
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
		if i == 0 && !pc.HasCurrent {
			pc.MoveTo(x0, y0)
		}
		pc.QuadraticTo(ncx, ncy, x2, y2)
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
	// reverse rotate and translate back to original coordinates
	return cx*cos - cy*sin + startX, cx*sin + cy*cos + startY
}

// DrawEllipticalArcPath is draws an arc centered at cx,cy with radii rx, ry, through
// given angle, either via the smaller or larger arc, depending on largeArc --
// returns in lx, ly the last points which are then set to the current cx, cy
// for the path drawer
func (pc *Context) DrawEllipticalArcPath(cx, cy, ocx, ocy, pcx, pcy, rx, ry, angle float32, largeArc, sweep bool) (lx, ly float32) {
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
		pc.CubicTo(lx+alpha*ldx, ly+alpha*ldy, px-alpha*dx, py-alpha*dy, px, py)
		lx, ly, ldx, ldy = px, py, dx, dy
	}
	return lx, ly
}

func (pc *Context) DrawEllipse(x, y, rx, ry float32) {
	pc.NewSubPath()
	pc.DrawEllipticalArc(x, y, rx, ry, 0, 2*mat32.Pi)
	pc.ClosePath()
}

func (pc *Context) DrawArc(x, y, r, angle1, angle2 float32) {
	pc.DrawEllipticalArc(x, y, r, r, angle1, angle2)
}

func (pc *Context) DrawCircle(x, y, r float32) {
	pc.NewSubPath()
	pc.DrawEllipticalArc(x, y, r, r, 0, 2*mat32.Pi)
	pc.ClosePath()
}

func (pc *Context) DrawRegularPolygon(n int, x, y, r, rotation float32) {
	angle := 2 * mat32.Pi / float32(n)
	rotation -= mat32.Pi / 2
	if n%2 == 0 {
		rotation += angle / 2
	}
	pc.NewSubPath()
	for i := 0; i < n; i++ {
		a := rotation + angle*float32(i)
		pc.LineTo(x+r*mat32.Cos(a), y+r*mat32.Sin(a))
	}
	pc.ClosePath()
}

// DrawImage draws the specified image at the specified point.
func (pc *Context) DrawImage(fmIm image.Image, x, y float32) {
	pc.DrawImageAnchored(fmIm, x, y, 0, 0)
}

// DrawImageAnchored draws the specified image at the specified anchor point.
// The anchor point is x - w * ax, y - h * ay, where w, h is the size of the
// image. Use ax=0.5, ay=0.5 to center the image at the specified point.
func (pc *Context) DrawImageAnchored(fmIm image.Image, x, y, ax, ay float32) {
	s := fmIm.Bounds().Size()
	x -= ax * float32(s.X)
	y -= ay * float32(s.Y)
	transformer := draw.BiLinear
	m := pc.CurXForm.Translate(x, y)
	s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
	if pc.Mask == nil {
		transformer.Transform(pc.Image, s2d, fmIm, fmIm.Bounds(), draw.Over, nil)
	} else {
		transformer.Transform(pc.Image, s2d, fmIm, fmIm.Bounds(), draw.Over, &draw.Options{
			DstMask:  pc.Mask,
			DstMaskP: image.Point{},
		})
	}
}

// DrawImageScaled draws the specified image starting at given upper-left point,
// such that the size of the image is rendered as specified by w, h parameters
// (an additional scaling is applied to the transform matrix used in rendering)
func (pc *Context) DrawImageScaled(fmIm image.Image, x, y, w, h float32) {
	s := fmIm.Bounds().Size()
	isz := mat32.NewVec2FmPoint(s)
	isc := mat32.Vec2{w, h}.Div(isz)

	transformer := draw.BiLinear
	m := pc.CurXForm.Translate(x, y).Scale(isc.X, isc.Y)
	s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
	if pc.Mask == nil {
		transformer.Transform(pc.Image, s2d, fmIm, fmIm.Bounds(), draw.Over, nil)
	} else {
		transformer.Transform(pc.Image, s2d, fmIm, fmIm.Bounds(), draw.Over, &draw.Options{
			DstMask:  pc.Mask,
			DstMaskP: image.Point{},
		})
	}
}

//////////////////////////////////////////////////////////////////////////////////
// Transformation Matrix Operations

// Identity resets the current transformation matrix to the identity matrix.
// This results in no translating, scaling, rotating, or shearing.
func (pc *Context) Identity() {
	pc.XForm = mat32.Identity2D()
}

// Translate updates the current matrix with a translation.
func (pc *Context) Translate(x, y float32) {
	pc.XForm = pc.XForm.Translate(x, y)
}

// Scale updates the current matrix with a scaling factor.
// Scaling occurs about the origin.
func (pc *Context) Scale(x, y float32) {
	pc.XForm = pc.XForm.Scale(x, y)
}

// ScaleAbout updates the current matrix with a scaling factor.
// Scaling occurs about the specified point.
func (pc *Context) ScaleAbout(sx, sy, x, y float32) {
	pc.Translate(x, y)
	pc.Scale(sx, sy)
	pc.Translate(-x, -y)
}

// Rotate updates the current matrix with a clockwise rotation.
// Rotation occurs about the origin. Angle is specified in radians.
func (pc *Context) Rotate(angle float32) {
	pc.XForm = pc.XForm.Rotate(angle)
}

// RotateAbout updates the current matrix with a clockwise rotation.
// Rotation occurs about the specified point. Angle is specified in radians.
func (pc *Context) RotateAbout(angle, x, y float32) {
	pc.Translate(x, y)
	pc.Rotate(angle)
	pc.Translate(-x, -y)
}

// Shear updates the current matrix with a shearing angle.
// Shearing occurs about the origin.
func (pc *Context) Shear(x, y float32) {
	pc.XForm = pc.XForm.Shear(x, y)
}

// ShearAbout updates the current matrix with a shearing angle.
// Shearing occurs about the specified point.
func (pc *Context) ShearAbout(sx, sy, x, y float32) {
	pc.Translate(x, y)
	pc.Shear(sx, sy)
	pc.Translate(-x, -y)
}

// InvertY flips the Y axis so that Y grows from bottom to top and Y=0 is at
// the bottom of the image.
func (pc *Context) InvertY() {
	pc.Translate(0, float32(pc.Image.Bounds().Size().Y))
	pc.Scale(1, -1)
}
