// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdraw

import (
	"embed"
	"fmt"
	"image"
	"image/color"
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
	MVP mat32.Mat4
	UVP mat32.Mat4
}

// Drawer is the vDraw implementation, which can be configured for
// different render targets (Surface, Framebuffer).  It manages
// an associated System.
type Drawer struct {
	Sys     vgpu.System   `desc:"drawing system"`
	Surf    *vgpu.Surface `desc:"surface if render target"`
	YIsDown bool          `desc:"render so the Y axis points down, with 0,0 at the upper left, which is the Vulkan standard.  default is Y is up, with 0,0 at bottom left, which is OpenGL default.  this must be set prior to configuring, the surface, as it determines the rendering parameters."`
	FlipY   bool          `desc:"flip the Y axis of the image when drawing"`
	SurfIdx uint32        `desc:"surface index for current render process"`
}

// SetImage sets given Go image as the drawing source.
// A standard Go image is rendered upright on a standard
// Vulkan surface. If flipY is true then the Image Y axis is
// efficiently flipped when rendering.
// A subsequent Copy, Scale or Draw call will render this image.
func (dw *Drawer) SetImage(img image.Image, flipY bool) {
	_, tx, _ := dw.Sys.Vars().ValByIdxTry(0, "Tex", 0)
	tx.SetGoImage(img, false) // use fast non-flipping
	dw.FlipY = flipY
	vars := dw.Sys.Vars()
	vars.BindValsStart(0)
	vars.BindStatVars(0) // gets images
	vars.BindValsEnd()
}

// Copy copies currently-set texture to render target.
// dp is the destination point,
// sr is the source region (set to tex.Format.Bounds() for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Copy(dp image.Point, sr image.Rectangle, op draw.Op) error {
	mat := mat32.Mat3{
		1, 0, 0,
		0, 1, 0,
		float32(dp.X - sr.Min.X), float32(dp.Y - sr.Min.Y), 1,
	}
	return dw.Draw(mat, sr, op)
}

// Scale copies currently-set texture to render target,
// scaling the region defined by src and sr to the destination
// such that sr in src-space is mapped to dr in dst-space.
// dr is the destination rectangle
// sr is the source region (set to tex.Format.Bounds() for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Scale(dr image.Rectangle, sr image.Rectangle, op draw.Op) error {
	rx := float32(dr.Dx()) / float32(sr.Dx())
	ry := float32(dr.Dy()) / float32(sr.Dy())
	mat := mat32.Mat3{
		rx, 0, 0,
		0, ry, 0,
		float32(dr.Min.X) - rx*float32(sr.Min.X),
		float32(dr.Min.Y) - ry*float32(sr.Min.Y), 1,
	}
	return dw.Draw(mat, sr, op)
}

// StartDraw starts image drawing rendering process on render target
func (dw *Drawer) StartImage() {
	dpl := dw.Sys.PipelineMap["draw"]
	if dw.Surf != nil {
		dw.SurfIdx = dw.Surf.AcquireNextImage()
		cmd := dpl.CmdPool.Buff
		vgpu.CmdReset(cmd)
		vgpu.CmdBegin(cmd)
		dpl.BeginRenderPass(cmd, dw.Surf.Frames[dw.SurfIdx])
		dpl.BindPipeline(cmd, 0)
	}
}

// EndDraw ends image drawing rendering process on render target
func (dw *Drawer) EndDraw() {
	dpl := dw.Sys.PipelineMap["draw"]
	cmd := dpl.CmdPool.Buff
	if dw.Surf != nil {
		dpl.EndRenderPass(cmd)
		vgpu.CmdEnd(cmd)
		dw.Surf.SubmitRender(cmd) // this is where it waits for the 16 msec
		dw.Surf.PresentImage(dw.SurfIdx)
	}
}

// StartFill starts color fill drawing rendering process on render target
func (dw *Drawer) StartFill() {
	fpl := dw.Sys.PipelineMap["fill"]
	if dw.Surf != nil {
		dw.SurfIdx = dw.Surf.AcquireNextImage()
		cmd := fpl.CmdPool.Buff
		vgpu.CmdReset(cmd)
		vgpu.CmdBegin(cmd)
		fpl.BeginRenderPass(cmd, dw.Surf.Frames[dw.SurfIdx])
		fpl.BindPipeline(cmd, 0)
	}
}

