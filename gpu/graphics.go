// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"log/slog"

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

// NewGraphicsPipeline returns a new GraphicsPipeline.
func NewGraphicsPipeline(name string) *GraphicsPipeline {
	pl := &GraphicsPipeline{}
	pl.Name = name
	return pl
}

// Init initializes pipeline as part of given System
func (pl *GraphicsPipeline) Init(sy *System) {
	pl.Sys = sy
	pl.InitPipeline()
}

func (pl *GraphicsPipeline) InitPipeline() {
	pl.SetGraphicsDefaults()
}

// BindPipeline binds this pipeline as the one to use for next commands in
// the given render pass.
// This also calls BindAllGroups, to bind the Current Value for all variables,
// excluding Vertex level variables: use BindVertex for that.
// Be sure to set the desired Current value prior to calling.
func (pl *GraphicsPipeline) BindPipeline(rp *wgpu.RenderPassEncoder) error {
	if pl.renderPipeline != nil {
		rp.SetPipeline(pl.renderPipeline)
		pl.BindAllGroups(rp)
		return nil
	}
	err := pl.Config(false)
	if err == nil {
		rp.SetPipeline(pl.renderPipeline)
		pl.BindAllGroups(rp)
		return nil
	}
	return err
}

// BindAllGroups binds the Current Value for all variables across all
// variable groups, as the Value to use by shader.
// Automatically called in BindPipeline at start of render for pipeline.
// Be sure to set Current index to correct value before calling!
func (pl *GraphicsPipeline) BindAllGroups(rp *wgpu.RenderPassEncoder) {
	vs := &pl.Sys.Vars
	ngp := vs.NGroups()
	for gi := 0; gi < ngp; gi++ {
		vg := vs.Groups[gi]
		rp.SetBindGroup(uint32(vg.Group), vg.bindGroup(), nil) // note: nil is dynamic offsets
	}
}

// BindGroup binds the Current Value for all variables in given
// variable group, as the Value to use by shader.
// Be sure to set Current index to correct value before calling!
func (pl *GraphicsPipeline) BindGroup(rp *wgpu.RenderPassEncoder, group int) {
	vs := &pl.Sys.Vars
	vg := vs.Groups[group]
	rp.SetBindGroup(uint32(vg.Group), vg.bindGroup(), nil) // note: nil is dynamic offsets
}

// BindDrawVertex binds the Current Value for all VertexGroup variables,
// as the vertex data, and then does a DrawIndexed call.
func (pl *GraphicsPipeline) BindDrawVertex(rp *wgpu.RenderPassEncoder) {
	pl.BindVertex(rp)
	pl.DrawIndexed(rp)
}

// BindVertex binds the Current Value for all VertexGroup variables,
// as the vertex data to use for next DrawIndexed call.
func (pl *GraphicsPipeline) BindVertex(rp *wgpu.RenderPassEncoder) {
	vs := &pl.Sys.Vars
	vg := vs.Groups[VertexGroup]
	if vg == nil {
		return
	}
	for _, vr := range vg.Vars {
		vl := vr.Values.CurrentValue()
		if vr.Role == Index {
			rp.SetIndexBuffer(vl.buffer, vr.Type.IndexType(), 0, wgpu.WholeSize)
		} else {
			rp.SetVertexBuffer(uint32(vr.Binding), vl.buffer, 0, wgpu.WholeSize)
		}
	}
}

