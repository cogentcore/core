// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"embed"
	"unsafe"

	"cogentcore.org/core/mat32"
	"cogentcore.org/core/vgpu"
	vk "github.com/goki/vulkan"
)

//go:embed shaders/*.spv
var content embed.FS

// CurRender holds info about the current render as updated by
// Use* methods -- determines which pipeline is used.
// Default is single color.
type CurRender struct {

	// index of descriptor collection to use -- for threaded / parallel rendering -- see vgup.Vars NDescs for more info
	DescIdx int

	// a texture was selected -- if true, overrides other options
	UseTexture bool

	// a per-vertex color was selected
	UseVtxColor bool

	// current model pose matrix
	ModelMtx mat32.Mat4

	// camera view and projection matrixes
	VPMtx Mtxs

	// current color surface properties
	Color Colors

	// texture parameters -- repeat, offset
	TexPars TexPars

	// index of currently-selected texture
	TexIdx int
}

// PushU is the push constants structure, holding everything that
// updates per object -- avoids any limitations on capacity.
type PushU struct {

	// Model Matrix: poses object in world coordinates
	ModelMtx mat32.Mat4

	// surface colors
	Color Colors

	// texture parameters
	Tex TexPars
}

// NewPush generates a new Push object based on current render settings
// unsafe.Pointer does not work having this be inside the CurRender obj itself
// so we create one afresh.
func (cr *CurRender) NewPush() *PushU {
	pu := &PushU{}
	pu.ModelMtx = cr.ModelMtx
	pu.Color = cr.Color
	// tex set specifically in tex
	return pu
}

// ConfigPipeline configures graphics settings on the pipeline
func (ph *Phong) ConfigPipeline(pl *vgpu.Pipeline) {
	pl.SetGraphicsDefaults()
	if ph.Wireframe {
		pl.SetRasterization(vk.PolygonModeLine, vk.CullModeNone, vk.FrontFaceCounterClockwise, 1.0)
	} else {
		pl.SetRasterization(vk.PolygonModeFill, vk.CullModeNone, vk.FrontFaceCounterClockwise, 1.0)
	}
}

// ConfigSys configures the vDraw System and pipelines.
func (ph *Phong) ConfigSys() {
	tpl := ph.Sys.NewPipeline("texture")
	ph.ConfigPipeline(tpl)
	opl := ph.Sys.NewPipeline("onecolor")
	ph.ConfigPipeline(opl)
	vpl := ph.Sys.NewPipeline("pervertex")
	ph.ConfigPipeline(vpl)

	tpl.AddShaderEmbed("texture_vert", vgpu.VertexShader, content, "shaders/texture_vert.spv")
	tpl.AddShaderEmbed("texture_frag", vgpu.FragmentShader, content, "shaders/texture_frag.spv")

	opl.AddShaderEmbed("onecolor_vert", vgpu.VertexShader, content, "shaders/onecolor_vert.spv")
	opl.AddShaderEmbed("onecolor_frag", vgpu.FragmentShader, content, "shaders/onecolor_frag.spv")

	vpl.AddShaderEmbed("pervertex_vert", vgpu.VertexShader, content, "shaders/pervertex_vert.spv")
	vpl.AddShaderEmbed("pervertex_frag", vgpu.FragmentShader, content, "shaders/pervertex_frag.spv")

	vars := ph.Sys.Vars()
	vars.NDescs = 1            // > 1 causes mysterious failures..
	pcset := vars.AddPushSet() // TexPush
	vset := vars.AddVertexSet()
	mtxset := vars.AddSet()   // set = 0
	nliteset := vars.AddSet() // set = 1
	liteset := vars.AddSet()  // set = 2
	txset := vars.AddSet()    // set = 3

	vec4sz := vgpu.Float32Vec4.Bytes()

	vset.Add("Pos", vgpu.Float32Vec3, 0, vgpu.Vertex, vgpu.VertexShader)
	vset.Add("Norm", vgpu.Float32Vec3, 0, vgpu.Vertex, vgpu.VertexShader)
	vset.Add("Tex", vgpu.Float32Vec2, 0, vgpu.Vertex, vgpu.VertexShader)
	vset.Add("Color", vgpu.Float32Vec4, 0, vgpu.Vertex, vgpu.VertexShader)
	vset.Add("Index", vgpu.Uint32, 0, vgpu.Index, vgpu.VertexShader)

	pcset.AddStruct("PushU", int(unsafe.Sizeof(PushU{})), 1, vgpu.Push, vgpu.VertexShader, vgpu.FragmentShader)

	mtxset.AddStruct("Mtxs", vgpu.Float32Mat4.Bytes()*2, 1, vgpu.Uniform, vgpu.VertexShader, vgpu.FragmentShader)

	nliteset.AddStruct("NLights", 4*4, 1, vgpu.Uniform, vgpu.FragmentShader)
	liteset.AddStruct("AmbLights", vec4sz*1, MaxLights, vgpu.Uniform, vgpu.FragmentShader)
	liteset.AddStruct("DirLights", vec4sz*2, MaxLights, vgpu.Uniform, vgpu.FragmentShader)
	liteset.AddStruct("PointLights", vec4sz*3, MaxLights, vgpu.Uniform, vgpu.FragmentShader)
	liteset.AddStruct("SpotLights", vec4sz*4, MaxLights, vgpu.Uniform, vgpu.FragmentShader)

	txset.Add("Tex", vgpu.ImageRGBA32, 1, vgpu.TextureRole, vgpu.FragmentShader)
	// tximgv.TextureOwns = true

	pcset.ConfigVals(1)
	mtxset.ConfigVals(1)
	nliteset.ConfigVals(1)
	liteset.ConfigVals(1)
}

// Push pushes given push constant data
func (ph *Phong) Push(pl *vgpu.Pipeline, push *PushU) {
	sy := &ph.Sys
	cmd := sy.CmdPool.Buff
	vars := sy.Vars()
	pvar, _ := vars.VarByNameTry(vgpu.PushSet, "PushU")
	pl.Push(cmd, pvar, unsafe.Pointer(push))
}
