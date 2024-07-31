// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

import (
	"embed"
	"unsafe"

	"cogentcore.org/core/gpu"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

//go:embed shaders/*.wgsl
var shaders embed.FS

// ConfigPipeline configures graphics settings on the pipeline
func (ph *Phong) configPipeline(pl *gpu.GraphicsPipeline) {
	pl.SetGraphicsDefaults()
	pl.SetCullMode(wgpu.CullModeNone)
	// if ph.Wireframe {
	// 	pl.SetRasterization(vk.PolygonModeLine, vk.CullModeNone, vk.FrontFaceCounterClockwise, 1.0)
	// } else {
	// 	pl.SetRasterization(vk.PolygonModeFill, vk.CullModeNone, vk.FrontFaceCounterClockwise, 1.0)
	// }
}

// Groups are the VarGroup numbers.
type Groups int32 //enums:enum

const (
	MatrixGroup Groups = iota
	ObjectGroup
	LightGroup
	TextureGroup
)

// configSystem configures the Phong System and pipelines.
func (ph *Phong) configSystem() {
	sy := ph.Sys
	tpl := sy.AddGraphicsPipeline("texture")
	ph.configPipeline(tpl)
	opl := sy.AddGraphicsPipeline("onecolor")
	ph.configPipeline(opl)
	vpl := sy.AddGraphicsPipeline("pervertex")
	ph.configPipeline(vpl)

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
	ogp := sy.Vars.AddGroup(gpu.Uniform, "Objects")        // group = 1
	lgp := sy.Vars.AddGroup(gpu.Uniform, "Lights")         // group = 2
	tgp := sy.Vars.AddGroup(gpu.SampledTexture, "Texture") // group = 3

	vector4sz := gpu.Float32Vector4.Bytes()

	vgp.Add("Pos", gpu.Float32Vector3, 0, gpu.VertexShader)
	vgp.Add("Norm", gpu.Float32Vector3, 0, gpu.VertexShader)
	vgp.Add("TexCoord", gpu.Float32Vector2, 0, gpu.VertexShader)
	vgp.Add("VertexColor", gpu.Float32Vector4, 0, gpu.VertexShader)
	ix := vgp.Add("Index", gpu.Uint32, 0, gpu.VertexShader)
	ix.Role = gpu.Index

	mgp.AddStruct("Matrix", gpu.Float32Matrix4.Bytes()*2, 1, gpu.VertexShader, gpu.FragmentShader)

	ov := ogp.AddStruct("Object", int(unsafe.Sizeof(Object{})), 1, gpu.VertexShader, gpu.FragmentShader)
	ov.DynamicOffset = true

	lgp.AddStruct("NLights", int(unsafe.Sizeof(NLights{})), 1, gpu.FragmentShader)
	lgp.AddStruct("AmbLights", vector4sz*1, MaxLights, gpu.FragmentShader)
	lgp.AddStruct("DirLights", vector4sz*2, MaxLights, gpu.FragmentShader)
	lgp.AddStruct("PointLights", vector4sz*3, MaxLights, gpu.FragmentShader)
	lgp.AddStruct("SpotLights", vector4sz*4, MaxLights, gpu.FragmentShader)

	tgp.Add("TexSampler", gpu.TextureRGBA32, 1, gpu.FragmentShader)
	mgp.SetNValues(1)
	lgp.SetNValues(1)
	tgp.SetNValues(1)
}
