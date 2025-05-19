// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"image"
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/pimage"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/text/shaped"
	"golang.org/x/image/draw"
)

/*
The original version borrowed heavily from: https://github.com/fogleman/gg
Copyright (C) 2016 Michael Fogleman

and https://github.com/srwiley/rasterx:
Copyright 2018 by the rasterx Authors. All rights reserved.
Created 2018 by S.R.Wiley

The new version is more strongly based on https://github.com/tdewolff/canvas
Copyright (c) 2015 Taco de Wolff, under an MIT License.
*/

// Painter provides the rendering state, styling parameters, and methods for
// painting. It accumulates all painting actions in a [render.Render]
// list, which should be obtained by a call to the [Painter.RenderDone] method
// when  done painting (resets list to start fresh).
//
// Pass this [render.Render] list to one or more [render.Renderers] to actually
// generate the resulting output. Renderers are independent of the Painter
// and the [render.Render] state is entirely self-contained, so rendering
// can be done in a separate goroutine etc.
//
// You must import _ "cogentcore.org/core/paint/renderers" to get the default
// renderers if using this outside of core which already does this for you.
// This sets the New*Renderer functions to point to default implementations.
type Painter struct {
	*State
	*styles.Paint
}

// NewPainter returns a new [Painter] with default styles and given size.
func NewPainter(size math32.Vector2) *Painter {
	pc := &Painter{&State{}, styles.NewPaint()}
	pc.State.Init(pc.Paint, size)
	pc.SetUnitContextExt(size.ToPointCeil())
	return pc
}

func (pc *Painter) Transform() math32.Matrix2 {
	return pc.Context().Transform.Mul(pc.Paint.Transform)
}

//////// Path basics

// MoveTo starts a new subpath within the current path starting at the
// specified point.
func (pc *Painter) MoveTo(x, y float32) {
	pc.State.Path.MoveTo(x, y)
}

// LineTo adds a line segment to the current path starting at the current
// point. If there is no current point, it is equivalent to MoveTo(x, y)
func (pc *Painter) LineTo(x, y float32) {
	pc.State.Path.LineTo(x, y)
}

// QuadTo adds a quadratic Bézier path with control point (cpx,cpy) and end point (x,y).
func (pc *Painter) QuadTo(cpx, cpy, x, y float32) {
	pc.State.Path.QuadTo(cpx, cpy, x, y)
}

// CubeTo adds a cubic Bézier path with control points
// (cpx1,cpy1) and (cpx2,cpy2) and end point (x,y).
func (pc *Painter) CubeTo(cp1x, cp1y, cp2x, cp2y, x, y float32) {
	pc.State.Path.CubeTo(cp1x, cp1y, cp2x, cp2y, x, y)
}

// ArcTo adds an arc with radii rx and ry, with rot the counter clockwise
// rotation with respect to the coordinate system in radians, large and sweep booleans
// (see https://developer.mozilla.org/en-US/docs/Web/SVG/Tutorial/Paths#Arcs),
// and (x,y) the end position of the pen. The start position of the pen was
// given by a previous command's end point.
func (pc *Painter) ArcTo(rx, ry, rot float32, large, sweep bool, x, y float32) {
	pc.State.Path.ArcTo(rx, ry, rot, large, sweep, x, y)
}

// Close closes a (sub)path with a LineTo to the start of the path
// (the most recent MoveTo command). It also signals the path closes
// as opposed to being just a LineTo command, which can be significant
// for stroking purposes for example.
func (pc *Painter) Close() {
	pc.State.Path.Close()
}

// Draw puts the current path on the render stack, capturing the style
// settings present at this point, which will be used to render the path,
// and creates a new current path.
func (pc *Painter) Draw() {
	pc.Paint.ToDots()
	pt := render.NewPath(pc.State.Path.Clone(), pc.Paint, pc.Context())
	pc.Render.Add(pt)
	pc.State.Path.Reset()
}

//////// basic shape functions

// note: the path shapes versions can be used when you want to add to an existing path
// using ppath.Join. These functions produce distinct standalone shapes, starting with
// a MoveTo generally.

// Line adds a separate line (MoveTo, LineTo).
func (pc *Painter) Line(x1, y1, x2, y2 float32) {
	pc.State.Path.Line(x1, y1, x2, y2)
}

// Polyline adds multiple connected lines, with no final Close.
func (pc *Painter) Polyline(points ...math32.Vector2) {
	pc.State.Path.Polyline(points...)
}

