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
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/pimage"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles/units"
)

// Renderer is an HTML canvas renderer.
type Renderer struct {
	canvas js.Value
	ctx    js.Value
	size   math32.Vector2
}

// New returns an HTMLCanvas renderer.
func New(size math32.Vector2) render.Renderer {
	rs := &Renderer{}
	rs.canvas = js.Global().Get("document").Call("getElementById", "app")
	rs.ctx = rs.canvas.Call("getContext", "2d")
	rs.SetSize(units.UnitDot, size)
	return rs
}

func (rs *Renderer) IsImage() bool      { return true }
func (rs *Renderer) Image() *image.RGBA { return nil } // TODO
func (rs *Renderer) Code() []byte       { return nil }

func (rs *Renderer) Size() (units.Units, math32.Vector2) {
	return units.UnitDot, rs.size // TODO: is Dot right?
}

func (rs *Renderer) SetSize(un units.Units, size math32.Vector2) {
	if rs.size == size {
		return
	}
	rs.size = size

	rs.canvas.Set("width", size.X)
	rs.canvas.Set("height", size.Y)

	// rs.ctx.Call("clearRect", 0, 0, size.X, size.Y)
	// rs.ctx.Set("imageSmoothingEnabled", true)
	// rs.ctx.Set("imageSmoothingQuality", "high")
}

// Render is the main rendering function.
func (rs *Renderer) Render(r render.Render) {
	for _, ri := range r {
		switch x := ri.(type) {
		case *render.Path:
			rs.RenderPath(x)
		case *pimage.Params:
			// x.Render(rs.image) TODO
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
			rs.ctx.Call("moveTo", end.X, rs.size.Y-end.Y)
		case ppath.LineTo:
			rs.ctx.Call("lineTo", end.X, rs.size.Y-end.Y)
		case ppath.QuadTo:
			cp := scanner.CP1()
			rs.ctx.Call("quadraticCurveTo", cp.X, rs.size.Y-cp.Y, end.X, rs.size.Y-end.Y)
		case ppath.CubeTo:
			cp1, cp2 := scanner.CP1(), scanner.CP2()
			rs.ctx.Call("bezierCurveTo", cp1.X, rs.size.Y-cp1.Y, cp2.X, rs.size.Y-cp2.Y, end.X, rs.size.Y-end.Y)
		case ppath.Close:
			rs.ctx.Call("closePath")
		}
	}
}

/*
func (rs *Renderer) toStyle(paint canvas.Paint) any {
	if paint.IsPattern() {
		// TODO
	} else if paint.IsGradient() {
		if g, ok := paint.Gradient.(*canvas.LinearGradient); ok {
			grad := rs.ctx.Call("createLinearGradient", g.Start.X, rs.size.Y-g.Start.Y, g.End.X, rs.size.Y-g.End.Y)
			for _, stop := range g.Stops {
				grad.Call("addColorStop", stop.Offset, canvas.CSSColor(stop.Color).String())
			}
			return grad
		} else if g, ok := paint.Gradient.(*canvas.RadialGradient); ok {
			grad := rs.ctx.Call("createRadialGradient", g.C0.X, rs.size.Y-g.C0.Y, g.R0, g.C1.X, rs.size.Y-g.C1.Y, g.R1)
			for _, stop := range g.Stops {
				grad.Call("addColorStop", stop.Offset, canvas.CSSColor(stop.Color).String())
			}
			return grad
		}
	}
	return canvas.CSSColor(paint.Color).String()
}
*/