// EndFill ends color filling rendering process on render target
func (dw *Drawer) EndFill() {
	fpl := dw.Sys.PipelineMap["fill"]
	cmd := fpl.CmdPool.Buff
	if dw.Surf != nil {
		fpl.EndRenderPass(cmd)
		vgpu.CmdEnd(cmd)
		dw.Surf.SubmitRender(cmd) // this is where it waits for the 16 msec
		dw.Surf.PresentImage(dw.SurfIdx)
	}
}

// Draw draws currently-set texture to render target.
// Must have called StartDraw first.
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling),
// sr is the source region (set to tex.Format.Bounds() for all)
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Draw(src2dst mat32.Mat3, sr image.Rectangle, op draw.Op) error {
	_, tx, _ := dw.Sys.Vars().ValByIdxTry(0, "Tex", 0)
	tmat := dw.ConfigMats(src2dst, tx.Texture.Format.Size, sr, op, dw.FlipY)

	matv, _ := dw.Sys.Vars().VarByNameTry(vgpu.PushConstSet, "Mats")

	dpl := dw.Sys.PipelineMap["draw"]

	cmd := dpl.CmdPool.Buff
	dpl.PushConstant(cmd, matv, vk.ShaderStageVertexBit, unsafe.Pointer(tmat))
	dpl.DrawVertex(cmd, 0)
	return nil
}

// ConfigSurface configures the Drawer to use given surface as a render target
func (dw *Drawer) ConfigSurface(sf *vgpu.Surface) {
	dw.Surf = sf
	dw.Sys.InitGraphics(sf.GPU, "vdraw.Drawer", &sf.Device)
	dw.Sys.RenderPass.NoClear = true
	dw.Sys.ConfigRenderPass(&dw.Surf.Format, vgpu.UndefType)
	sf.SetRenderPass(&dw.Sys.RenderPass)

	dw.ConfigSys()
}

func (dw *Drawer) Destroy() {
	dw.Sys.Destroy()
}

// DestSize returns the size of the render destination
func (dw *Drawer) DestSize() image.Point {
	if dw.Surf != nil {
		return dw.Surf.Format.Size
	}
	return image.Point{10, 10}
}

// FillRect fills color to render target, to given region.
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) FillRect(src color.Color, reg image.Rectangle, op draw.Op) error {
	mat := mat32.Mat3{
		1, 0, 0,
		0, 1, 0,
		0, 0, 1,
	}
	return dw.Fill(src, mat, reg, op)
}

// Fill fills color to render target.
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling),
// reg is the region to fill
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Fill(src color.Color, src2dst mat32.Mat3, reg image.Rectangle, op draw.Op) error {
	vars := dw.Sys.Vars()

	r, g, b, a := src.RGBA()
	clrv, fc, _ := vars.ValByIdxTry(1, "Color", 0)
	fcv := fc.Floats32()
	fcv.Set(0,
		float32(r)/65535,
		float32(g)/65535,
		float32(b)/65535,
		float32(a)/65535)
	fc.SetMod()

	tmat := dw.ConfigMats(src2dst, reg.Max, reg, op, false)

	matv, _ := vars.VarByNameTry(vgpu.PushConstSet, "Mats")

	vars.BindValsStart(0)
	vars.BindDynVal(1, clrv, fc)
	vars.BindVertexValIdx("Pos", 0)
	vars.BindVertexValIdx("Index", 0)
	vars.BindValsEnd()

	fpl := dw.Sys.PipelineMap["fill"]
	cmd := fpl.CmdPool.Buff
	fpl.PushConstant(cmd, matv, vk.ShaderStageVertexBit, unsafe.Pointer(tmat))
	fpl.BindDrawVertex(cmd, 0)

	return nil
}

///////////////////////////////////////////////////////////////////////
// Config