// Polyline adds multiple connected lines, with no final Close,
// with coordinates in Px units.
func (pc *Painter) PolylinePx(points ...math32.Vector2) {
	pu := &pc.UnitContext
	sz := len(points)
	if sz < 2 {
		return
	}
	p := &pc.State.Path
	p.MoveTo(pu.PxToDots(points[0].X), pu.PxToDots(points[0].Y))
	for i := 1; i < sz; i++ {
		p.LineTo(pu.PxToDots(points[i].X), pu.PxToDots(points[i].Y))
	}
}

// Polygon adds multiple connected lines with a final Close.
func (pc *Painter) Polygon(points ...math32.Vector2) {
	pc.Polyline(points...)
	pc.Close()
}

// Polygon adds multiple connected lines with a final Close,
// with coordinates in Px units.
func (pc *Painter) PolygonPx(points ...math32.Vector2) {
	pc.PolylinePx(points...)
	pc.Close()
}

// Rectangle adds a rectangle of width w and height h at position x,y.
func (pc *Painter) Rectangle(x, y, w, h float32) {
	pc.State.Path.Rectangle(x, y, w, h)
}

// RoundedRectangle adds a rectangle of width w and height h
// with rounded corners of radius r at postion x,y.
// A negative radius will cast the corners inwards (i.e. concave).
func (pc *Painter) RoundedRectangle(x, y, w, h, r float32) {
	pc.State.Path.RoundedRectangle(x, y, w, h, r)
}

// RoundedRectangleSides adds a standard rounded rectangle
// with a consistent border and with the given x and y position,
// width and height, and border radius for each corner.
func (pc *Painter) RoundedRectangleSides(x, y, w, h float32, r sides.Floats) {
	pc.State.Path.RoundedRectangleSides(x, y, w, h, r)
}

// BeveledRectangle adds a rectangle of width w and height h
// with beveled corners at distance r from the corner.
func (pc *Painter) BeveledRectangle(x, y, w, h, r float32) {
	pc.State.Path.BeveledRectangle(x, y, w, h, r)
}

// Circle adds a circle at given center coordinates of radius r.
func (pc *Painter) Circle(cx, cy, r float32) {
	pc.Ellipse(cx, cy, r, r)
}

// Ellipse adds an ellipse at given center coordinates of radii rx and ry.
func (pc *Painter) Ellipse(cx, cy, rx, ry float32) {
	pc.State.Path.Ellipse(cx, cy, rx, ry)
}

// CircularArc adds a circular arc centered at given coordinates with radius r
// and theta0 and theta1 as the angles in degrees of the ellipse
// (before rot is applied) between which the arc will run.
// If theta0 < theta1, the arc will run in a CCW direction.
// If the difference between theta0 and theta1 is bigger than 360 degrees,
// one full circle will be drawn and the remaining part of diff % 360,
// e.g. a difference of 810 degrees will draw one full circle and an arc
// over 90 degrees.
func (pc *Painter) CircularArc(x, y, r, theta0, theta1 float32) {
	pc.State.Path.EllipticalArc(x, y, r, r, 0, theta0, theta1)
}

// EllipticalArc adds an elliptical arc centered at given coordinates with
// radii rx and ry, with rot the counter clockwise rotation in degrees,
// and theta0 and theta1 the angles in degrees of the ellipse
// (before rot is applied) between which the arc will run.
// If theta0 < theta1, the arc will run in a CCW direction.
// If the difference between theta0 and theta1 is bigger than 360 degrees,
// one full circle will be drawn and the remaining part of diff % 360,
// e.g. a difference of 810 degrees will draw one full circle and an arc
// over 90 degrees.
func (pc *Painter) EllipticalArc(x, y, rx, ry, rot, theta0, theta1 float32) {
	pc.State.Path.EllipticalArc(x, y, rx, ry, rot, theta0, theta1)
}

// Triangle adds a triangle of radius r pointing upwards.
func (pc *Painter) Triangle(x, y, r float32) {
	pc.State.Path.RegularPolygon(3, r, true).Translate(x, y) // todo: just make these take a position.
}

// RegularPolygon adds a regular polygon with radius r.
// It uses n vertices/edges, so when n approaches infinity
// this will return a path that approximates a circle.
// n must be 3 or more. The up boolean defines whether
// the first point will point upwards or downwards.
func (pc *Painter) RegularPolygon(x, y float32, n int, r float32, up bool) {
	pc.State.Path.RegularPolygon(n, r, up).Translate(x, y)
}

