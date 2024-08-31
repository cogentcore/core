// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"math"

	"cogentcore.org/core/base/errors"
	"github.com/cogentcore/webgpu/wgpu"
)

// ComputeSystem manages a system of ComputePipelines that all share
// a common collection of Vars and Values.
type ComputeSystem struct {
	// optional name of this ComputeSystem
	Name string

	// vars represents all the data variables used by the system,
	// with one Var for each resource that is made visible to the shader,
	// indexed by Group (@group) and Binding (@binding).
	// Each Var has Value(s) containing specific instance values.
	// Access through the System.Vars() method.
	vars Vars

	// ComputePipelines by name
	ComputePipelines map[string]*ComputePipeline

	// CommandEncoder is the command encoder created in
	// [BeginComputePass], and released in [EndComputePass].
	CommandEncoder *wgpu.CommandEncoder

	// logical device for this ComputeSystem, which we own.
	device *Device

	// gpu is our GPU device, which has properties
	// and alignment factors.
	gpu *GPU
}

// NewComputeSystem returns a new ComputeSystem, initialized with
// its own new device that is owned by the system.
func NewComputeSystem(gp *GPU, name string) *ComputeSystem {
	sy := &ComputeSystem{}
	sy.init(gp, name)
	return sy
}

// System interface:

func (sy *ComputeSystem) Vars() *Vars     { return &sy.vars }
func (sy *ComputeSystem) Device() *Device { return sy.device }
func (sy *ComputeSystem) GPU() *GPU       { return sy.gpu }
func (sy *ComputeSystem) Render() *Render { return nil }

// init initializes the ComputeSystem
func (sy *ComputeSystem) init(gp *GPU, name string) {
	sy.gpu = gp
	sy.Name = name
	sy.device = errors.Log1(NewDevice(gp))
	sy.vars.device = *sy.device
	sy.vars.sys = sy
	sy.ComputePipelines = make(map[string]*ComputePipeline)
}

// WaitDone waits until device is done with current processing steps
func (sy *ComputeSystem) WaitDone() {
	sy.device.WaitDone()
}

func (sy *ComputeSystem) Release() {
	sy.WaitDone()
	for _, pl := range sy.ComputePipelines {
		pl.Release()
	}
	sy.ComputePipelines = nil
	sy.vars.Release()
	sy.gpu = nil
}

// AddComputePipeline adds a new ComputePipeline to the system
func (sy *ComputeSystem) AddComputePipeline(name string) *ComputePipeline {
	pl := NewComputePipeline(name, sy)
	sy.ComputePipelines[pl.Name] = pl
	return pl
}

// Config configures the entire system, after Pipelines and Vars
// have been initialized.  After this point, just need to set
// values for the vars, and then do compute passes.  This should
// not need to be called more than once.
func (sy *ComputeSystem) Config() {
	sy.vars.Config(sy.device)
	if Debug {
		fmt.Printf("%s\n", sy.vars.StringDoc())
	}
	for _, pl := range sy.ComputePipelines {
		pl.Config(true)
	}
}

// NewCommandEncoder returns a new CommandEncoder for encoding
// compute commands.  This is automatically called by
// BeginRenderPass and the result maintained in [CommandEncoder].
func (sy *ComputeSystem) NewCommandEncoder() (*wgpu.CommandEncoder, error) {
	cmd, err := sy.device.Device.CreateCommandEncoder(nil)
	if errors.Log(err) != nil {
		return nil, err
	}
	return cmd, nil
}

// BeginComputePass adds commands to the given command buffer
// to start the compute pass, returning the encoder object
// to which further compute commands should be added.
// Call [EndComputePass] when done.
func (sy *ComputeSystem) BeginComputePass() (*wgpu.ComputePassEncoder, error) {
	cmd, err := sy.NewCommandEncoder()
	if errors.Log(err) != nil {
		return nil, err
	}
	sy.CommandEncoder = cmd
	return cmd.BeginComputePass(nil), nil // note: optional name in the descriptor
}

// EndComputePass submits the current compute commands to the device
// Queue and releases the [CommandEncoder] and the given
// ComputePassEncoder.  You must call ce.End prior to calling this.
// Can insert other commands after ce.End, e.g., to copy data back
// from the GPU, prior to calling EndComputePass.
func (sy *GraphicsSystem) EndComputePass(ce *wgpu.ComputePassEncoder) error {
	cmd := sy.CommandEncoder
	sy.CommandEncoder = nil
	cmdBuffer, err := cmd.Finish(nil)
	if errors.Log(err) != nil {
		return err
	}
	sy.device.Queue.Submit(cmdBuffer)
	cmdBuffer.Release()
	ce.Release()
	cmd.Release()
	return nil
}

// Warps returns the number of warps (work goups of compute threads)
// that is sufficient to compute n elements, given specified number
// of threads per this dimension.
// It just rounds up to nearest even multiple of n divided by threads:
// Ceil(n / threads)
func Warps(n, threads int) int {
	return int(math.Ceil(float64(n) / float64(threads)))
}