// ConfigPipeline configures graphics settings on the pipeline
func (dw *Drawer) ConfigPipeline(pl *vgpu.Pipeline) {
	// gpu.Draw.Op(op)
	// gpu.Draw.DepthTest(false)
	// gpu.Draw.StencilTest(false)
	// gpu.Draw.Multisample(false)
	// app.drawProg.Activate()

	pl.SetGraphicsDefaults()
	pl.SetClearOff()
	if dw.YIsDown {
		pl.SetRasterization(vk.PolygonModeFill, vk.CullModeNone, vk.FrontFaceCounterClockwise, 1.0)
	} else {
		pl.SetRasterization(vk.PolygonModeFill, vk.CullModeNone, vk.FrontFaceClockwise, 1.0)
	}
}

// ConfigSys configures the vDraw System and pipelines.
func (dw *Drawer) ConfigSys() {
	dpl := dw.Sys.NewPipeline("draw")
	dw.ConfigPipeline(dpl)

	cb, _ := content.ReadFile("shaders/draw_vert.spv")
	dpl.AddShaderCode("draw_vert", vgpu.VertexShader, cb)
	cb, _ = content.ReadFile("shaders/draw_frag.spv")
	dpl.AddShaderCode("draw_frag", vgpu.FragmentShader, cb)

	fpl := dw.Sys.NewPipeline("fill")
	dw.ConfigPipeline(fpl)

	cb, _ = content.ReadFile("shaders/fill_vert.spv")
	fpl.AddShaderCode("fill_vert", vgpu.VertexShader, cb)
	cb, _ = content.ReadFile("shaders/fill_frag.spv")
	fpl.AddShaderCode("fill_frag", vgpu.FragmentShader, cb)

	vars := dw.Sys.Vars()
	vset := vars.AddVertexSet()
	pcset := vars.AddPushConstSet()
	txset := vars.AddSet()
	cset := vars.AddSet()

	nPts := 4
	nIdxs := 6

	posv := vset.Add("Pos", vgpu.Float32Vec3, nPts, vgpu.Vertex, vgpu.VertexShader)
	idxv := vset.Add("Index", vgpu.Uint16, nIdxs, vgpu.Index, vgpu.VertexShader)

	pcset.AddStruct("Mats", vgpu.Float32Mat4.Bytes()*2, 1, vgpu.PushConst, vgpu.VertexShader)

	tximgv := txset.Add("Tex", vgpu.ImageRGBA32, 1, vgpu.TextureRole, vgpu.FragmentShader)
	tximgv.TextureOwns = true

	cset.Add("Color", vgpu.Float32Vec4, 1, vgpu.Uniform, vgpu.FragmentShader)

	vset.ConfigVals(1)
	txset.ConfigVals(1)
	cset.ConfigVals(1)

	// note: add all values per above before doing Config
	dw.Sys.Config()

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
	rectIdx.CopyBytes(unsafe.Pointer(&idxs[0]))

	dw.Sys.Mem.SyncToGPU()

	vars.BindValsStart(0) // only one set of bindings
	vars.BindVertexValIdx("Pos", 0)
	vars.BindVertexValIdx("Index", 0)
	vars.BindValsEnd()
}

// ConfigMats configures the draw matrix for given draw parameters:
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling), txsz is the size of the texture to draw,
// sr is the source region (set to tex.Format.Bounds() for all)
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
// flipY inverts the Y axis of the source image.
func (dw *Drawer) ConfigMats(src2dst mat32.Mat3, txsz image.Point, sr image.Rectangle, op draw.Op, flipY bool) *Mats {
	var tmat Mats

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

	fmt.Printf("MVP: %v   UVP: %v  \n", tmat.MVP, tmat.UVP)
	// z := float32(1)
	// coords := []mat32.Vec4{
	// 	{0.0, 0.0, z, 1},
	// 	{0.0, 1.0, z, 1},
	// 	{1.0, 0.0, z, 1},
	// 	{1.0, 1.0, z, 1}}
	// for _, v := range coords {
	// 	tv := v.MulMat4(&tmat.MVP)
	// 	fmt.Printf("v: %v   tv: %v\n", v, tv)
	// }

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
//
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
