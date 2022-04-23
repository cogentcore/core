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

	ret = vk.BeginCommandBuffer(cBuff, &vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
	})
	IfPanic(NewError(ret))
	return cBuff
}

// Destroy
func (cp *CmdPool) Destroy(dv *Device) {
	if cp.Pool == nil {
		return
	}
	vk.DestroyCommandPool(dv.Device, cp.Pool, nil)
	cp.Pool = nil
}
