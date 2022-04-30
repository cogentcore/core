// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"log"

	vk "github.com/vulkan-go/vulkan"
)

// Pipeline manages a sequence of compute steps, which are fixed once configured.
// Each has an associated set of Vars, which could be maintained collectively for
// multiple different such piplines.
type Pipeline struct {
	Name       string             `desc:"unique name of this pipeline"`
	Sys        *System            `desc:"system that we belong to"`
	Device     Device             `desc:"device for this pipeline -- could be GPU or Compute"`
	CmdPool    CmdPool            `desc:"cmd pool specific to this pipeline"`
	Shaders    []*Shader          `desc:"shaders in order added -- should be execution order"`
	ShaderMap  map[string]*Shader `desc:"shaders loaded for this pipeline"`
	RenderPass RenderPass         `desc:"rendering info and depth buffer for this pipeline"`

	VkConfig   vk.GraphicsPipelineCreateInfo `desc:"vulkan pipeline configuration options"`
	VkPipeline vk.Pipeline                   `desc:"the created vulkan pipeline"`
	VkCache    vk.PipelineCache              `desc:"cache"`
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
	pl.CmdPool.Init(&pl.Sys.Device, 0)
	pl.CmdPool.MakeBuff(&pl.Sys.Device)
}

// Config is called once all the VkConfig options have been set
// using Set* methods, and the shaders have been loaded.
// The parent System has already done what it can for its config
func (pl *Pipeline) Config() {
	pl.ConfigStages()
	pl.VkConfig.SType = vk.StructureTypeGraphicsPipelineCreateInfo
	pl.VkConfig.PVertexInputState = pl.Sys.Vars.VkVertexConfig()
	pl.VkConfig.Layout = pl.Sys.Vars.VkDescLayout
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

	var pipelineCache vk.PipelineCache
	ret := vk.CreatePipelineCache(pl.Sys.Device.Device, &vk.PipelineCacheCreateInfo{
		SType: vk.StructureTypePipelineCacheCreateInfo,
	}, nil, &pipelineCache)
	IfPanic(NewError(ret))
	pl.VkCache = pipelineCache

	pipeline := make([]vk.Pipeline, 1)
	if pl.Sys.Compute {
		cfg := vk.ComputePipelineCreateInfo{
			SType:  vk.StructureTypeComputePipelineCreateInfo,
			Layout: pl.Sys.Vars.VkDescLayout,
			Stage:  pl.VkConfig.PStages[0], // note: only one allowed
		}
		ret = vk.CreateComputePipelines(pl.Sys.Device.Device, pl.VkCache, 1, []vk.ComputePipelineCreateInfo{cfg}, nil, pipeline)
	} else {
		ret = vk.CreateGraphicsPipelines(pl.Sys.Device.Device, pl.VkCache, 1, []vk.GraphicsPipelineCreateInfo{pl.VkConfig}, nil, pipeline)

	}
	IfPanic(NewError(ret))
	pl.VkPipeline = pipeline[0]

	pl.FreeShaders() // not needed once built
}

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
// Set options

// SetGraphicsDefaults configures all the default settings for a
// graphics rendering pipeline (not for a compute pipeline)
func (pl *Pipeline) SetGraphicsDefaults() {
	pl.SetDynamicState()
	pl.SetTopology(TriangleList, false)
	pl.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0)
	pl.SetColorBlend(true) // alpha blending
}

