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

// ComputeResetBindVars adds command to the default system CmdPool
// command buffer, to bind the Vars descriptors,
// for given collection of descriptors descIdx
// (see Vars NDescs for info).
// Required whenever variables have changed their mappings,
// before running a command.
func (sy *System) ComputeResetBindVars(descIdx int) {
	sy.CmdResetBindVars(sy.CmdPool.Buff, descIdx)
}

// ComputeResetBegin resets and begins the recording of commands
// on the default system CmdPool -- use prior to ComputeCommand
// if not needing to call ComputeBindVars
func (sy *System) ComputeResetBegin() {
	CmdResetBegin(sy.CmdPool.Buff)
}

// Warps returns the number of warps (thread groups) that is sufficient
// to compute n elements, given specified number of threads per this dimension.
// It just rounds up to nearest even multiple of n divided by threads:
// Ceil(n / threads)
func Warps(n, threads int) int {
	return int(math.Ceil(float64(n) / float64(threads)))
}

// ComputeCommand adds commands to run the compute shader for given
// number of *warps* (groups of threads) along 3 dimensions,
// which then generate indexes passed into the shader.
// In HLSL, the [numthreads(x, y, z)] directive specifies the number
// of threads allocated per warp -- the actual number of elements
// processed is threads * warps per each dimension. See Warps function.
// The hardware typically has 32 (NVIDIA, M1, M2) or 64 (AMD) hardware
// threads per warp, and so 64 is typically used as a default sum of
// threads per warp across all of the dimensions.
// Can use subsets of dimensions by using 1 for the other dimensions,
// and see ComputeCommand1D for a convenience method that automatically
// computes the number of warps for a 1D compute shader (everthing in x).
// Uses the system CmdPool -- must have a CmdBegin already executed,
// either via ComputeBindVars or ComputeResetBegin call.
// Must call CommandSubmit[Wait] to execute the command.
func (pl *Pipeline) ComputeCommand(nx, ny, nz int) {
	cmd := pl.Sys.CmdPool.Buff
	vk.CmdBindPipeline(cmd, vk.PipelineBindPointCompute, pl.VkPipeline)
	vk.CmdDispatch(cmd, uint32(nx), uint32(ny), uint32(nz))
}

// ComputeCommand1D adds commands to run the compute shader for given
// number of computational elements along the first (X) dimension,
// for given number *elements* and threads per warp (typically 64).
// See ComputeCommand for full info.
// This is just a convenience method for common 1D case that calls
// the Warps method for you.
func (pl *Pipeline) ComputeCommand1D(n, threads int) {
	pl.ComputeCommand(Warps(n, threads), 1, 1)
}

// ComputeSetEvent sets an event to be signalled when everything up to this point
// in the command buffer has completed.
// This is the best way to coordinate processing within a sequence of
// compute shader calls within a single command buffer.
// Returns an error if the named event was not found.
func (sy *System) ComputeSetEvent(event string) error {
	ev, err := sy.EventByNameTry(event)
	if err != nil {
		return err
	}
	vk.CmdSetEvent(sy.CmdPool.Buff, ev, vk.PipelineStageFlags(vk.PipelineStageComputeShaderBit))
	return nil
}

// ComputeWaitEvents waits until previous ComputeSetEvent event(s) have signalled.
// This is the best way to coordinate processing within a sequence of
// compute shader calls within a single command buffer.
// Returns an error if the named event was not found.
func (sy *System) ComputeWaitEvents(event ...string) error {
	evts := make([]vk.Event, len(event))
	for i, enm := range event {
		ev, err := sy.EventByNameTry(enm)
		if err != nil {
			return err
		}
		evts[i] = ev
	}
	flg := vk.PipelineStageFlags(vk.PipelineStageComputeShaderBit)
	vk.CmdWaitEvents(sy.CmdPool.Buff, uint32(len(evts)), evts, flg, flg, 0, nil, 0, nil, 0, nil)
	return nil
}

// ComputeCmdCopyToGPU records command to copy given regions
// in the Storage buffer memory from CPU to GPU, in one call.
// Use SyncRegValIdxFmCPU to get the regions.
func (sy *System) ComputeCmdCopyToGPU(regs ...MemReg) {
	sy.Mem.CmdTransferRegsToGPU(sy.CmdPool.Buff, sy.Mem.Buffs[StorageBuff], regs)
}

// ComputeCmdCopyFmGPU records command to copy given regions
// in the Storage buffer memory from GPU to CPU, in one call.
// Use SyncRegValIdxFmCPU to get the regions.
func (sy *System) ComputeCmdCopyFmGPU(regs ...MemReg) {
	sy.Mem.CmdTransferRegsFmGPU(sy.CmdPool.Buff, sy.Mem.Buffs[StorageBuff], regs)
}

/*
// ComputeCmdWaitMemory records command to wait for memory transfer to finish.
// use this after a ComputeCmdCopyToGPU, or FmGPU
func (sy *System) ComputeCmdWaitMemory() {
	vk.CmdPipelineBarrier(sy.CmdPool.Buff, []vk.MemoryBarrier{{
		SType: vk.StructureTypeMemoryBarrier,
		SrcAccessMask: ,
		DstAccessMask: ,
	}}
	sy.Mem.CmdTransferRegsToGPU(, sy.Mem.Buffs[StorageBuff], regs)
}
*/

// ComputeSubmitWait adds and End command and
// submits the current set of commands in the default system CmdPool,
// typically from ComputeCommand.
// Then waits for the commands to finish before returning
// control to the CPU.  Results will be available immediately
// thereafter for retrieving back fro the GPU.
// Uses ComputeFence to wait.
func (sy *System) ComputeSubmitWait() {
	fc, _ := sy.FenceByNameTry("ComputeWait") // always created
	CmdEnd(sy.CmdPool.Buff)
	CmdSubmitFence(sy.CmdPool.Buff, &sy.Device, fc)
	sy.ComputeWait()
}

// ComputeSubmitWaitSignal submits command in buffer to system device queue
// with given wait semaphore and given signal semaphore (by name) when done,
// and with given fence (use empty string for none).
// This will cause the GPU to wait until the wait semphaphore is
// signaled by a previous command with that semaphore as its signal.
// The optional fence is used typically at the end of a block of
// such commands, whenever the CPU needs to be sure the submitted GPU
// commands have completed.  Must use "ComputeWait" if using std
// ComputeWait function.
func (sy *System) ComputeSubmitWaitSignal(wait, signal, fence string) error {
	CmdEnd(sy.CmdPool.Buff)
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
	CmdSubmitWaitSignal(sy.CmdPool.Buff, &sy.Device, ws, ss, fc)
	return nil
}

// ComputeSubmitSignal submits command in buffer to system device queue
// with given signal semaphore (by name) when done,
// and with given fence (use empty string for none).
// The optional fence is used typically at the end of a block of
// such commands, whenever the CPU needs to be sure the submitted GPU
// commands have completed.  Must use "ComputeWait" if using std
// ComputeWait function.
func (sy *System) ComputeSubmitSignal(signal, fence string) error {
	CmdEnd(sy.CmdPool.Buff)
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
	CmdSubmitSignal(sy.CmdPool.Buff, &sy.Device, ss, fc)
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
