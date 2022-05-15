// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"embed"

	"github.com/goki/mat32"
	"github.com/goki/vgpu/vgpu"
	vk "github.com/goki/vulkan"
)

//go:embed shaders/*.spv
var content embed.FS

// Mats contains the projection matricies
type Mats struct {
	MVMat   mat32.Mat4 `desc:"Model * View Matrix: transforms into camera-centered, 3D coordinates"`
	MVPMat  mat32.Mat4 `desc:"Model * View * Projection Matrix: transforms into 2D render coordinates"`
	NormMat mat32.Mat4 `desc:"Normal Matrix: normal matrix has no offsets, for normal vector rotation only, based on MVMatrix"`
}

// Colors are the material colors with padding for direct uploading to shader
type Colors struct {
	Color       mat32.Vec3 `desc:"main color of surface, used for both ambient and diffuse color in standard Phong model -- alpha component determines transparency -- note that transparent objects require more complex rendering"`
	pad0        float32
	Emissive    mat32.Vec3 `desc:"color that surface emits independent of any lighting -- i.e., glow -- can be used for marking lights with an object"`
	pad1        float32
	Specular    mat32.Vec3 `desc:"shiny reflective color of surface -- set to white for shiny objects and to Color for non-shiny objects"`
	pad2        float32
	ShinyBright mat32.Vec3 `desc:"X = shininess factor, Y = brightness factor:  shiny = specular shininess factor -- how focally the surface shines back directional light -- this is an exponential factor, with 0 = very broad diffuse reflection, and higher values (typically max of 128 or so but can go higher) having a smaller more focal specular reflection.  Also set Specular color to affect overall shininess effect; bright = overall multiplier on final computed color value -- can be used to tune the overall brightness of various surfaces relative to each other for a given set of lighting parameters"`
}

// ConfigPipeline configures graphics settings on the pipeline
func (ph *Phong) ConfigPipeline(pl *vgpu.Pipeline) {
	// gpu.Draw.Op(op)
	// gpu.Draw.DepthTest(false)
	// gpu.Draw.StencilTest(false)
	// gpu.Draw.Multisample(false)
	// app.drawProg.Activate()

	pl.SetGraphicsDefaults()
	pl.SetClearOff()
	// if ph.YIsDown {
	pl.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0)
	// } else {
	// 	pl.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceClockwise, 1.0)
	// }
}

// ConfigSys configures the vDraw System and pipelines.
func (ph *Phong) ConfigSys() {
	tpl := ph.Sys.NewPipeline("texture")
	ph.ConfigPipeline(tpl)

	cb, _ := content.ReadFile("shaders/texture_vert.spv")
	tpl.AddShaderCode("texture_vert", vgpu.VertexShader, cb)
	cb, _ = content.ReadFile("shaders/texture_frag.spv")
	tpl.AddShaderCode("texture_frag", vgpu.FragmentShader, cb)

	vars := ph.Sys.Vars()
	pcset := vars.AddPushSet() // TexIdx
	vset := vars.AddVertexSet()
	matset := vars.AddSet()   // set = 0
	clrset := vars.AddSet()   // set = 1
	nliteset := vars.AddSet() // set = 2
	liteset := vars.AddSet()  // set = 3
	txset := vars.AddSet()    // set = 4

	vec4sz := vgpu.Float32Vec4.Bytes()

	vset.Add("Pos", vgpu.Float32Vec4, 0, vgpu.Vertex, vgpu.VertexShader)
	vset.Add("Norm", vgpu.Float32Vec3, 0, vgpu.Vertex, vgpu.VertexShader)
	vset.Add("Tex", vgpu.Float32Vec3, 0, vgpu.Vertex, vgpu.VertexShader)
	vset.Add("Color", vgpu.Float32Vec2, 0, vgpu.Vertex, vgpu.VertexShader)
	vset.Add("Index", vgpu.Uint16, nIdxs, vgpu.Index, vgpu.VertexShader)

	matset.AddStruct("Mats", vgpu.Float32Mat4.Bytes()*3, 1, vgpu.Uniform, vgpu.VertexShader)

	pcset.AddStruct("TexIdx", 4, 1, vgpu.Push, vgpu.FragmentShader)
	clrset.AddStruct("Color", vec4sz*4, 1, vgpu.Uniform, vgpu.FragmentShader)

	nliteset.AddStruct("NLights", 4*4, 1, vgpu.Uniform, vgpu.FragmentShader)
	liteset.AddStruct("AmbLights", vec4sz*1, 1, vgpu.Uniform, vgpu.FragmentShader)
	liteset.AddStruct("DirLights", vec4sz*2, 1, vgpu.Uniform, vgpu.FragmentShader)
	liteset.AddStruct("PointLights", vec4sz*3, 1, vgpu.Uniform, vgpu.FragmentShader)
	liteset.AddStruct("SpotLights", vec4sz*4, 1, vgpu.Uniform, vgpu.FragmentShader)

	txset.Add("Tex", vgpu.ImageRGBA32, 1, vgpu.TextureRole, vgpu.FragmentShader)
	// tximgv.TextureOwns = true

	liteset.ConfigVals(MaxLights)

	// vset.ConfigVals(1)
	// txset.ConfigVals(1)
	// cset.ConfigVals(ph.Impl.MaxColors)

	// note: add all values per above before doing Config
	// ph.Sys.Config()

	/*
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

		ph.Sys.Mem.SyncToGPU()

		vars.BindVertexValIdx("Pos", 0)
		vars.BindVertexValIdx("Index", 0)
	*/
}
