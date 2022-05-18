// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"

	vk "github.com/goki/vulkan"
)

// System manages a system of Pipelines that all share
// a common collection of Vars, Vals, and a Memory manager.
// For example, this could be a collection of different
// pipelines for different material types, or different
// compute operations performed on a common set of data.
// It maintains its own logical device and associated queue.
type System struct {
	Name        string               `desc:"optional name of this System"`
	GPU         *GPU                 `desc:"gpu device"`
	Device      Device               `desc:"logical device for this System -- has its own queues"`
	CmdPool     CmdPool              `desc:"cmd pool specific to this system"`
	Compute     bool                 `desc:"if true, this is a compute system -- otherwise is graphics"`
	Pipelines   []*Pipeline          `desc:"all pipelines"`
	PipelineMap map[string]*Pipeline `desc:"map of all pipelines -- names must be unique"`
	Mem         Memory               `desc:"manages all the memory for all the Vals"`
	RenderPass  RenderPass           `desc:"renderpass with depth buffer for this system"`
	Framebuffer Framebuffer          `desc:"shared framebuffer to render into, if not rendering into Surface"`
}

// InitGraphics initializes the System for graphics use, using
// the graphics device from the Surface associated with this system
// or another device can be initialized by calling
// sy.Device.Init(gp, vk.QueueGraphicsBit)
func (sy *System) InitGraphics(gp *GPU, name string, dev *Device) error {
	sy.GPU = gp
	sy.Name = name
	sy.Compute = false
	sy.Device = *dev
	sy.InitCmd()
	sy.Mem.Init(gp, &sy.Device)
	return nil
}

// InitCompute initializes the System for compute functionality,
// which creates its own Compute device.
func (sy *System) InitCompute(gp *GPU, name string) error {
	sy.GPU = gp
	sy.Name = name
	sy.Compute = true
	sy.Device.Init(gp, vk.QueueComputeBit)
	sy.InitCmd()
	sy.Mem.Init(gp, &sy.Device)
	return nil
}

// InitCmd initializes the command pool and buffer
func (sy *System) InitCmd() {
	sy.CmdPool.ConfigResettable(&sy.Device)
	sy.CmdPool.NewBuffer(&sy.Device)
}

// Vars returns a pointer to the vars for this pipeline, which has vals within it
func (sy *System) Vars() *Vars {
	return &sy.Mem.Vars
}

func (sy *System) Destroy() {
	for _, pl := range sy.Pipelines {
		pl.Destroy()
	}
	sy.CmdPool.Destroy(sy.Device.Device)
	sy.Mem.Destroy(sy.Device.Device)
	if sy.Compute {
		sy.Device.Destroy()
	} else {
		sy.RenderPass.Destroy()
	}
	sy.GPU = nil
}

// AddPipeline adds given pipeline
func (sy *System) AddPipeline(pl *Pipeline) {
	if sy.PipelineMap == nil {
		sy.PipelineMap = make(map[string]*Pipeline)
	}
	sy.Pipelines = append(sy.Pipelines, pl)
	sy.PipelineMap[pl.Name] = pl
}

// NewPipeline returns a new pipeline added to this System,
// initialized for use in this system.
func (sy *System) NewPipeline(name string) *Pipeline {
	pl := &Pipeline{Name: name}
	pl.Init(sy)
	sy.AddPipeline(pl)
	return pl
}

// ConfigRenderPass configures the renderpass, including the image
// format that we're rendering to, for a surface render target,
// and the depth buffer format (pass UndefType for no depth buffer).
func (sy *System) ConfigRenderPass(imgFmt *ImageFormat, depthFmt Types) {
	sy.RenderPass.Config(sy.Device.Device, imgFmt, depthFmt, false)
}

// ConfigRenderPassNonSurface configures the renderpass, including the image
// format that we're rendering to, for a RenderFrame non-surface target,
// and the depth buffer format (pass UndefType for no depth buffer).
func (sy *System) ConfigRenderPassNonSurface(imgFmt *ImageFormat, depthFmt Types) {
	sy.RenderPass.Config(sy.Device.Device, imgFmt, depthFmt, true)
}

// Config configures the entire system, after everything has been
// setup (Pipelines, Vars, etc).  Memory / Vals do not yet need to
// be configured and are not Config'd by this call.
func (sy *System) Config() {
	sy.Mem.Config(sy.Device.Device)
	if sy.GPU.Debug {
		fmt.Printf("%s\n", sy.Vars().StringDoc())
	}
	for _, pl := range sy.Pipelines {
		pl.Config()
	}
}

//////////////////////////////////////////////////////////////
// Set graphics options

// SetGraphicsDefaults configures all the default settings for all
// graphics rendering pipelines (not for a compute pipeline)
func (sy *System) SetGraphicsDefaults() {
	for _, pl := range sy.Pipelines {
		pl.SetGraphicsDefaults()
	}
	sy.SetClearColor(0, 0, 0, 1)
	sy.SetClearDepthStencil(1, 0)
}

// SetTopology sets the topology of vertex position data.
// TriangleList is the default.
// Also for Strip modes, restartEnable allows restarting a new
// strip by inserting a ??
// For all pipelines, to keep graphics settings consistent.
func (sy *System) SetTopology(topo Topologies, restartEnable bool) {
	for _, pl := range sy.Pipelines {
		pl.SetTopology(topo, restartEnable)
	}
}

