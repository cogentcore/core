// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build 3d

package glos

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/gpu"
	"golang.org/x/image/math/f64"
)

func (app *appImpl) initDrawProgs() error {
	if app.progInit {
		return nil
	}
	p := theGPU.NewProgram("draw")
	_, err := p.AddShader(gpu.VertexShader, "draw-vert", drawVertSrc)
	if err != nil {
		return err
	}
	_, err = p.AddShader(gpu.FragmentShader, "draw-frag", drawFragSrc)
	if err != nil {
		return err
	}
	p.AddUniform("mvp", gpu.UniType{Type: gpu.Float32, Mat: 3}, false, 0)
	p.AddUniform("uvp", gpu.UniType{Type: gpu.Float32, Mat: 3}, false, 0)
	p.AddUniform("sample", gpu.UniType{Type: gpu.Int}, false, 0)

	pv := p.AddInput("pos", gpu.VectorType{Type: gpu.Float32, Vec: 2}, gpu.VertexPosition)

	p.SetFragDataVar("outputColor")

	err = p.Compile()
	if err != nil {
		return err
	}
	app.drawProg = p

	b := theGPU.NewBufferMgr()
	vb := b.AddVectorsBuffer(gpu.StaticDraw)
	vb.AddVectors(pv, false)
	vb.SetLen(len(quadCoords))
	vb.SetAllData(quadCoords)
	vb.Activate()
	app.drawQuads = b

	p = theGPU.NewProgram("fill")
	_, err = p.AddShader(gpu.VertexShader, "fill-vert", fillVertSrc)
	if err != nil {
		return err
	}
	_, err = p.AddShader(gpu.FragmentShader, "fill-frag", fillFragSrc)
	if err != nil {
		return err
	}
	p.AddUniform("mvp", gpu.UniType{Type: gpu.Float32, Mat: 3}, false, 0)
	p.AddUniform("color", gpu.UniType{Type: gpu.Float32, Vec: 4}, false, 0)

	p.AddInput("pos", gpu.VectorType{Type: gpu.Float32, Vec: 2}, gpu.VertexPosition)

	p.SetFragDataVar("outputColor")

	err = p.Compile()
	if err != nil {
		return err
	}
	app.fillProg = p

	b = theGPU.NewBufferMgr()
	vb = b.AddVectorsBuffer(gpu.StaticDraw)
	vb.AddVectors(pv, false)
	vb.SetLen(len(quadCoords))
	vb.SetAllData(quadCoords)
	vb.Activate()
	app.fillQuads = b

	err = gpu.TheGPU.ErrCheck("initDrawProgs")
	if err != nil {
		return err
	}
	app.progInit = true
	return nil
}

func (w *windowImpl) draw(src2dst f64.Aff3, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {

	t := src.(*textureImpl)
	sr = sr.Intersect(t.Bounds())
	if sr.Empty() {
		return
	}

	theGPU.UseContext(w)
	defer theGPU.ClearContext(w)

	gpu.Draw.Op(op)
	theApp.drawProg.Activate()

	// todo: convert over to mat32 math..

	// Start with src-space left, top, right and bottom.
	srcL := float64(sr.Min.X)
	srcT := float64(sr.Min.Y)
	srcR := float64(sr.Max.X)
	srcB := float64(sr.Max.Y)
	// Transform to dst-space via the src2dst matrix, then to a MVP matrix.
	writeAff3(w.app.texture.mvp, w.mvp(
		src2dst[0]*srcL+src2dst[1]*srcT+src2dst[2],
		src2dst[3]*srcL+src2dst[4]*srcT+src2dst[5],
		src2dst[0]*srcR+src2dst[1]*srcT+src2dst[2],
		src2dst[3]*srcR+src2dst[4]*srcT+src2dst[5],
		src2dst[0]*srcL+src2dst[1]*srcB+src2dst[2],
		src2dst[3]*srcL+src2dst[4]*srcB+src2dst[5],
	))

	// OpenGL's fragment shaders' UV coordinates run from (0,0)-(1,1),
	// unlike vertex shaders' XY coordinates running from (-1,+1)-(+1,-1).
	//
	// We are drawing a rectangle PQRS, defined by two of its
	// corners, onto the entire texture. The two quads may actually
	// be equal, but in the general case, PQRS can be smaller.
	//
	//	(0,0) +---------------+ (1,0)
	//	      |  P +-----+ Q  |
	//	      |    |     |    |
	//	      |  S +-----+ R  |
	//	(0,1) +---------------+ (1,1)
	//
	// The PQRS quad is always axis-aligned. First of all, convert
	// from pixel space to texture space.
	tw := float64(t.size.X)
	th := float64(t.size.Y)
	px := float64(sr.Min.X-0) / tw
	py := float64(sr.Min.Y-0) / th
	qx := float64(sr.Max.X-0) / tw
	sy := float64(sr.Max.Y-0) / th
	// Due to axis alignment, qy = py and sx = px.
	//
	// The simultaneous equations are:
	//	  0 +   0 + a02 = px
	//	  0 +   0 + a12 = py
	//	a00 +   0 + a02 = qx
	//	a10 +   0 + a12 = qy = py
	//	  0 + a01 + a02 = sx = px
	//	  0 + a11 + a12 = sy
	writeAff3(w.app.texture.uvp, f64.Aff3{
		qx - px, 0, px,
		0, sy - py, py,
	})

	// todo: need gpu.Texture2D here
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, t.id)
	gl.Uniform1i(w.app.texture.sample, 0)

	theApp.drawQuads.Activate()
	gpu.Draw.TriangleStrips(0, 4)
}

