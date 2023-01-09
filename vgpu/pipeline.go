// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"log"
	"unsafe"

	"github.com/goki/ki/ints"
	vk "github.com/goki/vulkan"
)

// Pipeline manages Shader program(s) that accomplish a specific
// type of rendering or compute function, using Vars / Vals
// defined by the overall System.
// In the graphics context, each pipeline could handle a different
// class of materials (textures, Phong lighting, etc).
type Pipeline struct {
	Name      string             `desc:"unique name of this pipeline"`
	Sys       *System            `desc:"system that we belong to and manages all shared resources (Memory, Vars, Vals, etc), etc"`
	Shaders   []*Shader          `desc:"shaders in order added -- should be execution order"`
	ShaderMap map[string]*Shader `desc:"shaders loaded for this pipeline"`

	VkConfig   vk.GraphicsPipelineCreateInfo `desc:"vulkan pipeline configuration options"`
	VkPipeline vk.Pipeline                   `desc:"the created vulkan pipeline"`
	VkCache    vk.PipelineCache              `desc:"cache"`
}

// Vars returns a pointer to the vars for this pipeline, which has vals within it
func (pl *Pipeline) Vars() *Vars {
	return pl.Sys.Vars()
}

// AddShader adds Shader with given name and type to the pipeline
func (pl *Pipeline) AddShader(name string, typ ShaderTypes) *Shader {
	if pl.ShaderMap == nil {
		pl.ShaderMap = make(map[string]*Shader)
	}
	if sh, has := pl.ShaderMap[name]; has {
		log.Printf("vgpu.Pipeline AddShader: Shader named: %s already exists in pipline: %s\n", name, pl.Name)
		return sh
	}
	sh := &Shader{Name: name, Type: typ}
	pl.Shaders = append(pl.Shaders, sh)
	pl.ShaderMap[name] = sh
	return sh
}

// AddShaderFile adds Shader with given name and type to the pipeline,
// Opening SPV code from given filename
func (pl *Pipeline) AddShaderFile(name string, typ ShaderTypes, fname string) *Shader {
	sh := pl.AddShader(name, typ)
	sh.OpenFile(pl.Sys.Device.Device, fname)
	return sh
}

// AddShaderCode adds Shader with given name and type to the pipeline,
// Loading SPV code from given bytes
func (pl *Pipeline) AddShaderCode(name string, typ ShaderTypes, code []byte) *Shader {
	sh := pl.AddShader(name, typ)
	sh.OpenCode(pl.Sys.Device.Device, code)
	return sh
}

// ShaderByName returns Shader by name.
// Returns nil if not found (error auto logged).
func (pl *Pipeline) ShaderByName(name string) *Shader {
	sh, ok := pl.ShaderMap[name]
	if !ok {
		log.Printf("vgpu.Pipeline ShaderByName: Shader: %s not found in pipeline: %s\n", name, pl.Name)
		return nil
	}
	return sh
}

// FreeShaders is called after successful pipeline creation, to unload shader modules
// as they are no longer needed
func (pl *Pipeline) FreeShaders() {
	for _, sh := range pl.Shaders {
		sh.Free(pl.Sys.Device.Device)
	}
}

func (pl *Pipeline) Destroy() {
	pl.FreeShaders()
	pl.DestroyPipeline()
}

func (pl *Pipeline) DestroyPipeline() {
	vk.DestroyPipelineCache(pl.Sys.Device.Device, pl.VkCache, nil)
	vk.DestroyPipeline(pl.Sys.Device.Device, pl.VkPipeline, nil)
	pl.VkCache = nil
	pl.VkPipeline = nil
}

// Init initializes pipeline as part of given System
func (pl *Pipeline) Init(sy *System) {
	pl.Sys = sy
	pl.InitPipeline()
}

func (pl *Pipeline) InitPipeline() {
	pl.SetGraphicsDefaults()
}

