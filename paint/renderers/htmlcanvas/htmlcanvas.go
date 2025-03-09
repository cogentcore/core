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
	"github.com/cogentcore/webgpu/wgpu"
)

// Renderers is a list of all current HTML canvas renderers.
// It is used in core to delete inactive canvases.
var Renderers []*Renderer

// Renderer is an HTML canvas renderer.
type Renderer struct {
	Canvas js.Value
	ctx    js.Value
	size   math32.Vector2

	// style is a cached style of the most recently used styles for rendering,
	// which allows for avoiding unnecessary JS calls.
	style styles.Paint
}

// New returns an HTMLCanvas renderer. It makes a corresponding new HTML canvas element.
// It adds the renderer to [Renderers].
func New(size math32.Vector2) render.Renderer {
	rs := &Renderer{}
	rs.SetSize(units.UnitDot, size)
	Renderers = append(Renderers, rs)
	return rs
}

func (rs *Renderer) Size() (units.Units, math32.Vector2) {
	return units.UnitDot, rs.size // TODO: is Dot right?
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

func (rs *Renderer) SetCanvas(c js.Value) {
	rs.Canvas = c
	rs.ctx = rs.Canvas.Call("getContext", "2d")
	// todo: make this a font options.
	// rs.ctx.Set("imageSmoothingEnabled", false)
	// rs.ctx.Set("textRendering", "geometricPrecision")
}

// Render is the main rendering function.
func (rs *Renderer) Render(r render.Render) {
	for _, ri := range r {
		switch x := ri.(type) {
		case *render.Path:
			rs.RenderPath(x)
		case *pimage.Params:
			rs.RenderImage(x)
		case *render.Text:
			rs.RenderText(x)
		}
	}
}

func (rs *Renderer) writePath(pt *render.Path) {
	rs.ctx.Call("beginPath")
	for scanner := pt.Path.Scanner(); scanner.Scan(); {
		end := scanner.End()
		switch scanner.Cmd() {
		case ppath.MoveTo:
			rs.ctx.Call("moveTo", end.X, end.Y)
		case ppath.LineTo:
			rs.ctx.Call("lineTo", end.X, end.Y)
		case ppath.QuadTo:
			cp := scanner.CP1()
			rs.ctx.Call("quadraticCurveTo", cp.X, cp.Y, end.X, end.Y)
		case ppath.CubeTo:
			cp1, cp2 := scanner.CP1(), scanner.CP2()
			rs.ctx.Call("bezierCurveTo", cp1.X, cp1.Y, cp2.X, cp2.Y, end.X, end.Y)
		case ppath.Close:
			rs.ctx.Call("closePath")
		}
	}
}

func (rs *Renderer) imageToStyle(clr image.Image) any {
	if g, ok := clr.(gradient.Gradient); ok {
		if gl, ok := g.(*gradient.Linear); ok {
			grad := rs.ctx.Call("createLinearGradient", gl.Start.X, gl.Start.Y, gl.End.X, gl.End.Y) // TODO: are these params right?
			for _, stop := range gl.Stops {
				grad.Call("addColorStop", stop.Pos, colors.AsHex(stop.Color))
			}
			return grad
		} else if gr, ok := g.(*gradient.Radial); ok {
			grad := rs.ctx.Call("createRadialGradient", gr.Center.X, gr.Center.Y, gr.Radius, gr.Focal.X, gr.Focal.Y, gr.Radius) // TODO: are these params right?
			for _, stop := range gr.Stops {
				grad.Call("addColorStop", stop.Pos, colors.AsHex(stop.Color))
			}
			return grad
		}
	}
	// TODO: handle more cases for things like pattern functions and [image.RGBA] images?
	return colors.AsHex(colors.ToUniform(clr))
}

func (rs *Renderer) RenderPath(pt *render.Path) {
	if pt.Path.Empty() {
		return
	}
	pc := &pt.Context
	rs.ctx.Call("save") // save clip region prior to using
	br := pc.Bounds.Rect.ToRect()
	rs.ctx.Call("rect", br.Min.X, br.Min.Y, br.Dx(), br.Dy())
	rs.ctx.Call("clip")

	style := &pt.Context.Style
	p := pt.Path
	if !ppath.ArcToCubeImmediate {
		p = p.ReplaceArcs() // TODO: should we do this in writePath?
	}
	m := pt.Context.Transform                                       // TODO: do we need to do more transform handling of m?
	rs.ctx.Call("setTransform", m.XX, m.YX, m.XY, m.YY, m.X0, m.Y0) // TODO: cache

	strokeUnsupported := false
	// if m.IsSimilarity() { // TODO: implement
	if true {
		scale := math32.Sqrt(math32.Abs(m.Det()))
		// TODO: this is a hack to get the effect of [ppath.VectorEffectNonScalingStroke]
		style.Stroke.Width.Dots /= scale
		// style.Stroke.DashOffset, style.Stroke.Dashes = ppath.ScaleDash(style.Stroke.Width.Dots, style.Stroke.DashOffset, style.Stroke.Dashes)
	} else {
		strokeUnsupported = true
	}

	if style.HasFill() || (style.HasStroke() && !strokeUnsupported) {
		rs.writePath(pt)
	}

	if style.HasFill() {
		if style.Fill.Color != rs.style.Fill.Color {
			rs.ctx.Set("fillStyle", rs.imageToStyle(style.Fill.Color))
			rs.style.Fill.Color = style.Fill.Color
		}
		rs.ctx.Call("fill")
	}
	if style.HasStroke() && !strokeUnsupported {
		if style.Stroke.Cap != rs.style.Stroke.Cap {
			rs.ctx.Set("lineCap", style.Stroke.Cap.String())
			rs.style.Stroke.Cap = style.Stroke.Cap
		}

		if style.Stroke.Join != rs.style.Stroke.Join {
			rs.ctx.Set("lineJoin", style.Stroke.Join.String())
			if style.Stroke.Join == ppath.JoinMiter && !math32.IsNaN(style.Stroke.MiterLimit) {
				rs.ctx.Set("miterLimit", style.Stroke.MiterLimit)
			}
			rs.style.Stroke.Join = style.Stroke.Join
		}

		// TODO: all of this could be more efficient
		dashesEqual := len(style.Stroke.Dashes) == len(rs.style.Stroke.Dashes)
		if dashesEqual {
			for i, dash := range style.Stroke.Dashes {
				if dash != rs.style.Stroke.Dashes[i] {
					dashesEqual = false
					break
				}
			}
		}

		if !dashesEqual {
			dashes := []any{}
			for _, dash := range style.Stroke.Dashes {
				dashes = append(dashes, dash)
			}
			jsDashes := js.Global().Get("Array").New(dashes...)
			rs.ctx.Call("setLineDash", jsDashes)
			rs.style.Stroke.Dashes = style.Stroke.Dashes
		}

		if style.Stroke.DashOffset != rs.style.Stroke.DashOffset {
			rs.ctx.Set("lineDashOffset", style.Stroke.DashOffset)
			rs.style.Stroke.DashOffset = style.Stroke.DashOffset
		}

		if style.Stroke.Width.Dots != rs.style.Stroke.Width.Dots {
			rs.ctx.Set("lineWidth", style.Stroke.Width.Dots)
			rs.style.Stroke.Width = style.Stroke.Width
		}
		if style.Stroke.Color != rs.style.Stroke.Color {
			rs.ctx.Set("strokeStyle", rs.imageToStyle(style.Stroke.Color))
			rs.style.Stroke.Color = style.Stroke.Color
		}
		rs.ctx.Call("stroke")
	} else if style.HasStroke() {
		// stroke settings unsupported by HTML Canvas, draw stroke explicitly
		// TODO: check when this is happening, maybe remove or use rasterx?
		if len(style.Stroke.Dashes) > 0 {
			pt.Path = pt.Path.Dash(style.Stroke.DashOffset, style.Stroke.Dashes...)
		}
		pt.Path = pt.Path.Stroke(style.Stroke.Width.Dots, ppath.CapFromStyle(style.Stroke.Cap), ppath.JoinFromStyle(style.Stroke.Join), 1)
		rs.writePath(pt)
		if style.Stroke.Color != rs.style.Fill.Color {
			rs.ctx.Set("fillStyle", rs.imageToStyle(style.Stroke.Color))
			rs.style.Fill.Color = style.Stroke.Color
		}
		rs.ctx.Call("fill")
	}
	rs.ctx.Call("restore") // restore clip region
}

func jsAwait(v js.Value) (result js.Value, ok bool) { // TODO: use wgpu version
	// COPIED FROM https://go-review.googlesource.com/c/go/+/150917/
	if v.Type() != js.TypeObject || v.Get("then").Type() != js.TypeFunction {
		return v, true
	}

	done := make(chan struct{})

	onResolve := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		result = args[0]
		ok = true
		close(done)
		return nil
	})
	defer onResolve.Release()

	onReject := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		result = args[0]
		ok = false
		close(done)
		return nil
	})
	defer onReject.Release()

	v.Call("then", onResolve, onReject)
	<-done
	return
}

