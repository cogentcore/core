// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdraw

import (
	"embed"
	"fmt"
	"image"
	"image/draw"
	"unsafe"

	"github.com/goki/mat32"
	"github.com/goki/vgpu/vgpu"

	vk "github.com/vulkan-go/vulkan"
)

//go:embed shaders/*.spv
var content embed.FS

// Mats are the projection matricies
type Mats struct {
	MVP    mat32.Mat3 // 9 * 4 = 36 bytes
	align1 mat32.Vec3 // 3 * 4 = 12 bytes
	align2 mat32.Vec4 // 4 * 4 = 16 bytes = 64 byte alignment
	UVP    mat32.Mat3
}

// Drawer is the vDraw implementation, which can be configured for
// different render targets (Surface, Framebuffer).  It manages
// an associated System.
type Drawer struct {
	Sys  vgpu.System   `desc:"drawing system"`
	Surf *vgpu.Surface `desc:"surface if render target"`
}

// CopyImage copies given Go image to configured render target, using draw parameters:
// If flipY is true (default) then the Image Y axis is flipped
// when copying into the image data, so that images will appear
// upright in the standard OpenGL Y-is-up coordinate system.
// dp is the destination point, sr is the source region (set to tex.Format.Bounds() for all)
// op is the drawing operation: Src = copy source directly (blit), Over = alpha blend with existing
func (dr *Drawer) CopyImage(img image.Image, flipY bool, dp image.Point, sr image.Rectangle, op draw.Op) error {
	mat := mat32.Mat3{
		1, 0, 0,
		0, 1, 0,
		float32(dp.X - sr.Min.X), float32(dp.Y - sr.Min.Y), 1,
	}
	return dr.DrawImage(img, flipY, mat, sr, op)
}

// DrawImage draws given Go image to configured render target, using draw parameters:
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling), txsz is the size of the texture to draw,
// sr is the source region (set to tex.Format.Bounds() for all)
// op is the drawing operation: Src = copy source directly (blit), Over = alpha blend with existing
func (dr *Drawer) DrawImage(img image.Image, flipY bool, src2dst mat32.Mat3, sr image.Rectangle, op draw.Op) error {
	tx := dr.Sys.Mem.Vals.ValMap["Tex"]
	tx.SetGoImage(img, flipY)

	dr.ConfigMats(src2dst, tx.Texture.Format.Size, sr, op)
	dr.Sys.Mem.SyncToGPU()

	dr.Sys.SetVals(0, "RectPos", "Mats", "Tex", "FillColor")
	if err := dr.Sys.Vars.Validate(); err != nil {
		return err
	}
	dpl := dr.Sys.PipelineMap["draw"]

	if dr.Surf != nil {
		idx := dr.Surf.AcquireNextImage()
		dpl.FullStdRender(dpl.CmdPool.Buff, dr.Surf.Frames[idx])
		dr.Surf.SubmitRender(dpl.CmdPool.Buff) // this is where it waits for the 16 msec
		dr.Surf.PresentImage(idx)
	}
	return nil
}

// ConfigSurface configures the Drawer to use given surface as a render target
func (dr *Drawer) ConfigSurface(sf *vgpu.Surface) {
	dr.Surf = sf
	dr.Sys.InitGraphics(sf.GPU, "vdraw.Drawer", &sf.Device)
	dr.Sys.ConfigRenderPass(&dr.Surf.Format, vgpu.UndefType)
	sf.SetRenderPass(&dr.Sys.RenderPass)

	dr.ConfigSys()
}

func (dr *Drawer) Destroy() {
	dr.Sys.Destroy()
}

// DestSize returns the size of the render destination
func (dr *Drawer) DestSize() image.Point {
	if dr.Surf != nil {
		return dr.Surf.Format.Size
	}
	return image.Point{10, 10}
}

// ConfigPipeline configures graphics settings on the pipeline
func (dr *Drawer) ConfigPipeline(pl *vgpu.Pipeline) {
	// gpu.Draw.Op(op)
	// gpu.Draw.DepthTest(false)
	// gpu.Draw.CullFace(false, true, dstBotZero) // cull back face -- dstBotZero = CCW, !dstBotZero = CW
	// gpu.Draw.StencilTest(false)
	// gpu.Draw.Multisample(false)
	// app.drawProg.Activate()

	pl.SetGraphicsDefaults()
	pl.SetClearColor(0.2, 0.2, 0.2, 1)
	pl.SetRasterization(vk.PolygonModeFill, vk.CullModeNone, vk.FrontFaceCounterClockwise, 1.0)
}