/*
// ComputeCopyToGPU records command to copy given regions
// in the Storage buffer memory from CPU to GPU, in one call.
// Use SyncRegValueIndexFromCPU to get the regions.
func (sy *ComputeSystem) ComputeCopyToGPU(cmd *wgpu.ComputePassEncoder, regs ...MemReg) {
	sy.Mem.CmdTransferStorageRegsToGPU(cmd, regs)
}

// ComputeCopyFromGPU records command to copy given regions
// in the Storage buffer memory from GPU to CPU, in one call.
// Use SyncRegValueIndexFromCPU to get the regions.
func (sy *ComputeSystem) ComputeCopyFromGPU(cmd *wgpu.ComputePassEncoder, regs ...MemReg) {
	sy.Mem.CmdTransferStorageRegsFromGPU(cmd, regs)
}

// ComputeWaitMemWriteRead records pipeline barrier ensuring
// global memory writes from the shader have completed and are ready to read
// in the next step of a command queue.
// Use this instead of Events to synchronize steps of a computation.
func (sy *ComputeSystem) ComputeWaitMemWriteRead(cmd *wgpu.ComputePassEncoder) {
	shader := vk.PipelineStageFlags(vk.PipelineStageComputeShaderBit)
	vk.CmdPipelineBarrier(cmd, shader, shader, vk.DependencyFlags(0), 1,
		[]vk.MemoryBarrier{{
			SType:         vk.StructureTypeMemoryBarrier,
			SrcAccessMask: vk.AccessFlags(vk.AccessShaderWriteBit),
			DstAccessMask: vk.AccessFlags(vk.AccessShaderReadBit),
		}}, 0, nil, 0, nil)
}

// ComputeWaitMemHostToShader records pipeline barrier ensuring
// global memory writes from the host to shader have completed.
// Use this if the first commands are to copy memory from host,
// instead of creating a separate Event.
func (sy *ComputeSystem) ComputeWaitMemHostToShader(cmd *wgpu.ComputePassEncoder) {
	shader := vk.PipelineStageFlags(vk.PipelineStageComputeShaderBit)
	host := vk.PipelineStageFlags(vk.PipelineStageHostBit)
	vk.CmdPipelineBarrier(cmd, host, shader, vk.DependencyFlags(0), 1,
		[]vk.MemoryBarrier{{
			SType:         vk.StructureTypeMemoryBarrier,
			SrcAccessMask: vk.AccessFlags(vk.AccessHostWriteBit),
			DstAccessMask: vk.AccessFlags(vk.AccessShaderReadBit),
		}}, 0, nil, 0, nil)
}

// ComputeWaitMemShaderToHost records pipeline barrier ensuring
// global memory writes have completed from the compute shader,
// and are ready for the host to read.
// This is not necessary if a standard QueueWaitIdle is done at
// the end of a command (basically not really needed,
// but included for completeness).
func (sy *ComputeSystem) ComputeWaitMemShaderToHost(cmd *wgpu.ComputePassEncoder) {
	shader := vk.PipelineStageFlags(vk.PipelineStageComputeShaderBit)
	host := vk.PipelineStageFlags(vk.PipelineStageHostBit)
	vk.CmdPipelineBarrier(cmd, shader, host, vk.DependencyFlags(0), 1,
		[]vk.MemoryBarrier{{
			SType:         vk.StructureTypeMemoryBarrier,
			SrcAccessMask: vk.AccessFlags(vk.AccessShaderWriteBit),
			DstAccessMask: vk.AccessFlags(vk.AccessHostReadBit),
		}}, 0, nil, 0, nil)
}

// ComputeWaitMemoryBuff records pipeline barrier ensuring
// given buffer's memory writes have completed for given buffer from the compute shader,
// and are ready for the host to read.  Vulkan docs suggest that global memory
// buffer barrier is generally better to use (ComputeWaitMem*)
func (sy *ComputeSystem) ComputeWaitMemoryBuff(cmd *wgpu.ComputePassEncoder, buff *Buffer) {
	shader := vk.PipelineStageFlags(vk.PipelineStageComputeShaderBit)
	host := vk.PipelineStageFlags(vk.PipelineStageHostBit)
	vk.CmdPipelineBarrier(cmd, shader, host, vk.DependencyFlags(0), 0, nil, 1,
		[]vk.BufferMemoryBarrier{{
			SType:         vk.StructureTypeBufferMemoryBarrier,
			SrcAccessMask: vk.AccessFlags(vk.AccessShaderWriteBit),
			DstAccessMask: vk.AccessFlags(vk.AccessHostReadBit),
			Buffer:        buff.Dev,
			Size:          vk.DeviceSize(buff.Size),
		}}, 0, nil)
}

// ComputeSubmitWait submits the current set of commands
// in the default system CmdPool, typically from ComputeDispatch.
// Then waits for the commands to finish before returning
// control to the CPU.  Results will be available immediately
// thereafter for retrieving back from the GPU.
func (sy *ComputeSystem) ComputeSubmitWait(cmd *wgpu.ComputePassEncoder) {
	CmdSubmitWait(cmd, &sy.Device)
}

// ComputeCmdEnd adds an end to given command buffer
func (sy *ComputeSystem) ComputeCmdEnd(cmd *wgpu.ComputePassEncoder) {
	CmdEnd(cmd)
}

*/