/* todo: this is needed for rendering to something
func (s *SpinningCube) drawBuildCommandBuffer(res *as.SwapchainSurfaceFrames, cmd vk.CommandBuffer) {
	ret := vk.BeginCommandBuffer(cmd, &vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
		Flags: vk.CommandBufferUsageFlags(vk.CommandBufferUsageSimultaneousUseBit),
	})
	orPanic(as.NewError(ret))

	clearValues := make([]vk.ClearValue, 2)
	clearValues[1].SetDepthStencil(1, 0)
	clearValues[0].SetColor([]float32{
		0.2, 0.2, 0.2, 0.2,
	})
	vk.CmdBeginRenderPass(cmd, &vk.RenderPassBeginInfo{
		SType:       vk.StructureTypeRenderPassBeginInfo,
		RenderPass:  s.renderPass,
		Framebuffer: res.Framebuffer(),
		RenderArea: vk.Rect2D{
			Offset: vk.Offset2D{
				X: 0, Y: 0,
			},
			Extent: vk.Extent2D{
				Width:  s.width,
				Height: s.height,
			},
		},
		ClearValueCount: 2,
		PClearValues:    clearValues,
	}, vk.SubpassContentsInline)

	vk.CmdBindPipeline(cmd, vk.PipelineBindPointGraphics, s.pipeline)
	vk.CmdBindDescriptorSets(cmd, vk.PipelineBindPointGraphics, s.pipelineLayout,
		0, 1, []vk.DescriptorSet{res.DescriptorSet()}, 0, nil)
	vk.CmdSetViewport(cmd, 0, 1, []vk.Viewport{{
		Width:    float32(s.width),
		Height:   float32(s.height),
		MinDepth: 0.0,
		MaxDepth: 1.0,
	}})
	vk.CmdSetScissor(cmd, 0, 1, []vk.Rect2D{{
		Offset: vk.Offset2D{
			X: 0, Y: 0,
		},
		Extent: vk.Extent2D{
			Width:  s.width,
			Height: s.height,
		},
	}})

	vk.CmdDraw(cmd, 12*3, 1, 0, 0)
	// Note that ending the renderpass changes the image's layout from
	// vk.ImageLayoutColorAttachmentOptimal to vk.ImageLayoutPresentSrc
	vk.CmdEndRenderPass(cmd)

	graphicsQueueIndex := s.Context().Platform().GraphicsQueueFamilyIndex()
	presentQueueIndex := s.Context().Platform().PresentQueueFamilyIndex()
	if graphicsQueueIndex != presentQueueIndex {
		// Separate Present Queue Case
		//
		// We have to transfer ownership from the graphics queue family to the
		// present queue family to be able to present.  Note that we don't have
		// to transfer from present queue family back to graphics queue family at
		// the start of the next frame because we don't care about the image's
		// contents at that point.
		vk.CmdPipelineBarrier(cmd,
			vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
			vk.PipelineStageFlags(vk.PipelineStageBottomOfPipeBit),
			0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{{
				SType:               vk.StructureTypeImageMemoryBarrier,
				SrcAccessMask:       0,
				DstAccessMask:       vk.AccessFlags(vk.AccessColorAttachmentWriteBit),
				OldLayout:           vk.ImageLayoutPresentSrc,
				NewLayout:           vk.ImageLayoutPresentSrc,
				SrcQueueFamilyIndex: graphicsQueueIndex,
				DstQueueFamilyIndex: presentQueueIndex,
				SubresourceRange: vk.ImageSubresourceRange{
					AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit),
					LayerCount: 1,
					LevelCount: 1,
				},
				Image: res.Image(),
			}})
	}
	ret = vk.EndCommandBuffer(cmd)
	orPanic(as.NewError(ret))
}
*/

/////////////////////////////////////////////////////////////////

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
func (pl *Pipeline) SetRasterization(polygonMode vk.PolygonMode, cullMode vk.CullModeFlagBits, frontFace vk.FrontFace, lineWidth float32) {
	pl.VkConfig.PRasterizationState = &vk.PipelineRasterizationStateCreateInfo{
		SType:       vk.StructureTypePipelineRasterizationStateCreateInfo,
		PolygonMode: polygonMode,
		CullMode:    vk.CullModeFlags(cullMode),
		FrontFace:   frontFace,
		LineWidth:   lineWidth,
	}
}

// SetColorBlend determines the color blending function: either 1-source alpha (alphaBlend)
// or no blending: new color overwrites old.
// Default is alphaBlend = true
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

