// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdraw

import (
	"embed"
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
	tx := dw.Sys.Mem.Vals.ValMap["Tex"]
	tx.SetGoImage(img, false) // use fast non-flipping
	dw.FlipY = flipY
	dw.Sys.Mem.SyncToGPU()
	dw.Sys.SetVals(0, "Tex")
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

// Start starts rendering process on render target
func (dw *Drawer) Start() {
	dpl := dw.Sys.PipelineMap["draw"]
	if dw.Surf != nil {
		dw.SurfIdx = dw.Surf.AcquireNextImage()
		cmd := dpl.CmdPool.Buff
		vgpu.CmdReset(cmd)
		vgpu.CmdBegin(cmd)
		dpl.BeginRenderPass(cmd, dw.Surf.Frames[dw.SurfIdx])
		dpl.BindPipeline(cmd)
	}
}

// End ends rendering process on render target
func (dw *Drawer) End() {
	dpl := dw.Sys.PipelineMap["draw"]
	cmd := dpl.CmdPool.Buff
	if dw.Surf != nil {
		dpl.EndRenderPass(cmd)
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
	tx := dw.Sys.Mem.Vals.ValMap["Tex"]
	dw.ConfigMats(src2dst, tx.Texture.Format.Size, sr, op, dw.FlipY)
	if err := dw.Sys.Vars.Validate(); err != nil {
		return err
	}
	dpl := dw.Sys.PipelineMap["draw"]
	dpl.DrawVertex(dpl.CmdPool.Buff)
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
		pl.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0)
	} else {
		pl.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceClockwise, 1.0)
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

	posv := dw.Sys.Vars.Add("Pos", vgpu.Float32Vec2, vgpu.Vertex, 0, vgpu.VertexShader)
	// txcv := sy.Vars.Add("TexCoord", vgpu.Float32Vec2, vgpu.Vertex, 0, vgpu.VertexShader)
	idxv := dw.Sys.Vars.Add("Index", vgpu.Uint16, vgpu.Index, 0, vgpu.VertexShader)

	matv := dw.Sys.Vars.Add("Mats", vgpu.Struct, vgpu.Uniform, 0, vgpu.VertexShader)
	matv.SizeOf = vgpu.Float32Mat4.Bytes() * 2
	clrv := dw.Sys.Vars.Add("Color", vgpu.Float32Vec4, vgpu.Uniform, 0, vgpu.FragmentShader)
	tximgv := dw.Sys.Vars.Add("Tex", vgpu.ImageRGBA32, vgpu.TextureRole, 0, vgpu.FragmentShader)
	tximgv.TextureOwns = true

	nPts := 4
	nIdxs := 6
	rectPos := dw.Sys.Mem.Vals.Add("RectPos", posv, nPts)
	rectIdx := dw.Sys.Mem.Vals.Add("RectIdx", idxv, nIdxs)
	rectPos.Indexes = "RectIdx" // only need to set indexes for one vertex val

	// std vals
	dw.Sys.Mem.Vals.Add("Mats", matv, 1)
	dw.Sys.Mem.Vals.Add("FillColor", clrv, 1)
	tx := dw.Sys.Mem.Vals.Add("Tex", tximgv, 1)
	tx.Texture.Dev = dw.Sys.Device.Device // key for self-owning

	// note: add all values per above before doing Config
	dw.Sys.Config()
	dw.Sys.Mem.Config()

	// note: first val in set is offset
	rectPosA := rectPos.Floats32()
	rectPosA.Set(0,
		0.0, 0.0,
		0.0, 1.0,
		1.0, 0.0,
		1.0, 1.0)
	rectPos.Mod = true

	idxs := []uint16{0, 1, 2, 2, 1, 3} // triangle strip order
	rectIdx.CopyBytes(unsafe.Pointer(&idxs[0]))
	dw.Sys.SetVals(0, "RectPos", "Mats", "FillColor")
}

// ConfigMats configures the draw matrix for given draw parameters:
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling), txsz is the size of the texture to draw,
// sr is the source region (set to tex.Format.Bounds() for all)
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
// flipY inverts the Y axis of the source image.
func (dw *Drawer) ConfigMats(src2dst mat32.Mat3, txsz image.Point, sr image.Rectangle, op draw.Op, flipY bool) {
	sr = sr.Intersect(image.Rectangle{Max: txsz})
	if sr.Empty() {
		return
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
	var tmat Mats
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

	// fmt.Printf("MVP: %v   UVP: %v  \n", tmat.MVP, tmat.UVP)
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

	mat := dw.Sys.Mem.Vals.ValMap["Mats"]
	mat.CopyBytes(unsafe.Pointer(&tmat)) // sets mod
	dw.Sys.Mem.SyncToGPU()
	// dw.Sys.SetVals(0, "Mats")
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
	r, g, b, a := src.RGBA()
	fc := dw.Sys.Mem.Vals.ValMap["FillColor"]
	fcv := fc.Floats32()
	fcv.Set(0,
		float32(r)/65535,
		float32(g)/65535,
		float32(b)/65535,
		float32(a)/65535)
	fc.Mod = true

	dw.ConfigMats(src2dst, reg.Max, reg, op, false)
	dw.Sys.Mem.SyncToGPU()

	dw.Sys.SetVals(0, "Mats", "FillColor")
	if err := dw.Sys.Vars.Validate(); err != nil {
		return err
	}
	fpl := dw.Sys.PipelineMap["fill"]
	dpl := dw.Sys.PipelineMap["draw"]
	fpl.BindDrawVertex(dpl.CmdPool.Buff)
	return nil
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