func (rs *Renderer) RenderPath(pt *render.Path) {
	if pt.Path.Empty() {
		return
	}
	rs.ctx.Set("strokeStyle", colors.AsHex(colors.ToUniform(pt.Context.Style.Stroke.Color))) // TODO: remove
	rs.writePath(pt)
	rs.ctx.Call("stroke") // TODO: remove

	// style := &pt.Context.Style

	// strokeUnsupported := false
	// if m.IsSimilarity() {
	// 	scale := math.Sqrt(math.Abs(m.Det()))
	// 	style.StrokeWidth *= scale
	// 	style.DashOffset, style.Dashes = canvas.ScaleDash(style.StrokeWidth, style.DashOffset, style.Dashes)
	// } else {
	// 	strokeUnsupported = true
	// }

	/*
		if style.HasFill() || style.HasStroke() && !strokeUnsupported {
			rs.writePath(pt.Copy().Transform(m).ReplaceArcs())
		}

		if style.HasFill() {
			if !style.Fill.Equal(rs.style.Fill) {
				rs.ctx.Set("fillStyle", rs.toStyle(style.Fill))
				rs.style.Fill = style.Fill
			}
			rs.ctx.Call("fill")
		}
		if style.HasStroke() && !strokeUnsupported {
			if style.StrokeCapper != rs.style.StrokeCapper {
				if _, ok := style.StrokeCapper.(canvas.RoundCapper); ok {
					rs.ctx.Set("lineCap", "round")
				} else if _, ok := style.StrokeCapper.(canvas.SquareCapper); ok {
					rs.ctx.Set("lineCap", "square")
				} else if _, ok := style.StrokeCapper.(canvas.ButtCapper); ok {
					rs.ctx.Set("lineCap", "butt")
				} else {
					panic("HTML Canvas: line cap not support")
				}
				rs.style.StrokeCapper = style.StrokeCapper
			}

			if style.StrokeJoiner != rs.style.StrokeJoiner {
				if _, ok := style.StrokeJoiner.(canvas.BevelJoiner); ok {
					rs.ctx.Set("lineJoin", "bevel")
				} else if _, ok := style.StrokeJoiner.(canvas.RoundJoiner); ok {
					rs.ctx.Set("lineJoin", "round")
				} else if miter, ok := style.StrokeJoiner.(canvas.MiterJoiner); ok && !math.IsNaN(miter.Limit) && miter.GapJoiner == canvas.BevelJoin {
					rs.ctx.Set("lineJoin", "miter")
					rs.ctx.Set("miterLimit", miter.Limit)
				} else {
					panic("HTML Canvas: line join not support")
				}
				rs.style.StrokeJoiner = style.StrokeJoiner
			}

			dashesEqual := len(style.Dashes) == len(rs.style.Dashes)
			if dashesEqual {
				for i, dash := range style.Dashes {
					if dash != rs.style.Dashes[i] {
						dashesEqual = false
						break
					}
				}
			}

			if !dashesEqual {
				dashes := []interface{}{}
				for _, dash := range style.Dashes {
					dashes = append(dashes, dash)
				}
				jsDashes := js.Global().Get("Array").New(dashes...)
				rs.ctx.Call("setLineDash", jsDashes)
				rs.style.Dashes = style.Dashes
			}

			if style.DashOffset != rs.style.DashOffset {
				rs.ctx.Set("lineDashOffset", style.DashOffset)
				rs.style.DashOffset = style.DashOffset
			}

			if style.StrokeWidth != rs.style.StrokeWidth {
				rs.ctx.Set("lineWidth", style.StrokeWidth)
				rs.style.StrokeWidth = style.StrokeWidth
			}
			//if !style.Stroke.Equal(r.style.Stroke) {
			rs.ctx.Set("strokeStyle", rs.toStyle(style.Stroke))
			rs.style.Stroke = style.Stroke
			//}
			rs.ctx.Call("stroke")
		} else if style.HasStroke() {
			// stroke settings unsupported by HTML Canvas, draw stroke explicitly
			if style.IsDashed() {
				pt = pt.Dash(style.DashOffset, style.Dashes...)
			}
			pt = pt.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner, canvas.Tolerance)
			rs.writePath(pt.Transform(m).ReplaceArcs())
			if !style.Stroke.Equal(rs.style.Fill) {
				rs.ctx.Set("fillStyle", rs.toStyle(style.Stroke))
				rs.style.Fill = style.Stroke
			}
			rs.ctx.Call("fill")
		}
	*/
}

func (rs *Renderer) RenderText(text *render.Text) {
	// text.RenderAsPath(r, m, canvas.DefaultResolution)
}

/*
func jsAwait(v js.Value) (result js.Value, ok bool) {
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

// RenderImage renders an image to the canvas using a transformation matrix.
func (r *HTMLCanvas) RenderImage(img image.Image, m canvas.Matrix) {
	size := img.Bounds().Size()
	sp := img.Bounds().Min // starting point
	buf := make([]byte, 4*size.X*size.Y)
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			i := (y*size.X + x) * 4
			r, g, b, a := img.At(sp.X+x, sp.Y+y).RGBA()
			alpha := float64(a>>8) / 256.0
			buf[i+0] = byte(float64(r>>8) / alpha)
			buf[i+1] = byte(float64(g>>8) / alpha)
			buf[i+2] = byte(float64(b>>8) / alpha)
			buf[i+3] = byte(a >> 8)
		}
	}
	jsBuf := js.Global().Get("Uint8Array").New(len(buf))
	js.CopyBytesToJS(jsBuf, buf)
	jsBufClamped := js.Global().Get("Uint8ClampedArray").New(jsBuf)
	imageData := js.Global().Get("ImageData").New(jsBufClamped, size.X, size.Y)
	imageBitmapPromise := js.Global().Call("createImageBitmap", imageData)
	imageBitmap, ok := jsAwait(imageBitmapPromise)
	if !ok {
		panic("error while waiting for createImageBitmap promise")
	}

	origin := m.Dot(canvas.Point{0, float64(img.Bounds().Size().Y)}).Mul(r.dpm)
	m = m.Scale(r.dpm, r.dpm)
	r.ctx.Call("setTransform", m[0][0], m[0][1], m[1][0], m[1][1], origin.X, r.height-origin.Y)
	r.ctx.Call("drawImage", imageBitmap, 0, 0)
	r.ctx.Call("setTransform", 1.0, 0.0, 0.0, 1.0, 0.0, 0.0)
}
*/
