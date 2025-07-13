// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"cogentcore.org/core/base/errors"
	"github.com/cogentcore/webgpu/wgpu"
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

	// AlphaBlend determines whether to do alpha blending or not.
	AlphaBlend bool

	// renderPipeline is the configured, instantiated wgpu pipeline
	renderPipeline *wgpu.RenderPipeline
}

// NewGraphicsPipeline returns a new GraphicsPipeline.
func NewGraphicsPipeline(name string, sy *GraphicsSystem) *GraphicsPipeline {
	pl := &GraphicsPipeline{}
	pl.Name = name
	pl.System = sy
	pl.SetGraphicsDefaults()
	return pl
}

// BindAllGroups binds the Current Value for all variables across all
// variable groups, as the Value to use by shader.
// Automatically called in BindPipeline at start of render for pipeline.
// Be sure to set Current index to correct value before calling!
func (pl *GraphicsPipeline) BindAllGroups(rp *wgpu.RenderPassEncoder) {
	vs := pl.Vars()
	ngp := vs.NGroups()
	for gi := range ngp {
		pl.BindGroup(rp, gi)
	}
}

// BindGroup binds the Current Value for all variables in given
// variable group, as the Value to use by shader.
// Be sure to set Current index to correct value before calling!
func (pl *GraphicsPipeline) BindGroup(rp *wgpu.RenderPassEncoder, group int) {
	vg := pl.Vars().Groups[group]
	bg, dynOffs, err := pl.bindGroup(vg)
	if err == nil {
		rp.SetBindGroup(uint32(vg.Group), bg, dynOffs)
	}
}

// BindPipeline binds this pipeline as the one to use for next commands in
// the given render pass.
// This also calls BindAllGroups, to bind the Current Value for all variables,
// excluding Vertex level variables: use BindVertex for that.
// Be sure to set the desired Current value prior to calling.
func (pl *GraphicsPipeline) BindPipeline(rp *wgpu.RenderPassEncoder) error {
	if pl.renderPipeline == nil {
		err := pl.Config(false)
		if errors.Log(err) != nil {
			return err
		}
	}
	rp.SetPipeline(pl.renderPipeline)
	pl.BindAllGroups(rp)
	return nil
}

// BindDrawIndexed binds the Current Value for all VertexGroup variables,
// as the vertex data, and then does a DrawIndexed call.
func (pl *GraphicsPipeline) BindDrawIndexed(rp *wgpu.RenderPassEncoder) {
	pl.BindVertex(rp)
	pl.DrawIndexed(rp)
}

// BindVertex binds the Current Value for all VertexGroup variables,
// as the vertex data to use for next DrawIndexed call.
func (pl *GraphicsPipeline) BindVertex(rp *wgpu.RenderPassEncoder) {
	vs := pl.Vars()
	vg := vs.Groups[VertexGroup]
	if vg == nil {
		return
	}
	for _, vr := range vg.Vars {
		vl := vr.Values.CurrentValue()
		if vr.Role == Index {
			rp.SetIndexBuffer(vl.buffer, vr.Type.IndexType(), 0, wgpu.WholeSize)
		} else {
			if vl.buffer != nil {
				rp.SetVertexBuffer(uint32(vr.Binding), vl.buffer, 0, wgpu.WholeSize)
			}
		}
	}
}

// DrawVertex adds commands to the given command encoder
// to Draw based on current Index and Vertex values.
func (pl *GraphicsPipeline) DrawIndexed(rp *wgpu.RenderPassEncoder) {
	vs := pl.Vars()
	vg := vs.Groups[VertexGroup]
	if vg == nil {
		return
	}
	ix := vg.IndexVar()
	if ix == nil {
		return
	}
	iv := ix.Values.CurrentValue()
	rp.DrawIndexed(uint32(iv.dynamicN), 1, 0, 0, 0)
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

// Config is called once all the Config options have been set
// using Set* methods, and the shaders have been loaded.
// The parent GraphicsSystem has already done what it can for its config.
// The rebuild flag indicates whether pipelines should rebuild
func (pl *GraphicsPipeline) Config(rebuild bool) error {
	if pl.renderPipeline != nil {
		if !rebuild {
			return nil
		}
		pl.releasePipeline() // starting over: note: requires keeping shaders around
	}
	play, err := pl.bindLayout()
	if errors.Log(err) != nil {
		return err
	}
	defer play.Release()

	pl.Multisample.Count = uint32(pl.System.Render().Format.Samples)
	pd := &wgpu.RenderPipelineDescriptor{
		Label:       pl.Name,
		Layout:      play,
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
				Format:    pl.System.Render().Format.Format,
				Blend:     &wgpu.BlendStatePremultipliedAlphaBlending,
				WriteMask: wgpu.ColorWriteMaskAll, // todo
			}},
		}
		if !pl.AlphaBlend {
			pd.Fragment.Targets[0].Blend = &wgpu.BlendStateReplace
		}
	}
	if pl.System.Render().Depth.texture != nil {
		pd.DepthStencil = &wgpu.DepthStencilState{
			Format:            pl.System.Render().Depth.Format.Format,
			DepthWriteEnabled: true,
			DepthCompare:      wgpu.CompareFunctionLess,
			StencilFront: wgpu.StencilFaceState{
				Compare: wgpu.CompareFunctionAlways,
			},
			StencilBack: wgpu.StencilFaceState{
				Compare: wgpu.CompareFunctionAlways,
			},
		}
	}
	rp, err := pl.System.Device().Device.CreateRenderPipeline(pd)
	if errors.Log(err) != nil {
		return err
	}
	pl.renderPipeline = rp
	return nil
}

func (pl *GraphicsPipeline) Release() {
	pl.releaseShaders()
	pl.releasePipeline()
}

func (pl *GraphicsPipeline) releasePipeline() {
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
	pl.SetFrontFace(wgpu.FrontFaceCW)
	pl.SetCullMode(wgpu.CullModeBack)
	pl.SetAlphaBlend(true) // alpha blending
	pl.Multisample.Count = 1
	pl.Multisample.Mask = 0xFFFFFFFF              // todo
	pl.Multisample.AlphaToCoverageEnabled = false // todo
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
// Default is CW.
func (pl *GraphicsPipeline) SetFrontFace(face wgpu.FrontFace) *GraphicsPipeline {
	pl.Primitive.FrontFace = face
	return pl
}

// SetCullMode sets the face culling mode.
func (pl *GraphicsPipeline) SetCullMode(mode wgpu.CullMode) *GraphicsPipeline {
	pl.Primitive.CullMode = mode
	return pl
}

// SetLineWidth sets the rendering line width -- 1 is default.
func (pl *GraphicsPipeline) SetLineWidth(lineWidth float32) {
	// pl.VkConfig.PRasterizationState.LineWidth = lineWidth
}

// SetAlphaBlend determines the alpha (transparency) blending function:
// either 1-source alpha (alphaBlend) or no blending:
// new color overwrites old.  Default is alphaBlend = true
func (pl *GraphicsPipeline) SetAlphaBlend(alphaBlend bool) {
	pl.AlphaBlend = alphaBlend
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
