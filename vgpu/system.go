// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"log"

	vk "github.com/goki/vulkan"
)

// System manages a system of Pipelines that all share
// a common collection of Vars, Vals, and a Memory manager.
// For example, this could be a collection of different
// pipelines for different material types, or different
// compute operations performed on a common set of data.
// It maintains its own logical device and associated queue.
type System struct {
	Name        string                      `desc:"optional name of this System"`
	GPU         *GPU                        `desc:"gpu device"`
	Device      Device                      `desc:"logical device for this System, which is a non-owned copy of either Surface or RenderFrame device"`
	CmdPool     CmdPool                     `desc:"cmd pool specific to this system"`
	Compute     bool                        `desc:"if true, this is a compute system -- otherwise is graphics"`
	Pipelines   []*Pipeline                 `desc:"all pipelines"`
	PipelineMap map[string]*Pipeline        `desc:"map of all pipelines -- names must be unique"`
	Events      map[string]vk.Event         `desc:"map of events for synchronizing processing within a single command stream -- this is the best method for compute shaders to coordinate within a given sequence of shader runs in a single command stream"`
	Semaphores  map[string]vk.Semaphore     `desc:"map of semaphores for GPU-side sync between different submitted commands -- names must be unique -- note: better to use Events within one command if possible."`
	Fences      map[string]vk.Fence         `desc:"map of fences for CPU-GPU sync -- names must be unique.  WaitIdle implictly uses a fence so it is not essential to use this for simple wait case"`
	CmdBuffs    map[string]vk.CommandBuffer `desc:"map of command buffers, for persistent recorded commands -- names must be unique"`
	Mem         Memory                      `desc:"manages all the memory for all the Vals"`
	Render      Render                      `desc:"renderpass with depth buffer for this system"`
}

// InitGraphics initializes the System for graphics use, using
// the graphics device from the Surface associated with this system
// or another device can be initialized by calling
// sy.Device.Init(gp, vk.QueueGraphicsBit)
func (sy *System) InitGraphics(gp *GPU, name string, dev *Device) error {
	sy.GPU = gp
	sy.Render.Sys = sy
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
	sy.Render.Sys = sy
	sy.Name = name
	sy.Compute = true
	sy.Device.Init(gp, vk.QueueComputeBit)
	sy.InitCmd()
	sy.Mem.Init(gp, &sy.Device)
	sy.NewFence("ComputeWait") // always have this named fence avail for wait
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
	for _, ev := range sy.Events {
		vk.DestroyEvent(sy.Device.Device, ev, nil)
	}
	sy.Events = nil
	for _, sp := range sy.Semaphores {
		vk.DestroySemaphore(sy.Device.Device, sp, nil)
	}
	sy.Semaphores = nil
	for _, fc := range sy.Fences {
		vk.DestroyFence(sy.Device.Device, fc, nil)
	}
	sy.Fences = nil
	sy.CmdBuffs = nil
	for _, pl := range sy.Pipelines {
		pl.Destroy()
	}
	sy.Pipelines = nil
	sy.CmdPool.Destroy(sy.Device.Device)
	sy.Mem.Destroy(sy.Device.Device)
	if sy.Compute {
		sy.Device.Destroy()
	} else {
		sy.Render.Destroy()
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

// PipelineByNameTry returns pipeline by name with error for not found
func (sy *System) PipelineByNameTry(name string) (*Pipeline, error) {
	pl, ok := sy.PipelineMap[name]
	if !ok {
		err := fmt.Errorf("Pipeline named: %s not found", name)
		log.Println(err)
		return nil, err
	}
	return pl, nil
}

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
		return nil, err
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
		return nil, err
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
		return nil, err
	}
	return sp, nil
}

// NewCmdBuff returns a new fence using system device
func (sy *System) NewCmdBuff(name string) vk.CommandBuffer {
	cb := sy.CmdPool.NewBuffer(&sy.Device)
	if sy.CmdBuffs == nil {
		sy.CmdBuffs = make(map[string]vk.CommandBuffer)
	}
	sy.CmdBuffs[name] = cb
	return cb
}

// CmdBuffByNameTry returns fence by name with error for not found
func (sy *System) CmdBuffByNameTry(name string) (vk.CommandBuffer, error) {
	cb, ok := sy.CmdBuffs[name]
	if !ok {
		err := fmt.Errorf("CmdBuff named: %s not found", name)
		// log.Println(err)
		return nil, err
	}
	return cb, nil
}

// ConfigRender configures the renderpass, including the image
// format that we're rendering to, for a surface render target,
// and the depth buffer format (pass UndefType for no depth buffer).
func (sy *System) ConfigRender(imgFmt *ImageFormat, depthFmt Types) {
	sy.Render.Config(sy.Device.Device, imgFmt, depthFmt, false)
}

// ConfigRenderNonSurface configures the renderpass, including the image
// format that we're rendering to, for a RenderFrame non-surface target,
// and the depth buffer format (pass UndefType for no depth buffer).
func (sy *System) ConfigRenderNonSurface(imgFmt *ImageFormat, depthFmt Types) {
	sy.Render.Config(sy.Device.Device, imgFmt, depthFmt, true)
}

