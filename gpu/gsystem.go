// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"image"
	"image/color"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"github.com/cogentcore/webgpu/wgpu"
)

// GraphicsSystem manages a system of Pipelines that all share
// a common collection of Vars and Values.
// For example, this could be a collection of different
// pipelines for different material types.
// The System provides a simple top-level API for the whole
// render process.
type GraphicsSystem struct {
	// optional name of this GraphicsSystem
	Name string

	// vars represents all the data variables used by the system,
	// with one Var for each resource that is made visible to the shader,
	// indexed by Group (@group) and Binding (@binding).
	// Each Var has Value(s) containing specific instance values.
	// Access through the System.Vars() method.
	vars Vars

	// GraphicsPipelines by name
	GraphicsPipelines map[string]*GraphicsPipeline

	// Renderer is the rendering target for this system,
	// It is either a Surface or a RenderTexture.
	Renderer Renderer

	// CurrentCommandEncoder is the command encoder created in
	// [GraphicsSystem.BeginRenderPass], and released in [GraphicsSystem.EndRenderPass].
	CommandEncoder *wgpu.CommandEncoder

	// logical device for this GraphicsSystem, from the Renderer.
	device Device

	// gpu is our GPU device, which has properties
	// and alignment factors.
	gpu *GPU
}

// NewGraphicsSystem returns a new GraphicsSystem, using
// the given Renderer as the render target.
func NewGraphicsSystem(gp *GPU, name string, rd Renderer) *GraphicsSystem {
	sy := &GraphicsSystem{}
	sy.init(gp, name, rd)
	return sy
}

// System interface:

func (sy *GraphicsSystem) Vars() *Vars     { return &sy.vars }
func (sy *GraphicsSystem) Device() *Device { return &sy.device }
func (sy *GraphicsSystem) GPU() *GPU       { return sy.gpu }
func (sy *GraphicsSystem) Render() *Render { return sy.Renderer.Render() }

// init initializes the GraphicsSystem
func (sy *GraphicsSystem) init(gp *GPU, name string, rd Renderer) {
	sy.gpu = gp
	sy.Name = name
	sy.Renderer = rd
	sy.device = *rd.Device()
	sy.vars.device = sy.device
	sy.vars.sys = sy
	sy.GraphicsPipelines = make(map[string]*GraphicsPipeline)
}

// WaitDone waits until device is done with current processing steps
func (sy *GraphicsSystem) WaitDone() {
	sy.device.WaitDone()
}

func (sy *GraphicsSystem) Release() {
	sy.WaitDone()
	for _, pl := range sy.GraphicsPipelines {
		pl.Release()
	}
	sy.GraphicsPipelines = nil
	sy.vars.Release()
	sy.gpu = nil
}

// AddGraphicsPipeline adds a new GraphicsPipeline to the system
func (sy *GraphicsSystem) AddGraphicsPipeline(name string) *GraphicsPipeline {
	pl := NewGraphicsPipeline(name, sy)
	sy.GraphicsPipelines[pl.Name] = pl
	return pl
}

// When the render surface (e.g., window) is resized, call this function.
// WebGPU does not have any internal mechanism for tracking this, so we
// need to drive it from external events.
func (sy *GraphicsSystem) SetSize(size image.Point) {
	sy.Renderer.SetSize(size)
}

// Config configures the entire system, after Pipelines and Vars
// have been initialized.  After this point, just need to set
// values for the vars, and then do render passes.  This should
// not need to be called more than once.
func (sy *GraphicsSystem) Config() {
	sy.vars.Config(&sy.device)
	if Debug {
		fmt.Printf("%s\n", sy.vars.StringDoc())
	}
	for _, pl := range sy.GraphicsPipelines {
		pl.Config(true)
	}
}

//////////////////////////////////////////////////////////////
// Set graphics options

// SetGraphicsDefaults configures all the default settings for all
// graphics rendering pipelines (not for a compute pipeline)
func (sy *GraphicsSystem) SetGraphicsDefaults() *GraphicsSystem {
	for _, pl := range sy.GraphicsPipelines {
		pl.SetGraphicsDefaults()
	}
	sy.SetClearColor(colors.ToUniform(colors.Scheme.Background))
	sy.SetClearDepthStencil(1, 0)
	return sy
}

// SetTopology sets the topology of vertex position data.
// TriangleList is the default.
// Also for Strip modes, restartEnable allows restarting a new
// strip by inserting a ??
// For all pipelines, to keep graphics settings consistent.
func (sy *GraphicsSystem) SetTopology(topo Topologies, restartEnable bool) *GraphicsSystem {
	for _, pl := range sy.GraphicsPipelines {
		pl.SetTopology(topo, restartEnable)
	}
	return sy
}

// SetCullMode sets the face culling mode.
func (sy *GraphicsSystem) SetCullMode(mode wgpu.CullMode) *GraphicsSystem {
	for _, pl := range sy.GraphicsPipelines {
		pl.SetCullMode(mode)
	}
	return sy
}

