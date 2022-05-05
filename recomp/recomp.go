// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package recomp

import (
	"embed"
	"image"
	"image/color"
	"image/draw"
	"log"
	"unsafe"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/gpu"
	"github.com/goki/mat32"
	"github.com/goki/vgpu/vgpu"
)

//go:embed shaders/*.spv
var content embed.FS

// Mats are the projection matricies
type Mats struct {
	MVP    mat32.Mat3
	Align1 mat32.Vec3
	Align2 mat32.Vec4
	UVP    mat32.Mat3
}

// Config configures the rect composition System with a
// comp and fill pipeline.
func Config(sy *vgpu.System) {
	cpl := sy.NewPipeline("comp")
	cb, err := content.ReadFile("shaders/comp_vert.spv")
	cpl.AddShaderCode("comp_vert", vgpu.VertexShader, cb)
	cb, err = content.ReadFile("shaders/comp_frag.spv")
	cpl.AddShaderCode("comp_frag", vgpu.FragmentShader, cb)

	fpl := sy.NewPipeline("fill")
	cb, err := content.ReadFile("shaders/fill_vert.spv")
	fpl.AddShaderCode("fill_vert", vgpu.VertexShader, cb)
	cb, err = content.ReadFile("shaders/comp_frag.spv")
	fpl.AddShaderCode("fill_frag", vgpu.FragmentShader, cb)

	posv := sy.Vars.Add("Pos", vgpu.Float32Vec2, vgpu.Vertex, 0, vgpu.VertexShader)
	// txcv := sy.Vars.Add("TexCoord", vgpu.Float32Vec2, vgpu.Vertex, 0, vgpu.VertexShader)
	idxv := sy.Vars.Add("Index", vgpu.Uint16, vgpu.Index, 0, vgpu.VertexShader)

	matv := sy.Vars.Add("Mats", vgpu.Struct, vgpu.Uniform, 0, vgpu.VertexShader)
	matv.SizeOf = vgpu.Float32Mat4.Bytes() * 2 // no padding for these

	tximgv := sy.Vars.Add("Tex", vgpu.ImageRGBA32, vgpu.TextureRole, 0, vgpu.FragmentShader)
	clrv := sy.Vars.Add("Color", vgpu.Float32Vec4, vgpu.Uniform, 0, vgpu.FragmentShader)

	nPts := 4
	nIdxs := 6
	sqrPos := sy.Mem.Vals.Add("SqrPos", posv, nPts)
	sqrIdx := sy.Mem.Vals.Add("SqrIdx", idxv, nIdxs)
	sqrPos.Indexes = "SqrIdx" // only need to set indexes for one vertex val

	mat := sy.Mem.Vals.Add("Mats", matv, 1)

	img := sy.Mem.Vals.Add("Tex", tximgv, 1)
	clr := sy.Mem.Vals.Add("FillColor", clrv, 1)

	// note: add all values per above before doing Config
	sy.Config()
	sy.Mem.Config()

	// note: first val in set is offset
	sqrPosA := sqrPos.Floats32()
	sqrPosA.Set(0,
		0.0, 0.0,
		1.0, 0.0,
		1.0, 1.0,
		0.0, 1.0)
	sqrPos.Mod = true

	idxs := []uint16{0, 1, 2, 0, 2, 3}
	sqrIdx.CopyBytes(unsafe.Pointer(&idxs[0]))

	// cam.CopyBytes(unsafe.Pointer(&camo)) // sets mod

	sy.Mem.SyncToGPU()

	sy.SetVals(0, "SqrPos", "Camera", "TexImage")

	if sy.Vars.Validate() != nil {
		destroy()
		return
	}

	// cam.CopyBytes(unsafe.Pointer(&camo)) // sets mod
	// sy.Mem.SyncToGPU()

	idx := sf.AcquireNextImage()
	pl.FullStdRender(pl.CmdPool.Buff, sf.Frames[idx])
	sf.SubmitRender(pl.CmdPool.Buff) // this is where it waits for the 16 msec
	sf.PresentImage(idx)

}

