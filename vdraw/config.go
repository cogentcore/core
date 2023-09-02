// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdraw

import (
	"embed"
	"image"
	"image/draw"
	"unsafe"

	"github.com/goki/mat32"
	vk "github.com/goki/vulkan"
	"goki.dev/vgpu/v2/vgpu"
)

//go:embed shaders/*.spv
var content embed.FS

// Mtxs are the projection matricies
type Mtxs struct {
	MVP mat32.Mat4
	UVP mat32.Mat4
}

// DrawerImpl contains implementation state -- ignore..
type DrawerImpl struct {

	// surface index for current render process
	SurfIdx uint32 `desc:"surface index for current render process"`

	// maximum number of images per pass -- set by user at config
	MaxTextures int `desc:"maximum number of images per pass -- set by user at config"`

	// whether to render image with flipped Y
	FlipY bool `desc:"whether to render image with flipped Y"`

	// last draw operation used -- used for switching pipeline
	LastOp draw.Op `desc:"last draw operation used -- used for switching pipeline"`
}

// ConfigPipeline configures graphics settings on the pipeline
func (dw *Drawer) ConfigPipeline(pl *vgpu.Pipeline) {
	pl.SetGraphicsDefaults()
	if dw.YIsDown {
		pl.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0)
	} else {
		pl.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceClockwise, 1.0)
	}
}

// ConfigSys configures the vDraw System and pipelines.
func (dw *Drawer) ConfigSys() {
	sy := &dw.Sys

	// note: requires different pipelines for src vs. over draw op modes
	dpl := sy.NewPipeline("draw_src")
	dw.ConfigPipeline(dpl)
	dpl.SetColorBlend(false)

	cb, _ := content.ReadFile("shaders/draw_vert.spv")
	dpl.AddShaderCode("draw_vert", vgpu.VertexShader, cb)
	cb, _ = content.ReadFile("shaders/draw_frag.spv")
	dpl.AddShaderCode("draw_frag", vgpu.FragmentShader, cb)

	dpl = sy.NewPipeline("draw_over")
	dw.ConfigPipeline(dpl)
	dpl.SetColorBlend(true) // default

	cb, _ = content.ReadFile("shaders/draw_vert.spv")
	dpl.AddShaderCode("draw_vert", vgpu.VertexShader, cb)
	cb, _ = content.ReadFile("shaders/draw_frag.spv")
	dpl.AddShaderCode("draw_frag", vgpu.FragmentShader, cb)

	fpl := sy.NewPipeline("fill")
	dw.ConfigPipeline(fpl)

	cb, _ = content.ReadFile("shaders/fill_vert.spv")
	fpl.AddShaderCode("fill_vert", vgpu.VertexShader, cb)
	cb, _ = content.ReadFile("shaders/fill_frag.spv")
	fpl.AddShaderCode("fill_frag", vgpu.FragmentShader, cb)

	vars := sy.Vars()
	vars.NDescs = vgpu.NDescForTextures(dw.Impl.MaxTextures)
	vset := vars.AddVertexSet()
	pcset := vars.AddPushSet()
	txset := vars.AddSet() // 0

	nPts := 4
	nIdxs := 6

	posv := vset.Add("Pos", vgpu.Float32Vec2, nPts, vgpu.Vertex, vgpu.VertexShader)
	idxv := vset.Add("Index", vgpu.Uint16, nIdxs, vgpu.Index, vgpu.VertexShader)

	pcset.AddStruct("Mtxs", vgpu.Float32Mat4.Bytes()*2, 1, vgpu.Push, vgpu.VertexShader, vgpu.FragmentShader)
	// note: packing texidx into mvp[0][3] to fit within 128 byte limit

	tximgv := txset.Add("Tex", vgpu.ImageRGBA32, 1, vgpu.TextureRole, vgpu.FragmentShader)
	tximgv.TextureOwns = true

	vset.ConfigVals(1)
	txset.ConfigVals(dw.Impl.MaxTextures)

	// note: add all values per above before doing Config
	sy.Config()

	// note: first val in set is offset
	rectPos, _ := posv.Vals.ValByIdxTry(0)
	rectPosA := rectPos.Floats32()
	rectPosA.Set(0,
		0.0, 0.0,
		0.0, 1.0,
		1.0, 0.0,
		1.0, 1.0)
	rectPos.SetMod()

	rectIdx, _ := idxv.Vals.ValByIdxTry(0)
	idxs := []uint16{0, 1, 2, 2, 1, 3} // triangle strip order
	rectIdx.CopyFromBytes(unsafe.Pointer(&idxs[0]))

	sy.Mem.SyncToGPU()

	vars.BindVertexValIdx("Pos", 0)
	vars.BindVertexValIdx("Index", 0)
}