// Config is called once all the VkConfig options have been set
// using Set* methods, and the shaders have been loaded.
// The parent System has already done what it can for its config
func (pl *Pipeline) Config() {
	if pl.VkPipeline != nil {
		return // note: it is not possible to reconfig without loading shaders!
	}
	pl.ConfigStages()
	if pl.Sys.Compute {
		pl.ConfigCompute()
		return
	}

	vars := pl.Vars()

	pl.VkConfig.SType = vk.StructureTypeGraphicsPipelineCreateInfo
	pl.VkConfig.PVertexInputState = vars.VkVertexConfig()
	pl.VkConfig.Layout = vars.VkDescLayout
	pl.VkConfig.RenderPass = pl.Sys.Render.VkClearPass
	pl.VkConfig.PMultisampleState = &vk.PipelineMultisampleStateCreateInfo{
		SType:                vk.StructureTypePipelineMultisampleStateCreateInfo,
		RasterizationSamples: pl.Sys.Render.Format.Samples,
	}
	pl.VkConfig.PViewportState = &vk.PipelineViewportStateCreateInfo{
		SType:         vk.StructureTypePipelineViewportStateCreateInfo,
		ScissorCount:  1,
		ViewportCount: 1,
	}
	if pl.Sys.Render.HasDepth {
		pl.VkConfig.PDepthStencilState = &vk.PipelineDepthStencilStateCreateInfo{
			SType:                 vk.StructureTypePipelineDepthStencilStateCreateInfo,
			DepthTestEnable:       vk.True,
			DepthWriteEnable:      vk.True,
			DepthCompareOp:        vk.CompareOpLessOrEqual,
			DepthBoundsTestEnable: vk.False,
			Back: vk.StencilOpState{
				FailOp:    vk.StencilOpKeep,
				PassOp:    vk.StencilOpKeep,
				CompareOp: vk.CompareOpAlways,
			},
			StencilTestEnable: vk.False,
			Front: vk.StencilOpState{
				FailOp:    vk.StencilOpKeep,
				PassOp:    vk.StencilOpKeep,
				CompareOp: vk.CompareOpAlways,
			},
		}
	}

	var pipelineCache vk.PipelineCache
	ret := vk.CreatePipelineCache(pl.Sys.Device.Device, &vk.PipelineCacheCreateInfo{
		SType: vk.StructureTypePipelineCacheCreateInfo,
	}, nil, &pipelineCache)
	IfPanic(NewError(ret))
	pl.VkCache = pipelineCache

	pipeline := make([]vk.Pipeline, 1)
	ret = vk.CreateGraphicsPipelines(pl.Sys.Device.Device, pl.VkCache, 1, []vk.GraphicsPipelineCreateInfo{pl.VkConfig}, nil, pipeline)

	IfPanic(NewError(ret))
	pl.VkPipeline = pipeline[0]

	pl.FreeShaders() // not needed once built
}

// ConfigCompute does the configuration for a Compute pipeline
func (pl *Pipeline) ConfigCompute() {
	var pipelineCache vk.PipelineCache
	ret := vk.CreatePipelineCache(pl.Sys.Device.Device, &vk.PipelineCacheCreateInfo{
		SType: vk.StructureTypePipelineCacheCreateInfo,
	}, nil, &pipelineCache)
	IfPanic(NewError(ret))
	pl.VkCache = pipelineCache

	pipeline := make([]vk.Pipeline, 1)
	cfg := vk.ComputePipelineCreateInfo{
		SType:  vk.StructureTypeComputePipelineCreateInfo,
		Layout: pl.Vars().VkDescLayout,
		Stage:  pl.VkConfig.PStages[0], // note: only one allowed
	}
	ret = vk.CreateComputePipelines(pl.Sys.Device.Device, pl.VkCache, 1, []vk.ComputePipelineCreateInfo{cfg}, nil, pipeline)
	IfPanic(NewError(ret))
	pl.VkPipeline = pipeline[0]

	pl.FreeShaders() // not needed once built
}

// ConfigStages configures the shader stages
func (pl *Pipeline) ConfigStages() {
	ns := len(pl.Shaders)
	pl.VkConfig.StageCount = uint32(ns)
	stgs := make([]vk.PipelineShaderStageCreateInfo, ns)
	for i, sh := range pl.Shaders {
		stgs[i] = vk.PipelineShaderStageCreateInfo{
			SType:  vk.StructureTypePipelineShaderStageCreateInfo,
			Stage:  ShaderStageFlags[sh.Type],
			Module: sh.VkModule,
			PName:  "main\x00",
		}
	}
	pl.VkConfig.PStages = stgs
}

//////////////////////////////////////////////////////////////
// Set graphics options