// draw draws to current render target (could be window or framebuffer / texture)
// proper context must have already been established outside this call!
// dstBotZero is true if destination has Y=0 at bottom
func Composite(dstSz image.Point, src2dst mat32.Mat3, tx image.Image, sr image.Rectangle) {
	// tx := src.(*textureImpl)
	sr = sr.Intersect(tx.Bounds())
	if sr.Empty() {
		return
	}

	// srcBotZero := src.BotZero()
	// if opts != nil && opts.FlipY {
	// 	srcBotZero = !srcBotZero
	// }

	// gpu.Draw.Op(op)
	// gpu.Draw.DepthTest(false)
	// gpu.Draw.CullFace(false, true, dstBotZero) // cull back face -- dstBotZero = CCW, !dstBotZero = CW
	// gpu.Draw.StencilTest(false)
	// gpu.Draw.Multisample(false)
	// app.drawProg.Activate()

	// Start with src-space left, top, right and bottom.
	srcL := float32(sr.Min.X)
	srcT := float32(sr.Min.Y)
	srcR := float32(sr.Max.X)
	srcB := float32(sr.Max.Y)

	// Transform to dst-space via the src2dst matrix, then to a MVP matrix.
	matMVP := calcMVP(dstSz.X, dstSz.Y,
		src2dst[0]*srcL+src2dst[3]*srcT+src2dst[6],
		src2dst[1]*srcL+src2dst[4]*srcT+src2dst[7],
		src2dst[0]*srcR+src2dst[3]*srcT+src2dst[6],
		src2dst[1]*srcR+src2dst[4]*srcT+src2dst[7],
		src2dst[0]*srcL+src2dst[3]*srcB+src2dst[6],
		src2dst[1]*srcL+src2dst[4]*srcB+src2dst[7],
		dstBotZero,
	)
	// fmt.Printf("trgTex: %v  matMVP: %v\n", trgTex, matMVP)

	var tmat Mats

	mat := sy.Mem.Vals["Mats"]
	mat.CopyBytes(unsafe.Pointer(&matMVP)) // sets mod

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
	tw := float32(tx.size.X)
	th := float32(tx.size.Y)
	px := float32(sr.Min.X-0) / tw
	py := float32(sr.Min.Y-0) / th
	qx := float32(sr.Max.X-0) / tw
	sy := float32(sr.Max.Y-0) / th
	// Due to axis alignment, qy = py and sx = px.
	//
	// The simultaneous equations are:
	//	  0 +   0 + a02 = px
	//	  0 +   0 + a12 = py
	//	a00 +   0 + a02 = qx
	//	a10 +   0 + a12 = qy = py
	//	  0 + a01 + a02 = sx = px
	//	  0 + a11 + a12 = sy

	var matUVP mat32.Mat3
	if srcBotZero {
		matUVP = mat32.Mat3{
			qx - px, 0, 0,
			0, py - sy, 0, // py - sy
			px, sy, 1}
	} else {
		matUVP = mat32.Mat3{
			qx - px, 0, 0,
			0, sy - py, 0, // sy - py
			px, py, 1,
		}
	}
	err = app.drawProg.UniformByName("uvp").SetValue(matUVP)
	if err != nil {
		return
	}
	// fmt.Printf("matUVP: %v\n", matUVP)

	tx.Activate(0)
	err = app.drawProg.UniformByName("tex").SetValue(int32(0))
	if err != nil {
		log.Println(err)
	}

	qbuff.Activate()
	gpu.Draw.TriangleStrips(0, 4)
}

// fill fills to current render target (could be window or framebuffer / texture)
// proper context must have already been established outside this call!
// dstBotZero is true if flipping Y axis
func (app *appImpl) fill(mvp mat32.Mat3, src color.Color, op draw.Op, qbuff gpu.BufferMgr, dstBotZero bool) {
	gpu.Draw.Op(op)
	gpu.Draw.CullFace(false, true, dstBotZero) // dstBotZero = CCW, else CW
	gpu.Draw.DepthTest(false)
	gpu.Draw.StencilTest(false)
	gpu.Draw.Multisample(false)
	app.fillProg.Activate()

	app.fillProg.UniformByName("mvp").SetValue(mvp)

	r, g, b, a := src.RGBA()

	clvec4 := mat32.NewVec4(
		float32(r)/65535,
		float32(g)/65535,
		float32(b)/65535,
		float32(a)/65535,
	)

	app.fillProg.UniformByName("color").SetValue(clvec4)

	qbuff.Activate()
	gpu.Draw.TriangleStrips(0, 4)
}

