// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"errors"
	// "fmt"
	"github.com/golang/freetype/raster"
	"github.com/rcoreilly/goki/gi/units"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/math/fixed"
	"image"
	"math"
	"reflect"
)

/*
This borrows very heavily from: https://github.com/fogleman/gg

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
	Off         bool          `desc:"node and everything below it are off, non-rendering"`
	StyleSet    bool          `desc:"have the styles already been set?"`
	UnContext   units.Context `desc:"units context -- parameters necessary for anchoring relative units"`
	StrokeStyle StrokeStyle
	FillStyle   FillStyle
	FontStyle   FontStyle
	TextStyle   TextStyle
	XForm       XFormMatrix2D `xml:"-" json:"-" desc:"our additions to transform -- pushed to render state"`
}

func (pc *Paint) Defaults() {
	pc.Off = false
	pc.StyleSet = false
	pc.StrokeStyle.Defaults()
	pc.FillStyle.Defaults()
	pc.FontStyle.Defaults()
	pc.TextStyle.Defaults()
	pc.XForm = Identity2D()
}

func NewPaint() Paint {
	p := Paint{}
	p.Defaults()
	return p
}

// default style can be used when property specifies "default"
var PaintDefault = NewPaint()

// set paint values based on given property map (name: value pairs), inheriting elements as appropriate from parent, and also having a default style for the "initial" setting
func (pc *Paint) SetStyle(parent, defs *Paint, props map[string]interface{}) {
	// nil interface is special and != interface{} of a nil ptr!
	pfi := interface{}(nil)
	dfi := interface{}(nil)
	if parent != nil {
		pfi = interface{}(parent)
	}
	if defs != nil {
		dfi = interface{}(defs)
	}
	inherit := !pc.StyleSet // we only inherit if not already set
	WalkStyleStruct(pc, pfi, dfi, "", inherit, props, StyleField)
	pc.StrokeStyle.SetStylePost()
	pc.FillStyle.SetStylePost()
	pc.FontStyle.SetStylePost()
	pc.TextStyle.SetStylePost()
}

// set the unit context based on size of viewport and parent element (from bbox)
// and then cache everything out in terms of raw pixel dots for rendering -- call at start of
// render
func (pc *Paint) SetUnitContext(vp *Viewport2D, el Vec2D) {
	pc.UnContext.Defaults() // todo: need to get screen information and true dpi
	if vp != nil && vp.Render.Image != nil {
		sz := vp.Render.Image.Bounds().Size()
		pc.UnContext.SetSizes(float64(sz.X), float64(sz.Y), el.X, el.Y)
	}
	pc.FontStyle.SetUnitContext(&pc.UnContext)
	pc.ToDots()
}

// call ToDots on all units.Value fields in the style (recursively) -- need to have set the
// UnContext first -- only after layout at render time is that possible
func (s *Paint) ToDots() {
	valtyp := reflect.TypeOf(units.Value{})

	WalkStyleStruct(s, nil, nil, "", false, nil,
		func(sf reflect.StructField, vf, pf, df reflect.Value,
			hasPar bool, tag string, inherit bool, props map[string]interface{}) {
			if vf.Kind() == reflect.Struct && vf.Type() == valtyp {
				uv := vf.Addr().Interface().(*units.Value)
				uv.ToDots(&s.UnContext)
			}
		})
}

//////////////////////////////////////////////////////////////////////////////////
// State query

// does the current Paint have an active stroke to render?
func (pc *Paint) HasStroke() bool {
	return pc.StrokeStyle.On
}

// does the current Paint have an active fill to render?
func (pc *Paint) HasFill() bool {
	return pc.FillStyle.On
}

// does the current Paint not have either a stroke or fill?  in which case, often we just skip it
func (pc *Paint) HasNoStrokeOrFill() bool {
	return (!pc.StrokeStyle.On && !pc.FillStyle.On)
}

// convenience for final draw for shapes when done
func (pc *Paint) FillStrokeClear(rs *RenderState) {
	if pc.HasFill() {
		pc.FillPreserve(rs)
	}
	if pc.HasStroke() {
		pc.StrokePreserve(rs)
	}
	pc.ClearPath(rs)
}

//////////////////////////////////////////////////////////////////////////////////
// RenderState

// The RenderState holds all the current rendering state information used while painting -- a viewport just has one of these
type RenderState struct {
	XForm       XFormMatrix2D     `desc:"current transform"`
	StrokePath  raster.Path       `desc:"current stroke path"`
	FillPath    raster.Path       `desc:"current fill path"`
	Start       Vec2D             `desc:"starting point, for close path"`
	Current     Vec2D             `desc:"current point"`
	HasCurrent  bool              `desc:"is current point current?"`
	Image       *image.RGBA       `desc:"pointer to image to render into"`
	Mask        *image.Alpha      `desc:"current mask"`
	Bounds      image.Rectangle   `desc:"boundaries to restrict drawing to -- much faster than clip mask for basic square region exclusion -- used for restricting drawing"`
	XFormStack  []XFormMatrix2D   `desc:"stack of transforms"`
	BoundsStack []image.Rectangle `desc:"stack of bounds -- every render starts with a push onto this stack, and finishes with a pop"`
	ClipStack   []*image.Alpha    `desc:"stack of clips, if needed"`
}

func (rs *RenderState) Defaults() {
	rs.XForm = Identity2D()
}

// push current xform onto stack and apply new xform on top of it
func (rs *RenderState) PushXForm(xf XFormMatrix2D) {
	if rs.XFormStack == nil {
		rs.XFormStack = make([]XFormMatrix2D, 0, 100)
	}
	rs.XFormStack = append(rs.XFormStack, rs.XForm)
	rs.XForm = rs.XForm.Multiply(xf)
}

// pop xform off the stack and set to current xform
func (rs *RenderState) PopXForm() {
	if rs.XFormStack == nil || len(rs.XFormStack) == 0 {
		rs.XForm = Identity2D()
		return
	}
	sz := len(rs.XFormStack)
	rs.XForm = rs.XFormStack[sz-1]
	rs.XFormStack = rs.XFormStack[:sz-1]
}

// push current bounds onto stack and set new bounds
func (rs *RenderState) PushBounds(b image.Rectangle) {
	if rs.BoundsStack == nil {
		rs.BoundsStack = make([]image.Rectangle, 0, 100)
	}
	if rs.Bounds.Empty() { // note: method name should be IsEmpty!
		rs.Bounds = rs.Image.Bounds()
	}
	rs.BoundsStack = append(rs.BoundsStack, rs.Bounds)
	rs.Bounds = b
}

// pop bounds off the stack and set to current bounds
func (rs *RenderState) PopBounds() {
	if rs.BoundsStack == nil || len(rs.BoundsStack) == 0 {
		rs.Bounds = rs.Image.Bounds()
		return
	}
	sz := len(rs.BoundsStack)
	rs.Bounds = rs.BoundsStack[sz-1]
	rs.BoundsStack = rs.BoundsStack[:sz-1]
}

// push current Mask onto the clip stack
func (rs *RenderState) PushClip() {
	if rs.Mask == nil {
		return
	}
	if rs.ClipStack == nil {
		rs.ClipStack = make([]*image.Alpha, 0, 10)
	}
	rs.ClipStack = append(rs.ClipStack, rs.Mask)
}

// pop Mask off the clip stack and set to current mask
func (rs *RenderState) PopClip() {
	if rs.ClipStack == nil || len(rs.ClipStack) == 0 {
		rs.Mask = nil // implied
		return
	}
	sz := len(rs.ClipStack)
	rs.Mask = rs.ClipStack[sz-1]
	rs.ClipStack[sz-1] = nil
	rs.ClipStack = rs.ClipStack[:sz-1]
}

//////////////////////////////////////////////////////////////////////////////////
// Path Manipulation

// TransformPoint multiplies the specified point by the current transform matrix,
// returning a transformed position.
func (pc *Paint) TransformPoint(rs *RenderState, x, y float64) Vec2D {
	tx, ty := rs.XForm.TransformPoint(x, y)
	return Vec2D{tx, ty}
}

// get the bounding box for an element in pixel int coordinates
func (pc *Paint) BoundingBox(rs *RenderState, minX, minY, maxX, maxY float64) image.Rectangle {
	sw := 0.0
	if pc.HasStroke() {
		sw = 0.5 * pc.StrokeStyle.Width.Dots
	}
	tx1, ty1 := rs.XForm.TransformPoint(minX-sw, minY-sw)
	tx2, ty2 := rs.XForm.TransformPoint(maxX+sw, maxY+sw)
	return image.Rect(int(tx1), int(ty1), int(tx2), int(ty2))
}

// get the bounding box for a slice of points
func (pc *Paint) BoundingBoxFromPoints(rs *RenderState, points []Vec2D) image.Rectangle {
	sz := len(points)
	if sz == 0 {
		return image.Rectangle{}
	}
	tx, ty := rs.XForm.TransformPointToInt(points[0].X, points[0].Y)
	bb := image.Rect(tx, ty, tx, ty)
	for i := 1; i < sz; i++ {
		tx, ty := rs.XForm.TransformPointToInt(points[i].X, points[i].Y)
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
func (pc *Paint) MoveTo(rs *RenderState, x, y float64) {
	if rs.HasCurrent {
		rs.FillPath.Add1(rs.Start.Fixed())
	}
	p := pc.TransformPoint(rs, x, y)
	rs.StrokePath.Start(p.Fixed())
	rs.FillPath.Start(p.Fixed())
	rs.Start = p
	rs.Current = p
	rs.HasCurrent = true
}

// LineTo adds a line segment to the current path starting at the current
// point. If there is no current point, it is equivalent to MoveTo(x, y)
func (pc *Paint) LineTo(rs *RenderState, x, y float64) {
	if !rs.HasCurrent {
		pc.MoveTo(rs, x, y)
	} else {
		p := pc.TransformPoint(rs, x, y)
		rs.StrokePath.Add1(p.Fixed())
		rs.FillPath.Add1(p.Fixed())
		rs.Current = p
	}
}

// QuadraticTo adds a quadratic bezier curve to the current path starting at
// the current point. If there is no current point, it first performs
// MoveTo(x1, y1)
func (pc *Paint) QuadraticTo(rs *RenderState, x1, y1, x2, y2 float64) {
	if !rs.HasCurrent {
		pc.MoveTo(rs, x1, y1)
	}
	p1 := pc.TransformPoint(rs, x1, y1)
	p2 := pc.TransformPoint(rs, x2, y2)
	rs.StrokePath.Add2(p1.Fixed(), p2.Fixed())
	rs.FillPath.Add2(p1.Fixed(), p2.Fixed())
	rs.Current = p2
}

// CubicTo adds a cubic bezier curve to the current path starting at the
// current point. If there is no current point, it first performs
// MoveTo(x1, y1). Because freetype/raster does not support cubic beziers,
// this is emulated with many small line segments.
func (pc *Paint) CubicTo(rs *RenderState, x1, y1, x2, y2, x3, y3 float64) {
	if !rs.HasCurrent {
		pc.MoveTo(rs, x1, y1)
	}
	x0, y0 := rs.Current.X, rs.Current.Y
	x1, y1 = rs.XForm.TransformPoint(x1, y1)
	x2, y2 = rs.XForm.TransformPoint(x2, y2)
	x3, y3 = rs.XForm.TransformPoint(x3, y3)
	points := CubicBezier(x0, y0, x1, y1, x2, y2, x3, y3)
	previous := rs.Current.Fixed()
	for _, p := range points[1:] {
		f := p.Fixed()
		if f == previous {
			// TODO: this fixes some rendering issues but not all
			continue
		}
		previous = f
		rs.StrokePath.Add1(f)
		rs.FillPath.Add1(f)
		rs.Current = p
	}
}

// ClosePath adds a line segment from the current point to the beginning
// of the current subpath. If there is no current point, this is a no-op.
func (pc *Paint) ClosePath(rs *RenderState) {
	if rs.HasCurrent {
		rs.StrokePath.Add1(rs.Start.Fixed())
		rs.FillPath.Add1(rs.Start.Fixed())
		rs.Current = rs.Start
	}
}

// ClearPath clears the current path. There is no current point after this
// operation.
func (pc *Paint) ClearPath(rs *RenderState) {
	rs.StrokePath.Clear()
	rs.FillPath.Clear()
	rs.HasCurrent = false
}

// NewSubPath starts a new subpath within the current path. There is no current
// point after this operation.
func (pc *Paint) NewSubPath(rs *RenderState) {
	if rs.HasCurrent {
		rs.FillPath.Add1(rs.Start.Fixed())
	}
	rs.HasCurrent = false
}

// Path Drawing

func (pc *Paint) capper() raster.Capper {
	switch pc.StrokeStyle.Cap {
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
	switch pc.StrokeStyle.Join {
	case LineJoinRound:
		return raster.RoundJoiner
	default: // all others for now.. -- todo: support more joiners!!??
		return raster.BevelJoiner
	}
	return nil
}

func (pc *Paint) stroke(rs *RenderState, painter raster.Painter) {
	path := rs.StrokePath
	if len(pc.StrokeStyle.Dashes) > 0 {
		path = dashed(path, pc.StrokeStyle.Dashes)
	} else {
		// TODO: this is a temporary workaround to remove tiny segments
		// that result in rendering issues
		path = rasterPath(flattenPath(path))
	}
	sz := rs.Image.Bounds().Size()
	r := raster.NewRasterizer(sz.X, sz.Y)
	r.UseNonZeroWinding = true
	r.AddStroke(path, fix(pc.StrokeStyle.Width.Dots), pc.capper(), pc.joiner())
	r.Rasterize(painter)
}

func (pc *Paint) fill(rs *RenderState, painter raster.Painter) {
	path := rs.FillPath
	if rs.HasCurrent {
		path = make(raster.Path, len(rs.FillPath))
		copy(path, rs.FillPath)
		path.Add1(rs.Start.Fixed())
	}
	sz := rs.Image.Bounds().Size()
	r := raster.NewRasterizer(sz.X, sz.Y)
	r.UseNonZeroWinding = (pc.FillStyle.Rule == FillRuleNonZero)
	r.AddPath(path)
	r.Rasterize(painter)
}

// StrokePreserve strokes the current path with the current color, line width,
// line cap, line join and dash settings. The path is preserved after this
// operation.
func (pc *Paint) StrokePreserve(rs *RenderState) {
	painter := newPaintServerPainter(rs.Image, rs.Mask, pc.StrokeStyle.Server, rs.Bounds)
	pc.stroke(rs, painter)
}

// Stroke strokes the current path with the current color, line width,
// line cap, line join and dash settings. The path is cleared after this
// operation.
func (pc *Paint) Stroke(rs *RenderState) {
	pc.StrokePreserve(rs)
	pc.ClearPath(rs)
}

// FillPreserve fills the current path with the current color. Open subpaths
// are implicity closed. The path is preserved after this operation.
func (pc *Paint) FillPreserve(rs *RenderState) {
	painter := newPaintServerPainter(rs.Image, rs.Mask, pc.FillStyle.Server, rs.Bounds)
	pc.fill(rs, painter)
}

// Fill fills the current path with the current color. Open subpaths
// are implicity closed. The path is cleared after this operation.
func (pc *Paint) Fill(rs *RenderState) {
	pc.FillPreserve(rs)
	pc.ClearPath(rs)
}

// ClipPreserve updates the clipping region by intersecting the current
// clipping region with the current path as it would be filled by pc.Fill().
// The path is preserved after this operation.
func (pc *Paint) ClipPreserve(rs *RenderState) {
	clip := image.NewAlpha(rs.Image.Bounds())
	painter := raster.NewAlphaOverPainter(clip)
	pc.fill(rs, painter)
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
func (pc *Paint) SetMask(rs *RenderState, mask *image.Alpha) error {
	if mask.Bounds() != rs.Image.Bounds() {
		return errors.New("mask size must match context size")
	}
	rs.Mask = mask
	return nil
}

// AsMask returns an *image.Alpha representing the alpha channel of this
// context. This can be useful for advanced clipping operations where you first
// render the mask geometry and then use it as a mask.
func (pc *Paint) AsMask(rs *RenderState) *image.Alpha {
	b := rs.Image.Bounds()
	mask := image.NewAlpha(b)
	draw.Draw(mask, b, rs.Image, image.ZP, draw.Src)
	return mask
}

// Clip updates the clipping region by intersecting the current
// clipping region with the current path as it would be filled by pc.Fill().
// The path is cleared after this operation.
func (pc *Paint) Clip(rs *RenderState) {
	pc.ClipPreserve(rs)
	pc.ClearPath(rs)
}

// ResetClip clears the clipping region.
func (pc *Paint) ResetClip(rs *RenderState) {
	rs.Mask = nil
}

//////////////////////////////////////////////////////////////////////////////////
// Convenient Drawing Functions

// Clear fills the entire image with the current fill color.
func (pc *Paint) Clear(rs *RenderState) {
	src := image.NewUniform(&pc.FillStyle.Color)
	draw.Draw(rs.Image, rs.Image.Bounds(), src, image.ZP, draw.Src)
}

// SetPixel sets the color of the specified pixel using the current stroke color.
func (pc *Paint) SetPixel(rs *RenderState, x, y int) {
	rs.Image.Set(x, y, &pc.StrokeStyle.Color)
}

func (pc *Paint) DrawLine(rs *RenderState, x1, y1, x2, y2 float64) {
	pc.MoveTo(rs, x1, y1)
	pc.LineTo(rs, x2, y2)
}

func (pc *Paint) DrawPolyline(rs *RenderState, points []Vec2D) {
	sz := len(points)
	if sz < 2 {
		return
	}
	pc.MoveTo(rs, points[0].X, points[0].Y)
	for i := 1; i < sz; i++ {
		pc.LineTo(rs, points[i].X, points[i].Y)
	}
}

func (pc *Paint) DrawPolygon(rs *RenderState, points []Vec2D) {
	pc.DrawPolyline(rs, points)
	pc.ClosePath(rs)
}

func (pc *Paint) DrawRectangle(rs *RenderState, x, y, w, h float64) {
	pc.NewSubPath(rs)
	pc.MoveTo(rs, x, y)
	pc.LineTo(rs, x+w, y)
	pc.LineTo(rs, x+w, y+h)
	pc.LineTo(rs, x, y+h)
	pc.ClosePath(rs)
}

func (pc *Paint) DrawRoundedRectangle(rs *RenderState, x, y, w, h, r float64) {
	x0, x1, x2, x3 := x, x+r, x+w-r, x+w
	y0, y1, y2, y3 := y, y+r, y+h-r, y+h
	pc.NewSubPath(rs)
	pc.MoveTo(rs, x1, y0)
	pc.LineTo(rs, x2, y0)
	pc.DrawArc(rs, x2, y1, r, Radians(270), Radians(360))
	pc.LineTo(rs, x3, y2)
	pc.DrawArc(rs, x2, y2, r, Radians(0), Radians(90))
	pc.LineTo(rs, x1, y3)
	pc.DrawArc(rs, x1, y2, r, Radians(90), Radians(180))
	pc.LineTo(rs, x0, y1)
	pc.DrawArc(rs, x1, y1, r, Radians(180), Radians(270))
	pc.ClosePath(rs)
}

func (pc *Paint) DrawEllipticalArc(rs *RenderState, x, y, rx, ry, angle1, angle2 float64) {
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
		if i == 0 && !rs.HasCurrent {
			pc.MoveTo(rs, x0, y0)
		}
		pc.QuadraticTo(rs, cx, cy, x2, y2)
	}
}

func (pc *Paint) DrawEllipse(rs *RenderState, x, y, rx, ry float64) {
	pc.NewSubPath(rs)
	pc.DrawEllipticalArc(rs, x, y, rx, ry, 0, 2*math.Pi)
	pc.ClosePath(rs)
}

func (pc *Paint) DrawArc(rs *RenderState, x, y, r, angle1, angle2 float64) {
	pc.DrawEllipticalArc(rs, x, y, r, r, angle1, angle2)
}

func (pc *Paint) DrawCircle(rs *RenderState, x, y, r float64) {
	pc.NewSubPath(rs)
	pc.DrawEllipticalArc(rs, x, y, r, r, 0, 2*math.Pi)
	pc.ClosePath(rs)
}

func (pc *Paint) DrawRegularPolygon(rs *RenderState, n int, x, y, r, rotation float64) {
	angle := 2 * math.Pi / float64(n)
	rotation -= math.Pi / 2
	if n%2 == 0 {
		rotation += angle / 2
	}
	pc.NewSubPath(rs)
	for i := 0; i < n; i++ {
		a := rotation + angle*float64(i)
		pc.LineTo(rs, x+r*math.Cos(a), y+r*math.Sin(a))
	}
	pc.ClosePath(rs)
}

// DrawImage draws the specified image at the specified point.
func (pc *Paint) DrawImage(rs *RenderState, fmIm image.Image, x, y int) {
	pc.DrawImageAnchored(rs, fmIm, x, y, 0, 0)
}

// DrawImageAnchored draws the specified image at the specified anchor point.
// The anchor point is x - w * ax, y - h * ay, where w, h is the size of the
// image. Use ax=0.5, ay=0.5 to center the image at the specified point.
func (pc *Paint) DrawImageAnchored(rs *RenderState, fmIm image.Image, x, y int, ax, ay float64) {
	s := rs.Image.Bounds().Size()
	x -= int(ax * float64(s.X))
	y -= int(ay * float64(s.Y))
	transformer := draw.BiLinear
	fx, fy := float64(x), float64(y)
	m := rs.XForm.Translate(fx, fy)
	s2d := f64.Aff3{m.XX, m.XY, m.X0, m.YX, m.YY, m.Y0}
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
// Text Functions

func (pc *Paint) SetFontFace(fontFace font.Face) {
	pc.FontStyle.Face = fontFace
	pc.FontStyle.Height = float64(fontFace.Metrics().Height) / 64.0
}

func (pc *Paint) LoadFontFace(path string, points float64) error {
	face, err := FontLibrary.Font(path, points)
	if err == nil {
		pc.SetFontFace(face)
	}
	return err
}

func (pc *Paint) FontHeight() float64 {
	return pc.FontStyle.Height
}

func (pc *Paint) drawString(rs *RenderState, im *image.RGBA, bounds image.Rectangle, s string, x, y float64) {
	if int(y) < bounds.Min.Y || int(y) > bounds.Max.Y {
		return
	}
	d := &font.Drawer{
		Dst:  im,
		Src:  image.NewUniform(&pc.StrokeStyle.Color),
		Face: pc.FontStyle.Face,
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
		nxt := d.Dot.X + advance
		if nxt.Ceil() > bounds.Max.X || d.Dot.X.Floor() < bounds.Min.X {
			d.Dot.X = nxt
			continue
		}
		sr := dr.Sub(dr.Min)
		transformer := draw.BiLinear
		fx, fy := float64(dr.Min.X), float64(dr.Min.Y)
		m := rs.XForm.Translate(fx, fy)
		s2d := f64.Aff3{m.XX, m.XY, m.X0, m.YX, m.YY, m.Y0}
		transformer.Transform(d.Dst, s2d, d.Src, sr, draw.Over, &draw.Options{
			SrcMask:  mask,
			SrcMaskP: maskp,
		})
		d.Dot.X = nxt
		prevC = c
	}
}

// DrawString according to current settings -- width is needed for alignment
// -- if non-zero, then x position is for the left edge of the width box, and
// alignment is WRT that width -- otherwise x position is as in
// DrawStringAnchored
func (pc *Paint) DrawString(rs *RenderState, s string, x, y, width float64) {
	ax, ay := pc.TextStyle.AlignFactors()
	if width > 0.0 {
		x += ax * width // re-offset for width
	}
	if pc.TextStyle.WordWrap {
		pc.DrawStringWrapped(rs, s, x, y, ax, ay, width, pc.TextStyle.EffLineHeight())
	} else {
		pc.DrawStringAnchored(rs, s, x, y, ax, ay, width)
	}
}

func (pc *Paint) DrawStringLines(rs *RenderState, lines []string, x, y, width, height float64) {
	ax, ay := pc.TextStyle.AlignFactors()
	pc.DrawStringLinesAnchored(rs, lines, x, y, ax, ay, width, height, pc.TextStyle.EffLineHeight())
}

// DrawStringAnchored draws the specified text at the specified anchor point.
// The anchor point is x - w * ax, y - h * ay, where w, h is the size of the
// text. Use ax=0.5, ay=0.5 to center the text at the specified point.
func (pc *Paint) DrawStringAnchored(rs *RenderState, s string, x, y, ax, ay, width float64) {
	w, h := pc.MeasureString(s)
	x -= ax * w
	y += ay * h
	// fmt.Printf("ds bounds: %v point x,y %v, %v\n", rs.Bounds, x, y)
	if rs.Mask == nil {
		pc.drawString(rs, rs.Image, rs.Bounds, s, x, y)
	} else {
		im := image.NewRGBA(rs.Image.Bounds())
		pc.drawString(rs, im, rs.Bounds, s, x, y)
		draw.DrawMask(rs.Image, rs.Image.Bounds(), im, image.ZP, rs.Mask, image.ZP, draw.Over)
	}
}

// DrawStringWrapped word-wraps the specified string to the given max width
// and then draws it at the specified anchor point using the given line
// spacing and text alignment.
func (pc *Paint) DrawStringWrapped(rs *RenderState, s string, x, y, ax, ay, width, lineHeight float64) {
	lines, h := pc.MeasureStringWrapped(s, width, lineHeight)
	pc.DrawStringLinesAnchored(rs, lines, x, y, ax, ay, width, h, lineHeight)
}

func (pc *Paint) DrawStringLinesAnchored(rs *RenderState, lines []string, x, y, ax, ay, width, h, lineHeight float64) {
	x -= ax * width
	y -= ay * h
	ax, ay = pc.TextStyle.AlignFactors()
	// ay = 1
	for _, line := range lines {
		pc.DrawStringAnchored(rs, line, x, y, ax, ay, width)
		y += pc.FontStyle.Height * lineHeight
	}
}

// todo: all of these measurements are failing to take into account transforms -- maybe that's ok -- keep the font non-scaled?  maybe add an option for that actually..

// MeasureString returns the rendered width and height of the specified text
// given the current font face.
func (pc *Paint) MeasureString(s string) (w, h float64) {
	if pc.FontStyle.Face == nil {
		pc.FontStyle.LoadFont(&pc.UnContext, "")
	}
	d := &font.Drawer{
		Face: pc.FontStyle.Face,
	}
	a := d.MeasureString(s)
	return float64(a >> 6), pc.FontStyle.Height
}

func (pc *Paint) MeasureStringWrapped(s string, width, lineHeight float64) ([]string, float64) {
	lines := pc.WordWrap(s, width)
	h := float64(len(lines)) * pc.FontStyle.Height * lineHeight
	h -= (lineHeight - 1) * pc.FontStyle.Height
	return lines, h
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
func (pc *Paint) InvertY(rs *RenderState) {
	pc.Translate(0, float64(rs.Image.Bounds().Size().Y))
	pc.Scale(1, -1)
}

////////////////////////////////////////////////////////////////////////////////////
// Internal -- might want to export these later depending

func flattenPath(p raster.Path) [][]Vec2D {
	var result [][]Vec2D
	var path []Vec2D
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
			path = append(path, Vec2D{x, y})
			cx, cy = x, y
			i += 4
		case 1:
			x := unfix(p[i+1])
			y := unfix(p[i+2])
			path = append(path, Vec2D{x, y})
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

func dashPath(paths [][]Vec2D, dashes []float64) [][]Vec2D {
	var result [][]Vec2D
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
		var segment []Vec2D
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

func rasterPath(paths [][]Vec2D) raster.Path {
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