func (w *windowImpl) fill(mvp f64.Aff3, src color.Color, op draw.Op) {
	w.RunOnWin(func() {
		theGPU.UseContext(w)
		defer theGPU.ClearContext(w)

		doFill(w.app, mvp, src, op)
	})
}

func doFill(app *appImpl, mvp f64.Aff3, src color.Color, op draw.Op) {
	useOp(op)
	gl.UseProgram(app.fill.program)

	writeAff3(app.fill.mvp, mvp)

	r, g, b, a := src.RGBA()
	gl.Uniform4f(
		app.fill.color,
		float32(r)/65535,
		float32(g)/65535,
		float32(b)/65535,
		float32(a)/65535,
	)

	gl.BindBuffer(gl.ARRAY_BUFFER, app.fill.quad)
	gl.EnableVertexAttribArray(uint32(app.fill.pos))
	gl.VertexAttribPointer(uint32(app.fill.pos), 2, gl.FLOAT, false, 0, gl.PtrOffset(0))

	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	gl.DisableVertexAttribArray(uint32(app.fill.pos))
}

func (w *windowImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	minX := float64(dr.Min.X)
	minY := float64(dr.Min.Y)
	maxX := float64(dr.Max.X)
	maxY := float64(dr.Max.Y)
	w.fill(w.mvp(
		minX, minY,
		maxX, minY,
		minX, maxY,
	), src, op)
}

func (w *windowImpl) DrawUniform(src2dst f64.Aff3, src color.Color, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	minX := float64(sr.Min.X)
	minY := float64(sr.Min.Y)
	maxX := float64(sr.Max.X)
	maxY := float64(sr.Max.Y)
	w.fill(w.mvp(
		src2dst[0]*minX+src2dst[1]*minY+src2dst[2],
		src2dst[3]*minX+src2dst[4]*minY+src2dst[5],
		src2dst[0]*maxX+src2dst[1]*minY+src2dst[2],
		src2dst[3]*maxX+src2dst[4]*minY+src2dst[5],
		src2dst[0]*minX+src2dst[1]*maxY+src2dst[2],
		src2dst[3]*minX+src2dst[4]*maxY+src2dst[5],
	), src, op)
}

func writeAff3(u int32, a f64.Aff3) {
	var m [9]float32
	m[0*3+0] = float32(a[0*3+0])
	m[0*3+1] = float32(a[1*3+0])
	m[0*3+2] = 0
	m[1*3+0] = float32(a[0*3+1])
	m[1*3+1] = float32(a[1*3+1])
	m[1*3+2] = 0
	m[2*3+0] = float32(a[0*3+2])
	m[2*3+1] = float32(a[1*3+2])
	m[2*3+2] = 1
	gl.UniformMatrix3fv(u, 1, false, &m[0])
	gpu.TheGPU.ErrCheck("writeaff3")
}

var quadCoords = mat32.ArrayF32{
	0, 0, // top left
	1, 0, // top right
	0, 1, // bottom left
	1, 1, // bottom right
}

const drawVertSrc = `
#version 330

uniform mat3 mvp;
uniform mat3 uvp;

in vec2 pos;

out vec2 uv;

void main() {
	vec3 p = vec3(pos, 1);
	gl_Position = vec4(mvp * p, 1);
	uv = (uvp * vec3(pos, 1)).xy;
}
` + "\x00"

const drawFragSrc = `
#version 330

precision mediump float;

uniform sampler2D sample;

in vec2 uv;

out vec4 outputColor;

void main() {
	outputColor = texture(sample, uv);
}
` + "\x00"

const fillVertSrc = `
#version 330

uniform mat3 mvp;

in vec2 pos;

void main() {
	vec3 p = vec3(pos, 1);
	gl_Position = vec4(mvp * p, 1);
}
` + "\x00"

const fillFragSrc = `
#version 330

precision mediump float;

uniform vec4 color;

out vec4 outputColor;

void main() {
	outputColor = color;
}
` + "\x00"
