// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

import (
	"embed"

	"cogentcore.org/core/gpu"
	"cogentcore.org/core/math32"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

//go:embed shaders/*.wgsl
var shaders embed.FS

// Current holds info about the current render as updated by
// Use* methods. Determines which pipeline is used.
// Default is single color.
type Current struct {

	// a texture was selected, if true, overrides other options
	UseTexture bool

	// a per-vertex color was selected
	UseVtxColor bool

	// current model pose matrix
	ModelMtx math32.Matrix4

	// camera view and projection matrixes
	VPMtx Mtxs

	// current color surface properties
	Color Colors

	// texture parameters -- repeat, offset
	TexPars TexPars

	// index of currently selected texture
	TexIndex int
}

// PushU is the push constants structure, holding everything that
// updates per object -- avoids any limitations on capacity.
type PushU struct {

	// Model Matrix: poses object in world coordinates
	ModelMtx math32.Matrix4

	// surface colors
	Color Colors

	// texture parameters
	Tex TexPars
}

// NewPush generates a new Push object based on current render settings
// unsafe.Pointer does not work having this be inside the Current obj itself
// so we create one afresh.
func (cr *Current) NewPush() *PushU {
	pu := &PushU{}
	pu.ModelMtx = cr.ModelMtx
	pu.Color = cr.Color
	// tex set specifically in tex
	return pu
}

// ConfigPipeline configures graphics settings on the pipeline
func (ph *Phong) ConfigPipeline(pl *gpu.Pipeline) {
	pl.SetGraphicsDefaults()
	pl.SetCullMode(wgpu.CullModeNone)
	// if ph.Wireframe {
	// 	pl.SetRasterization(vk.PolygonModeLine, vk.CullModeNone, vk.FrontFaceCounterClockwise, 1.0)
	// } else {
	// 	pl.SetRasterization(vk.PolygonModeFill, vk.CullModeNone, vk.FrontFaceCounterClockwise, 1.0)
	// }
}

// ConfigSys configures the vDraw System and pipelines.
func (ph *Phong) ConfigSys() {
	sy := ph.Sys
	tpl := sy.NewPipeline("texture")
	ph.ConfigPipeline(tpl)
	opl := sy.NewPipeline("onecolor")
	ph.ConfigPipeline(opl)
	vpl := sy.NewPipeline("pervertex")
	ph.ConfigPipeline(vpl)

	sh := tpl.AddShader("texture")
	sh.OpenFileFS(shaders, "shaders/texture.wgsl")
	tpl.AddEntry(sh, gpu.VertexShader, "vs_main")
	tpl.AddEntry(sh, gpu.FragmentShader, "fs_main")

	sh = opl.AddShader("onecolor")
	sh.OpenFileFS(shaders, "shaders/onecolor.wgsl")
	opl.AddEntry(sh, gpu.VertexShader, "vs_main")
	opl.AddEntry(sh, gpu.FragmentShader, "fs_main")

	sh = opl.AddShader("pervertex")
	sh.OpenFileFS(shaders, "shaders/pervertex.wgsl")
	vpl.AddEntry(sh, gpu.VertexShader, "vs_main")
	vpl.AddEntry(sh, gpu.FragmentShader, "fs_main")

	vgp := sy.Vars.AddVertexGroup()
	mgp := sy.Vars.AddGroup(gpu.Uniform, "Matrix")         // group = 0
	lgp := sy.Vars.AddGroup(gpu.Uniform, "Lights")         // group = 1
	cgp := sy.Vars.AddGroup(gpu.Uniform, "Color")          // group = 2
	tgp := sy.Vars.AddGroup(gpu.SampledTexture, "Texture") // group = 3

	vector4sz := gpu.Float32Vector4.Bytes()

	vgp.Add("Pos", gpu.Float32Vector3, 0, gpu.VertexShader)
	vgp.Add("Norm", gpu.Float32Vector3, 0, gpu.VertexShader)
	vgp.Add("TexCoord", gpu.Float32Vector2, 0, gpu.VertexShader)
	vgp.Add("VertexColor", gpu.Float32Vector4, 0, gpu.VertexShader)
	mx := vgp.Add("ModelMtx", gpu.Float32Matrix4, 0, gpu.VertexShader) // serialized 4xVector
	ix := vgp.Add("Index", gpu.Uint32, 0, gpu.VertexShader)
	ix.Role = gpu.Index

	mgp.AddStruct("Matrix", gpu.Float32Matrix4.Bytes()*2, 1, gpu.VertexShader, gpu.FragmentShader)

	lgp.AddStruct("NLights", reflectx.SizeOf(NLights), 1, gpu.FragmentShader)
	lgp.AddStruct("AmbLights", vector4sz*1, MaxLights, gpu.FragmentShader)
	lgp.AddStruct("DirLights", vector4sz*2, MaxLights, gpu.FragmentShader)
	lgp.AddStruct("PointLights", vector4sz*3, MaxLights, gpu.FragmentShader)
	lgp.AddStruct("SpotLights", vector4sz*4, MaxLights, gpu.FragmentShader)

	// note: could combine color + lights but no need to -- under 4 limit
	cv := cgp.AddStruct("Color", reflectx.SizeOf(Colors), 1, gpu.FragmentShader)
	cv.DynamicOffset = true

	tgp.Add("TexSampler", gpu.TextureRGBA32, 1, gpu.FragmentShader)
	mgp.SetNValues(1)
	lgp.SetNValues(1)
	tgp.SetNValues(1)
}
