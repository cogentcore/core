// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"image/color"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// System manages a system of Pipelines that all share
// a common collection of Vars and Values.
// For example, this could be a collection of different
// pipelines for different material types, or different
// compute operations performed on a common set of data.
// It maintains its own logical device and associated queue.
type System struct {
	// optional name of this System
	Name string

	// Vars represents all the resources used by the system, with one
	// Var for each resource that is made visible to the shader,
	// indexed by Group (@group) and Binding (@binding).
	Vars Vars

	// if true, this is a pure compute system -- otherwise has graphics
	Compute bool

	// GraphicsPipelines by name
	GraphicsPipelines map[string]*GraphicsPipeline

	// renderpass with depth buffer for this system
	Render Render

	// GPU is needed to access some properties and alignment factors.
	GPU *GPU

	// logical device for this System.
	// This is owned by us for a Compute device.
	Device Device
}

// NewGraphicsSystem returns a new System for graphics use, using
// the graphics Device from the Surface or RenderFrame depending
// on the target of rendering.the Surface associated with this system.
func NewGraphicsSystem(gp *GPU, name string, dev *Device) *System {
	sy := &System{}
	sy.initGraphics(gp, name, dev)
	return sy
}

// NewComputeSystem returns a new System for purely compute use,
// creating a new device specific to this system.
func NewComputeSystem(gp *GPU, name string) *System {
	sy := &System{}
	sy.initCompute(gp, name)
	return sy
}

// init initializes the System
func (sy *System) init(gp *GPU, name string, dev *Device) {
	sy.GPU = gp
	sy.Render.sys = sy
	sy.Name = name
	sy.Device = *dev
	sy.Vars.device = *dev
	sy.Vars.sys = sy
}

// initGraphics initializes the System for graphics use, using
// the graphics device from the Surface associated with this system.
func (sy *System) initGraphics(gp *GPU, name string, dev *Device) error {
	sy.init(gp, name, dev)
	sy.Compute = false
	return nil
}

// initCompute initializes the System for compute functionality,
// which creates its own Compute device.
func (sy *System) initCompute(gp *GPU, name string) error {
	dev, err := NewDevice(gp)
	if err != nil {
		return err
	}
	sy.init(gp, name, dev)
	sy.Compute = true
	// sy.NewFence("ComputeWait") // always have this named fence avail for wait
	return nil
}

func (sy *System) NewCommandEncoder() *wgpu.CommandEncoder {
	ce, err := sy.Device.Device.CreateCommandEncoder(nil)
	if errors.Log(err) != nil {
		return nil
	}
	return ce
}

// WaitDone waits until device is done with current processing steps
func (sy *System) WaitDone() {
	sy.Device.Device.Poll(true, nil)
}

func (sy *System) Release() {
	if sy.Device.Device != nil {
		sy.WaitDone()
	}
	// for _, ev := range sy.Events {
	// 	vk.ReleaseEvent(sy.Device.Device, ev, nil)
	// }
	// sy.Events = nil
	// for _, sp := range sy.Semaphores {
	// 	vk.ReleaseSemaphore(sy.Device.Device, sp, nil)
	// }
	// sy.Semaphores = nil
	// for _, fc := range sy.Fences {
	// 	vk.ReleaseFence(sy.Device.Device, fc, nil)
	// }
	// sy.Fences = nil
	// sy.CmdBuffs = nil
	if sy.GraphicsPipelines != nil {
		for _, pl := range sy.GraphicsPipelines {
			pl.Release()
		}
		sy.GraphicsPipelines = nil
	}
	sy.Vars.Release()
	if sy.Compute {
		sy.Device.Release()
	} else {
		sy.Render.Release()
	}
	sy.GPU = nil
}

// AddGraphicsPipeline adds a new GraphicsPipeline to the system
func (sy *System) AddGraphicsPipeline(name string) *GraphicsPipeline {
	if sy.GraphicsPipelines == nil {
		sy.GraphicsPipelines = make(map[string]*GraphicsPipeline)
	}
	pl := NewGraphicsPipeline(name)
	pl.Init(sy)
	sy.GraphicsPipelines[pl.Name] = pl
	return pl
}