// fillRect fills given rectangle, where dstSz is overall size of the destination (e.g., window)
// dstBotZero is true if destination has Y=0 at bottom
func (app *appImpl) fillRect(dstSz image.Point, dr image.Rectangle, src color.Color, op draw.Op, qbuff gpu.BufferMgr, dstBotZero bool) {
	minX := float32(dr.Min.X)
	minY := float32(dr.Min.Y)
	maxX := float32(dr.Max.X)
	maxY := float32(dr.Max.Y)

	mvp := calcMVP(dstSz.X, dstSz.Y,
		minX, minY,
		maxX, minY,
		minX, maxY, dstBotZero,
	)
	app.fill(mvp, src, op, qbuff, dstBotZero)
}

// drawUniform does a fill-like uniform color fill but with an arbitrary src2dst transform
// dstBotZero is true if destination has Y=0 at bottom
func (app *appImpl) drawUniform(dstSz image.Point, src2dst mat32.Mat3, src color.Color, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions, qbuff gpu.BufferMgr, dstBotZero bool) {
	minX := float32(sr.Min.X)
	minY := float32(sr.Min.Y)
	maxX := float32(sr.Max.X)
	maxY := float32(sr.Max.Y)

	// Transform to dst-space via the src2dst matrix, then to a MVP matrix.
	mvp := calcMVP(dstSz.X, dstSz.Y,
		src2dst[0]*minX+src2dst[3]*minY+src2dst[6],
		src2dst[1]*minX+src2dst[4]*minY+src2dst[7],
		src2dst[0]*maxX+src2dst[3]*minY+src2dst[6],
		src2dst[1]*maxX+src2dst[4]*minY+src2dst[7],
		src2dst[0]*minX+src2dst[3]*maxY+src2dst[6],
		src2dst[1]*minX+src2dst[4]*maxY+src2dst[7],
		dstBotZero,
	)
	app.fill(mvp, src, op, qbuff, dstBotZero)
}

// calcMVP returns the Model View Projection matrix that maps the quadCoords
// unit square, (0, 0) to (1, 1), to a quad QV, such that QV in vertex shader
// space corresponds to the quad QP in pixel space, where QP is defined by
// three of its four corners - the arguments to this function. The three
// corners are nominally the top-left, top-right and bottom-left, but there is
// no constraint that e.g. tlx < trx.
//
// In pixel space, the window ranges from (0, 0) to (widthPx, heightPx). The
// Y-axis points downwards (unless flipped).
//
// In vertex shader space, the window ranges from (-1, +1) to (+1, -1), which
// is a 2-unit by 2-unit square. The Y-axis points upwards.
//
// if dstBotZero is true, then the y=0 is at bottom in dest, else top
//
func calcMVP(widthPx, heightPx int, tlx, tly, trx, try, blx, bly float32, dstBotZero bool) mat32.Mat3 {
	// Convert from pixel coords to vertex shader coords.
	invHalfWidth := 2 / float32(widthPx)
	invHalfHeight := 2 / float32(heightPx)
	if dstBotZero {
		tlx = tlx*invHalfWidth - 1
		tly = 1 - tly*invHalfHeight // 1 - min
		trx = trx*invHalfWidth - 1
		try = 1 - try*invHalfHeight // 1 - min
		blx = blx*invHalfWidth - 1
		bly = 1 - bly*invHalfHeight // 1 - (min + max)
	} else {
		tlx = tlx*invHalfWidth - 1
		tly = tly*invHalfHeight - 1
		trx = trx*invHalfWidth - 1
		try = try*invHalfHeight - 1
		blx = blx*invHalfWidth - 1
		bly = bly*invHalfHeight - 1
	}

	// The resultant affine matrix:
	//	- maps (0, 0) to (tlx, tly).
	//	- maps (1, 0) to (trx, try).
	//	- maps (0, 1) to (blx, bly).
	return mat32.Mat3{
		trx - tlx, try - tly, 0,
		blx - tlx, bly - tly, 0,
		tlx, tly, 1,
	}
}

// Note: arranged in CCW order for dstBotZero = true
// if !dstBotZero then need to reverse culling!
var quadCoords = mat32.ArrayF32{
	0, 0, // top left
	0, 1, // bottom left
	1, 0, // top right
	1, 1, // bottom right
}
