// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"log/slog"
	"unsafe"

	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// GraphicsPipeline is a Pipeline specifically for the Graphics stack.
// In this context, each pipeline could handle a different
// class of materials (textures, Phong lighting, etc).
// There must be two shader-names 
type GraphicsPipeline struct {
	Pipeline

	// Primitive has various settings for graphics primitives,
	// e.g., TriangleList
	Primitive wgpu.PrimitiveState

	Multisample wgpu.MultisampleState

	renderPipeline *wgpu.RenderPipeline
}

// VertexEntry returns the [ShaderEntry] for [VertexShader].
// Can be nil if no vertex shader defined.
func (pl *GraphicsPipeline) VertexEntry() *ShaderEntry {
	return pl.EntryByType(VertexShader)
}

// FragmentEntry returns the [ShaderEntry] for [FragmentShader].
// Can be nil if no vertex shader defined.
func (pl *GraphicsPipeline) FragmentEntry() *ShaderEntry {
	return pl.EntryByType(FragmentShader)
}

// Config is called once all the VkConfig options have been set
// using Set* methods, and the shaders have been loaded.
// The parent System has already done what it can for its config.
// The rebuild flag indicates whether pipelines should rebuild,
// e.g., based on NTextures changing.
func (pl *GraphicsPipeline) Config(rebuild bool) {
	if pl.RenderPipeline != nil {
		if !rebuild {
			return
		}
		pl.ReleasePipeline() // starting over: note: requires keeping shaders around
	}
	err := pl.BindLayout()
	if err != nil {
		return err
	}
	pd := &wgpu.RenderPipelineDescriptor{
		Label:  pl.Name,
		Layout: pl.layout,
		Primitive: pl.Primitive,
		Multisample: pl.Multisample
	}
	ve := pl.VertexEntry()
	if ve != nil {
		vtxLay := pl.Vars().VertexLayout()
		// todo: err if vtxlay is nil
		pd.Vertex = &wgpu.VertexState{
			Module:     ve.Shader.module,
			EntryPoint: ve.Entry,
			Buffers:    vtxLay,
		}
	}
	fe := pl.FragmentEntry()
	if fe != nil {
		pd.Fragment = &wgpu.FragmentState{
			Module:     ve.Shader.module,
			EntryPoint: ve.Entry
			Targets: []wgpu.ColorTargetState{{
				Format:    s.config.Format, // todo
				Blend:     &wgpu.BlendState_Replace, // todo
				WriteMask: wgpu.ColorWriteMask_All,  // todo
			}},
		},
	})
	rp, err := s.device.CreateRenderPipeline(pd)
	if err != nil {
		slog.Error(err)
		return err
	}
	pl.renderPipeline = rp
	return nil
}

// pl.VkConfig.SType = vk.StructureTypeGraphicsPipelineCreateInfo
// pl.VkConfig.PVertexInputState = vars.VkVertexConfig()
// pl.VkConfig.Layout = vars.VkDescLayout
// pl.VkConfig.RenderPass = pl.Sys.Render.VkClearPass
// pl.VkConfig.PMultisampleState = &vk.PipelineMultisampleStateCreateInfo{
// 	SType:                vk.StructureTypePipelineMultisampleStateCreateInfo,
// 	RasterizationSamples: pl.Sys.Render.Format.Samples,
// }
// pl.VkConfig.PViewportState = &vk.PipelineViewportStateCreateInfo{
// 	SType:         vk.StructureTypePipelineViewportStateCreateInfo,
// 	ScissorCount:  1,
// 	ViewportCount: 1,
// }
// if pl.Sys.Render.HasDepth {
// 	pl.VkConfig.PDepthStencilState = &vk.PipelineDepthStencilStateCreateInfo{
// 		SType:                 vk.StructureTypePipelineDepthStencilStateCreateInfo,
// 		DepthTestEnable:       vk.True,
// 		DepthWriteEnable:      vk.True,
// 		DepthCompareOp:        vk.CompareOpLessOrEqual,
// 		DepthBoundsTestEnable: vk.False,
// 		Back: vk.StencilOpState{
// 			FailOp:    vk.StencilOpKeep,
// 			PassOp:    vk.StencilOpKeep,
// 			CompareOp: vk.CompareOpAlways,
// 		},
// 		StencilTestEnable: vk.False,
// 		Front: vk.StencilOpState{
// 			FailOp:    vk.StencilOpKeep,
// 			PassOp:    vk.StencilOpKeep,
// 			CompareOp: vk.CompareOpAlways,
// 		},
// 	}
// }