/*
// NewSemaphore returns a new semaphore using system device
func (sy *System) NewSemaphore(name string) vk.Semaphore {
	sp := NewSemaphore(sy.Device.Device)
	if sy.Semaphores == nil {
		sy.Semaphores = make(map[string]vk.Semaphore)
	}
	sy.Semaphores[name] = sp
	return sp
}

// SemaphoreByNameTry returns semaphore by name with error for not found
func (sy *System) SemaphoreByNameTry(name string) (vk.Semaphore, error) {
	sp, ok := sy.Semaphores[name]
	if !ok {
		err := fmt.Errorf("Semaphore named: %s not found", name)
		log.Println(err)
		return vk.NullSemaphore, err
	}
	return sp, nil
}

// NewEvent returns a new event using system device
func (sy *System) NewEvent(name string) vk.Event {
	sp := NewEvent(sy.Device.Device)
	if sy.Events == nil {
		sy.Events = make(map[string]vk.Event)
	}
	sy.Events[name] = sp
	return sp
}

// EventByNameTry returns event by name with error for not found
func (sy *System) EventByNameTry(name string) (vk.Event, error) {
	sp, ok := sy.Events[name]
	if !ok {
		err := fmt.Errorf("Event named: %s not found", name)
		log.Println(err)
		return vk.NullEvent, err
	}
	return sp, nil
}

// NewFence returns a new fence using system device
func (sy *System) NewFence(name string) vk.Fence {
	sp := NewFence(sy.Device.Device)
	if sy.Fences == nil {
		sy.Fences = make(map[string]vk.Fence)
	}
	sy.Fences[name] = sp
	return sp
}

// FenceByNameTry returns fence by name with error for not found
func (sy *System) FenceByNameTry(name string) (vk.Fence, error) {
	sp, ok := sy.Fences[name]
	if !ok {
		err := fmt.Errorf("Fence named: %s not found", name)
		log.Println(err)
		return vk.NullFence, err
	}
	return sp, nil
}
*/

// NewCmdBuff returns a new command encoder using system device
func (sy *System) NewCmdBuff(name string) *wgpu.CommandEncoder {
	// cb := sy.CmdPool.NewBuffer(&sy.Device)
	// if sy.CmdBuffs == nil {
	// 	sy.CmdBuffs = make(map[string]*wgpu.CommandEncoder)
	// }
	// sy.CmdBuffs[name] = cb
	// return cb
	return nil
}

// CmdBuffByNameTry returns command encoder by name with error for not found
func (sy *System) CmdBuffByNameTry(name string) (*wgpu.CommandEncoder, error) {
	// cb, ok := sy.CmdBuffs[name]
	// if !ok {
	// 	err := fmt.Errorf("CmdBuff named: %s not found", name)
	// 	// log.Println(err)
	// 	return nil, err
	// }
	// return cb, nil
	return nil, nil
}

// ConfigRender configures the renderpass, including the texture
// format that we're rendering to, for a surface render target,
// and the depth buffer format (pass UndefType for no depth buffer).
func (sy *System) ConfigRender(renderFormat *TextureFormat, depthFmt Types) {
	sy.Render.Config(&sy.Device, renderFormat, depthFmt, false)
}

// ConfigRenderNonSurface configures the renderpass, including the texture
// format that we're rendering to, for a RenderFrame non-surface target,
// and the depth buffer format (pass UndefType for no depth buffer).
func (sy *System) ConfigRenderNonSurface(renderFormat *TextureFormat, depthFmt Types) {
	sy.Render.Config(&sy.Device, renderFormat, depthFmt, true)
}

// Config configures the entire system, after everything has been
// setup (Pipelines, Vars, etc).
func (sy *System) Config() {
	sy.Vars.Config(&sy.Device)
	if Debug {
		fmt.Printf("%s\n", sy.Vars.StringDoc())
	}
	if sy.GraphicsPipelines != nil {
		for _, pl := range sy.GraphicsPipelines {
			pl.Config(true)
		}
	}
}

//////////////////////////////////////////////////////////////
// Set graphics options

// SetGraphicsDefaults configures all the default settings for all
// graphics rendering pipelines (not for a compute pipeline)
func (sy *System) SetGraphicsDefaults() *System {
	for _, pl := range sy.GraphicsPipelines {
		pl.SetGraphicsDefaults()
	}
	sy.SetClearColor(colors.Black)
	sy.SetClearDepthStencil(1, 0)
	return sy
}

// SetTopology sets the topology of vertex position data.
// TriangleList is the default.
// Also for Strip modes, restartEnable allows restarting a new
// strip by inserting a ??
// For all pipelines, to keep graphics settings consistent.
func (sy *System) SetTopology(topo Topologies, restartEnable bool) *System {
	for _, pl := range sy.GraphicsPipelines {
		pl.SetTopology(topo, restartEnable)
	}
	return sy
}

// SetRasterization sets various options for how to rasterize shapes:
// Defaults are: vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0
// For all pipelines, to keep graphics settings consistent.
// func (sy *System) SetRasterization(polygonMode vk.PolygonMode, cullMode vk.CullModeFlagBits, frontFace vk.FrontFace, lineWidth float32) {
// 	for _, pl := range sy.GraphicsPipelines {
// 		pl.SetRasterization(polygonMode, cullMode, frontFace, lineWidth)
// 	}
// }