// SetFrontFace sets the winding order for what counts as a front face.
func (sy *GraphicsSystem) SetFrontFace(face wgpu.FrontFace) *GraphicsSystem {
	for _, pl := range sy.GraphicsPipelines {
		pl.SetFrontFace(face)
	}
	return sy
}

// SetLineWidth sets the rendering line width -- 1 is default.
func (sy *GraphicsSystem) SetLineWidth(lineWidth float32) *GraphicsSystem {
	for _, pl := range sy.GraphicsPipelines {
		pl.SetLineWidth(lineWidth)
	}
	return sy
}

// SetAlphaBlend determines the alpha (transparency) blending function:
// either 1-source alpha (alphaBlend) or no blending:
// new color overwrites old.  Default is alphaBlend = true
// For all pipelines, to keep graphics settings consistent.
func (sy *GraphicsSystem) SetAlphaBlend(alphaBlend bool) *GraphicsSystem {
	for _, pl := range sy.GraphicsPipelines {
		pl.SetAlphaBlend(alphaBlend)
	}
	return sy
}

// SetClearColor sets the RGBA colors to set when starting new render
// For all pipelines, to keep graphics settings consistent.
func (sy *GraphicsSystem) SetClearColor(c color.Color) *GraphicsSystem {
	sy.Render().ClearColor = c
	return sy
}

// SetClearDepthStencil sets the depth and stencil values when starting new render
// For all pipelines, to keep graphics settings consistent.
func (sy *GraphicsSystem) SetClearDepthStencil(depth float32, stencil uint32) *GraphicsSystem {
	rd := sy.Render()
	rd.ClearDepth = depth
	rd.ClearStencil = stencil
	return sy
}

//////////////////////////////////////////////////////////////////////////
// Rendering

// NewCommandEncoder returns a new CommandEncoder for encoding
// rendering commands.  This is automatically called by
// BeginRenderPass and the result maintained in CurrentCommandEncoder.
func (sy *GraphicsSystem) NewCommandEncoder() (*wgpu.CommandEncoder, error) {
	cmd, err := sy.device.Device.CreateCommandEncoder(nil)
	if errors.Log(err) != nil {
		return nil, err
	}
	return cmd, nil
}

func (sy *GraphicsSystem) beginRenderPass() (*Render, *wgpu.TextureView, error) {
	rd := sy.Renderer
	view, err := rd.GetCurrentTexture()
	if errors.Log(err) != nil {
		return nil, nil, err
	}
	cmd, err := sy.NewCommandEncoder()
	if errors.Log(err) != nil {
		return nil, nil, err
	}
	sy.CommandEncoder = cmd
	return rd.Render(), view, nil
}

// BeginRenderPass adds commands to the given command buffer
// to start the render pass using the Renderer configured for
// this system, and returns the encoder object to which further
// rendering commands should be added.
// Call [EndRenderPass] when done.
// This version Clears the target texture first, using ClearValues.
func (sy *GraphicsSystem) BeginRenderPass() (*wgpu.RenderPassEncoder, error) {
	rd, view, err := sy.beginRenderPass()
	if errors.Log(err) != nil {
		return nil, err
	}
	return rd.BeginRenderPass(sy.CommandEncoder, view), nil
}

// BeginRenderPassNoClear adds commands to the given command buffer
// to start the render pass using the Renderer configured for
// this system, and returns the encoder object to which further
// rendering commands should be added.
// Call [EndRenderPass] when done.
// This version does NOT clear the target texture first,
// so the prior render output is carried over.
func (sy *GraphicsSystem) BeginRenderPassNoClear() (*wgpu.RenderPassEncoder, error) {
	rd, view, err := sy.beginRenderPass()
	if errors.Log(err) != nil {
		return nil, err
	}
	return rd.BeginRenderPassNoClear(sy.CommandEncoder, view), nil
}

// SubmitRender submits the current render commands to the device
// Queue and releases the [CurrentCommandEncoder] and the given
// RenderPassEncoder.  You must call rp.End prior to calling this.
// Can insert other commands after rp.End, e.g., to copy the rendered image,
// prior to calling SubmitRender.
func (sy *GraphicsSystem) SubmitRender(rp *wgpu.RenderPassEncoder) error {
	cmd := sy.CommandEncoder
	sy.CommandEncoder = nil
	rp.Release() // must happen before Finish
	cmdBuffer, err := cmd.Finish(nil)
	if errors.Log(err) != nil {
		return err
	}
	sy.device.Queue.Submit(cmdBuffer)
	cmdBuffer.Release()
	cmd.Release()
	for _, pl := range sy.GraphicsPipelines {
		pl.releaseOldBindGroups()
	}
	return nil
}

// EndRenderPass ends the render pass started by [BeginRenderPass],
// by calling [SubmitRender] to submit the rendering commands to the
// device, and calling Present() on the Renderer to show results.
func (sy *GraphicsSystem) EndRenderPass(rp *wgpu.RenderPassEncoder) {
	sy.SubmitRender(rp)
	sy.Renderer.Present()
}
