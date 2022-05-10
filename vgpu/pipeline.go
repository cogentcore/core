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
	vk "github.com/vulkan-go/vulkan"
)

// Pipeline manages a sequence of compute steps, which are fixed once configured.
// Each has an associated set of Vars, which could be maintained collectively for
// multiple different such piplines.
type Pipeline struct {
	Name       string             `desc:"unique name of this pipeline"`
	Sys        *System            `desc:"system that we belong to -- use for device, vars, etc"`
	CmdPool    CmdPool            `desc:"cmd pool specific to this pipeline"`
	Shaders    []*Shader          `desc:"shaders in order added -- should be execution order"`
	ShaderMap  map[string]*Shader `desc:"shaders loaded for this pipeline"`
	RenderPass RenderPass         `desc:"rendering info and depth buffer for this pipeline"`
	ClearVals  []vk.ClearValue    `desc:"values for clearing image when starting render pass"`

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

	vk.DestroyPipelineCache(pl.Sys.Device.Device, pl.VkCache, nil)
	pl.RenderPass.Destroy()
	vk.DestroyPipeline(pl.Sys.Device.Device, pl.VkPipeline, nil)
	pl.CmdPool.Destroy(pl.Sys.Device.Device)
}

// Init initializes pipeline as part of given System
func (pl *Pipeline) Init(sy *System) {
	pl.Sys = sy
	pl.InitPipeline()
}

func (pl *Pipeline) InitPipeline() {
	pl.CmdPool.ConfigResettable(&pl.Sys.Device)
	pl.CmdPool.NewBuffer(&pl.Sys.Device)
}

