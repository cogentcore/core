// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"errors"
	"image"
	"image/color"
	"io"
	"math"
	"slices"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/raster"
	"cogentcore.org/core/styles"
	"github.com/anthonynsimon/bild/clone"
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
	pc.Bounds = img.Rect

	pc.Defaults()
	pc.SetUnitContextExt(sz)

	return pc
}

// NewContextFromImage returns a new [Context] associated with an [image.RGBA]
// copy of the given [image.Image]. It does not render directly onto the given
// image; see [NewContextFromRGBA] for a version that renders directly.
func NewContextFromImage(img *image.RGBA) *Context {
	pc := &Context{&State{}, &styles.Paint{}}

	pc.Init(img.Rect.Dx(), img.Rect.Dy(), img)

	pc.Defaults()
	pc.SetUnitContextExt(img.Rect.Size())

	return pc
}

// NewContextFromRGBA returns a new [Context] associated with the given [image.RGBA].
// It renders directly onto the given image; see [NewContextFromImage] for a version
// that makes a copy.
func NewContextFromRGBA(img image.Image) *Context {
	pc := &Context{&State{}, &styles.Paint{}}

	r := clone.AsRGBA(img)
	pc.Init(r.Rect.Dx(), r.Rect.Dy(), r)

	pc.Defaults()
	pc.SetUnitContextExt(r.Rect.Size())

	return pc
}

// FillStrokeClear is a convenience final stroke and clear draw for shapes when done
func (pc *Context) FillStrokeClear() {
	if pc.SVGOut != nil {
		io.WriteString(pc.SVGOut, pc.SVGPath())
	}
	pc.FillPreserve()
	pc.StrokePreserve()
	pc.ClearPath()
}

//////////////////////////////////////////////////////////////////////////////////
// Path Manipulation

// TransformPoint multiplies the specified point by the current transform matrix,
// returning a transformed position.
func (pc *Context) TransformPoint(x, y float32) math32.Vector2 {
	return pc.CurrentTransform.MulVector2AsPoint(math32.Vec2(x, y))
}

// BoundingBox computes the bounding box for an element in pixel int
// coordinates, applying current transform
func (pc *Context) BoundingBox(minX, minY, maxX, maxY float32) image.Rectangle {
	sw := float32(0.0)
	if pc.StrokeStyle.Color != nil {
		sw = 0.5 * pc.StrokeWidth()
	}
	tmin := pc.CurrentTransform.MulVector2AsPoint(math32.Vec2(minX, minY))
	tmax := pc.CurrentTransform.MulVector2AsPoint(math32.Vec2(maxX, maxY))
	tp1 := math32.Vec2(tmin.X-sw, tmin.Y-sw).ToPointFloor()
	tp2 := math32.Vec2(tmax.X+sw, tmax.Y+sw).ToPointCeil()
	return image.Rect(tp1.X, tp1.Y, tp2.X, tp2.Y)
}