// ConfigSys configures the vDraw System and pipelines.
func (dr *Drawer) ConfigSys() {
	dpl := dr.Sys.NewPipeline("draw")
	dr.ConfigPipeline(dpl)

	cb, _ := content.ReadFile("shaders/draw_vert.spv")
	dpl.AddShaderCode("draw_vert", vgpu.VertexShader, cb)
	cb, _ = content.ReadFile("shaders/draw_frag.spv")
	dpl.AddShaderCode("draw_frag", vgpu.FragmentShader, cb)

	fpl := dr.Sys.NewPipeline("fill")
	dr.ConfigPipeline(fpl)

	cb, _ = content.ReadFile("shaders/fill_vert.spv")
	fpl.AddShaderCode("fill_vert", vgpu.VertexShader, cb)
	cb, _ = content.ReadFile("shaders/fill_frag.spv")
	fpl.AddShaderCode("fill_frag", vgpu.FragmentShader, cb)

	posv := dr.Sys.Vars.Add("Pos", vgpu.Float32Vec2, vgpu.Vertex, 0, vgpu.VertexShader)
	// txcv := sy.Vars.Add("TexCoord", vgpu.Float32Vec2, vgpu.Vertex, 0, vgpu.VertexShader)
	idxv := dr.Sys.Vars.Add("Index", vgpu.Uint16, vgpu.Index, 0, vgpu.VertexShader)

	matv := dr.Sys.Vars.Add("Mats", vgpu.Struct, vgpu.Uniform, 0, vgpu.VertexShader)
	matv.SizeOf = vgpu.Float32Mat4.Bytes() * 2 // no padding for these
	clrv := dr.Sys.Vars.Add("Color", vgpu.Float32Vec4, vgpu.Uniform, 0, vgpu.FragmentShader)
	tximgv := dr.Sys.Vars.Add("Tex", vgpu.ImageRGBA32, vgpu.TextureRole, 0, vgpu.FragmentShader)
	tximgv.TextureOwns = true

	nPts := 4
	nIdxs := 6
	rectPos := dr.Sys.Mem.Vals.Add("RectPos", posv, nPts)
	rectIdx := dr.Sys.Mem.Vals.Add("RectIdx", idxv, nIdxs)
	rectPos.Indexes = "RectIdx" // only need to set indexes for one vertex val

	// std vals
	dr.Sys.Mem.Vals.Add("Mats", matv, 1)
	dr.Sys.Mem.Vals.Add("FillColor", clrv, 1)
	tx := dr.Sys.Mem.Vals.Add("Tex", tximgv, 1)
	tx.Texture.Dev = dr.Sys.Device.Device // key for self-owning

	// note: add all values per above before doing Config
	dr.Sys.Config()
	dr.Sys.Mem.Config()

	// note: first val in set is offset
	rectPosA := rectPos.Floats32()
	rectPosA.Set(0,
		0.0, 0.0,
		1.0, 0.0,
		1.0, 1.0,
		0.0, 1.0)
	rectPos.Mod = true

	idxs := []uint16{0, 1, 2, 0, 2, 3}
	rectIdx.CopyBytes(unsafe.Pointer(&idxs[0]))
}

// ConfigMats configures the draw matrix for given draw parameters:
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling), txsz is the size of the texture to draw,
// sr is the source region (set to tex.Format.Bounds() for all)
// op is the drawing operation: Src = copy source directly (blit), Over = alpha blend with existing
func (dr *Drawer) ConfigMats(src2dst mat32.Mat3, txsz image.Point, sr image.Rectangle, op draw.Op) {
	sr = sr.Intersect(image.Rectangle{Max: txsz})
	if sr.Empty() {
		return
	}

	destSz := dr.DestSize()

	dstBotZero := true
	srcBotZero := true

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
		dstBotZero,
	)
	var tmat Mats
	tmat.MVP = matMVP // todo render direct

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

	if srcBotZero {
		tmat.UVP = mat32.Mat3{
			qx - px, 0, 0,
			0, py - sy, 0, // py - sy
			px, sy, 1}
	} else {
		tmat.UVP = mat32.Mat3{
			qx - px, 0, 0,
			0, sy - py, 0, // sy - py
			px, py, 1,
		}
	}

	fmt.Printf("matUVP: %v  matMVP: %v\n", tmat.UVP, matMVP)

	// coords := []mat32.Vec3{}

	mat := dr.Sys.Mem.Vals.ValMap["Mats"]
	mat.CopyBytes(unsafe.Pointer(&tmat)) // sets mod
}

/*
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
*/

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