// SetRasterization sets various options for how to rasterize shapes:
// Defaults are: vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0
// For all pipelines, to keep graphics settings consistent.
func (sy *System) SetRasterization(polygonMode vk.PolygonMode, cullMode vk.CullModeFlagBits, frontFace vk.FrontFace, lineWidth float32) {
	for _, pl := range sy.Pipelines {
		pl.SetRasterization(polygonMode, cullMode, frontFace, lineWidth)
	}
}

// SetCullFace sets the face culling mode: true = back, false = front
// use CullBack, CullFront constants
func (sy *System) SetCullFace(back bool) {
	for _, pl := range sy.Pipelines {
		pl.SetCullFace(back)
	}
}

// SetFrontFace sets the winding order for what counts as a front face
// true = CCW, false = CW
func (sy *System) SetFrontFace(ccw bool) {
	for _, pl := range sy.Pipelines {
		pl.SetFrontFace(ccw)
	}
}

// SetLineWidth sets the rendering line width -- 1 is default.
func (sy *System) SetLineWidth(lineWidth float32) {
	for _, pl := range sy.Pipelines {
		pl.SetLineWidth(lineWidth)
	}
}

// SetColorBlend determines the color blending function:
// either 1-source alpha (alphaBlend) or no blending:
// new color overwrites old.  Default is alphaBlend = true
// For all pipelines, to keep graphics settings consistent.
func (sy *System) SetColorBlend(alphaBlend bool) {
	for _, pl := range sy.Pipelines {
		pl.SetColorBlend(alphaBlend)
	}
}

// SetClearOff turns off clearing at start of rendering.
// call SetClearColor to turn back on.
func (sy *System) SetClearOff() {
	sy.RenderPass.SetClearOff()
}

// SetClearColor sets the RGBA colors to set when starting new render
// For all pipelines, to keep graphics settings consistent.
func (sy *System) SetClearColor(r, g, b, a float32) {
	sy.RenderPass.SetClearColor(r, g, b, a)
}

// SetClearDepthStencil sets the depth and stencil values when starting new render
// For all pipelines, to keep graphics settings consistent.
func (sy *System) SetClearDepthStencil(depth float32, stencil uint32) {
	sy.RenderPass.SetClearDepthStencil(depth, stencil)
}

//////////////////////////////////////////////////////////////////////////
// Rendering

// CmdBindVars adds command to the given command buffer
// to bind the Vars descriptors, for given collection of descriptors descIdx
// (see Vars NDescs for info).
func (sy *System) CmdBindVars(cmd vk.CommandBuffer, descIdx int) {
	vars := sy.Vars()
	if len(vars.SetMap) == 0 {
		return
	}
	dset := vars.VkDescSets[descIdx]
	doff := vars.DynOffs[descIdx]

	if sy.Compute {
		vk.CmdBindDescriptorSets(cmd, vk.PipelineBindPointCompute, vars.VkDescLayout,
			0, uint32(len(dset)), dset, uint32(len(doff)), doff)
	} else {
		vk.CmdBindDescriptorSets(cmd, vk.PipelineBindPointGraphics, vars.VkDescLayout,
			0, uint32(len(dset)), dset, uint32(len(doff)), doff)
	}

}

// CmdResetBindVars adds command to the given command buffer
// to bind the Vars descriptors, for given collection of descriptors descIdx
// (see Vars NDescs for info).
func (sy *System) CmdResetBindVars(cmd vk.CommandBuffer, descIdx int) {
	CmdResetBegin(cmd)
	sy.CmdBindVars(cmd, descIdx)
}

// BeginRenderPass adds commands to the given command buffer
// to start the render pass on given framebuffer.
// Optionally clears the frame according to the current ClearVals.
// Also Binds descriptor sets to command buffer for given collection
// of descriptors descIdx (see Vars NDescs for info).
func (sy *System) BeginRenderPass(cmd vk.CommandBuffer, fr *Framebuffer, descIdx int) {
	sy.CmdBindVars(cmd, descIdx)
	sy.RenderPass.BeginRenderPass(cmd, fr)
}

// ResetBeginRenderPass adds commands to the given command buffer
// to reset command buffer and call begin on it, then starts
// the render pass on given framebuffer (BeginRenderPass)
// Optionally clears the frame according to the current ClearVals.
// Also Binds descriptor sets to command buffer for given collection
// of descriptors descIdx (see Vars NDescs for info).
func (sy *System) ResetBeginRenderPass(cmd vk.CommandBuffer, fr *Framebuffer, descIdx int) {
	CmdResetBegin(cmd)
	sy.BeginRenderPass(cmd, fr, descIdx)
}

// EndRenderPass adds commands to the given command buffer
// to end the render pass.  It does not call EndCommandBuffer,
// in case any further commands are to be added.
func (sy *System) EndRenderPass(cmd vk.CommandBuffer) {
	// Note that ending the renderpass changes the image's layout from
	// vk.ImageLayoutColorAttachmentOptimal to vk.ImageLayoutPresentSrc
	vk.CmdEndRenderPass(cmd)
}
