// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"errors"
	"github.com/golang/freetype/raster"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/math/fixed"
	"image"
	"math"
)

/*
This is mostly just restructured version of: https://github.com/fogleman/gg

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

// The Paint object provides the full context (parameters) and functions for
// painting onto an image -- image is always passed as an argument so it can be
// applied to anything
type Paint struct {
	Stroke     PaintStroke
	Fill       PaintFill
	Font       PaintFont
	TextLayout PaintTextLayout
	// below are rendering state
	StrokePath raster.Path
	FillPath   raster.Path
	Start      Point2D
	Current    Point2D
	HasCurrent bool
	XForm      XFormMatrix2D
	Bounds     image.Rectangle `desc:"image bounding rectangle -- should be 0,0,size X,Y"`
	Mask       *image.Alpha
}

func (pc *Paint) Defaults(bound image.Rectangle) {
	pc.Stroke.Defaults()
	pc.Fill.Defaults()
	pc.Font.Defaults()
	pc.TextLayout.Defaults()
	pc.XForm = Identity2D()
	pc.Bounds = bound
}

// update the Paint Stroke and Fill from the properties of a given node -- because Paint stack captures all the relevant inheritance, this does NOT look for inherited properties
func (pc *Paint) SetFromNode(g *Node2DBase) {
	pc.Stroke.SetFromNode(g)
	pc.Fill.SetFromNode(g)
	pc.Font.SetFromNode(g)
	pc.TextLayout.SetFromNode(g)
}

//////////////////////////////////////////////////////////////////////////////////
// Path Manipulation

// TransformPoint multiplies the specified point by the current transform matrix,
// returning a transformed position.
func (pc *Paint) TransformPoint(x, y float64) Point2D {
	tx, ty := pc.XForm.TransformPoint(x, y)
	return Point2D{tx, ty}
}

// get the bounding box for an element in pixel int coordinates
func (pc *Paint) BoundingBox(minX, minY, maxX, maxY float64) image.Rectangle {
	tx1, ty1 := pc.XForm.TransformPoint(minX, minY)
	tx2, ty2 := pc.XForm.TransformPoint(maxX, maxY)
	return image.Rect(int(tx1), int(ty1), int(tx2), int(ty2))
}

// get the bounding box for a slice of points
func (pc *Paint) BoundingBoxFromPoints(points []Point2D) image.Rectangle {
	sz := len(points)
	if sz == 0 {
		return image.Rectangle{}
	}
	tx, ty := pc.XForm.TransformPointToInt(points[0].X, points[0].Y)
	bb := image.Rect(tx, ty, tx, ty)
	for i := 1; i < sz; i++ {
		tx, ty := pc.XForm.TransformPointToInt(points[i].X, points[i].Y)
		if tx < bb.Min.X {
			bb.Min.X = tx
		} else if tx > bb.Max.X {
			bb.Max.X = tx
		}
		if ty < bb.Min.Y {
			bb.Min.Y = ty
		} else if ty > bb.Max.Y {
			bb.Max.Y = ty
		}
	}
	return bb
}

// MoveTo starts a new subpath within the current path starting at the
// specified point.
func (pc *Paint) MoveTo(x, y float64) {
	if pc.HasCurrent {
		pc.FillPath.Add1(pc.Start.Fixed())
	}
	p := pc.TransformPoint(x, y)
	pc.StrokePath.Start(p.Fixed())
	pc.FillPath.Start(p.Fixed())
	pc.Start = p
	pc.Current = p
	pc.HasCurrent = true
}

// LineTo adds a line segment to the current path starting at the current
// point. If there is no current point, it is equivalent to MoveTo(x, y)
func (pc *Paint) LineTo(x, y float64) {
	if !pc.HasCurrent {
		pc.MoveTo(x, y)
	} else {
		p := pc.TransformPoint(x, y)
		pc.StrokePath.Add1(p.Fixed())
		pc.FillPath.Add1(p.Fixed())
		pc.Current = p
	}
}

// QuadraticTo adds a quadratic bezier curve to the current path starting at
// the current point. If there is no current point, it first performs
// MoveTo(x1, y1)
func (pc *Paint) QuadraticTo(x1, y1, x2, y2 float64) {
	if !pc.HasCurrent {
		pc.MoveTo(x1, y1)
	}
	p1 := pc.TransformPoint(x1, y1)
	p2 := pc.TransformPoint(x2, y2)
	pc.StrokePath.Add2(p1.Fixed(), p2.Fixed())
	pc.FillPath.Add2(p1.Fixed(), p2.Fixed())
	pc.Current = p2
}

// CubicTo adds a cubic bezier curve to the current path starting at the
// current point. If there is no current point, it first performs
// MoveTo(x1, y1). Because freetype/raster does not support cubic beziers,
// this is emulated with many small line segments.
func (pc *Paint) CubicTo(x1, y1, x2, y2, x3, y3 float64) {
	if !pc.HasCurrent {
		pc.MoveTo(x1, y1)
	}
	x0, y0 := pc.Current.X, pc.Current.Y
	x1, y1 = pc.XForm.TransformPoint(x1, y1)
	x2, y2 = pc.XForm.TransformPoint(x2, y2)
	x3, y3 = pc.XForm.TransformPoint(x3, y3)
	points := CubicBezier(x0, y0, x1, y1, x2, y2, x3, y3)
	previous := pc.Current.Fixed()
	for _, p := range points[1:] {
		f := p.Fixed()
		if f == previous {
			// TODO: this fixes some rendering issues but not all
			continue
		}
		previous = f
		pc.StrokePath.Add1(f)
		pc.FillPath.Add1(f)
		pc.Current = p
	}
}

// ClosePath adds a line segment from the current point to the beginning
// of the current subpath. If there is no current point, this is a no-op.
func (pc *Paint) ClosePath() {
	if pc.HasCurrent {
		pc.StrokePath.Add1(pc.Start.Fixed())
		pc.FillPath.Add1(pc.Start.Fixed())
		pc.Current = pc.Start
	}
}

// ClearPath clears the current path. There is no current point after this
// operation.
func (pc *Paint) ClearPath() {
	pc.StrokePath.Clear()
	pc.FillPath.Clear()
	pc.HasCurrent = false
}

// NewSubPath starts a new subpath within the current path. There is no current
// point after this operation.
func (pc *Paint) NewSubPath() {
	if pc.HasCurrent {
		pc.FillPath.Add1(pc.Start.Fixed())
	}
	pc.HasCurrent = false
}

// Path Drawing

func (pc *Paint) capper() raster.Capper {
	switch pc.Stroke.Cap {
	case LineCapButt:
		return raster.ButtCapper
	case LineCapRound:
		return raster.RoundCapper
	case LineCapSquare:
		return raster.SquareCapper
	}
	return nil
}

func (pc *Paint) joiner() raster.Joiner {
	switch pc.Stroke.Join {
	case LineJoinRound:
		return raster.RoundJoiner
	default: // all others for now.. -- todo: support more joiners!!??
		return raster.BevelJoiner
	}
	return nil
}

func (pc *Paint) stroke(painter raster.Painter) {
	path := pc.StrokePath
	if len(pc.Stroke.Dashes) > 0 {
		path = dashed(path, pc.Stroke.Dashes)
	} else {
		// TODO: this is a temporary workaround to remove tiny segments
		// that result in rendering issues
		path = rasterPath(flattenPath(path))
	}
	sz := pc.Bounds.Size()
	r := raster.NewRasterizer(sz.X, sz.Y)
	r.UseNonZeroWinding = true
	r.AddStroke(path, fix(pc.Stroke.Width), pc.capper(), pc.joiner())
	r.Rasterize(painter)
}

func (pc *Paint) fill(painter raster.Painter) {
	path := pc.FillPath
	if pc.HasCurrent {
		path = make(raster.Path, len(pc.FillPath))
		copy(path, pc.FillPath)
		path.Add1(pc.Start.Fixed())
	}
	sz := pc.Bounds.Size()
	r := raster.NewRasterizer(sz.X, sz.Y)
	r.UseNonZeroWinding = (pc.Fill.Rule == FillRuleNonZero)
	r.AddPath(path)
	r.Rasterize(painter)
}

// StrokePreserve strokes the current path with the current color, line width,
// line cap, line join and dash settings. The path is preserved after this
// operation.
func (pc *Paint) StrokePreserve(im *image.RGBA) {
	painter := newPaintServerPainter(im, pc.Mask, pc.Stroke.Server)
	pc.stroke(painter)
}

// Stroke strokes the current path with the current color, line width,
// line cap, line join and dash settings. The path is cleared after this
// operation.
func (pc *Paint) StrokeImage(im *image.RGBA) {
	pc.StrokePreserve(im)
	pc.ClearPath()
}

// FillPreserve fills the current path with the current color. Open subpaths
// are implicity closed. The path is preserved after this operation.
func (pc *Paint) FillPreserve(im *image.RGBA) {
	painter := newPaintServerPainter(im, pc.Mask, pc.Fill.Server)
	pc.fill(painter)
}

// Fill fills the current path with the current color. Open subpaths
// are implicity closed. The path is cleared after this operation.
func (pc *Paint) FillImage(im *image.RGBA) {
	pc.FillPreserve(im)
	pc.ClearPath()
}

// ClipPreserve updates the clipping region by intersecting the current
// clipping region with the current path as it would be filled by pc.Fill().
// The path is preserved after this operation.
func (pc *Paint) ClipPreserve() {
	clip := image.NewAlpha(pc.Bounds)
	painter := raster.NewAlphaOverPainter(clip)
	pc.fill(painter)
	if pc.Mask == nil {
		pc.Mask = clip
	} else {
		mask := image.NewAlpha(pc.Bounds)
		draw.DrawMask(mask, mask.Bounds(), clip, image.ZP, pc.Mask, image.ZP, draw.Over)
		pc.Mask = mask
	}
}

// SetMask allows you to directly set the *image.Alpha to be used as a clipping
// mask. It must be the same size as the context, else an error is returned
// and the mask is unchanged.
func (pc *Paint) SetMask(mask *image.Alpha) error {
	if mask.Bounds() != pc.Bounds {
		return errors.New("mask size must match context size")
	}
	pc.Mask = mask
	return nil
}

// AsMask returns an *image.Alpha representing the alpha channel of this
// context. This can be useful for advanced clipping operations where you first
// render the mask geometry and then use it as a mask.
func (pc *Paint) AsMask(im *image.RGBA) *image.Alpha {
	mask := image.NewAlpha(pc.Bounds)
	draw.Draw(mask, im.Bounds(), im, image.ZP, draw.Src)
	return mask
}

// Clip updates the clipping region by intersecting the current
// clipping region with the current path as it would be filled by pc.Fill().
// The path is cleared after this operation.
func (pc *Paint) Clip() {
	pc.ClipPreserve()
	pc.ClearPath()
}

// ResetClip clears the clipping region.
func (pc *Paint) ResetClip() {
	pc.Mask = nil
}

//////////////////////////////////////////////////////////////////////////////////
// Convenient Drawing Functions

// Clear fills the entire image with the current fill color.
func (pc *Paint) Clear(im *image.RGBA) {
	src := image.NewUniform(pc.Fill.Color)
	draw.Draw(im, im.Bounds(), src, image.ZP, draw.Src)
}

// SetPixel sets the color of the specified pixel using the current stroke color.
func (pc *Paint) SetPixel(im *image.RGBA, x, y int) {
	im.Set(x, y, pc.Stroke.Color)
}

func (pc *Paint) DrawLine(x1, y1, x2, y2 float64) {
	pc.MoveTo(x1, y1)
	pc.LineTo(x2, y2)
}

func (pc *Paint) DrawPolyline(points []Point2D) {
	sz := len(points)
	if sz < 2 {
		return
	}
	pc.MoveTo(points[0].X, points[0].Y)
	for i := 1; i < sz; i++ {
		pc.LineTo(points[i].X, points[i].Y)
	}
}

func (pc *Paint) DrawPolygon(points []Point2D) {
	pc.DrawPolyline(points)
	pc.ClosePath()
}

func (pc *Paint) DrawRectangle(x, y, w, h float64) {
	pc.NewSubPath()
	pc.MoveTo(x, y)
	pc.LineTo(x+w, y)
	pc.LineTo(x+w, y+h)
	pc.LineTo(x, y+h)
	pc.ClosePath()
}

func (pc *Paint) DrawRoundedRectangle(x, y, w, h, r float64) {
	x0, x1, x2, x3 := x, x+r, x+w-r, x+w
	y0, y1, y2, y3 := y, y+r, y+h-r, y+h
	pc.NewSubPath()
	pc.MoveTo(x1, y0)
	pc.LineTo(x2, y0)
	pc.DrawArc(x2, y1, r, Radians(270), Radians(360))
	pc.LineTo(x3, y2)
	pc.DrawArc(x2, y2, r, Radians(0), Radians(90))
	pc.LineTo(x1, y3)
	pc.DrawArc(x1, y2, r, Radians(90), Radians(180))
	pc.LineTo(x0, y1)
	pc.DrawArc(x1, y1, r, Radians(180), Radians(270))
	pc.ClosePath()
}

func (pc *Paint) DrawEllipticalArc(x, y, rx, ry, angle1, angle2 float64) {
	const n = 16
	for i := 0; i < n; i++ {
		p1 := float64(i+0) / n
		p2 := float64(i+1) / n
		a1 := angle1 + (angle2-angle1)*p1
		a2 := angle1 + (angle2-angle1)*p2
		x0 := x + rx*math.Cos(a1)
		y0 := y + ry*math.Sin(a1)
		x1 := x + rx*math.Cos(a1+(a2-a1)/2)
		y1 := y + ry*math.Sin(a1+(a2-a1)/2)
		x2 := x + rx*math.Cos(a2)
		y2 := y + ry*math.Sin(a2)
		cx := 2*x1 - x0/2 - x2/2
		cy := 2*y1 - y0/2 - y2/2
		if i == 0 && !pc.HasCurrent {
			pc.MoveTo(x0, y0)
		}
		pc.QuadraticTo(cx, cy, x2, y2)
	}
}

func (pc *Paint) DrawEllipse(x, y, rx, ry float64) {
	pc.NewSubPath()
	pc.DrawEllipticalArc(x, y, rx, ry, 0, 2*math.Pi)
	pc.ClosePath()
}

func (pc *Paint) DrawArc(x, y, r, angle1, angle2 float64) {
	pc.DrawEllipticalArc(x, y, r, r, angle1, angle2)
}

func (pc *Paint) DrawCircle(x, y, r float64) {
	pc.NewSubPath()
	pc.DrawEllipticalArc(x, y, r, r, 0, 2*math.Pi)
	pc.ClosePath()
}

func (pc *Paint) DrawRegularPolygon(n int, x, y, r, rotation float64) {
	angle := 2 * math.Pi / float64(n)
	rotation -= math.Pi / 2
	if n%2 == 0 {
		rotation += angle / 2
	}
	pc.NewSubPath()
	for i := 0; i < n; i++ {
		a := rotation + angle*float64(i)
		pc.LineTo(x+r*math.Cos(a), y+r*math.Sin(a))
	}
	pc.ClosePath()
}

// DrawImage draws the specified image at the specified point.
func (pc *Paint) DrawImage(toIm *image.RGBA, fmIm image.Image, x, y int) {
	pc.DrawImageAnchored(toIm, fmIm, x, y, 0, 0)
}

// DrawImageAnchored draws the specified image at the specified anchor point.
// The anchor point is x - w * ax, y - h * ay, where w, h is the size of the
// image. Use ax=0.5, ay=0.5 to center the image at the specified point.
func (pc *Paint) DrawImageAnchored(toIm *image.RGBA, fmIm image.Image, x, y int, ax, ay float64) {
	s := pc.Bounds.Size()
	x -= int(ax * float64(s.X))
	y -= int(ay * float64(s.Y))
	transformer := draw.BiLinear
	fx, fy := float64(x), float64(y)
	m := pc.XForm.Translate(fx, fy)
	s2d := f64.Aff3{m.XX, m.XY, m.X0, m.YX, m.YY, m.Y0}
	if pc.Mask == nil {
		transformer.Transform(toIm, s2d, fmIm, fmIm.Bounds(), draw.Over, nil)
	} else {
		transformer.Transform(toIm, s2d, fmIm, fmIm.Bounds(), draw.Over, &draw.Options{
			DstMask:  pc.Mask,
			DstMaskP: image.ZP,
		})
	}
}

//////////////////////////////////////////////////////////////////////////////////
// Text Functions

func (pc *Paint) SetFontFace(fontFace font.Face) {
	pc.Font.Face = fontFace
	pc.Font.Height = float64(fontFace.Metrics().Height) / 64.0
}

func (pc *Paint) LoadFontFace(path string, points float64) error {
	face, err := LoadFontFace(path, points)
	if err == nil {
		pc.Font.Face = face
		pc.Font.Height = points * 72 / 96
	}
	return err
}

func (pc *Paint) FontHeight() float64 {
	return pc.Font.Height
}

func (pc *Paint) drawString(im *image.RGBA, s string, x, y float64) {
	d := &font.Drawer{
		Dst:  im,
		Src:  image.NewUniform(pc.Stroke.Color),
		Face: pc.Font.Face,
		Dot:  fixp(x, y),
	}
	// based on Drawer.DrawString() in golang.org/x/image/font/font.go
	prevC := rune(-1)
	for _, c := range s {
		if prevC >= 0 {
			d.Dot.X += d.Face.Kern(prevC, c)
		}
		dr, mask, maskp, advance, ok := d.Face.Glyph(d.Dot, c)
		if !ok {
			// TODO: is falling back on the U+FFFD glyph the responsibility of
			// the Drawer or the Face?
			// TODO: set prevC = '\ufffd'?
			continue
		}
		sr := dr.Sub(dr.Min)
		transformer := draw.BiLinear
		fx, fy := float64(dr.Min.X), float64(dr.Min.Y)
		m := pc.XForm.Translate(fx, fy)
		s2d := f64.Aff3{m.XX, m.XY, m.X0, m.YX, m.YY, m.Y0}
		transformer.Transform(d.Dst, s2d, d.Src, sr, draw.Over, &draw.Options{
			SrcMask:  mask,
			SrcMaskP: maskp,
		})
		d.Dot.X += advance
		prevC = c
	}
}

// DrawString according to current settings -- width is only needed for wrap case
func (pc *Paint) DrawString(im *image.RGBA, s string, x, y, width float64) {
	var ax, ay float64
	switch pc.TextLayout.Align {
	case TextAlignLeft:
	case TextAlignCenter:
		ax = 0.5 // todo: determine if font is horiz or vert..
	case TextAlignRight:
		ax = 1.0
	}
	if pc.TextLayout.Wrap {
		pc.DrawStringAnchored(im, s, x, y, ax, ay)
	} else {
		pc.DrawStringWrapped(im, s, x, y, ax, ay, width, pc.TextLayout.Spacing.Y, pc.TextLayout.Align)
	}
}

// DrawStringAnchored draws the specified text at the specified anchor point.
// The anchor point is x - w * ax, y - h * ay, where w, h is the size of the
// text. Use ax=0.5, ay=0.5 to center the text at the specified point.
func (pc *Paint) DrawStringAnchored(im *image.RGBA, s string, x, y, ax, ay float64) {
	w, h := pc.MeasureString(s)
	x -= ax * w
	y += ay * h
	if pc.Mask == nil {
		pc.drawString(im, s, x, y)
	} else {
		im := image.NewRGBA(pc.Bounds)
		pc.drawString(im, s, x, y)
		draw.DrawMask(im, im.Bounds(), im, image.ZP, pc.Mask, image.ZP, draw.Over)
	}
}

// DrawStringWrapped word-wraps the specified string to the given max width
// and then draws it at the specified anchor point using the given line
// spacing and text alignment.
func (pc *Paint) DrawStringWrapped(im *image.RGBA, s string, x, y, ax, ay, width, lineSpacing float64, align TextAlign) {
	lines := pc.WordWrap(s, width)
	h := float64(len(lines)) * pc.Font.Height * lineSpacing
	h -= (lineSpacing - 1) * pc.Font.Height
	x -= ax * width
	y -= ay * h
	switch align {
	case TextAlignLeft:
		ax = 0
	case TextAlignCenter:
		ax = 0.5
		x += width / 2
	case TextAlignRight:
		ax = 1
		x += width
	}
	ay = 1
	for _, line := range lines {
		pc.DrawStringAnchored(im, line, x, y, ax, ay)
		y += pc.Font.Height * lineSpacing
	}
}

// MeasureString returns the rendered width and height of the specified text
// given the current font face.
func (pc *Paint) MeasureString(s string) (w, h float64) {
	d := &font.Drawer{
		Face: pc.Font.Face,
	}
	a := d.MeasureString(s)
	return float64(a >> 6), pc.Font.Height
}

// WordWrap wraps the specified string to the given max width and current
// font face.
func (pc *Paint) WordWrap(s string, w float64) []string {
	return wordWrap(pc, s, w)
}

//////////////////////////////////////////////////////////////////////////////////
// Transformation Matrix Operations

// Identity resets the current transformation matrix to the identity matrix.
// This results in no translating, scaling, rotating, or shearing.
func (pc *Paint) Identity() {
	pc.XForm = Identity2D()
}

// Translate updates the current matrix with a translation.
func (pc *Paint) Translate(x, y float64) {
	pc.XForm = pc.XForm.Translate(x, y)
}

// Scale updates the current matrix with a scaling factor.
// Scaling occurs about the origin.
func (pc *Paint) Scale(x, y float64) {
	pc.XForm = pc.XForm.Scale(x, y)
}

// ScaleAbout updates the current matrix with a scaling factor.
// Scaling occurs about the specified point.
func (pc *Paint) ScaleAbout(sx, sy, x, y float64) {
	pc.Translate(x, y)
	pc.Scale(sx, sy)
	pc.Translate(-x, -y)
}

// Rotate updates the current matrix with a clockwise rotation.
// Rotation occurs about the origin. Angle is specified in radians.
func (pc *Paint) Rotate(angle float64) {
	pc.XForm = pc.XForm.Rotate(angle)
}

// RotateAbout updates the current matrix with a clockwise rotation.
// Rotation occurs about the specified point. Angle is specified in radians.
func (pc *Paint) RotateAbout(angle, x, y float64) {
	pc.Translate(x, y)
	pc.Rotate(angle)
	pc.Translate(-x, -y)
}

// Shear updates the current matrix with a shearing angle.
// Shearing occurs about the origin.
func (pc *Paint) Shear(x, y float64) {
	pc.XForm = pc.XForm.Shear(x, y)
}

// ShearAbout updates the current matrix with a shearing angle.
// Shearing occurs about the specified point.
func (pc *Paint) ShearAbout(sx, sy, x, y float64) {
	pc.Translate(x, y)
	pc.Shear(sx, sy)
	pc.Translate(-x, -y)
}

// InvertY flips the Y axis so that Y grows from bottom to top and Y=0 is at
// the bottom of the image.
func (pc *Paint) InvertY() {
	pc.Translate(0, float64(pc.Bounds.Size().Y))
	pc.Scale(1, -1)
}

////////////////////////////////////////////////////////////////////////////////////
// Internal -- might want to export these later depending

func flattenPath(p raster.Path) [][]Point2D {
	var result [][]Point2D
	var path []Point2D
	var cx, cy float64
	for i := 0; i < len(p); {
		switch p[i] {
		case 0:
			if len(path) > 0 {
				result = append(result, path)
				path = nil
			}
			x := unfix(p[i+1])
			y := unfix(p[i+2])
			path = append(path, Point2D{x, y})
			cx, cy = x, y
			i += 4
		case 1:
			x := unfix(p[i+1])
			y := unfix(p[i+2])
			path = append(path, Point2D{x, y})
			cx, cy = x, y
			i += 4
		case 2:
			x1 := unfix(p[i+1])
			y1 := unfix(p[i+2])
			x2 := unfix(p[i+3])
			y2 := unfix(p[i+4])
			points := QuadraticBezier(cx, cy, x1, y1, x2, y2)
			path = append(path, points...)
			cx, cy = x2, y2
			i += 6
		case 3:
			x1 := unfix(p[i+1])
			y1 := unfix(p[i+2])
			x2 := unfix(p[i+3])
			y2 := unfix(p[i+4])
			x3 := unfix(p[i+5])
			y3 := unfix(p[i+6])
			points := CubicBezier(cx, cy, x1, y1, x2, y2, x3, y3)
			path = append(path, points...)
			cx, cy = x3, y3
			i += 8
		default:
			panic("bad path")
		}
	}
	if len(path) > 0 {
		result = append(result, path)
	}
	return result
}

func dashPath(paths [][]Point2D, dashes []float64) [][]Point2D {
	var result [][]Point2D
	if len(dashes) == 0 {
		return paths
	}
	if len(dashes) == 1 {
		dashes = append(dashes, dashes[0])
	}
	for _, path := range paths {
		if len(path) < 2 {
			continue
		}
		previous := path[0]
		pathIndex := 1
		dashIndex := 0
		segmentLength := 0.0
		var segment []Point2D
		segment = append(segment, previous)
		for pathIndex < len(path) {
			dashLength := dashes[dashIndex]
			point := path[pathIndex]
			d := previous.Distance(point)
			maxd := dashLength - segmentLength
			if d > maxd {
				t := maxd / d
				p := previous.Interpolate(point, t)
				segment = append(segment, p)
				if dashIndex%2 == 0 && len(segment) > 1 {
					result = append(result, segment)
				}
				segment = nil
				segment = append(segment, p)
				segmentLength = 0
				previous = p
				dashIndex = (dashIndex + 1) % len(dashes)
			} else {
				segment = append(segment, point)
				previous = point
				segmentLength += d
				pathIndex++
			}
		}
		if dashIndex%2 == 0 && len(segment) > 1 {
			result = append(result, segment)
		}
	}
	return result
}

func rasterPath(paths [][]Point2D) raster.Path {
	var result raster.Path
	for _, path := range paths {
		var previous fixed.Point26_6
		for i, point := range path {
			f := point.Fixed()
			if i == 0 {
				result.Start(f)
			} else {
				dx := f.X - previous.X
				dy := f.Y - previous.Y
				if dx < 0 {
					dx = -dx
				}
				if dy < 0 {
					dy = -dy
				}
				if dx+dy > 8 {
					// TODO: this is a hack for cases where two points are
					// too close - causes rendering issues with joins / caps
					result.Add1(f)
				}
			}
			previous = f
		}
	}
	return result
}

func dashed(path raster.Path, dashes []float64) raster.Path {
	return rasterPath(dashPath(flattenPath(path), dashes))
}