// ConfigMtxs configures the draw matrix for given draw parameters:
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling), txsz is the size of the texture to draw,
// sr is the source region (set to tex.Format.Bounds() for all)
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
// flipY inverts the Y axis of the source image.
func (dw *Drawer) ConfigMtxs(src2dst mat32.Mat3, txsz image.Point, sr image.Rectangle, op draw.Op, flipY bool) *Mtxs {
	var tmat Mtxs

	if dw.YIsDown {
		flipY = !flipY
	}

	sr = sr.Intersect(image.Rectangle{Max: txsz})
	if sr.Empty() {
		tmat.MVP.SetIdentity()
		tmat.UVP.SetIdentity()
		return &tmat
	}

	destSz := dw.DestSize()

	// Start with src-space left, top, right and bottom.
	srcL := float32(sr.Min.X)
	srcT := float32(sr.Min.Y)
	srcR := float32(sr.Max.X)
	srcB := float32(sr.Max.Y)

	// Transform to dst-space via the src2dst matrix, then to a MVP matrix.
	matMVP := calcMVP(destSz.X, destSz.Y,
		src2dst[0]*srcL+src2dst[3]*srcT+src2dst[6],
		src2dst[1]*srcL+src2dst[4]*srcT+src2dst[7],
		src2dst[0]*srcR+src2dst[3]*srcT+src2dst[6],
		src2dst[1]*srcR+src2dst[4]*srcT+src2dst[7],
		src2dst[0]*srcL+src2dst[3]*srcB+src2dst[6],
		src2dst[1]*srcL+src2dst[4]*srcB+src2dst[7],
		dw.YIsDown,
	)
	tmat.MVP.SetFromMat3(&matMVP) // todo render direct

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
	tw := float32(txsz.X)
	th := float32(txsz.Y)
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

	if flipY { // note: reversed from openGL for vulkan
		tmat.UVP.SetFromMat3(&mat32.Mat3{
			qx - px, 0, 0,
			0, sy - py, 0, // sy - py
			px, py, 1})
	} else {
		tmat.UVP.SetFromMat3(&mat32.Mat3{
			qx - px, 0, 0,
			0, py - sy, 0, // py - sy
			px, sy, 1})
	}

	return &tmat
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
// if yisdown is true, then the y=0 is at top in dest, else bottom
func calcMVP(widthPx, heightPx int, tlx, tly, trx, try, blx, bly float32, yisdown bool) mat32.Mat3 {
	// Convert from pixel coords to vertex shader coords.
	invHalfWidth := 2 / float32(widthPx)
	invHalfHeight := 2 / float32(heightPx)
	if yisdown {
		tlx = tlx*invHalfWidth - 1
		tly = tly*invHalfHeight - 1
		trx = trx*invHalfWidth - 1
		try = try*invHalfHeight - 1
		blx = blx*invHalfWidth - 1
		bly = bly*invHalfHeight - 1
	} else {
		tlx = tlx*invHalfWidth - 1
		tly = 1 - tly*invHalfHeight // 1 - min
		trx = trx*invHalfWidth - 1
		try = 1 - try*invHalfHeight // 1 - min
		blx = blx*invHalfWidth - 1
		bly = 1 - bly*invHalfHeight // 1 - (min + max)
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