// SetCullMode sets the face culling mode.
func (sy *System) SetCullMode(mode wgpu.CullMode) *System {
	for _, pl := range sy.GraphicsPipelines {
		pl.SetCullMode(mode)
	}
	return sy
}

// SetFrontFace sets the winding order for what counts as a front face.
func (sy *System) SetFrontFace(face wgpu.FrontFace) *System {
	for _, pl := range sy.GraphicsPipelines {
		pl.SetFrontFace(face)
	}
	return sy
}

// SetLineWidth sets the rendering line width -- 1 is default.
func (sy *System) SetLineWidth(lineWidth float32) *System {
	for _, pl := range sy.GraphicsPipelines {
		pl.SetLineWidth(lineWidth)
	}
	return sy
}

// SetColorBlend determines the color blending function:
// either 1-source alpha (alphaBlend) or no blending:
// new color overwrites old.  Default is alphaBlend = true
// For all pipelines, to keep graphics settings consistent.
func (sy *System) SetColorBlend(alphaBlend bool) *System {
	for _, pl := range sy.GraphicsPipelines {
		pl.SetColorBlend(alphaBlend)
	}
	return sy
}

// SetClearColor sets the RGBA colors to set when starting new render
// For all pipelines, to keep graphics settings consistent.
func (sy *System) SetClearColor(c color.Color) *System {
	sy.Render.ClearColor = c
	return sy
}

// SetClearDepthStencil sets the depth and stencil values when starting new render
// For all pipelines, to keep graphics settings consistent.
func (sy *System) SetClearDepthStencil(depth float32, stencil uint32) *System {
	sy.Render.ClearDepth = depth
	sy.Render.ClearStencil = stencil
	return sy
}

//////////////////////////////////////////////////////////////////////////
// Rendering

// CmdBindVars adds command to the given command buffer
// to bind the Vars descriptors, for given collection of descriptors descIndex
// (see Vars NDescs for info).
func (sy *System) CmdBindVars(cmd *wgpu.CommandEncoder, descIndex int) {
	// vars := sy.Vars()
	// if len(vars.SetMap) == 0 {
	// 	return
	// }
	// vars.BindDescIndex = descIndex
	// dset := vars.VkDescSets[descIndex]
	// doff := vars.DynOffs[descIndex]
	// if sy.Compute {
	// 	vk.CmdBindDescriptorSets(cmd, vk.PipelineBindPointCompute, vars.VkDescLayout,
	// 		0, uint32(len(dset)), dset, 0, nil)
	// } else {
	// 	vk.CmdBindDescriptorSets(cmd, vk.PipelineBindPointGraphics, vars.VkDescLayout,
	// 		0, uint32(len(dset)), dset, uint32(len(doff)), doff)
	// }
}

// CmdResetBindVars adds command to the given command buffer
// to bind the Vars descriptors, for given collection of descriptors descIndex
// (see Vars NDescs for info).
func (sy *System) CmdResetBindVars(cmd *wgpu.CommandEncoder, descIndex int) {
	// CmdResetBegin(cmd)
	// sy.CmdBindVars(cmd, descIndex)
}

// BeginRenderPass adds commands to the given command buffer
// to start the render pass on given TextureView.
// Clears the frame first, according to the ClearValues.
func (sy *System) BeginRenderPass(cmd *wgpu.CommandEncoder, view *wgpu.TextureView) *wgpu.RenderPassEncoder {
	// sy.CmdBindVars(cmd, descIndex)
	return sy.Render.BeginRenderPass(cmd, view)
}

// BeginRenderPassNoClear adds commands to the given command buffer
// to start the render pass on given framebuffer.
// does NOT clear the frame first -- loads prior state.
// Also Binds descriptor sets to command buffer for given collection
// of descriptors descIndex (see Vars NDescs for info).
func (sy *System) BeginRenderPassNoClear(cmd *wgpu.CommandEncoder, view *wgpu.TextureView) *wgpu.RenderPassEncoder {
	// sy.CmdBindVars(cmd, descIndex)
	return sy.Render.BeginRenderPassNoClear(cmd, view)
}

// EndRenderPass adds commands to the given command buffer
// to end the render pass.  It does not call EndCommandBuffer,
// in case any further commands are to be added.
func (sy *System) EndRenderPass(cmd *wgpu.CommandEncoder) {
	// Note that ending the renderpass changes the image's layout from
	// vk.TextureLayoutColorAttachmentOptimal to vk.TextureLayoutPresentSrc
	// vk.CmdEndRenderPass(cmd)
}

/////////////////////////////////////////////
// Compute utils

// CmdSubmitWait does SubmitWait on CmdPool
func (sy *System) CmdSubmitWait() {
	// sy.CmdPool.SubmitWait(&sy.Device)
}