// Config configures the entire system, after everything has been
// setup (Pipelines, Vars, etc).  Memory / Vals do not yet need to
// be configured and are not Config'd by this call.
func (sy *System) Config() {
	sy.Mem.Config(sy.Device.Device)
	if Debug {
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

// SetClearColor sets the RGBA colors to set when starting new render
// For all pipelines, to keep graphics settings consistent.
func (sy *System) SetClearColor(r, g, b, a float32) {
	sy.Render.SetClearColor(r, g, b, a)
}

// SetClearDepthStencil sets the depth and stencil values when starting new render
// For all pipelines, to keep graphics settings consistent.
func (sy *System) SetClearDepthStencil(depth float32, stencil uint32) {
	sy.Render.SetClearDepthStencil(depth, stencil)
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
	vars.BindDescIdx = descIdx
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

// CmdBindTextureVarIdx returns the txIdx needed to select the given Texture value
// at valIdx in given variable in given set index, for use in a shader (i.e., pass
// txIdx as a push constant to the shader to select this texture).  If there are
// more than MaxTexturesPerSet textures, then it may need to select a different
// descIdx where that val has been allocated -- the descIdx is returned, and
// switched is true if it had to issue a CmdBindVars to given command buffer
// to bind to that desc set, updating BindDescIdx.  Typically other vars are
// bound to the same vals across sets, so this should not affect them, but
// that is not necessarily the case, so other steps might need to be taken.
// If the texture is not valid, a -1 is returned for txIdx, and an error is logged.
func (sy *System) CmdBindTextureVarIdx(cmd vk.CommandBuffer, setIdx int, varNm string, valIdx int) (txIdx, descIdx int, switched bool, err error) {
	vars := sy.Vars()
	txv, _, _ := vars.ValByIdxTry(setIdx, varNm, valIdx)

	descIdx = valIdx / MaxTexturesPerSet
	if descIdx != vars.BindDescIdx {
		sy.CmdBindVars(cmd, descIdx)
		vars.BindDescIdx = descIdx
		switched = true
	}
	stIdx := descIdx * MaxTexturesPerSet
	txIdx = txv.TextureValidIdx(stIdx, valIdx)
	if txIdx < 0 {
		err = fmt.Errorf("vgpu.CmdBindTextureVarIdx: Texture var %s image val at index %d (starting at idx: %d) is not valid", varNm, valIdx, stIdx)
		log.Println(err) // this is always bad
	}
	return
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
// Clears the frame first, according to the ClearVals.
// Also Binds descriptor sets to command buffer for given collection
// of descriptors descIdx (see Vars NDescs for info).
func (sy *System) BeginRenderPass(cmd vk.CommandBuffer, fr *Framebuffer, descIdx int) {
	sy.CmdBindVars(cmd, descIdx)
	sy.Render.BeginRenderPass(cmd, fr)
}

// ResetBeginRenderPass adds commands to the given command buffer
// to reset command buffer and call begin on it, then starts
// the render pass on given framebuffer (BeginRenderPass)
// Clears the frame first, according to the ClearVals.
// Also Binds descriptor sets to command buffer for given collection
// of descriptors descIdx (see Vars NDescs for info).
func (sy *System) ResetBeginRenderPass(cmd vk.CommandBuffer, fr *Framebuffer, descIdx int) {
	CmdResetBegin(cmd)
	sy.BeginRenderPass(cmd, fr, descIdx)
}

// BeginRenderPassNoClear adds commands to the given command buffer
// to start the render pass on given framebuffer.
// does NOT clear the frame first -- loads prior state.
// Also Binds descriptor sets to command buffer for given collection
// of descriptors descIdx (see Vars NDescs for info).
func (sy *System) BeginRenderPassNoClear(cmd vk.CommandBuffer, fr *Framebuffer, descIdx int) {
	sy.CmdBindVars(cmd, descIdx)
	sy.Render.BeginRenderPassNoClear(cmd, fr)
}

// ResetBeginRenderPassNoClear adds commands to the given command buffer
// to reset command buffer and call begin on it, then starts
// the render pass on given framebuffer (BeginRenderPass)
// does NOT clear the frame first -- loads prior state.
// Also Binds descriptor sets to command buffer for given collection
// of descriptors descIdx (see Vars NDescs for info).
func (sy *System) ResetBeginRenderPassNoClear(cmd vk.CommandBuffer, fr *Framebuffer, descIdx int) {
	CmdResetBegin(cmd)
	sy.BeginRenderPassNoClear(cmd, fr, descIdx)
}

// EndRenderPass adds commands to the given command buffer
// to end the render pass.  It does not call EndCommandBuffer,
// in case any further commands are to be added.
func (sy *System) EndRenderPass(cmd vk.CommandBuffer) {
	// Note that ending the renderpass changes the image's layout from
	// vk.ImageLayoutColorAttachmentOptimal to vk.ImageLayoutPresentSrc
	vk.CmdEndRenderPass(cmd)
}

/////////////////////////////////////////////
// Memory utils

// MemCmdStart starts a one-time memory command using the
// Memory CmdPool and Device associated with this System
// Use this for other random memory transfer commands.
func (sy *System) MemCmdStart() vk.CommandBuffer {
	cmd := sy.Mem.CmdPool.NewBuffer(&sy.Device)
	CmdBeginOneTime(cmd)
	return cmd
}

// MemCmdEndSubmitWaitFree submits current one-time memory command
// using the Memory CmdPool and Device associated with this System
// Use this for other random memory transfer commands.
func (sy *System) MemCmdEndSubmitWaitFree() {
	sy.Mem.CmdPool.EndSubmitWaitFree(&sy.Device)
}

/////////////////////////////////////////////
// Compute utils

// CmdSubmitWait does SubmitWait on CmdPool
func (sy *System) CmdSubmitWait() {
	sy.CmdPool.SubmitWait(&sy.Device)
}