func (pl *GraphicsPipeline) DrawIndexed(rp *wgpu.RenderPassEncoder) {
	vs := &pl.Sys.Vars
	vg := vs.Groups[VertexGroup]
	if vg == nil {
		return
	}
	ix := vg.IndexVar()
	if ix == nil {
		return
	}
	iv := ix.Values.CurrentValue()
	rp.DrawIndexed(uint32(iv.N), 1, 0, 0, 0)
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
func (pl *GraphicsPipeline) Config(rebuild bool) error {
	if pl.renderPipeline != nil {
		if !rebuild {
			return nil
		}
		pl.ReleasePipeline() // starting over: note: requires keeping shaders around
	}
	err := pl.BindLayout()
	if err != nil {
		return err
	}
	pd := &wgpu.RenderPipelineDescriptor{
		Label:       pl.Name,
		Layout:      pl.layout,
		Primitive:   pl.Primitive,
		Multisample: pl.Multisample,
	}
	ve := pl.VertexEntry()
	if ve != nil {
		vtxLay := pl.Vars().VertexLayout()
		// todo: err if vtxlay is nil
		pd.Vertex = wgpu.VertexState{
			Module:     ve.Shader.module,
			EntryPoint: ve.Entry,
			Buffers:    vtxLay,
		}
	}
	fe := pl.FragmentEntry()
	if fe != nil {
		pd.Fragment = &wgpu.FragmentState{
			Module:     fe.Shader.module,
			EntryPoint: fe.Entry,
			Targets: []wgpu.ColorTargetState{{
				Format:    pl.Sys.Render.Format.Format,
				Blend:     &wgpu.BlendStateReplace, // todo
				WriteMask: wgpu.ColorWriteMaskAll,  // todo
			}},
		}
	}
	rp, err := pl.Sys.device.Device.CreateRenderPipeline(pd)
	if err != nil {
		slog.Error(err.Error())
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
	if pl.layout != nil {
		pl.layout.Release()
		pl.layout = nil
	}
	if pl.renderPipeline != nil {
		pl.renderPipeline.Release()
		pl.renderPipeline = nil
	}
}

//////////////////////////////////////////////////////////////
// Set graphics options

// SetGraphicsDefaults configures all the default settings for a
// graphics rendering pipeline (not for a compute pipeline)
func (pl *GraphicsPipeline) SetGraphicsDefaults() *GraphicsPipeline {
	pl.SetTopology(TriangleList, false)
	pl.SetFrontFace(wgpu.FrontFaceCCW)
	pl.SetCullMode(wgpu.CullModeBack)
	pl.SetColorBlend(true) // alpha blending
	pl.SetMultisample(1)
	// pl.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0)
	return pl
}

// SetTopology sets the topology of vertex position data.
// TriangleList is the default.
// Also for Strip modes, restartEnable allows restarting a new
// strip by inserting a ??
func (pl *GraphicsPipeline) SetTopology(topo Topologies, restartEnable bool) *GraphicsPipeline {
	pl.Primitive.Topology = topo.Primitive()
	return pl
}

// SetFrontFace sets the winding order for what counts as a front face.
func (pl *GraphicsPipeline) SetFrontFace(face wgpu.FrontFace) *GraphicsPipeline {
	pl.Primitive.FrontFace = face
	return pl
}

// SetCullMode sets the face culling mode.
func (pl *GraphicsPipeline) SetCullMode(mode wgpu.CullMode) *GraphicsPipeline {
	pl.Primitive.CullMode = mode
	return pl
}

func (pl *GraphicsPipeline) SetMultisample(ms int) *GraphicsPipeline {
	pl.Multisample.Count = uint32(max(1, ms))
	pl.Multisample.Mask = 0xFFFFFFFF              // todo
	pl.Multisample.AlphaToCoverageEnabled = false // todo
	return pl
}

// SetLineWidth sets the rendering line width -- 1 is default.
func (pl *GraphicsPipeline) SetLineWidth(lineWidth float32) {
	// pl.VkConfig.PRasterizationState.LineWidth = lineWidth
}

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

// Push pushes given value as a push constant for given
// registered push constant variable.
// Note: it is *essential* to use a local, stack variable for the push value
// as cgo will likely complain if it is inside some other structure.
// BindPipeline must have been called before this.
// func (pl *GraphicsPipeline) Push(cmd *wgpu.CommandEncoder, vr *Var, val unsafe.Pointer) {
// 	vs := pl.Vars()
// 	vk.CmdPushConstants(cmd, vs.VkDescLayout, vk.ShaderStageFlags(vr.Shaders), uint32(vr.Offset), uint32(vr.SizeOf), val)
// }

// DrawVertex adds commands to the given command buffer
// to bind vertex / index values and Draw based on current BindVertexValue
// setting for any Vertex (and associated Index) Vars,
// for given descIndex set of descriptors (see Vars NDescs for info).
func (pl *GraphicsPipeline) DrawVertex(cmd *wgpu.CommandEncoder, descIndex int) {
	vs := pl.Vars()
	if !vs.hasVertex {
		return
	}
	// st := vs.SetMap[VertexSet]
	// var offs []vk.DeviceSize
	// var idxVar *Var
	// var idxValue *Value
	// if len(st.RoleMap[Index]) == 1 {
	// 	idxVar = st.RoleMap[Index][0]
	// 	idxValue, _ = idxVar.BindValue(descIndex)
	// }
	// vtxn := 0
	// for _, vr := range st.Vars {
	// 	vl, err := vr.BindValue(descIndex)
	// 	if err != nil || vr.Role != Vertex {
	// 		continue
	// 	}
	// 	offs = append(offs, vk.DeviceSize(vl.Offset))
	// 	if vtxn == 0 {
	// 		vtxn = vl.N
	// 	} else {
	// 		vtxn = min(vtxn, vl.N)
	// 	}
	// }
	// mbuf := pl.Sys.Mem.Buffs[IndexBuffer].Dev
	// vtxbuf := make([]vk.Buffer, len(offs))
	// for i := range vtxbuf {
	// 	vtxbuf[i] = mbuf
	// }
	// vk.CmdBindVertexBuffers(cmd, 0, uint32(len(offs)), vtxbuf, offs)
	// if idxValue != nil {
	// 	vktyp := idxVar.Type.VkIndexType()
	// 	vk.CmdBindIndexBuffer(cmd, mbuf, vk.DeviceSize(idxValue.Offset), vktyp)
	// 	vk.CmdDrawIndexed(cmd, uint32(idxValue.N), 1, 0, 0, 0)
	// } else {
	// 	vk.CmdDraw(cmd, uint32(vtxn), 1, 0, 0)
	// }
}

// Topologies are the different vertex topology
type Topologies int32 //enum:enum

const (
	PointList Topologies = iota
	LineList
	LineStrip
	TriangleList
	TriangleStrip
)

func (tp Topologies) Primitive() wgpu.PrimitiveTopology {
	return WebGPUTopologies[tp]
}

var WebGPUTopologies = map[Topologies]wgpu.PrimitiveTopology{
	PointList:     wgpu.PrimitiveTopologyPointList,
	LineList:      wgpu.PrimitiveTopologyLineList,
	LineStrip:     wgpu.PrimitiveTopologyLineStrip,
	TriangleList:  wgpu.PrimitiveTopologyTriangleList,
	TriangleStrip: wgpu.PrimitiveTopologyTriangleStrip,
}