// SetGraphicsDefaults configures all the default settings for a
// graphics rendering pipeline (not for a compute pipeline)
func (pl *Pipeline) SetGraphicsDefaults() {
	pl.SetDynamicState()
	pl.SetTopology(TriangleList, false)
	pl.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0)
	pl.SetColorBlend(true) // alpha blending
}

// SetDynamicState sets dynamic state (Scissor, Viewport, what else?)
func (pl *Pipeline) SetDynamicState() {
	pl.VkConfig.PDynamicState = &vk.PipelineDynamicStateCreateInfo{
		SType:             vk.StructureTypePipelineDynamicStateCreateInfo,
		DynamicStateCount: 2,
		PDynamicStates: []vk.DynamicState{
			vk.DynamicStateScissor,
			vk.DynamicStateViewport,
		},
	}
}

// SetTopology sets the topology of vertex position data.
// TriangleList is the default.
// Also for Strip modes, restartEnable allows restarting a new
// strip by inserting a ??
func (pl *Pipeline) SetTopology(topo Topologies, restartEnable bool) {
	rese := vk.False
	if restartEnable {
		rese = vk.True
	}
	pl.VkConfig.PInputAssemblyState = &vk.PipelineInputAssemblyStateCreateInfo{
		SType:                  vk.StructureTypePipelineInputAssemblyStateCreateInfo,
		Topology:               vk.PrimitiveTopology(topo),
		PrimitiveRestartEnable: vk.Bool32(rese),
	}
}

// SetRasterization sets various options for how to rasterize shapes:
// Defaults are: vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0
// There are also separate methods for CullFace, FrontFace, and LineWidth
func (pl *Pipeline) SetRasterization(polygonMode vk.PolygonMode, cullMode vk.CullModeFlagBits, frontFace vk.FrontFace, lineWidth float32) {
	pl.VkConfig.PRasterizationState = &vk.PipelineRasterizationStateCreateInfo{
		SType:       vk.StructureTypePipelineRasterizationStateCreateInfo,
		PolygonMode: polygonMode,
		CullMode:    vk.CullModeFlags(cullMode),
		FrontFace:   frontFace,
		LineWidth:   lineWidth,
	}
}

const (
	// CullBack is for SetCullFace function
	CullBack = true

	// CullFront is for SetCullFace function
	CullFront = false

	// CCW is for SetFrontFace function
	CCW = true

	// CW is for SetFrontFace function
	CW = false
)

// SetCullFace sets the face culling mode: true = back, false = front
// use CullBack, CullFront constants
func (pl *Pipeline) SetCullFace(back bool) {
	cm := vk.CullModeFrontBit
	if back {
		cm = vk.CullModeBackBit
	}
	pl.VkConfig.PRasterizationState.CullMode = vk.CullModeFlags(cm)
}

// SetFrontFace sets the winding order for what counts as a front face
// true = CCW, false = CW
func (pl *Pipeline) SetFrontFace(ccw bool) {
	cm := vk.FrontFaceClockwise
	if ccw {
		cm = vk.FrontFaceCounterClockwise
	}
	pl.VkConfig.PRasterizationState.FrontFace = cm
}

// SetLineWidth sets the rendering line width -- 1 is default.
func (pl *Pipeline) SetLineWidth(lineWidth float32) {
	pl.VkConfig.PRasterizationState.LineWidth = lineWidth
}

// SetColorBlend determines the color blending function:
// either 1-source alpha (alphaBlend) or no blending:
// new color overwrites old.  Default is alphaBlend = true
func (pl *Pipeline) SetColorBlend(alphaBlend bool) {
	var cb vk.PipelineColorBlendAttachmentState
	cb.ColorWriteMask = 0xF

	if alphaBlend {
		cb.BlendEnable = vk.True
		cb.SrcColorBlendFactor = vk.BlendFactorOne // vk.BlendFactorSrcAlpha -- that is traditional
		cb.DstColorBlendFactor = vk.BlendFactorOneMinusSrcAlpha
		cb.ColorBlendOp = vk.BlendOpAdd
		cb.SrcAlphaBlendFactor = vk.BlendFactorOne
		cb.DstAlphaBlendFactor = vk.BlendFactorZero
		cb.AlphaBlendOp = vk.BlendOpAdd
		cb.ColorWriteMask = 0xF
	} else {
		cb.BlendEnable = vk.False
	}

	pl.VkConfig.PColorBlendState = &vk.PipelineColorBlendStateCreateInfo{
		SType:           vk.StructureTypePipelineColorBlendStateCreateInfo,
		LogicOpEnable:   vk.False,
		AttachmentCount: 1,
		PAttachments:    []vk.PipelineColorBlendAttachmentState{cb},
	}
}