func (pl *GraphicsPipeline) Release() {
	pl.ReleaseShaders()
	pl.ReleasePipeline()
}

func (pl *GraphicsPipeline) ReleasePipeline() {
	if pl.Layout != nil {
		pl.Layout.Release()
		pl.Layout = nil
	}
	if pl.RenderPipeline != nil {
		pl.RenderPipeline.Release()
		pl.RenderPipeline = nil
	}
}

// Init initializes pipeline as part of given System
func (pl *GraphicsPipeline) Init(sy *System) {
	pl.Sys = sy
	pl.InitPipeline()
}

func (pl *GraphicsPipeline) InitPipeline() {
	pl.SetGraphicsDefaults()
}

//////////////////////////////////////////////////////////////
// Set graphics options

// SetGraphicsDefaults configures all the default settings for a
// graphics rendering pipeline (not for a compute pipeline)
func (pl *GraphicsPipeline) SetGraphicsDefaults() {
	pl.SetDynamicState()
	pl.SetTopology(TriangleList, false)
	pl.SetFrontFace(true)
	pl.SetCullFace(true)
	// pl.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0)
	// pl.SetColorBlend(true) // alpha blending
}

// SetTopology sets the topology of vertex position data.
// TriangleList is the default.
// Also for Strip modes, restartEnable allows restarting a new
// strip by inserting a ??
func (pl *GraphicsPipeline) SetTopology(topo Topologies, restartEnable bool) {
	pl.Primitive.Topology = topo.Primitive()
}

// SetFrontFace sets the winding order for what counts as a front face
// true = CCW, false = CW
func (pl *GraphicsPipeline) SetFrontFace(ccw bool) {
	cm := wgpu.FrontFace_CW
	if ccw {
		cm = wgpu.FrontFace_CCW
	}
	pl.Primitive.FrontFace = cm
}

// SetCullFace sets the face culling mode: true = back, false = front
// use CullBack, CullFront constants
func (pl *GraphicsPipeline) SetCullFace(back bool) {
	cm := wgpu.CullMode_Front
	if back {
		cm = wgpu.CullMode_Back
	}
	pl.Primitive.CullMode = cm
}

