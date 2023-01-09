// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import vk "github.com/goki/vulkan"

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

// ComputeCommand adds commands to run the compute shader for given
// number of computational elements along 3 dimensions,
// which are passed as indexes into the shader.
// Can use subsets of dimensions by using 1 for the other dimensions.
// Uses the system CmdPool -- must have a CmdBegin already executed,
// either via ComputeBindVars or ComputeResetBegin call.
// Must call CommandSubmit[Wait] to execute the command.
func (pl *Pipeline) ComputeCommand(nx, ny, nz int) {
	cmd := pl.Sys.CmdPool.Buff
	vk.CmdBindPipeline(cmd, vk.PipelineBindPointCompute, pl.VkPipeline)
	vk.CmdDispatch(cmd, uint32(nx), uint32(ny), uint32(nz))
	CmdEnd(cmd)
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
