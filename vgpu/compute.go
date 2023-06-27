// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"embed"
	"math"

	vk "github.com/goki/vulkan"
)

// https://www.reddit.com/r/vulkan/comments/us9p72/multiple_command_buffers_vs_multiple_batches_of/

// NewComputePipelineEmbed returns a new pipeline added to this System,
// using given file name from given embed.FS filesystem as a ComputeShader.
func (sy *System) NewComputePipelineEmbed(name string, efs embed.FS, fname string) *Pipeline {
	pl := sy.NewPipeline(name)
	pl.AddShaderEmbed(name, ComputeShader, efs, fname)
	return pl
}

// ComputeCmdBuff returns the default compute command buffer: CmdPool.Buff
// which can be used for executing arbitrary compute commands.
func (sy *System) ComputeCmdBuff() vk.CommandBuffer {
	return sy.CmdPool.Buff
}

// ComputeResetBindVars adds command to the given
// command buffer, to bind the Vars descriptors,
// for given collection of descriptors descIdx
// (see Vars NDescs for info).
// Required whenever variables have changed their mappings,
// before running a command.
func (sy *System) ComputeResetBindVars(cmd vk.CommandBuffer, descIdx int) {
	sy.CmdResetBindVars(cmd, descIdx)
}

// ComputeResetBegin resets and begins the recording of commands
// the given command buffer -- use prior to ComputeCommand
// if not needing to call ComputeBindVars
func (sy *System) ComputeResetBegin(cmd vk.CommandBuffer) {
	CmdResetBegin(cmd)
}

// Warps returns the number of warps (thread groups) that is sufficient
// to compute n elements, given specified number of threads per this dimension.
// It just rounds up to nearest even multiple of n divided by threads:
// Ceil(n / threads)
func Warps(n, threads int) int {
	return int(math.Ceil(float64(n) / float64(threads)))
}

// ComputeDispatch adds commands to given cmd buffer to run the compute
// shader for given number of *warps* (groups of threads) along 3 dimensions,
// which then generate indexes passed into the shader.
// In HLSL, the [numthreads(x, y, z)] directive specifies the number
// of threads allocated per warp -- the actual number of elements
// processed is threads * warps per each dimension. See Warps function.
// The hardware typically has 32 (NVIDIA, M1, M2) or 64 (AMD) hardware
// threads per warp, and so 64 is typically used as a default sum of
// threads per warp across all of the dimensions.
// Can use subsets of dimensions by using 1 for the other dimensions,
// and see ComputeDispatch1D for a convenience method that automatically
// computes the number of warps for a 1D compute shader (everthing in x).
// Must have a CmdBegin already executed, either via ComputeBindVars
// or ComputeResetBegin call.
// Must call CommandSubmit[Wait] to execute the command.
func (pl *Pipeline) ComputeDispatch(cmd vk.CommandBuffer, nx, ny, nz int) {
	vk.CmdBindPipeline(cmd, vk.PipelineBindPointCompute, pl.VkPipeline)
	vk.CmdDispatch(cmd, uint32(nx), uint32(ny), uint32(nz))
}

// ComputeDispatch1D adds commands to run the compute shader for given
// number of computational elements along the first (X) dimension,
// for given number *elements* and threads per warp (typically 64).
// See ComputeDispatch for full info.
// This is just a convenience method for common 1D case that calls
// the Warps method for you.
func (pl *Pipeline) ComputeDispatch1D(cmd vk.CommandBuffer, n, threads int) {
	pl.ComputeDispatch(cmd, Warps(n, threads), 1, 1)
}

// ComputeSetEvent sets an event to be signalled when everything
// up to this point in the named command buffer has completed.
// This is the best way to coordinate processing within a sequence of
// compute shader calls within a single command buffer.
// Returns an error if the named event was not found.
func (sy *System) ComputeSetEvent(cmd vk.CommandBuffer, event string) error {
	ev, err := sy.EventByNameTry(event)
	if err != nil {
		return err
	}
	vk.CmdSetEvent(cmd, ev, vk.PipelineStageFlags(vk.PipelineStageComputeShaderBit))
	return nil
}

// ComputeWaitEvents waits until previous ComputeSetEvent event(s) have signalled.
// This is the best way to coordinate processing within a sequence of
// compute shader calls within named command buffer.
// However, use ComputeWaitMem* calls (e.g., WriteRead) to ensure memory writes
// have completed, instead of creating an Event.
// Returns an error if the named event was not found.
func (sy *System) ComputeWaitEvents(cmd vk.CommandBuffer, event ...string) error {
	evts := make([]vk.Event, len(event))
	for i, enm := range event {
		ev, err := sy.EventByNameTry(enm)
		if err != nil {
			return err
		}
		evts[i] = ev
	}
	flg := vk.PipelineStageFlags(vk.PipelineStageComputeShaderBit)
	vk.CmdWaitEvents(cmd, uint32(len(evts)), evts, flg, flg, 0, nil, 0, nil, 0, nil)
	return nil
}

