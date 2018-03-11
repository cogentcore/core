// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	"github.com/rcoreilly/goki/ki"
	"golang.org/x/image/font"
	"image"
	"image/draw"
	"image/png"
	"io"
	"log"
	"os"
	"reflect"
)

// Viewport2D provides an image and a stack of Paint contexts for drawing onto the image
// with a convenience forwarding of the Paint methods operating on the current Paint
type Viewport2D struct {
	Node2DBase
	ViewBox ViewBox2D   `svg:"viewBox",desc:"viewbox within any parent Viewport2D"`
	Paints  []*Paint    `json:"-",desc:"paint stack for rendering"`
	Pixels  *image.RGBA `json:"-",desc:"pixels that we render into"`
	Backing *image.RGBA `json:"-",desc:"if non-nil, this is what goes behind our image -- copied from our region in parent image -- allows us to re-render cleanly into parent, even with transparency"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Viewport2D = ki.KiTypes.AddType(&Viewport2D{})

// NewViewport2D creates a new image.RGBA with the specified width and height
// and prepares a context for rendering onto that image.
func NewViewport2D(width, height int) *Viewport2D {
	return NewViewport2DForRGBA(image.NewRGBA(image.Rect(0, 0, width, height)))
}

// NewViewport2DForImage copies the specified image into a new image.RGBA
// and prepares a context for rendering onto that image.
func NewViewport2DForImage(im image.Image) *Viewport2D {
	return NewViewport2DForRGBA(imageToRGBA(im))
}

// NewViewport2DForRGBA prepares a context for rendering onto the specified image.
// No copy is made.
func NewViewport2DForRGBA(im *image.RGBA) *Viewport2D {
	vp := &Viewport2D{
		ViewBox: ViewBox2D{Size: image.Point{X: im.Bounds().Size().X,
			Y: im.Bounds().Size().Y}},
		Pixels: im,
	}
	vp.PushNewPaint()
	return vp
}

// resize viewport, creating a new image (no point in trying to resize the image -- need to re-render) -- updates ViewBox
func (vp *Viewport2D) Resize(width, heigth int) {
	vp.UpdateStart()

}

////////////////////////////////////////////////////////////////////////////////////////
//  Main Rendering code

// gets the current Paint at top of stack
func (vp *Viewport2D) CurPaint() *Paint {
	return vp.Paints[len(vp.Paints)-1]
}

// Push a new paint on top of stack -- always copies current values
func (vp *Viewport2D) PushNewPaint() *Paint {
	c := &Paint{}
	if len(vp.Paints) > 0 {
		*c = *vp.CurPaint() // always copy current settings
	} else {
		c.Defaults(vp.Pixels.Bounds())
	}
	vp.Paints = append(vp.Paints, c)
	return c
}

// Pop current top-level paint off the stack -- cannot go less than 1
func (vp *Viewport2D) PopPaint() {
	sz := len(vp.Paints)
	if sz == 1 {
		return
	}
	vp.Paints[sz-1] = nil
	vp.Paints = vp.Paints[:sz-1]
}

// does the current Paint have an active stroke to render?
func (vp *Viewport2D) HasStroke() bool {
	pc := vp.CurPaint()
	return pc.Stroke.On
}

// does the current Paint have an active fill to render?
func (vp *Viewport2D) HasFill() bool {
	pc := vp.CurPaint()
	return pc.Fill.On
}

// does the current Paint not have either a stroke or fill?  in which case, we just skip it
func (vp *Viewport2D) HasNoStrokeOrFill() bool {
	pc := vp.CurPaint()
	return (!pc.Stroke.On && !pc.Fill.On)
}

// draw our image into parents -- called at right place in Render
func (vp *Viewport2D) DrawIntoParent(parVp *Viewport2D) {
	r := vp.ViewBox.Bounds()
	if vp.Backing != nil {
		draw.Draw(parVp.Pixels, r, vp.Backing, image.ZP, draw.Src)
	}
	draw.Draw(parVp.Pixels, r, vp.Pixels, image.ZP, draw.Src)
}

// copy our backing image from parent -- called at right place in Render
func (vp *Viewport2D) CopyBacking(parVp *Viewport2D) {
	r := vp.ViewBox.Bounds()
	if vp.Backing == nil {
		vp.Backing = image.NewRGBA(vp.ViewBox.SizeRect())
	}
	draw.Draw(vp.Backing, r, parVp.Pixels, image.ZP, draw.Src)
}

func (vp *Viewport2D) DrawIntoWindow() {
	wini := vp.FindParentByType(reflect.TypeOf(Window{}))
	if wini != nil {
		win := (wini).(*Window)
		// width, height := win.Win.Size() // todo: update size of our window
		s := win.Win.Screen()
		s.CopyRGBA(vp.Pixels, vp.Pixels.Bounds())
		win.Win.FlushImage()
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Node2D interface

func (vp *Viewport2D) GiNode2D() *Node2DBase {
	return &vp.Node2DBase
}

func (vp *Viewport2D) GiViewport2D() *Viewport2D {
	return vp
}

func (vp *Viewport2D) Node2DBBox(parVp *Viewport2D) image.Rectangle {
	return vp.ViewBox.Bounds()
}

// viewport has a special render function that handles all the items below
func (vp *Viewport2D) Render2D(parVp *Viewport2D) bool {
	last_level := 0
	vp.FunDown(last_level, vp, func(k ki.Ki, level int, d interface{}) bool {
		gii, ok := k.(Node2D)
		if !ok { // error message already in InitNode2D
			return false // going into a different type of thing, bail
		}
		gi := gii.GiNode2D()
		if k == vp.This {
			bb := gii.Node2DBBox(parVp) // only update our bbox
			if parVp != nil {
				gi.WinBBox = bb.Add(image.Point{parVp.WinBBox.Min.X, parVp.WinBBox.Min.Y})
			} else {
				gi.WinBBox = bb
			}
			return true
		}
		disp := gi.PropDisplay()
		if !disp { // go no further
			gi.ZeroWinBBox()
			return false
		}
		// from here we need to update context
		if level > last_level {
			vp.PushNewPaint()
		} else if level < last_level {
			vp.PopPaint()
		}
		last_level = level
		cont := true // whether to continue down the stack at this point
		vis := gi.PropVisible()
		if vis {
			if gi.IsLeaf() { // each terminal leaf gets its own context
				vp.PushNewPaint()
			}
			vp.SetPaintFromNode(gi)
			gi.XForm = vp.CurPaint().XForm // cache current xform -- not clear if needed
			bb := gii.Node2DBBox(vp)       // only update our bbox
			gi.WinBBox = bb.Add(image.Point{vp.WinBBox.Min.X, vp.WinBBox.Min.Y})
			cont = gii.Render2D(vp) // if this is itself a vp, we need to stop
			if gi.IsLeaf() {
				vp.PopPaint()
			}
		} else {
			gi.XForm = vp.CurPaint().XForm // cache current xform even if not visible?  maybe nil?
			gi.ZeroWinBBox()               // not visible
		}
		return cont
	})
	if parVp != nil {
		vp.CopyBacking(parVp) // full re-render is when we copy the backing
		vp.DrawIntoParent(parVp)
	} else { // top-level, try drawing into window
		vp.DrawIntoWindow()
	}
	return false // we tell parent render to not continue down any further -- we just did it all
}

// viewport has a special render function that handles all the items below
func (vp *Viewport2D) InitNode2D(parVp *Viewport2D) bool {
	vp.FunDown(0, vp, func(k ki.Ki, level int, d interface{}) bool {
		if level == 0 || k == vp.This { // don't process us!
			return true
		}
		gii, ok := (interface{}(k)).(Node2D)
		if !ok {
			// todo: need to detect the thing that wraps a 3D node inside a 2D, and stop there
			log.Printf("Node %v in Viewport2D does NOT implement Node2D interface -- it should!\n", k.PathUnique())
			return true
		}
		cont := gii.InitNode2D(vp)
		return cont

	})
	return false // we tell parent render to not continue down any further -- we just did it all
}

////////////////////////////////////////////////////////////////////////////////////////
// Top-level API

func (vp *Viewport2D) RenderTopLevel() {
	vp.Render2D(nil) // we are the top
}

func (vp *Viewport2D) InitTopLevel() {
	vp.InitNode2D(nil) // we are the top
}

// SavePNG encodes the image as a PNG and writes it to disk.
func (vp *Viewport2D) SavePNG(path string) error {
	return SavePNG(path, vp.Pixels)
}

// EncodePNG encodes the image as a PNG and writes it to the provided io.Writer.
func (vp *Viewport2D) EncodePNG(w io.Writer) error {
	return png.Encode(w, vp.Pixels)
}

//////////////////////////////////////////////////////////////////////////////////
//                       Below are largely Paint wrappers

//////////////////////////////////////////////////////////////////////////////////
// Path Manipulation

// update the Paint Stroke and Fill from the properties of a given node -- because Paint stack captures all the relevant inheritance, this does NOT look for inherited properties
func (vp *Viewport2D) SetPaintFromNode(g *Node2DBase) {
	pc := vp.CurPaint()
	pc.SetFromNode(g)
}

// get the bounding box for an element in pixel int coordinates
func (vp *Viewport2D) BoundingBox(minX, minY, maxX, maxY float64) image.Rectangle {
	pc := vp.CurPaint()
	return pc.BoundingBox(minX, minY, maxX, maxY)
}

// get the bounding box for a slice of points
func (vp *Viewport2D) BoundingBoxFromPoints(points []Point2D) image.Rectangle {
	pc := vp.CurPaint()
	return pc.BoundingBoxFromPoints(points)
}

// MoveTo starts a new subpath within the current path starting at the
// specified point.
func (vp *Viewport2D) MoveTo(x, y float64) {
	pc := vp.CurPaint()
	pc.MoveTo(x, y)
}

// LineTo adds a line segment to the current path starting at the current
// point. If there is no current point, it is equivalent to MoveTo(x, y)
func (vp *Viewport2D) LineTo(x, y float64) {
	pc := vp.CurPaint()
	pc.LineTo(x, y)
}

// QuadraticTo adds a quadratic bezier curve to the current path starting at
// the current point. If there is no current point, it first performs
// MoveTo(x1, y1)
func (vp *Viewport2D) QuadraticTo(x1, y1, x2, y2 float64) {
	pc := vp.CurPaint()
	pc.QuadraticTo(x1, y1, x2, y2)
}

// CubicTo adds a cubic bezier curve to the current path starting at the
// current point. If there is no current point, it first performs
// MoveTo(x1, y1). Because freetype/raster does not support cubic beziers,
// this is emulated with many small line segments.
func (vp *Viewport2D) CubicTo(x1, y1, x2, y2, x3, y3 float64) {
	pc := vp.CurPaint()
	pc.CubicTo(x1, y1, x2, y2, x3, y3)
}

// ClosePath adds a line segment from the current point to the beginning
// of the current subpath. If there is no current point, this is a no-op.
func (vp *Viewport2D) ClosePath() {
	pc := vp.CurPaint()
	pc.ClosePath()
}

// ClearPath clears the current path. There is no current point after this
// operation.
func (vp *Viewport2D) ClearPath() {
	pc := vp.CurPaint()
	pc.ClearPath()
}

// NewSubPath starts a new subpath within the current path. There is no current
// point after this operation.
func (vp *Viewport2D) NewSubPath() {
	pc := vp.CurPaint()
	pc.NewSubPath()
}

// StrokePreserve strokes the current path with the current color, line width,
// line cap, line join and dash settings. The path is preserved after this
// operation.
func (vp *Viewport2D) StrokePreserve() {
	pc := vp.CurPaint()
	pc.StrokePreserve(vp.Pixels)
}

// Stroke strokes the current path with the current color, line width,
// line cap, line join and dash settings. The path is cleared after this
// operation.
func (vp *Viewport2D) Stroke() {
	pc := vp.CurPaint()
	pc.StrokeImage(vp.Pixels)
}

// FillPreserve fills the current path with the current color. Open subpaths
// are implicity closed. The path is preserved after this operation.
func (vp *Viewport2D) FillPreserve() {
	pc := vp.CurPaint()
	pc.FillPreserve(vp.Pixels)
}

// Fill fills the current path with the current color. Open subpaths
// are implicity closed. The path is cleared after this operation.
func (vp *Viewport2D) Fill() {
	pc := vp.CurPaint()
	pc.FillImage(vp.Pixels)
}

// ClipPreserve updates the clipping region by intersecting the current
// clipping region with the current path as it would be filled by vp.Fill().
// The path is preserved after this operation.
func (vp *Viewport2D) ClipPreserve() {
	pc := vp.CurPaint()
	pc.ClipPreserve()
}

// SetMask allows you to directly set the *image.Alpha to be used as a clipping
// mask. It must be the same size as the context, else an error is returned
// and the mask is unchanged.
func (vp *Viewport2D) SetMask(mask *image.Alpha) error {
	pc := vp.CurPaint()
	return pc.SetMask(mask)
}

// AsMask returns an *image.Alpha representing the alpha channel of this
// context. This can be useful for advanced clipping operations where you first
// render the mask geometry and then use it as a mask.
func (vp *Viewport2D) AsMask() *image.Alpha {
	pc := vp.CurPaint()
	return pc.AsMask(vp.Pixels)
}

// Clip updates the clipping region by intersecting the current
// clipping region with the current path as it would be filled by vp.Fill().
// The path is cleared after this operation.
func (vp *Viewport2D) Clip() {
	pc := vp.CurPaint()
	pc.Clip()
}

// ResetClip clears the clipping region.
func (vp *Viewport2D) ResetClip() {
	pc := vp.CurPaint()
	pc.ResetClip()
}

//////////////////////////////////////////////////////////////////////////////////
// Convenient Drawing Functions

// Clear fills the entire image with the current color.
func (vp *Viewport2D) Clear() {
	pc := vp.CurPaint()
	pc.Clear(vp.Pixels)
}

// SetPixel sets the color of the specified pixel using the current color.
func (vp *Viewport2D) SetPixel(x, y int) {
	pc := vp.CurPaint()
	pc.SetPixel(vp.Pixels, x, y)
}

// DrawPoint is like DrawCircle but ensures that a circle of the specified
// size is drawn regardless of the current transformation matrix. The position
// is still transformed, but not the shape of the point.
func (vp *Viewport2D) DrawPoint(x, y, r float64) {
	pc := vp.PushNewPaint()
	p := pc.TransformPoint(x, y)
	pc.Identity()
	pc.DrawCircle(p.X, p.Y, r)
	vp.PopPaint()
}

func (vp *Viewport2D) DrawLine(x1, y1, x2, y2 float64) {
	pc := vp.CurPaint()
	pc.DrawLine(x1, y1, x2, y2)
}

func (vp *Viewport2D) DrawPolyline(points []Point2D) {
	pc := vp.CurPaint()
	pc.DrawPolyline(points)
}

func (vp *Viewport2D) DrawPolygon(points []Point2D) {
	pc := vp.CurPaint()
	pc.DrawPolygon(points)
}

func (vp *Viewport2D) DrawRectangle(x, y, w, h float64) {
	pc := vp.CurPaint()
	pc.DrawRectangle(x, y, w, h)
}

func (vp *Viewport2D) DrawRoundedRectangle(x, y, w, h, r float64) {
	pc := vp.CurPaint()
	pc.DrawRoundedRectangle(x, y, w, h, r)
}

func (vp *Viewport2D) DrawEllipticalArc(x, y, rx, ry, angle1, angle2 float64) {
	pc := vp.CurPaint()
	pc.DrawEllipticalArc(x, y, rx, ry, angle1, angle2)
}

func (vp *Viewport2D) DrawEllipse(x, y, rx, ry float64) {
	pc := vp.CurPaint()
	pc.DrawEllipse(x, y, rx, ry)
}

func (vp *Viewport2D) DrawArc(x, y, r, angle1, angle2 float64) {
	pc := vp.CurPaint()
	pc.DrawArc(x, y, r, angle1, angle2)
}

func (vp *Viewport2D) DrawCircle(x, y, r float64) {
	pc := vp.CurPaint()
	pc.DrawCircle(x, y, r)
}

func (vp *Viewport2D) DrawRegularPolygon(n int, x, y, r, rotation float64) {
	pc := vp.CurPaint()
	pc.DrawRegularPolygon(n, x, y, r, rotation)
}

// DrawImage draws the specified image at the specified point.
func (vp *Viewport2D) DrawImage(im image.Image, x, y int) {
	pc := vp.CurPaint()
	pc.DrawImage(vp.Pixels, im, x, y)
}

// DrawImageAnchored draws the specified image at the specified anchor point.
// The anchor point is x - w * ax, y - h * ay, where w, h is the size of the
// image. Use ax=0.5, ay=0.5 to center the image at the specified point.
func (vp *Viewport2D) DrawImageAnchored(im image.Image, x, y int, ax, ay float64) {
	pc := vp.CurPaint()
	pc.DrawImageAnchored(vp.Pixels, im, x, y, ax, ay)
}

//////////////////////////////////////////////////////////////////////////////////
// Text Functions

func (vp *Viewport2D) SetFontFace(fontFace font.Face) {
	pc := vp.CurPaint()
	pc.SetFontFace(fontFace)
}

func (vp *Viewport2D) LoadFontFace(path string, points float64) error {
	pc := vp.CurPaint()
	return pc.LoadFontFace(path, points)
}

func (vp *Viewport2D) FontHeight() float64 {
	pc := vp.CurPaint()
	return pc.FontHeight()
}

// DrawString draws the specified text at the specified point.
func (vp *Viewport2D) DrawString(s string, x, y, width float64) {
	pc := vp.CurPaint()
	pc.DrawString(vp.Pixels, s, x, y, width)
}

// DrawStringAnchored draws the specified text at the specified anchor point.
// The anchor point is x - w * ax, y - h * ay, where w, h is the size of the
// text. Use ax=0.5, ay=0.5 to center the text at the specified point.
func (vp *Viewport2D) DrawStringAnchored(s string, x, y, ax, ay float64) {
	pc := vp.CurPaint()
	pc.DrawStringAnchored(vp.Pixels, s, x, y, ax, ay)
}

// DrawStringWrapped word-wraps the specified string to the given max width
// and then draws it at the specified anchor point using the given line
// spacing and text alignment.
func (vp *Viewport2D) DrawStringWrapped(s string, x, y, ax, ay, width, lineSpacing float64, align TextAlign) {
	pc := vp.CurPaint()
	pc.DrawStringWrapped(vp.Pixels, s, x, y, ax, ay, width, lineSpacing, align)
}

// MeasureString returns the rendered width and height of the specified text
// given the current font face.
func (vp *Viewport2D) MeasureString(s string) (w, h float64) {
	pc := vp.CurPaint()
	return pc.MeasureString(s)
}

// WordWrap wraps the specified string to the given max width and current
// font face.
func (vp *Viewport2D) WordWrap(s string, w float64) []string {
	return wordWrap(vp, s, w)
}

//////////////////////////////////////////////////////////////////////////////////
// Transformation Matrix Operations

// Identity resets the current transformation matrix to the identity matrix.
// This results in no translating, scaling, rotating, or shearing.
func (vp *Viewport2D) Identity() {
	pc := vp.CurPaint()
	pc.Identity()
}

// Translate updates the current matrix with a translation.
func (vp *Viewport2D) Translate(x, y float64) {
	pc := vp.CurPaint()
	pc.Translate(x, y)
}

// Scale updates the current matrix with a scaling factor.
// Scaling occurs about the origin.
func (vp *Viewport2D) Scale(x, y float64) {
	pc := vp.CurPaint()
	pc.Scale(x, y)
}

// ScaleAbout updates the current matrix with a scaling factor.
// Scaling occurs about the specified point.
func (vp *Viewport2D) ScaleAbout(sx, sy, x, y float64) {
	pc := vp.CurPaint()
	pc.ScaleAbout(sx, sy, x, y)
}

// Rotate updates the current matrix with a clockwise rotation.
// Rotation occurs about the origin. Angle is specified in radians.
func (vp *Viewport2D) Rotate(angle float64) {
	pc := vp.CurPaint()
	pc.Rotate(angle)
}

// RotateAbout updates the current matrix with a clockwise rotation.
// Rotation occurs about the specified point. Angle is specified in radians.
func (vp *Viewport2D) RotateAbout(angle, x, y float64) {
	pc := vp.CurPaint()
	pc.RotateAbout(angle, x, y)
}

// Shear updates the current matrix with a shearing angle.
// Shearing occurs about the origin.
func (vp *Viewport2D) Shear(x, y float64) {
	pc := vp.CurPaint()
	pc.Shear(x, y)
}

// ShearAbout updates the current matrix with a shearing angle.
// Shearing occurs about the specified point.
func (vp *Viewport2D) ShearAbout(sx, sy, x, y float64) {
	pc := vp.CurPaint()
	pc.ShearAbout(sx, sy, x, y)
}

// InvertY flips the Y axis so that Y grows from bottom to top and Y=0 is at
// the bottom of the image.
func (vp *Viewport2D) InvertY() {
	pc := vp.CurPaint()
	pc.InvertY()
}

//////////////////////////////////////////////////////////////////////////////////
//  Image utilities

func LoadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	im, _, err := image.Decode(file)
	return im, err
}

func LoadPNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return png.Decode(file)
}

func SavePNG(path string, im image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, im)
}

func imageToRGBA(src image.Image) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Rect, src, image.ZP, draw.Src)
	return dst
}
