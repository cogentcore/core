// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"embed"
	"math"

	vk "github.com/goki/vulkan"
)

// todo: more work needed to figure out how to actually run
// more efficiently -- Submit is the biggie time-wise.

// https://www.reddit.com/r/vulkan/comments/us9p72/multiple_command_buffers_vs_multiple_batches_of/

// NewComputePipelineEmbed returns a new pipeline added to this System,
// using given file name from given embed.FS filesystem as a ComputeShader.
func (sy *System) NewComputePipelineEmbed(name string, efs embed.FS, fname string) *Pipeline {
	pl := sy.NewPipeline(name)
	pl.AddShaderEmbed(name, ComputeShader, efs, fname)
	return pl
}

// ComputeBindVars adds command to the default system CmdPool
// command buffer, to bind the Vars descriptors,
// for given collection of descriptors descIdx
// (see Vars NDescs for info).
// Required whenever variables have changed their mappings,
// before running a command.
func (sy *System) ComputeBindVars(descIdx int) {
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
	CmdEnd(cmd)
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

// ComputeSubmit submits the current set of commands in
// the default system CmdPool, typically from ComputeCommand.
// does NOT wait for the commands to finish before returning
// control to the CPU.  Can call this multiple times to
// run the same command iteratively, followed by a final wait,
// for example.
func (sy *System) ComputeSubmit() {
	CmdSubmit(sy.CmdPool.Buff, &sy.Device)
}

// ComputeSubmitWait submits the current set of commands in
// the default system CmdPool, typically from ComputeCommand.
// and then waits for the commands to finish before returning
// control to the CPU.  Results will be available immediately
// thereafter for retrieving back fro the GPU.
func (sy *System) ComputeSubmitWait() {
	CmdSubmitWait(sy.CmdPool.Buff, &sy.Device)
}

// ComputeWait waits for previously-submitted commands to finish.
// Results will be available immediately thereafter for
// retrieving back fro the GPU.
func (sy *System) ComputeWait() {
	CmdWait(&sy.Device)
}
