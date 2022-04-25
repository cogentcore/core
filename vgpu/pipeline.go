// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"fmt"
	"log"

	vk "github.com/vulkan-go/vulkan"
)

// Pipeline manages a sequence of compute steps, which are fixed once configured.
// Each has an associated set of Vars, which could be maintained collectively for
// multiple different such piplines.
type Pipeline struct {
	Name      string             `desc:"unique name of this pipeline"`
	Sys       *System            `desc:"system that we belong to"`
	Device    Device             `desc:"device for this pipeline -- could be GPU or Compute"`
	CmdPool   CmdPool            `desc:"cmd pool specific to this pipeline"`
	Shaders   []*Shader          `desc:"shaders in order added -- should be execution order"`
	ShaderMap map[string]*Shader `desc:"shaders loaded for this pipeline"`

	VkConfig     vk.GraphicsPipelineCreateInfo `desc:"vulkan pipeline configuration options"`
	VkPipeline   vk.Pipeline                   `desc:"the created vulkan pipeline"`
	VkCache      vk.PipelineCache              `desc:"cache"`
	VkRenderPass vk.RenderPass                 `desc:"render pass config"`
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
		sh.Free()
	}
}

func (pl *Pipeline) Destroy() {
	pl.FreeShaders()
	pl.CmdPool.Destroy(&pl.Sys.Device)
}

// Init initializes pipeline as part of given System
func (pl *Pipeline) Init(sy *System) {
	pl.Sys = sy
	pl.InitPipeline()
}

func (pl *Pipeline) InitPipeline() {
	pl.CmdPool.Init(&pl.Sys.Device, 0)
	pl.CmdPool.Buff = pl.CmdPool.MakeBuff(&pl.Sys.Device)
}

// Config is called once all the VkConfig options have been set
// using Set* methods, and the shaders have been loaded.
// The parent System has already done what it can for its config
func (pl *Pipeline) Config() {
	pl.ConfigStages()
	pl.VkConfig.SType = vk.StructureTypeGraphicsPipelineCreateInfo
	pl.VkConfig.PVertexInputState = pl.Sys.Vars.VkVertexConfig()
	pl.VkConfig.Layout = pl.Sys.Vars.VkDescLayout

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
			Stage:  pl.VkConfig.PStages[0], // note: only one allowefd
		}
		ret = vk.CreateComputePipelines(pl.Sys.Device.Device, pl.VkCache, 1, []vk.ComputePipelineCreateInfo{cfg}, nil, pipeline)
	} else {
		ret = vk.CreateGraphicsPipelines(pl.Sys.Device.Device, pl.VkCache, 1, []vk.GraphicsPipelineCreateInfo{pl.VkConfig}, nil, pipeline)

	}
	IfPanic(NewError(ret))
	pl.VkPipeline = pipeline[0]

	pl.FreeShaders()
}

func (pl *Pipeline) ConfigStages() {
	ns := len(pl.Shaders)
	pl.VkConfig.StageCount = uint32(ns)
	stgs := make([]vk.PipelineShaderStageCreateInfo, ns)
	for i, sh := range pl.Shaders {
		stgs[i] = vk.PipelineShaderStageCreateInfo{
			SType:  vk.StructureTypePipelineShaderStageCreateInfo,
			Stage:  vk.ShaderStageFlagBits(sh.Type),
			Module: sh.VkModule,
			PName:  "main\x00",
		}
		fmt.Printf("sh type: %v\n", sh.Type)
	}
	pl.VkConfig.PStages = stgs
}

//////////////////////////////////////////////////////////////
// Set options

// SetGraphicsDefaults configures all the default settings for a
// graphics rendering pipeline (not for a compute pipeline)
func (pl *Pipeline) SetGraphicsDefaults() {
	pl.SetRenderPass()
	pl.SetDynamicState()
	pl.SetTopology(TriangleList, false)
	pl.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0)
	pl.SetColorBlend(true) // alpha blending
	pl.SetMultisample(4)
}