////////////////////////////////////////////////////////
// Graphics render

// BindPipeline adds commands to the given command buffer to bind
// this pipeline to command buffer.
// System BeginRenderPass must have been called at some point before this.
func (pl *Pipeline) BindPipeline(cmd vk.CommandBuffer) {
	vk.CmdBindPipeline(cmd, vk.PipelineBindPointGraphics, pl.VkPipeline)
}

// Push pushes given value as a push constant for given
// registered push constant variable.
// Note: it is *essential* to use a local, stack variable for the push value
// as cgo will likely complain if it is inside some other structure.
// BindPipeline must have been called before this.
func (pl *Pipeline) Push(cmd vk.CommandBuffer, vr *Var, val unsafe.Pointer) {
	vs := pl.Vars()
	vk.CmdPushConstants(cmd, vs.VkDescLayout, vk.ShaderStageFlags(vr.Shaders), uint32(vr.Offset), uint32(vr.SizeOf), val)
}

// Draw adds CmdDraw command to the given command buffer
// BindPipeline must have been called before this.
// SeeDrawVertex for more typical case using Vertex (and Index) variables.
func (pl *Pipeline) Draw(cmd vk.CommandBuffer, vtxCount, instanceCount, firstVtx, firstInstance int) {
	vk.CmdDraw(cmd, uint32(vtxCount), uint32(instanceCount), uint32(firstVtx), uint32(firstInstance))
}

// DrawVertex adds commands to the given command buffer
// to bind vertex / index values and Draw based on current BindVertexVal
// setting for any Vertex (and associated Index) Vars,
// for given descIdx set of descriptors (see Vars NDescs for info).
func (pl *Pipeline) DrawVertex(cmd vk.CommandBuffer, descIdx int) {
	vs := pl.Vars()
	if !vs.HasVertex {
		return
	}
	st := vs.SetMap[VertexSet]
	var offs []vk.DeviceSize
	var idxVar *Var
	var idxVal *Val
	if len(st.RoleMap[Index]) == 1 {
		idxVar = st.RoleMap[Index][0]
		idxVal, _ = idxVar.BindVal(descIdx)
	}
	vtxn := 0
	for _, vr := range st.Vars {
		vl, err := vr.BindVal(descIdx)
		if err != nil || vr.Role != Vertex {
			continue
		}
		offs = append(offs, vk.DeviceSize(vl.Offset))
		if vtxn == 0 {
			vtxn = vl.N
		} else {
			vtxn = ints.MinInt(vtxn, vl.N)
		}
	}
	mbuf := pl.Sys.Mem.Buffs[VtxIdxBuff].Dev
	vtxbuf := make([]vk.Buffer, len(offs))
	for i := range vtxbuf {
		vtxbuf[i] = mbuf
	}
	vk.CmdBindVertexBuffers(cmd, 0, uint32(len(offs)), vtxbuf, offs)
	if idxVal != nil {
		vktyp := idxVar.Type.VkIndexType()
		vk.CmdBindIndexBuffer(cmd, mbuf, vk.DeviceSize(idxVal.Offset), vktyp)
		vk.CmdDrawIndexed(cmd, uint32(idxVal.N), 1, 0, 0, 0)
	} else {
		vk.CmdDraw(cmd, uint32(vtxn), 1, 0, 0)
	}
}

// BindDrawVertex adds commands to the given command buffer
// to bind this pipeline, and then bind vertex / index values and Draw
// based on current vals for any Vertex (and associated Index) Vars.
// for given descIdx set of descriptors (see Vars NDescs for info).
// This is the standard unit of drawing between Begin and End.
func (pl *Pipeline) BindDrawVertex(cmd vk.CommandBuffer, descIdx int) {
	pl.BindPipeline(cmd)
	pl.DrawVertex(cmd, descIdx)
}
