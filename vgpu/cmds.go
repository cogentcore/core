// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import vk "github.com/vulkan-go/vulkan"

// CmdPool is a command pool and buffer
type CmdPool struct {
	Pool vk.CommandPool
	Buff vk.CommandBuffer
}

// Init initializes the pool
func (cp *CmdPool) Init(dv *Device, flags vk.CommandPoolCreateFlagBits) {
	var cmdPool vk.CommandPool
	ret := vk.CreateCommandPool(dv.Device, &vk.CommandPoolCreateInfo{
		SType:            vk.StructureTypeCommandPoolCreateInfo,
		QueueFamilyIndex: dv.QueueIndex,
		Flags:            vk.CommandPoolCreateFlags(flags),
	}, nil, &cmdPool)
	IfPanic(NewError(ret))
	cp.Pool = cmdPool
}

// MakeBuff makes a buffer in pool
func (cp *CmdPool) MakeBuff(dv *Device) vk.CommandBuffer {
	var cmdBuff = make([]vk.CommandBuffer, 1)
	ret := vk.AllocateCommandBuffers(dv.Device, &vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		CommandPool:        cp.Pool,
		Level:              vk.CommandBufferLevelPrimary,
		CommandBufferCount: 1,
	}, cmdBuff)
	IfPanic(NewError(ret))
	cBuff := cmdBuff[0]
	cp.Buff = cBuff
	return cBuff
}

// BeginCmd does BeginCommandBuffer on buffer
func (cp *CmdPool) BeginCmd() {
	ret := vk.BeginCommandBuffer(cp.Buff, &vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
	})
	IfPanic(NewError(ret))
}

// BeginCmdOneTime does BeginCommandBuffer with OneTimeSubmit set on buffer
func (cp *CmdPool) BeginCmdOneTime() {
	ret := vk.BeginCommandBuffer(cp.Buff, &vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
		Flags: vk.CommandBufferUsageFlags(vk.CommandBufferUsageOneTimeSubmitBit),
	})
	IfPanic(NewError(ret))
}

// SubmitWait does End, Submit, WaitIdle on Buffer
func (cp *CmdPool) SubmitWait(dev *Device) {
	cp.EndCmd()
	cp.Submit(dev)
	vk.QueueWaitIdle(dev.Queue)
}

// SubmitWaitFree does End, Submit, WaitIdle, Free on Buffer
func (cp *CmdPool) SubmitWaitFree(dev *Device) {
	cp.SubmitWait(dev)
	cp.FreeBuffer(dev)
}

// EndCmd does EndCommandBuffer on buffer
func (cp *CmdPool) EndCmd() {
	vk.EndCommandBuffer(cp.Buff)
}

// Submit submits commands in buffer to given device queue
func (cp *CmdPool) Submit(dev *Device) {
	cmdBu := []vk.CommandBuffer{cp.Buff}
	ret := vk.QueueSubmit(dev.Queue, 1, []vk.SubmitInfo{{
		SType:              vk.StructureTypeSubmitInfo,
		CommandBufferCount: 1,
		PCommandBuffers:    cmdBu,
	}}, vk.NullFence)
	IfPanic(NewError(ret))
}

// FreeBuffer frees the current Buff buffer
func (cp *CmdPool) FreeBuffer(dev *Device) {
	cmdBu := []vk.CommandBuffer{cp.Buff}
	vk.FreeCommandBuffers(dev.Device, cp.Pool, 1, cmdBu)
	cp.Buff = nil
}

// Destroy
func (cp *CmdPool) Destroy(dev vk.Device) {
	if cp.Pool == nil {
		return
	}
	vk.DestroyCommandPool(dev, cp.Pool, nil)
	cp.Pool = nil
}