// SetRenderPass todo: what is it, what are opts?
func (pl *Pipeline) SetRenderPass() {
	// The initial layout for the color and depth attachments will be vk.LayoutUndefined
	// because at the start of the renderpass, we don't care about their contents.
	// At the start of the subpass, the color attachment's layout will be transitioned
	// to vk.LayoutColorAttachmentOptimal and the depth stencil attachment's layout
	// will be transitioned to vk.LayoutDepthStencilAttachmentOptimal.  At the end of
	// the renderpass, the color attachment's layout will be transitioned to
	// vk.LayoutPresentSrc to be ready to present.  This is all done as part of
	// the renderpass, no barriers are necessary.
	var renderPass vk.RenderPass
	ret := vk.CreateRenderPass(pl.Sys.Device.Device, &vk.RenderPassCreateInfo{
		SType:           vk.StructureTypeRenderPassCreateInfo,
		AttachmentCount: 2,
		PAttachments: []vk.AttachmentDescription{{
			// Format:         s.Context().SwapchainDimensions().Format, // todo!
			Samples:        vk.SampleCount1Bit,
			LoadOp:         vk.AttachmentLoadOpClear,
			StoreOp:        vk.AttachmentStoreOpStore,
			StencilLoadOp:  vk.AttachmentLoadOpDontCare,
			StencilStoreOp: vk.AttachmentStoreOpDontCare,
			InitialLayout:  vk.ImageLayoutUndefined,
			FinalLayout:    vk.ImageLayoutPresentSrc,
		}, {
			// Format:         s.depth.format, // todo!
			Samples:        vk.SampleCount1Bit,
			LoadOp:         vk.AttachmentLoadOpClear,
			StoreOp:        vk.AttachmentStoreOpDontCare,
			StencilLoadOp:  vk.AttachmentLoadOpDontCare,
			StencilStoreOp: vk.AttachmentStoreOpDontCare,
			InitialLayout:  vk.ImageLayoutUndefined,
			FinalLayout:    vk.ImageLayoutDepthStencilAttachmentOptimal,
		}},
		SubpassCount: 1,
		PSubpasses: []vk.SubpassDescription{{
			PipelineBindPoint:    vk.PipelineBindPointGraphics,
			ColorAttachmentCount: 1,
			PColorAttachments: []vk.AttachmentReference{{
				Attachment: 0,
				Layout:     vk.ImageLayoutColorAttachmentOptimal,
			}},
			PDepthStencilAttachment: &vk.AttachmentReference{
				Attachment: 1,
				Layout:     vk.ImageLayoutDepthStencilAttachmentOptimal,
			},
		}},
	}, nil, &renderPass)
	IfPanic(NewError(ret))
	pl.VkRenderPass = renderPass
	pl.VkConfig.RenderPass = pl.VkRenderPass
}

/* todo: this is needed for rendering to something
func (s *SpinningCube) drawBuildCommandBuffer(res *as.SwapchainImageResources, cmd vk.CommandBuffer) {
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

// SetMultisample sets the number of multisampling to decrease aliasing
// Default is 4, which is typically sufficient.  Values must be power of 2.
func (pl *Pipeline) SetMultisample(nsamp int) {
	ns := vk.SampleCount1Bit
	switch nsamp {
	case 2:
		ns = vk.SampleCount2Bit
	case 4:
		ns = vk.SampleCount4Bit
	case 8:
		ns = vk.SampleCount8Bit
	case 16:
		ns = vk.SampleCount16Bit
	case 32:
		ns = vk.SampleCount32Bit
	case 64:
		ns = vk.SampleCount64Bit
	}
	pl.VkConfig.PMultisampleState = &vk.PipelineMultisampleStateCreateInfo{
		SType:                vk.StructureTypePipelineMultisampleStateCreateInfo,
		RasterizationSamples: ns,
	}
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
			sf.ImageResources[idx].CmdBuff,
		},
		SignalSemaphoreCount: 1,
		PSignalSemaphores: []vk.Semaphore{
			sf.DrawCompleteSemaphores[sf.FrameIndex],
		},
	}}, nullFence)
	IfPanic(NewError(ret))
}
*/