func (pl *Pipeline) RunGraphics(fr *Framebuffer, queueIndex uint32) {
	cmd := pl.CmdPool.BeginCmdOneTime()
	vk.CmdBindPipeline(cmd, vk.PipelineBindPointGraphics, pl.VkPipeline)
	clearValues := make([]vk.ClearValue, 2)
	clearValues[1].SetDepthStencil(1, 0)
	clearValues[0].SetColor([]float32{
		0.2, 0.2, 0.2, 0.2,
	})

	w, h := fr.Image.Format.Size32()

	vk.CmdBeginRenderPass(cmd, &vk.RenderPassBeginInfo{
		SType:       vk.StructureTypeRenderPassBeginInfo,
		RenderPass:  pl.Sys.RenderPass.RenderPass,
		Framebuffer: fr.Framebuffer,
		RenderArea: vk.Rect2D{
			Offset: vk.Offset2D{X: 0, Y: 0},
			Extent: vk.Extent2D{Width: w, Height: h},
		},
		ClearValueCount: 2,
		PClearValues:    clearValues,
	}, vk.SubpassContentsInline)

	vk.CmdBindPipeline(cmd, vk.PipelineBindPointGraphics, pl.VkPipeline)

	if len(pl.Sys.Vars.Vars) > 0 {
		vk.CmdBindDescriptorSets(cmd, vk.PipelineBindPointCompute, pl.Sys.Vars.VkDescLayout,
			0, uint32(len(pl.Sys.Vars.VkDescSets)), pl.Sys.Vars.VkDescSets, uint32(len(pl.Sys.Vars.DynOffs)), pl.Sys.Vars.DynOffs)
	}

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

	vk.CmdDraw(cmd, 3, 1, 0, 0) // todo: need to have this all info from input!

	// Note that ending the renderpass changes the image's layout from
	// vk.ImageLayoutColorAttachmentOptimal to vk.ImageLayoutPresentSrc
	vk.CmdEndRenderPass(cmd)

	/*
		// Separate Present Queue Case
		//
		// We have to transfer ownership from the graphics queue family to the
		// present queue family to be able to present.  Note that we don't have
		// to transfer from present queue family back to graphics queue family at
		// the start of the next frame because we don't care about the image's
		// contents at that point.
		vk.CmdPipelineBarrier(cmd,
			vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
			vk.PipelineStageFlags(vk.PipelineStageBottomOfPipeBit),
			0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{{
				SType:               vk.StructureTypeImageMemoryBarrier,
				SrcAccessMask:       0,
				DstAccessMask:       vk.AccessFlags(vk.AccessColorAttachmentWriteBit),
				OldLayout:           vk.ImageLayoutPresentSrc,
				NewLayout:           vk.ImageLayoutPresentSrc,
				SrcQueueFamilyIndex: pl.Sys.Device.QueueIndex,
				DstQueueFamilyIndex: uint32(queueIndex),
				SubresourceRange: vk.ImageSubresourceRange{
					AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit),
					LayerCount: 1,
					LevelCount: 1,
				},
				Image: fr.Image.Image,
			}})
	*/
	ret := vk.EndCommandBuffer(cmd)
	IfPanic(NewError(ret))

	var fence vk.Fence
	ret = vk.CreateFence(pl.Sys.Device.Device, &vk.FenceCreateInfo{
		SType: vk.StructureTypeFenceCreateInfo,
	}, nil, &fence)
	IfPanic(NewError(ret))

	cmdBufs := []vk.CommandBuffer{cmd}
	ret = vk.QueueSubmit(pl.Sys.Device.Queue, 1, []vk.SubmitInfo{{
		SType:              vk.StructureTypeSubmitInfo,
		CommandBufferCount: 1,
		PCommandBuffers:    cmdBufs,
	}}, fence)
	IfPanic(NewError(ret))

	ret = vk.WaitForFences(pl.Sys.Device.Device, 1, []vk.Fence{fence}, vk.True, vk.MaxUint64)
	IfPanic(NewError(ret))
	vk.DestroyFence(pl.Sys.Device.Device, fence, nil)
}

// RunCompute runs the compute shader for given of computational elements
// along 3 dimensions, which are passed as indexes into the shader.
// The values have to be bound to the vars prior to calling this.
func (pl *Pipeline) RunCompute(nx, ny, nz int) {
	cmd := pl.CmdPool.BeginCmdOneTime()
	vk.CmdBindPipeline(cmd, vk.PipelineBindPointCompute, pl.VkPipeline)

	vk.CmdBindDescriptorSets(cmd, vk.PipelineBindPointCompute, pl.Sys.Vars.VkDescLayout,
		0, uint32(len(pl.Sys.Vars.VkDescSets)), pl.Sys.Vars.VkDescSets, uint32(len(pl.Sys.Vars.DynOffs)), pl.Sys.Vars.DynOffs)

	vk.CmdDispatch(cmd, uint32(nx), uint32(ny), uint32(nz))
	pl.CmdPool.SubmitWait(&pl.Sys.Device)
}

/*
func (pl *Pipeline) Run() {
	graphicsQueue := sf.GPU.GraphicsQueue
	var nullFence vk.Fence
	ret = vk.QueueSubmit(graphicsQueue, 1, []vk.SubmitInfo{{
		SType: vk.StructureTypeSubmitInfo,
		PWaitDstStageMask: []vk.SurfaceStageFlags{
			vk.SurfaceStageFlags(vk.SurfaceStageColorAttachmentOutputBit),
		},
		WaitSemaphoreCount: 1,
		PWaitSemaphores: []vk.Semaphore{
			sf.ImageAcquiredSemaphores[sf.FrameIndex],
		},
		CommandBufferCount: 1,
		PCommandBuffers: []vk.CommandBuffer{
			sf.SurfaceFrames[idx].CmdBuff,
		},
		SignalSemaphoreCount: 1,
		PSignalSemaphores: []vk.Semaphore{
			sf.DrawCompleteSemaphores[sf.FrameIndex],
		},
	}}, nullFence)
	IfPanic(NewError(ret))
}
*/