// Config is called once all the VkConfig options have been set
// using Set* methods, and the shaders have been loaded.
// The parent System has already done what it can for its config
func (pl *Pipeline) Config() {
	pl.ConfigStages()
	if pl.Sys.Compute {
		pl.ConfigCompute()
		return
	}

	vars := pl.Vars()

	pl.VkConfig.SType = vk.StructureTypeGraphicsPipelineCreateInfo
	pl.VkConfig.PVertexInputState = vars.VkVertexConfig()
	pl.VkConfig.Layout = vars.VkDescLayout
	pl.VkConfig.RenderPass = pl.Sys.RenderPass.RenderPass
	pl.VkConfig.PMultisampleState = &vk.PipelineMultisampleStateCreateInfo{
		SType:                vk.StructureTypePipelineMultisampleStateCreateInfo,
		RasterizationSamples: pl.Sys.RenderPass.Format.Samples,
	}
	pl.VkConfig.PViewportState = &vk.PipelineViewportStateCreateInfo{
		SType:         vk.StructureTypePipelineViewportStateCreateInfo,
		ScissorCount:  1,
		ViewportCount: 1,
	}
	if pl.Sys.RenderPass.HasDepth {
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
	pl.SetClearColor(0, 0, 0, 1)
	pl.SetClearDepthStencil(1, 0)
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
		cb.SrcColorBlendFactor = vk.BlendFactorSrcAlpha
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

// SetClearOff turns off clearing at start of rendering.
// call SetClearColor to turn back on.
func (pl *Pipeline) SetClearOff() {
	pl.ClearVals = nil
}

// SetClearColor sets the RGBA colors to set when starting new render
func (pl *Pipeline) SetClearColor(r, g, b, a float32) {
	if len(pl.ClearVals) == 0 {
		pl.ClearVals = make([]vk.ClearValue, 2)
	}
	pl.ClearVals[0].SetColor([]float32{r, g, b, a})
}

// SetClearDepthStencil sets the depth and stencil values when starting new render
func (pl *Pipeline) SetClearDepthStencil(depth float32, stencil uint32) {
	if len(pl.ClearVals) == 0 {
		pl.ClearVals = make([]vk.ClearValue, 2)
	}
	pl.ClearVals[1].SetDepthStencil(depth, stencil)
}

////////////////////////////////////////////////////////
// Graphics render

// BeginRenderPass adds commands to the given command buffer
// to start the render pass on given framebuffer.
// Clears the frame according to the ClearVals.
func (pl *Pipeline) BeginRenderPass(cmd vk.CommandBuffer, fr *Framebuffer) {
	w, h := fr.Image.Format.Size32()
	vk.CmdBeginRenderPass(cmd, &vk.RenderPassBeginInfo{
		SType:       vk.StructureTypeRenderPassBeginInfo,
		RenderPass:  pl.Sys.RenderPass.RenderPass,
		Framebuffer: fr.Framebuffer,
		RenderArea: vk.Rect2D{
			Offset: vk.Offset2D{X: 0, Y: 0},
			Extent: vk.Extent2D{Width: w, Height: h},
		},
		ClearValueCount: uint32(len(pl.ClearVals)),
		PClearValues:    pl.ClearVals,
	}, vk.SubpassContentsInline)

	vk.CmdSetViewport(cmd, 0, 1, []vk.Viewport{{
		Width:    float32(w),
		Height:   float32(h),
		MinDepth: 0.0,
		MaxDepth: 1.0,
	}})

	vk.CmdSetScissor(cmd, 0, 1, []vk.Rect2D{{
		Offset: vk.Offset2D{X: 0, Y: 0},
		Extent: vk.Extent2D{Width: w, Height: h},
	}})
}

// BindPipeline adds commands to the given command buffer to bind
// this pipeline to command buffer and bind descriptor sets for variable
// values to use, as determined by the descIdx index (see Vars NDescs for info).
// BeginRenderPass must have been called at some point before this.
func (pl *Pipeline) BindPipeline(cmd vk.CommandBuffer, descIdx int) {
	vk.CmdBindPipeline(cmd, vk.PipelineBindPointGraphics, pl.VkPipeline)
	vs := pl.Vars()
	if len(vs.SetMap) > 0 {
		dset := vs.VkDescSets[descIdx]
		doff := vs.DynOffs[descIdx]
		vk.CmdBindDescriptorSets(cmd, vk.PipelineBindPointGraphics, vs.VkDescLayout,
			0, uint32(len(dset)), dset, uint32(len(doff)), doff)
	}
}

// PushConstant pushes given value as a push constant for given
// registered push constant variable.
// BindPipeline must have been called before this.
func (pl *Pipeline) PushConstant(cmd vk.CommandBuffer, vr *Var, shader vk.ShaderStageFlagBits, val unsafe.Pointer) {
	vs := pl.Vars()
	vk.CmdPushConstants(cmd, vs.VkDescLayout, vk.ShaderStageFlags(shader), uint32(vr.Offset), uint32(vr.SizeOf), val)
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
	pl.BindPipeline(cmd, descIdx)
	pl.DrawVertex(cmd, descIdx)
}

// EndRenderPass adds commands to the given command buffer
// to end the render pass.  It does not call EndCommandBuffer,
// in case any further commands are to be added.
func (pl *Pipeline) EndRenderPass(cmd vk.CommandBuffer) {
	// Note that ending the renderpass changes the image's layout from
	// vk.ImageLayoutColorAttachmentOptimal to vk.ImageLayoutPresentSrc
	vk.CmdEndRenderPass(cmd)
}

// FullStdRender adds commands to the given command buffer
// to perform a full standard render using Vertex input vars:
// CmdReset, CmdBegin, BeginRenderPass, BindPipeline,
// DrawVertex, EndRenderPass, EndCmd.
// for given descIdx set of descriptors (see Vars NDescs for info).
// This is mainly for demo / informational purposes as usually multiple
// pipeline draws are performed between Begin and End.
func (pl *Pipeline) FullStdRender(cmd vk.CommandBuffer, fr *Framebuffer, descIdx int) {
	CmdReset(cmd)
	CmdBegin(cmd)
	pl.BeginRenderPass(cmd, fr)
	pl.BindDrawVertex(cmd, descIdx)
	pl.EndRenderPass(cmd)
	CmdEnd(cmd)
}

///////////////////////////////////////////////////////////////////////////////////////
//  Compute

// ComputeCommand adds commands to run the compute shader for given
// number of computational elements along 3 dimensions,
// which are passed as indexes into the shader.
// The values have to be bound to the vars prior to calling this.
// descIdx index determines which group of var val bindings to use
// (see Vars NDescs for info).
func (pl *Pipeline) ComputeCommand(cmd vk.CommandBuffer, descIdx int, nx, ny, nz int) {
	vk.CmdBindPipeline(cmd, vk.PipelineBindPointCompute, pl.VkPipeline)

	vs := pl.Vars()
	if len(vs.SetMap) > 0 {
		dset := vs.VkDescSets[descIdx]
		doff := vs.DynOffs[descIdx]
		vk.CmdBindDescriptorSets(cmd, vk.PipelineBindPointCompute, vs.VkDescLayout,
			0, uint32(len(dset)), dset, uint32(len(doff)), doff)
	}

	vk.CmdDispatch(cmd, uint32(nx), uint32(ny), uint32(nz))
}

// RunComputeWait runs the compute shader for given
// number of computational elements along 3 dimensions,
// which are passed as indexes into the shader.
// for given descIdx set of descriptors (see Vars NDescs for info).
// The values have to be bound to the vars prior to calling this.
// Submits the run command and waits for the queue to finish so the
// results will be available immediately after this.
func (pl *Pipeline) RunComputeWait(cmd vk.CommandBuffer, descIdx int, nx, ny, nz int) {
	CmdReset(cmd)
	CmdBegin(cmd)
	pl.ComputeCommand(cmd, descIdx, nx, ny, nz)
	CmdSubmitWait(cmd, &pl.Sys.Device)
}