func (pl *GraphicsPipeline) SetMultisample(ms int) {
	pl.Multisample.Count = max(1, ms)
	pl.Multsample.Mask = 0xFFFFFFFF // todo
	pl.AlphaToCoverageEnabled = false // todo
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

// SetLineWidth sets the rendering line width -- 1 is default.
// func (pl *GraphicsPipeline) SetLineWidth(lineWidth float32) {
// 	pl.VkConfig.PRasterizationState.LineWidth = lineWidth
// }

// SetColorBlend determines the color blending function:
// either 1-source alpha (alphaBlend) or no blending:
// new color overwrites old.  Default is alphaBlend = true
func (pl *GraphicsPipeline) SetColorBlend(alphaBlend bool) {
	// todo:
	// var cb vk.PipelineColorBlendAttachmentState
	// cb.ColorWriteMask = 0xF
// 
	// if alphaBlend {
	// 	cb.BlendEnable = vk.True
	// 	cb.SrcColorBlendFactor = vk.BlendFactorOne // vk.BlendFactorSrcAlpha -- that is traditional
	// 	cb.DstColorBlendFactor = vk.BlendFactorOneMinusSrcAlpha
	// 	cb.ColorBlendOp = vk.BlendOpAdd
	// 	cb.SrcAlphaBlendFactor = vk.BlendFactorOne
	// 	cb.DstAlphaBlendFactor = vk.BlendFactorZero
	// 	cb.AlphaBlendOp = vk.BlendOpAdd
	// 	cb.ColorWriteMask = 0xF
	// } else {
	// 	cb.BlendEnable = vk.False
	// }
}

////////////////////////////////////////////////////////
// Graphics render

// todo: here

// BindPipeline adds commands to the given command buffer to bind
// this pipeline to command buffer.
// System BeginRenderPass must have been called at some point before this.
func (pl *GraphicsPipeline) BindPipeline(cmd *wgpu.CommandEncoder) {
	vk.CmdBindPipeline(cmd, vk.PipelineBindPointGraphics, pl.VkPipeline)
}

// Push pushes given value as a push constant for given
// registered push constant variable.
// Note: it is *essential* to use a local, stack variable for the push value
// as cgo will likely complain if it is inside some other structure.
// BindPipeline must have been called before this.
// func (pl *GraphicsPipeline) Push(cmd *wgpu.CommandEncoder, vr *Var, val unsafe.Pointer) {
// 	vs := pl.Vars()
// 	vk.CmdPushConstants(cmd, vs.VkDescLayout, vk.ShaderStageFlags(vr.Shaders), uint32(vr.Offset), uint32(vr.SizeOf), val)
// }

// Draw adds CmdDraw command to the given command buffer
// BindPipeline must have been called before this.
// SeeDrawVertex for more typical case using Vertex (and Index) variables.
func (pl *GraphicsPipeline) Draw(cmd *wgpu.CommandEncoder, vtxCount, instanceCount, firstVtx, firstInstance int) {
	vk.CmdDraw(cmd, uint32(vtxCount), uint32(instanceCount), uint32(firstVtx), uint32(firstInstance))
}

// DrawVertex adds commands to the given command buffer
// to bind vertex / index values and Draw based on current BindVertexValue
// setting for any Vertex (and associated Index) Vars,
// for given descIndex set of descriptors (see Vars NDescs for info).
func (pl *GraphicsPipeline) DrawVertex(cmd *wgpu.CommandEncoder, descIndex int) {
	vs := pl.Vars()
	if !vs.HasVertex {
		return
	}
	st := vs.SetMap[VertexSet]
	var offs []vk.DeviceSize
	var idxVar *Var
	var idxValue *Value
	if len(st.RoleMap[Index]) == 1 {
		idxVar = st.RoleMap[Index][0]
		idxValue, _ = idxVar.BindValue(descIndex)
	}
	vtxn := 0
	for _, vr := range st.Vars {
		vl, err := vr.BindValue(descIndex)
		if err != nil || vr.Role != Vertex {
			continue
		}
		offs = append(offs, vk.DeviceSize(vl.Offset))
		if vtxn == 0 {
			vtxn = vl.N
		} else {
			vtxn = min(vtxn, vl.N)
		}
	}
	mbuf := pl.Sys.Mem.Buffs[IndexBuffer].Dev
	vtxbuf := make([]vk.Buffer, len(offs))
	for i := range vtxbuf {
		vtxbuf[i] = mbuf
	}
	vk.CmdBindVertexBuffers(cmd, 0, uint32(len(offs)), vtxbuf, offs)
	if idxValue != nil {
		vktyp := idxVar.Type.VkIndexType()
		vk.CmdBindIndexBuffer(cmd, mbuf, vk.DeviceSize(idxValue.Offset), vktyp)
		vk.CmdDrawIndexed(cmd, uint32(idxValue.N), 1, 0, 0, 0)
	} else {
		vk.CmdDraw(cmd, uint32(vtxn), 1, 0, 0)
	}
}

// BindDrawVertex adds commands to the given command buffer
// to bind this pipeline, and then bind vertex / index values and Draw
// based on current vals for any Vertex (and associated Index) Vars.
// for given descIndex set of descriptors (see Vars NDescs for info).
// This is the standard unit of drawing between Begin and End.
func (pl *GraphicsPipeline) BindDrawVertex(cmd *wgpu.CommandEncoder, descIndex int) {
	pl.BindPipeline(cmd)
	pl.DrawVertex(cmd, descIndex)
}


// Topologies are the different vertex topology
type Topologies int32 //enum:enum

const (
	PointList       Topologies = iota
	LineList                   
	LineStrip                  
	TriangleList               
	TriangleStrip              
)

func(tp Topologies) Primitive() wgpu.PrimitiveTopology {
	return WebGPUTopologies[tp]
}

var WebGPUTopologies = map[Topologies]wgpu.PrimitiveTopology{
	PointList: wgpu.PrimitiveTopology_PointList,
	LineList: wgpu.PrimitiveTopology_LineList,
	LineStrip: wgpu.PrimitiveTopology_LineStrip,
	TriangleList: wgpu.PrimitiveTopology_TriangleList
	TriangleStrip: wgpu.PrimitiveTopology_TriangleStrip
}