// BoundingBoxFromPoints computes the bounding box for a slice of points
func (pc *Context) BoundingBoxFromPoints(points []math32.Vector2) image.Rectangle {
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

func (pc *Context) capfunc() raster.CapFunc {
	switch pc.StrokeStyle.Cap {
	case styles.LineCapButt:
		return raster.ButtCap
	case styles.LineCapRound:
		return raster.RoundCap
	case styles.LineCapSquare:
		return raster.SquareCap
	case styles.LineCapCubic:
		return raster.CubicCap
	case styles.LineCapQuadratic:
		return raster.QuadraticCap
	}
	return nil
}

func (pc *Context) joinmode() raster.JoinMode {
	switch pc.StrokeStyle.Join {
	case styles.LineJoinMiter:
		return raster.Miter
	case styles.LineJoinMiterClip:
		return raster.MiterClip
	case styles.LineJoinRound:
		return raster.Round
	case styles.LineJoinBevel:
		return raster.Bevel
	case styles.LineJoinArcs:
		return raster.Arc
	case styles.LineJoinArcsClip:
		return raster.ArcClip
	}
	return raster.Arc
}

// StrokeWidth obtains the current stoke width subject to transform (or not
// depending on VecEffNonScalingStroke)
func (pc *Context) StrokeWidth() float32 {
	dw := pc.StrokeStyle.Width.Dots
	if dw == 0 {
		return dw
	}
	if pc.VectorEffect == styles.VectorEffectNonScalingStroke {
		return dw
	}
	scx, scy := pc.CurrentTransform.ExtractScale()
	sc := 0.5 * (math32.Abs(scx) + math32.Abs(scy))
	lw := math32.Max(sc*dw, pc.StrokeStyle.MinWidth.Dots)
	return lw
}

// StrokePreserve strokes the current path with the current color, line width,
// line cap, line join and dash settings. The path is preserved after this
// operation.
func (pc *Context) StrokePreserve() {
	if pc.Raster == nil || pc.StrokeStyle.Color == nil {
		return
	}

	dash := slices.Clone(pc.StrokeStyle.Dashes)
	if dash != nil {
		scx, scy := pc.CurrentTransform.ExtractScale()
		sc := 0.5 * (math32.Abs(scx) + math32.Abs(scy))
		for i := range dash {
			dash[i] *= sc
		}
	}

	pc.Raster.SetStroke(
		math32.ToFixed(pc.StrokeWidth()),
		math32.ToFixed(pc.StrokeStyle.MiterLimit),
		pc.capfunc(), nil, nil, pc.joinmode(), // todo: supports leading / trailing caps, and "gaps"
		dash, 0)
	pc.Scanner.SetClip(pc.Bounds)
	pc.Path.AddTo(pc.Raster)
	fbox := pc.Raster.Scanner.GetPathExtent()
	pc.LastRenderBBox = image.Rectangle{Min: image.Point{fbox.Min.X.Floor(), fbox.Min.Y.Floor()},
		Max: image.Point{fbox.Max.X.Ceil(), fbox.Max.Y.Ceil()}}
	if g, ok := pc.StrokeStyle.Color.(gradient.Gradient); ok {
		g.Update(pc.StrokeStyle.Opacity, math32.B2FromRect(pc.LastRenderBBox), pc.CurrentTransform)
		pc.Raster.SetColor(pc.StrokeStyle.Color)
	} else {
		if pc.StrokeStyle.Opacity < 1 {
			pc.Raster.SetColor(gradient.ApplyOpacityImage(pc.StrokeStyle.Color, pc.StrokeStyle.Opacity))
		} else {
			pc.Raster.SetColor(pc.StrokeStyle.Color)
		}
	}

	pc.Raster.Draw()
	pc.Raster.Clear()
}

// Stroke strokes the current path with the current color, line width,
// line cap, line join and dash settings. The path is cleared after this
// operation.
func (pc *Context) Stroke() {
	if pc.SVGOut != nil && pc.StrokeStyle.Color != nil {
		io.WriteString(pc.SVGOut, pc.SVGPath())
	}
	pc.StrokePreserve()
	pc.ClearPath()
}

// FillPreserve fills the current path with the current color. Open subpaths
// are implicitly closed. The path is preserved after this operation.
func (pc *Context) FillPreserve() {
	if pc.Raster == nil || pc.FillStyle.Color == nil {
		return
	}

	rf := &pc.Raster.Filler
	rf.SetWinding(pc.FillStyle.Rule == styles.FillRuleNonZero)
	pc.Scanner.SetClip(pc.Bounds)
	pc.Path.AddTo(rf)
	fbox := pc.Scanner.GetPathExtent()
	pc.LastRenderBBox = image.Rectangle{Min: image.Point{fbox.Min.X.Floor(), fbox.Min.Y.Floor()},
		Max: image.Point{fbox.Max.X.Ceil(), fbox.Max.Y.Ceil()}}
	if g, ok := pc.FillStyle.Color.(gradient.Gradient); ok {
		g.Update(pc.FillStyle.Opacity, math32.B2FromRect(pc.LastRenderBBox), pc.CurrentTransform)
		rf.SetColor(pc.FillStyle.Color)
	} else {
		if pc.FillStyle.Opacity < 1 {
			rf.SetColor(gradient.ApplyOpacityImage(pc.FillStyle.Color, pc.FillStyle.Opacity))
		} else {
			rf.SetColor(pc.FillStyle.Color)
		}
	}
	rf.Draw()
	rf.Clear()
}

// Fill fills the current path with the current color. Open subpaths
// are implicitly closed. The path is cleared after this operation.
func (pc *Context) Fill() {
	if pc.SVGOut != nil {
		io.WriteString(pc.SVGOut, pc.SVGPath())
	}

	pc.FillPreserve()
	pc.ClearPath()
}

// FillBox performs an optimized fill of the given
// rectangular region with the given image.
func (pc *Context) FillBox(pos, size math32.Vector2, img image.Image) {
	pc.DrawBox(pos, size, img, draw.Over)
}

// BlitBox performs an optimized overwriting fill (blit) of the given
// rectangular region with the given image.
func (pc *Context) BlitBox(pos, size math32.Vector2, img image.Image) {
	pc.DrawBox(pos, size, img, draw.Src)
}

// DrawBox performs an optimized fill/blit of the given rectangular region
// with the given image, using the given draw operation.
func (pc *Context) DrawBox(pos, size math32.Vector2, img image.Image, op draw.Op) {
	if img == nil {
		img = colors.C(color.RGBA{})
	}
	pos = pc.CurrentTransform.MulVector2AsPoint(pos)
	size = pc.CurrentTransform.MulVector2AsVector(size)
	b := pc.Bounds.Intersect(math32.RectFromPosSizeMax(pos, size))
	if g, ok := img.(gradient.Gradient); ok {
		g.Update(pc.FillStyle.Opacity, math32.B2FromRect(b), pc.CurrentTransform)
	} else {
		img = gradient.ApplyOpacityImage(img, pc.FillStyle.Opacity)
	}
	draw.Draw(pc.Image, b, img, b.Min, op)
}

// BlurBox blurs the given already drawn region with the given blur radius.
// The blur radius passed to this function is the actual Gaussian
// standard deviation (σ). This means that you need to divide a CSS-standard
// blur radius value by two before passing it this function
// (see https://stackoverflow.com/questions/65454183/how-does-blur-radius-value-in-box-shadow-property-affect-the-resulting-blur).
func (pc *Context) BlurBox(pos, size math32.Vector2, blurRadius float32) {
	rect := math32.RectFromPosSizeMax(pos, size)
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
	pc.FillPreserve()
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
	src := pc.FillStyle.Color
	draw.Draw(pc.Image, pc.Image.Bounds(), src, image.Point{}, draw.Src)
}

// SetPixel sets the color of the specified pixel using the current stroke color.
func (pc *Context) SetPixel(x, y int) {
	pc.Image.Set(x, y, pc.StrokeStyle.Color.At(x, y))
}

func (pc *Context) DrawLine(x1, y1, x2, y2 float32) {
	pc.MoveTo(x1, y1)
	pc.LineTo(x2, y2)
}

func (pc *Context) DrawPolyline(points []math32.Vector2) {
	sz := len(points)
	if sz < 2 {
		return
	}
	pc.MoveTo(points[0].X, points[0].Y)
	for i := 1; i < sz; i++ {
		pc.LineTo(points[i].X, points[i].Y)
	}
}

func (pc *Context) DrawPolylinePxToDots(points []math32.Vector2) {
	pu := &pc.UnitContext
	sz := len(points)
	if sz < 2 {
		return
	}
	pc.MoveTo(pu.PxToDots(points[0].X), pu.PxToDots(points[0].Y))
	for i := 1; i < sz; i++ {
		pc.LineTo(pu.PxToDots(points[i].X), pu.PxToDots(points[i].Y))
	}
}

func (pc *Context) DrawPolygon(points []math32.Vector2) {
	pc.DrawPolyline(points)
	pc.ClosePath()
}

func (pc *Context) DrawPolygonPxToDots(points []math32.Vector2) {
	pc.DrawPolylinePxToDots(points)
	pc.ClosePath()
}

// DrawBorder is a higher-level function that draws, strokes, and fills
// an potentially rounded border box with the given position, size, and border styles.
func (pc *Context) DrawBorder(x, y, w, h float32, bs styles.Border) {
	r := bs.Radius.Dots()
	if styles.SidesAreSame(bs.Style) && styles.SidesAreSame(bs.Color) && styles.SidesAreSame(bs.Width.Dots().Sides) {
		// set the color if it is not nil and the stroke style
		// is not set to the correct color
		if bs.Color.Top != nil && bs.Color.Top != pc.StrokeStyle.Color {
			pc.StrokeStyle.Color = bs.Color.Top
		}
		pc.StrokeStyle.Width = bs.Width.Top
		pc.StrokeStyle.ApplyBorderStyle(bs.Style.Top)
		if styles.SidesAreZero(r.Sides) {
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
	min := math32.Min(w/2, h/2)
	r.Top = math32.Clamp(r.Top, 0, min)
	r.Right = math32.Clamp(r.Right, 0, min)
	r.Bottom = math32.Clamp(r.Bottom, 0, min)
	r.Left = math32.Clamp(r.Left, 0, min)

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
	if bs.Color.Top != pc.StrokeStyle.Color {
		pc.StrokeStyle.Color = bs.Color.Top
	}
	pc.StrokeStyle.Width = bs.Width.Top
	pc.LineTo(xtri, ytr)
	if r.Right != 0 {
		pc.DrawArc(xtri, ytri, r.Right, math32.DegToRad(270), math32.DegToRad(360))
	}
	// if the color or width is changing for the next one, we have to stroke now
	if bs.Color.Top != bs.Color.Right || bs.Width.Top.Dots != bs.Width.Right.Dots {
		pc.Stroke()
		pc.NewSubPath()
		pc.MoveTo(xtr, ytri)
	}

	if bs.Color.Right != pc.StrokeStyle.Color {
		pc.StrokeStyle.Color = bs.Color.Right
	}
	pc.StrokeStyle.Width = bs.Width.Right
	pc.LineTo(xbr, ybri)
	if r.Bottom != 0 {
		pc.DrawArc(xbri, ybri, r.Bottom, math32.DegToRad(0), math32.DegToRad(90))
	}
	if bs.Color.Right != bs.Color.Bottom || bs.Width.Right.Dots != bs.Width.Bottom.Dots {
		pc.Stroke()
		pc.NewSubPath()
		pc.MoveTo(xbri, ybr)
	}

	if bs.Color.Bottom != pc.StrokeStyle.Color {
		pc.StrokeStyle.Color = bs.Color.Bottom
	}
	pc.StrokeStyle.Width = bs.Width.Bottom
	pc.LineTo(xbli, ybl)
	if r.Left != 0 {
		pc.DrawArc(xbli, ybli, r.Left, math32.DegToRad(90), math32.DegToRad(180))
	}
	if bs.Color.Bottom != bs.Color.Left || bs.Width.Bottom.Dots != bs.Width.Left.Dots {
		pc.Stroke()
		pc.NewSubPath()
		pc.MoveTo(xbl, ybli)
	}

	if bs.Color.Left != pc.StrokeStyle.Color {
		pc.StrokeStyle.Color = bs.Color.Left
	}
	pc.StrokeStyle.Width = bs.Width.Left
	pc.LineTo(xtl, ytli)
	if r.Top != 0 {
		pc.DrawArc(xtli, ytli, r.Top, math32.DegToRad(180), math32.DegToRad(270))
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

// DrawRoundedRectangle draws a standard rounded rectangle
// with a consistent border and with the given x and y position,
// width and height, and border radius for each corner.
func (pc *Context) DrawRoundedRectangle(x, y, w, h float32, r styles.SideFloats) {
	// clamp border radius values
	min := math32.Min(w/2, h/2)
	r.Top = math32.Clamp(r.Top, 0, min)
	r.Right = math32.Clamp(r.Right, 0, min)
	r.Bottom = math32.Clamp(r.Bottom, 0, min)
	r.Left = math32.Clamp(r.Left, 0, min)

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
		pc.DrawArc(xtri, ytri, r.Right, math32.DegToRad(270), math32.DegToRad(360))
	}

	pc.LineTo(xbr, ybri)
	if r.Bottom != 0 {
		pc.DrawArc(xbri, ybri, r.Bottom, math32.DegToRad(0), math32.DegToRad(90))
	}

	pc.LineTo(xbli, ybl)
	if r.Left != 0 {
		pc.DrawArc(xbli, ybli, r.Left, math32.DegToRad(90), math32.DegToRad(180))
	}

	pc.LineTo(xtl, ytli)
	if r.Top != 0 {
		pc.DrawArc(xtli, ytli, r.Top, math32.DegToRad(180), math32.DegToRad(270))
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
	x = math32.Floor(x)
	y = math32.Floor(y)
	w = math32.Ceil(w)
	h = math32.Ceil(h)
	br := math32.Ceil(radiusFactor * blurSigma)
	br = math32.Clamp(br, 1, w/2-2)
	br = math32.Clamp(br, 1, h/2-2)
	// radiusFactor = math32.Ceil(br / blurSigma)
	radiusFactor = br / blurSigma
	blurs := EdgeBlurFactors(blurSigma, radiusFactor)

	origStroke := pc.StrokeStyle
	origFill := pc.FillStyle
	origOpacity := pc.FillStyle.Opacity

	pc.StrokeStyle.Color = nil
	pc.DrawRoundedRectangle(x+br, y+br, w-2*br, h-2*br, r)
	pc.FillStrokeClear()
	pc.StrokeStyle.Color = pc.FillStyle.Color
	pc.FillStyle.Color = nil
	pc.StrokeStyle.Width.Dots = 1.5 // 1.5 is the key number: 1 makes lines very transparent overall
	for i, b := range blurs {
		bo := br - float32(i)
		pc.StrokeStyle.Opacity = b * origOpacity
		pc.DrawRoundedRectangle(x+bo, y+bo, w-2*bo, h-2*bo, r)
		pc.Stroke()

	}
	pc.StrokeStyle = origStroke
	pc.FillStyle = origFill
}

// DrawEllipticalArc draws arc between angle1 and angle2 (radians)
// along an ellipse. Because the y axis points down, angles are clockwise,
// and the rendering draws segments progressing from angle1 to angle2
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
		x0 := cx + rx*math32.Cos(a1)
		y0 := cy + ry*math32.Sin(a1)
		x1 := cx + rx*math32.Cos((a1+a2)/2)
		y1 := cy + ry*math32.Sin((a1+a2)/2)
		x2 := cx + rx*math32.Cos(a2)
		y2 := cy + ry*math32.Sin(a2)
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
	bCosEta := b * math32.Cos(eta)
	aSinEta := a * math32.Sin(eta)
	px = -aSinEta*cosTheta - bCosEta*sinTheta
	py = -aSinEta*sinTheta + bCosEta*cosTheta
	return
}

// ellipsePointAt gives points for parameterized ellipse; a, b, radii, eta
// parameter, center cx, cy
func ellipsePointAt(a, b, sinTheta, cosTheta, eta, cx, cy float32) (px, py float32) {
	aCosEta := a * math32.Cos(eta)
	bSinEta := b * math32.Sin(eta)
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
	cos, sin := math32.Cos(rotX), math32.Sin(rotX)

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
		nry := math32.Sqrt(midlenSq)
		if *rx == *ry {
			*rx = nry // prevents roundoff
		} else {
			*rx = *rx * nry / *ry
		}
		*ry = nry
	} else {
		hr = math32.Sqrt(*ry**ry-midlenSq) / math32.Sqrt(midlenSq)
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
	startAngle := math32.Atan2(pcy-cy, pcx-cx) - rotX
	endAngle := math32.Atan2(ocy-cy, ocx-cx) - rotX
	deltaTheta := endAngle - startAngle
	arcBig := math32.Abs(deltaTheta) > math.Pi

	// Approximate ellipse using cubic bezier splines
	etaStart := math32.Atan2(math32.Sin(startAngle)/ry, math32.Cos(startAngle)/rx)
	etaEnd := math32.Atan2(math32.Sin(endAngle)/ry, math32.Cos(endAngle)/rx)
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
	segs := int(math32.Abs(deltaEta)/MaxDx) + 1
	dEta := deltaEta / float32(segs) // span of each segment
	// Approximate the ellipse using a set of cubic bezier curves by the method of
	// L. Maisonobe, "Drawing an elliptical arc using polylines, quadratic
	// or cubic Bezier curves", 2003
	// https://www.spaceroots.org/documents/ellipse/elliptical-arc.pdf
	tde := math32.Tan(dEta / 2)
	alpha := math32.Sin(dEta) * (math32.Sqrt(4+3*tde*tde) - 1) / 3 // Math is fun!
	lx, ly = pcx, pcy
	sinTheta, cosTheta := math32.Sin(rotX), math32.Cos(rotX)
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

// DrawEllipse draws an ellipse at the given position with the given radii.
func (pc *Context) DrawEllipse(x, y, rx, ry float32) {
	pc.NewSubPath()
	pc.DrawEllipticalArc(x, y, rx, ry, 0, 2*math32.Pi)
	pc.ClosePath()
}

// DrawArc draws an arc at the given position with the given radius
// and angles in radians.  Because the y axis points down, angles are clockwise,
// and the rendering draws segments progressing from angle1 to angle2
func (pc *Context) DrawArc(x, y, r, angle1, angle2 float32) {
	pc.DrawEllipticalArc(x, y, r, r, angle1, angle2)
}

// DrawCircle draws a circle at the given position with the given radius.
func (pc *Context) DrawCircle(x, y, r float32) {
	pc.NewSubPath()
	pc.DrawEllipticalArc(x, y, r, r, 0, 2*math32.Pi)
	pc.ClosePath()
}

// DrawRegularPolygon draws a regular polygon with the given number of sides
// at the given position with the given rotation.
func (pc *Context) DrawRegularPolygon(n int, x, y, r, rotation float32) {
	angle := 2 * math32.Pi / float32(n)
	rotation -= math32.Pi / 2
	if n%2 == 0 {
		rotation += angle / 2
	}
	pc.NewSubPath()
	for i := 0; i < n; i++ {
		a := rotation + angle*float32(i)
		pc.LineTo(x+r*math32.Cos(a), y+r*math32.Sin(a))
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
	m := pc.CurrentTransform.Translate(x, y)
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
	isz := math32.Vector2FromPoint(s)
	isc := math32.Vec2(w, h).Div(isz)

	transformer := draw.BiLinear
	m := pc.CurrentTransform.Translate(x, y).Scale(isc.X, isc.Y)
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
	pc.Transform = math32.Identity2()
}

// Translate updates the current matrix with a translation.
func (pc *Context) Translate(x, y float32) {
	pc.Transform = pc.Transform.Translate(x, y)
}

// Scale updates the current matrix with a scaling factor.
// Scaling occurs about the origin.
func (pc *Context) Scale(x, y float32) {
	pc.Transform = pc.Transform.Scale(x, y)
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
	pc.Transform = pc.Transform.Rotate(angle)
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
	pc.Transform = pc.Transform.Shear(x, y)
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
