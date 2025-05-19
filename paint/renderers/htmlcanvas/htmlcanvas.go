// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

//go:build js

package htmlcanvas

import (
	"image"
	"syscall/js"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/pimage"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// Renderer is an HTML canvas renderer.
type Renderer struct {
	Canvas js.Value
	ctx    js.Value
	size   math32.Vector2

	// curRect is the rectangle of the current object.
	curRect image.Rectangle
}

// New returns an HTMLCanvas renderer.
func New(size math32.Vector2) render.Renderer {
	rs := &Renderer{}
	rs.SetSize(units.UnitDot, size)
	return rs
}

func (rs *Renderer) Size() (units.Units, math32.Vector2) {
	return units.UnitDot, rs.size
}

func (rs *Renderer) SetSize(un units.Units, size math32.Vector2) {
	if rs.size == size {
		return
	}
	// TODO: truncate/round here? (HTML doesn't support fractional width/height)
	rs.size = size

	return // TODO(newpaint)

	// rs.ctx.Call("clearRect", 0, 0, size.X, size.Y)
	// rs.ctx.Set("imageSmoothingQuality", "high")
}

// todo: could return wrapped JS image grabbed from canvas.
func (rs *Renderer) Image() image.Image { return nil }
func (rs *Renderer) Source() []byte     { return nil }

func (rs *Renderer) SetCanvas(c js.Value) {
	rs.Canvas = c
	rs.ctx = rs.Canvas.Call("getContext", "2d", "alpha", "true")
	// todo: make this a font options.
	// rs.ctx.Set("imageSmoothingEnabled", false)
	rs.ctx.Set("textRendering", "geometricPrecision")
	// rs.ctx.Set("willReadFrequently", "false")
	// rs.ctx.Set("desynchronized", "false")
}

// Render is the main rendering function.
func (rs *Renderer) Render(r render.Render) render.Renderer {
	for _, ri := range r {
		switch x := ri.(type) {
		case *render.Path:
			rs.RenderPath(x)
		case *pimage.Params:
			rs.RenderImage(x)
		case *render.Text:
			rs.RenderText(x)
		case *render.ContextPush:
			rs.PushContext(x)
		case *render.ContextPop:
			rs.PopContext(x)
		}
	}
	return rs
}

func (rs *Renderer) PushContext(pt *render.ContextPush) {
	pc := &pt.Context
	rs.ctx.Call("save") // save clip region prior to using
	br := pc.Bounds.Rect.ToRect()
	rs.ctx.Call("rect", br.Min.X, br.Min.Y, br.Dx(), br.Dy())
	rs.ctx.Call("clip")
}

func (rs *Renderer) PopContext(pt *render.ContextPop) {
	rs.ctx.Call("restore") // restore clip region
}

func (rs *Renderer) setTransform(ctx *render.Context) {
	m := ctx.Transform
	rs.ctx.Call("setTransform", m.XX, m.YX, m.XY, m.YY, m.X0, m.Y0)
}

func (rs *Renderer) setFill(clr image.Image) {
	rs.ctx.Set("fillStyle", rs.imageToStyle(clr))
}

func (rs *Renderer) setStroke(stroke *styles.Stroke) {
	rs.ctx.Set("lineCap", stroke.Cap.String())
	rs.ctx.Set("lineJoin", stroke.Join.String())
	if stroke.Join == ppath.JoinMiter && !math32.IsNaN(stroke.MiterLimit) {
		rs.ctx.Set("miterLimit", stroke.MiterLimit)
	}
	dashes := []any{}
	for _, dash := range stroke.Dashes {
		dashes = append(dashes, dash)
	}
	jsDashes := js.Global().Get("Array").New(dashes...)
	rs.ctx.Call("setLineDash", jsDashes)
	rs.ctx.Set("lineDashOffset", stroke.DashOffset)
	rs.ctx.Set("lineWidth", stroke.Width.Dots)
	rs.ctx.Set("strokeStyle", rs.imageToStyle(stroke.Color))
}

// toJSer is satisfied by [gradient.Gradient] on JS.
type toJSer interface {
	ToJS(ctx js.Value) js.Value
}

func (rs *Renderer) imageToStyle(clr image.Image) any {
	if g, ok := clr.(gradient.Gradient); ok {
		g.Update(1, math32.B2FromRect(rs.curRect), math32.Identity2()) // TODO: opacity, transform?
		return g.(toJSer).ToJS(rs.ctx)
	}
	// TODO: handle more cases for things like pattern functions and [image.RGBA] images?
	return colors.AsHex(colors.ToUniform(clr))
}