// ComputeCopyToGPU records command to copy given regions
// in the Storage buffer memory from CPU to GPU, in one call.
// Use SyncRegValIdxFmCPU to get the regions.
func (sy *System) ComputeCopyToGPU(cmd vk.CommandBuffer, regs ...MemReg) {
	sy.Mem.CmdTransferStorageRegsToGPU(cmd, regs)
}

// ComputeCopyFmGPU records command to copy given regions
// in the Storage buffer memory from GPU to CPU, in one call.
// Use SyncRegValIdxFmCPU to get the regions.
func (sy *System) ComputeCopyFmGPU(cmd vk.CommandBuffer, regs ...MemReg) {
	sy.Mem.CmdTransferStorageRegsFmGPU(cmd, regs)
}

// ComputeWaitMemWriteRead records pipeline barrier ensuring
// global memory writes from the shader have completed and are ready to read
// in the next step of a command queue.
// Use this instead of Events to synchronize steps of a computation.
func (sy *System) ComputeWaitMemWriteRead(cmd vk.CommandBuffer) {
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
func (sy *System) ComputeWaitMemHostToShader(cmd vk.CommandBuffer) {
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
func (sy *System) ComputeWaitMemShaderToHost(cmd vk.CommandBuffer) {
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
func (sy *System) ComputeWaitMemoryBuff(cmd vk.CommandBuffer, buff *MemBuff) {
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
func (sy *System) ComputeSubmitWait(cmd vk.CommandBuffer) {
	CmdSubmitWait(cmd, &sy.Device)
}

// ComputeCmdEnd adds an end to given command buffer
func (sy *System) ComputeCmdEnd(cmd vk.CommandBuffer) {
	CmdEnd(cmd)
}

// ComputeSubmitWaitSignal submits command in given buffer to system device queue
// with given wait semaphore and given signal semaphore (by name) when done,
// and with given fence (use empty string for none).
// This will cause the GPU to wait until the wait semphaphore is
// signaled by a previous command with that semaphore as its signal.
// The optional fence is used typically at the end of a block of
// such commands, whenever the CPU needs to be sure the submitted GPU
// commands have completed.  Must use "ComputeWait" if using std
// ComputeWait function.
func (sy *System) ComputeSubmitWaitSignal(cmd vk.CommandBuffer, wait, signal, fence string) error {
	CmdEnd(cmd)
	ws, err := sy.SemaphoreByNameTry(wait)
	if err != nil {
		return err
	}
	ss, err := sy.SemaphoreByNameTry(signal)
	if err != nil {
		return err
	}
	fc := vk.NullFence
	if fence != "" {
		fc, err = sy.FenceByNameTry(fence)
		if err != nil {
			return err
		}
	}
	CmdSubmitWaitSignal(cmd, &sy.Device, ws, ss, fc)
	return nil
}

// ComputeSubmitSignal submits command in buffer to system device queue
// with given signal semaphore (by name) when done,
// and with given fence (use empty string for none).
// The optional fence is used typically at the end of a block of
// such commands, whenever the CPU needs to be sure the submitted GPU
// commands have completed.  Must use "ComputeWait" if using std
// ComputeWait function.
func (sy *System) ComputeSubmitSignal(cmd vk.CommandBuffer, signal, fence string) error {
	CmdEnd(cmd)
	ss, err := sy.SemaphoreByNameTry(signal)
	if err != nil {
		return err
	}
	fc := vk.NullFence
	if fence != "" {
		fc, err = sy.FenceByNameTry(fence)
		if err != nil {
			return err
		}
	}
	CmdSubmitSignal(cmd, &sy.Device, ss, fc)
	return nil
}

// ComputeWaitFence waits for given fence (by name), and resets the fence
func (sy *System) ComputeWaitFence(fence string) error {
	fc, err := sy.FenceByNameTry(fence)
	if err != nil {
		return err
	}
	vk.WaitForFences(sy.Device.Device, 1, []vk.Fence{fc}, vk.True, vk.MaxUint64)
	vk.ResetFences(sy.Device.Device, 1, []vk.Fence{fc})
	return nil
}

// ComputeWait waits for the standard ComputeWait fence
func (sy *System) ComputeWait() error {
	fc, _ := sy.FenceByNameTry("ComputeWait")
	vk.WaitForFences(sy.Device.Device, 1, []vk.Fence{fc}, vk.True, vk.MaxUint64)
	vk.ResetFences(sy.Device.Device, 1, []vk.Fence{fc})
	return nil
}