// RegularStarPolygon adds a regular star polygon with radius r.
// It uses n vertices of density d. This will result in a
// self-intersection star in counter clockwise direction.
// If n/2 < d the star will be clockwise and if n and d are not coprime
// a regular polygon will be obtained, possible with multiple windings.
// n must be 3 or more and d 2 or more. The up boolean defines whether
// the first point will point upwards or downwards.
func (pc *Painter) RegularStarPolygon(x, y float32, n, d int, r float32, up bool) {
	pc.State.Path.RegularStarPolygon(n, d, r, up).Translate(x, y)
}

// StarPolygon returns a star polygon of n points with alternating
// radius R and r. The up boolean defines whether the first point
// will be point upwards or downwards.
func (pc *Painter) StarPolygon(x, y float32, n int, R, r float32, up bool) {
	pc.State.Path.StarPolygon(n, R, r, up).Translate(x, y)
}

// Grid returns a stroked grid of width w and height h,
// with grid line thickness r, and the number of cells horizontally
// and vertically as nx and ny respectively.
func (pc *Painter) Grid(x, y, w, h float32, nx, ny int, r float32) {
	pc.State.Path.Grid(w, y, nx, ny, r).Translate(x, y)
}

// ClampBorderRadius returns the given border radius clamped to fit based
// on the given width and height of the object.
func ClampBorderRadius(r sides.Floats, w, h float32) sides.Floats {
	min := math32.Min(w/2, h/2)
	r.Top = math32.Clamp(r.Top, 0, min)
	r.Right = math32.Clamp(r.Right, 0, min)
	r.Bottom = math32.Clamp(r.Bottom, 0, min)
	r.Left = math32.Clamp(r.Left, 0, min)
	return r
}

// Border is a higher-level function that draws, strokes, and fills
// an potentially rounded border box with the given position, size, and border styles.
func (pc *Painter) Border(x, y, w, h float32, bs styles.Border) {
	origStroke := pc.Stroke
	origFill := pc.Fill
	defer func() {
		pc.Stroke = origStroke
		pc.Fill = origFill
	}()
	r := bs.Radius.Dots()
	if sides.AreSame(bs.Style) && sides.AreSame(bs.Color) && sides.AreSame(bs.Width.Dots().Sides) {
		// set the color if it is not nil and the stroke style
		// is not set to the correct color
		if bs.Color.Top != nil && bs.Color.Top != pc.Stroke.Color {
			pc.Stroke.Color = bs.Color.Top
		}
		pc.Stroke.Width = bs.Width.Top
		pc.Stroke.ApplyBorderStyle(bs.Style.Top)
		if sides.AreZero(r.Sides) {
			pc.Rectangle(x, y, w, h)
		} else {
			pc.RoundedRectangleSides(x, y, w, h, r)
		}
		pc.Draw()
		return
	}

	// use consistent rounded rectangle for fill, and then draw borders side by side
	pc.RoundedRectangleSides(x, y, w, h, r)
	pc.Draw()

	r = ClampBorderRadius(r, w, h)

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

	pc.MoveTo(xtli, ytl)

	// set the color if it is not the same as the already set color
	if bs.Color.Top != pc.Stroke.Color {
		pc.Stroke.Color = bs.Color.Top
	}
	pc.Stroke.Width = bs.Width.Top
	pc.LineTo(xtri, ytr)
	if r.Right != 0 {
		pc.CircularArc(xtri, ytri, r.Right, math32.DegToRad(270), math32.DegToRad(360))
	}
	// if the color or width is changing for the next one, we have to stroke now
	if bs.Color.Top != bs.Color.Right || bs.Width.Top.Dots != bs.Width.Right.Dots {
		pc.Draw()
		pc.MoveTo(xtr, ytri)
	}

	if bs.Color.Right != pc.Stroke.Color {
		pc.Stroke.Color = bs.Color.Right
	}
	pc.Stroke.Width = bs.Width.Right
	pc.LineTo(xbr, ybri)
	if r.Bottom != 0 {
		pc.CircularArc(xbri, ybri, r.Bottom, math32.DegToRad(0), math32.DegToRad(90))
	}
	if bs.Color.Right != bs.Color.Bottom || bs.Width.Right.Dots != bs.Width.Bottom.Dots {
		pc.Draw()
		pc.MoveTo(xbri, ybr)
	}

	if bs.Color.Bottom != pc.Stroke.Color {
		pc.Stroke.Color = bs.Color.Bottom
	}
	pc.Stroke.Width = bs.Width.Bottom
	pc.LineTo(xbli, ybl)
	if r.Left != 0 {
		pc.CircularArc(xbli, ybli, r.Left, math32.DegToRad(90), math32.DegToRad(180))
	}
	if bs.Color.Bottom != bs.Color.Left || bs.Width.Bottom.Dots != bs.Width.Left.Dots {
		pc.Draw()
		pc.MoveTo(xbl, ybli)
	}

	if bs.Color.Left != pc.Stroke.Color {
		pc.Stroke.Color = bs.Color.Left
	}
	pc.Stroke.Width = bs.Width.Left
	pc.LineTo(xtl, ytli)
	if r.Top != 0 {
		pc.CircularArc(xtli, ytli, r.Top, math32.DegToRad(180), math32.DegToRad(270))
	}
	pc.LineTo(xtli, ytl)
	pc.Draw()
}