func (rs *Renderer) RenderImage(pimg *pimage.Params) {
	if pimg.Source == nil {
		return
	}
	// TODO: for some reason we are getting a non-nil interface of a nil [image.RGBA]
	if r, ok := pimg.Source.(*image.RGBA); ok && r == nil {
		return
	}

	// Fast path for [image.Uniform]
	if u, ok := pimg.Source.(*image.Uniform); ok && pimg.Mask == nil {
		// TODO: caching?
		rs.style.Fill.Color = u
		rs.ctx.Set("fillStyle", rs.imageToStyle(u))
		rs.ctx.Call("fillRect", pimg.Rect.Min.X, pimg.Rect.Min.Y, pimg.Rect.Dx(), pimg.Rect.Dy())
		return
	}

	// TODO: images possibly comparatively not performant on web, so there
	// might be a better path for things like FillBox.
	// TODO: have a fast path for [image.RGBA]?
	// size := pimg.Rect.Size() // TODO: is this right?
	// TODO: clean this up
	jsBuf := wgpu.BytesToJS(pimg.Source.(*image.RGBA).Pix)
	sbb := pimg.Source.Bounds()
	imageData := js.Global().Get("ImageData").New(jsBuf, sbb.Dx(), sbb.Dy())
	imageBitmapPromise := js.Global().Call("createImageBitmap", imageData)
	imageBitmap, ok := jsAwait(imageBitmapPromise)
	if !ok {
		panic("error while waiting for createImageBitmap promise")
	}

	sw := min(pimg.Rect.Dx(), sbb.Dx())
	sh := min(pimg.Rect.Dy(), sbb.Dy())
	// origin := m.Dot(canvas.Point{0, float64(img.Bounds().Size().Y)}).Mul(rs.dpm)
	// m = m.Scale(rs.dpm, rs.dpm)
	// rs.ctx.Call("setTransform", m[0][0], m[0][1], m[1][0], m[1][1], origin.X, rs.height-origin.Y)
	rs.ctx.Call("drawImage", imageBitmap, pimg.SourcePos.X, pimg.SourcePos.Y, sw, sh, pimg.Rect.Min.X, pimg.Rect.Min.Y, sw, sh)
	// rs.ctx.Call("putImageData", imageData, pimg.Rect.Min.X, pimg.Rect.Min.Y)
	// rs.ctx.Call("setTransform", 1.0, 0.0, 0.0, 1.0, 0.0, 0.0)
}