// RoundedShadowBlur draws a standard rounded rectangle
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
func (pc *Painter) RoundedShadowBlur(blurSigma, radiusFactor, x, y, w, h float32, r sides.Floats) {
	if blurSigma <= 0 || radiusFactor <= 0 {
		pc.RoundedRectangleSides(x, y, w, h, r)
		pc.Draw()
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

	origStroke := pc.Stroke
	origFill := pc.Fill
	origOpacity := pc.Fill.Opacity

	pc.Stroke.Color = nil
	pc.RoundedRectangleSides(x+br, y+br, w-2*br, h-2*br, r)
	pc.Draw()
	pc.Stroke.Color = pc.Fill.Color
	pc.Fill.Color = nil
	pc.Stroke.Width.Dots = 1.5 // 1.5 is the key number: 1 makes lines very transparent overall
	for i, b := range blurs {
		bo := br - float32(i)
		pc.Stroke.Opacity = b * origOpacity
		pc.RoundedRectangleSides(x+bo, y+bo, w-2*bo, h-2*bo, r)
		pc.Draw()

	}
	pc.Stroke = origStroke
	pc.Fill = origFill
}

//////// Image drawing

// FillBox performs an optimized fill of the given
// rectangular region with the given image. It is equivalent
// to [Painter.DrawBox] with [draw.Over].
func (pc *Painter) FillBox(pos, size math32.Vector2, img image.Image) {
	pc.DrawBox(pos, size, img, draw.Over)
}

// BlitBox performs an optimized overwriting fill (blit) of the given
// rectangular region with the given image. It is equivalent
// to [Painter.DrawBox] with [draw.Src].
func (pc *Painter) BlitBox(pos, size math32.Vector2, img image.Image) {
	pc.DrawBox(pos, size, img, draw.Src)
}

// DrawBox performs an optimized fill/blit of the given rectangular region
// with the given image, using the given draw operation.
// If the image is nil, a new transparent color is used.
func (pc *Painter) DrawBox(pos, size math32.Vector2, img image.Image, op draw.Op) {
	if img == nil {
		img = colors.Uniform(color.RGBA{})
	}
	pos = pc.Transform().MulVector2AsPoint(pos)
	size = pc.Transform().MulVector2AsVector(size)
	br := math32.RectFromPosSizeMax(pos, size)
	cb := pc.Context().Bounds.Rect.ToRect()
	b := cb.Intersect(br)
	if b.Size() == (image.Point{}) {
		return
	}
	if g, ok := img.(gradient.Gradient); ok {
		g.Update(pc.Fill.Opacity, math32.B2FromRect(b), pc.Transform())
	} else {
		img = gradient.ApplyOpacity(img, pc.Fill.Opacity)
	}
	pc.Render.Add(pimage.NewDraw(b, img, b.Min, op))
}

// BlurBox blurs the given already drawn region with the given blur radius.
// The blur radius passed to this function is the actual Gaussian
// standard deviation (σ). This means that you need to divide a CSS-standard
// blur radius value by two before passing it this function
// (see https://stackoverflow.com/questions/65454183/how-does-blur-radius-value-in-box-shadow-property-affect-the-resulting-blur).
func (pc *Painter) BlurBox(pos, size math32.Vector2, blurRadius float32) {
	rect := math32.RectFromPosSizeMax(pos, size)
	pc.Render.Add(pimage.NewBlur(rect, blurRadius))
}

// SetMask allows you to directly set the *image.Alpha to be used as a clipping
// mask. It must be the same size as the context, else an error is returned
// and the mask is unchanged.
func (pc *Painter) SetMask(mask *image.Alpha) error {
	// if mask.Bounds() != pc.Image.Bounds() {
	// 	return errors.New("mask size must match context size")
	// }
	pc.Mask = mask
	return nil
}

// AsMask returns an *image.Alpha representing the alpha channel of this
// context. This can be useful for advanced clipping operations where you first
// render the mask geometry and then use it as a mask.
// func (pc *Painter) AsMask() *image.Alpha {
// 	b := pc.Image.Bounds()
// 	mask := image.NewAlpha(b)
// 	draw.Draw(mask, b, pc.Image, image.Point{}, draw.Src)
// 	return mask
// }

// Clear fills the entire image with the current fill color.
func (pc *Painter) Clear() {
	src := pc.Fill.Color
	pc.Render.Add(pimage.NewClear(src, image.Point{}, draw.Src))
}

// SetPixel sets the color of the specified pixel using the current stroke color.
func (pc *Painter) SetPixel(x, y int) {
	pc.Render.Add(pimage.NewSetPixel(image.Point{x, y}, pc.Stroke.Color))
}

// DrawImage draws the given image at the specified starting point,
// using the bounds of the source image in rectangle rect, using
// the given draw operration: Over = overlay (alpha blend with destination)
// Src = copy source directly, overwriting destination pixels.
func (pc *Painter) DrawImage(src image.Image, rect image.Rectangle, srcStart image.Point, op draw.Op) {
	pc.Render.Add(pimage.NewDraw(rect, src, srcStart, op))
}

// DrawImageAnchored draws the specified image at the specified anchor point.
// The anchor point is x - w * ax, y - h * ay, where w, h is the size of the
// image. Use ax=0.5, ay=0.5 to center the image at the specified point.
func (pc *Painter) DrawImageAnchored(src image.Image, x, y, ax, ay float32) {
	s := src.Bounds().Size()
	x -= ax * float32(s.X)
	y -= ay * float32(s.Y)
	m := pc.Transform().Translate(x, y)
	if pc.Mask == nil {
		pc.Render.Add(pimage.NewTransform(m, src.Bounds(), src, draw.Over))
	} else {
		pc.Render.Add(pimage.NewTransformMask(m, src.Bounds(), src, draw.Over, pc.Mask, image.Point{}))
	}
}

// DrawImageScaled draws the specified image starting at given upper-left point,
// such that the size of the image is rendered as specified by w, h parameters
// (an additional scaling is applied to the transform matrix used in rendering)
func (pc *Painter) DrawImageScaled(src image.Image, x, y, w, h float32) {
	s := src.Bounds().Size()
	isz := math32.FromPoint(s)
	isc := math32.Vec2(w, h).Div(isz)

	m := pc.Transform().Translate(x, y).Scale(isc.X, isc.Y)
	if pc.Mask == nil {
		pc.Render.Add(pimage.NewTransform(m, src.Bounds(), src, draw.Over))
	} else {
		pc.Render.Add(pimage.NewTransformMask(m, src.Bounds(), src, draw.Over, pc.Mask, image.Point{}))
	}
}

// BoundingBox computes the bounding box for an element in pixel int
// coordinates, applying current transform
func (pc *Painter) BoundingBox(minX, minY, maxX, maxY float32) image.Rectangle {
	sw := float32(0.0)
	// if pc.Stroke.Color != nil {// todo
	// 	sw = 0.5 * pc.StrokeWidth()
	// }
	tmin := pc.Transform().MulVector2AsPoint(math32.Vec2(minX, minY))
	tmax := pc.Transform().MulVector2AsPoint(math32.Vec2(maxX, maxY))
	tp1 := math32.Vec2(tmin.X-sw, tmin.Y-sw).ToPointFloor()
	tp2 := math32.Vec2(tmax.X+sw, tmax.Y+sw).ToPointCeil()
	return image.Rect(tp1.X, tp1.Y, tp2.X, tp2.Y)
}

// BoundingBoxFromPoints computes the bounding box for a slice of points
func (pc *Painter) BoundingBoxFromPoints(points []math32.Vector2) image.Rectangle {
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

/////// DrawText

// DrawText adds given [shaped] text lines to the rendering list,
// at given position. Note that all rendering is subject to the
// current active transform, including the position:
// e.g., use math32.Rotate2DAround to just rotate the text at a given
// absolute position offset.
func (pc *Painter) DrawText(tx *shaped.Lines, pos math32.Vector2) {
	pc.Render.Add(render.NewText(tx, pc.Paint, pc.Context(), pos))
}
