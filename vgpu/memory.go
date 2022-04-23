// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"log"
	"unsafe"

	vk "github.com/vulkan-go/vulkan"
)

// MemReg is a region of memory
type MemReg struct {
	Offset int
	Size   int
}

// Memory manages memory for the GPU, using separate buffers for
// Images (Textures) vs. other values.
type Memory struct {
	GPU     *GPU
	Device  Device  `desc:"logical device that this memory is managed for: a Surface or GPU itself"`
	CmdPool CmdPool `desc:"command pool for memory transfers"`

	Vals   Vals `desc:"values encoded in this Memory -- no Images here!"`
	Images Vals `desc:"Image-type values"`

	BuffSize    int             `desc:"allocated buffer size"`
	BuffHost    vk.Buffer       `desc:"logical descriptor for host CPU-visible memory, for staging"`
	BuffHostMem vk.DeviceMemory `desc:"host CPU-visible memory, for staging"`
	BuffDev     vk.Buffer       `desc:"logical descriptor for device GPU-local memory, for computation"`
	BuffDevMem  vk.DeviceMemory `desc:"device GPU-local memory, for computation"`

	Active bool `desc:"device memory is allocated and tranferred -- ready for use"`
}

// Init configures the Memory for use with given gpu, device, and associated queueindex
func (mm *Memory) Init(gp *GPU, device *Device) {
	mm.GPU = gp
	mm.Device = *device
	mm.CmdPool.Init(device, vk.CommandPoolCreateTransientBit)
}

func (mm *Memory) Destroy() {
	mm.Free()
	mm.GPU = nil
}

// Config should be called after all Vals have been configured
// and are ready to go with their initial data.
// Does: Alloc(), AllocDev(), CopyToStaging(), TransferAllToGPU()
// func (mm *Memory) Config() {
// 	mm.Alloc()
// 	mm.AllocDev()
// 	mm.CopyBuffsToStaging()
// 	mm.TransferBuffAllToGPU()
// 	mm.Active = true
// }

// Alloc allocates memory for all Vars and Images
func (mm *Memory) Alloc() {
	bsz := mm.Vals.MemSize()
	if bsz != mm.BuffSize {
		mm.BuffHost = mm.MakeBuffer(mm.BuffSize, vk.BufferUsageTransferSrcBit)
		mm.BuffDev = mm.MakeBuffer(mm.BuffSize, vk.BufferUsageTransferDstBit|vk.BufferUsageVertexBufferBit|vk.BufferUsageIndexBufferBit)
		mm.BuffHostMem = mm.AllocMem(mm.BuffHost, vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit)
		mm.BuffSize = bsz
	}
	var buffPtr unsafe.Pointer
	ret := vk.MapMemory(mm.Device.Device, mm.BuffHostMem, 0, vk.DeviceSize(mm.BuffSize), 0, &buffPtr)
	if IsError(ret) {
		log.Printf("vulkan Memory:CopyBuffs warning: failed to map device memory for data (len=%d)", mm.BuffSize)
		return
	}
	mm.Vals.Alloc(buffPtr, 0)
}

// AllocDev allocates memory on the device
func (mm *Memory) AllocDev() {
	mm.BuffDevMem = mm.AllocMem(mm.BuffDev, vk.MemoryPropertyDeviceLocalBit)
}

// MakeBuffer makes a buffer of given size, usage
func (mm *Memory) MakeBuffer(size int, usage vk.BufferUsageFlagBits) vk.Buffer {
	var buffer vk.Buffer
	ret := vk.CreateBuffer(mm.Device.Device, &vk.BufferCreateInfo{
		SType: vk.StructureTypeBufferCreateInfo,
		Usage: vk.BufferUsageFlags(usage),
		Size:  vk.DeviceSize(size),
	}, nil, &buffer)
	IfPanic(NewError(ret))
	return buffer
}

// AllocMem allocates memory for given buffer, with given properties
func (mm *Memory) AllocMem(buffer vk.Buffer, props vk.MemoryPropertyFlagBits) vk.DeviceMemory {
	// Ask device about its memory requirements.
	var memReqs vk.MemoryRequirements
	vk.GetBufferMemoryRequirements(mm.Device.Device, buffer, &memReqs)
	memReqs.Deref()

	memProps := mm.GPU.MemoryProps
	memType, ok := FindRequiredMemoryType(memProps, vk.MemoryPropertyFlagBits(memReqs.MemoryTypeBits), props)
	if !ok {
		log.Println("vulkan warning: failed to find required memory type")
	}

	var memory vk.DeviceMemory
	// Allocate device memory and bind to the buffer.
	ret := vk.AllocateMemory(mm.Device.Device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memType,
	}, nil, &memory)
	IfPanic(NewError(ret))
	vk.BindBufferMemory(mm.Device.Device, buffer, memory, 0)
	return memory
}

// FreeBuffMem frees given device memory to nil
func (mm *Memory) FreeBuffMem(memory *vk.DeviceMemory) {
	if *memory == nil {
		return
	}
	vk.FreeMemory(mm.Device.Device, *memory, nil)
	*memory = nil
}

// Free frees any allocated memory -- returns true if freed
func (mm *Memory) Free() bool {
	if mm.BuffSize == 0 {
		return false
	}
	vk.UnmapMemory(mm.Device.Device, mm.BuffHostMem)
	mm.Vals.Free()
	mm.FreeBuffMem(&mm.BuffDevMem)
	vk.DestroyBuffer(mm.Device.Device, mm.BuffDev, nil)
	mm.FreeBuffMem(&mm.BuffHostMem)
	vk.DestroyBuffer(mm.Device.Device, mm.BuffHost, nil)
	mm.BuffSize = 0
	mm.BuffHost = nil
	mm.BuffDev = nil
	mm.Active = false
	return true
}

// Deactivate deactivates device memory
func (mm *Memory) Deactivate() {
	mm.FreeBuffMem(&mm.BuffDevMem)
	mm.Active = false
}

// Activate ensures device memory is ready to use
// assumes the staging memory is configured.
// Call Sync after this if needed.
func (mm *Memory) Activate() {
	if mm.Active {
		return
	}
	if mm.BuffDevMem == nil {
		mm.AllocDev()
		mm.TransferAllToGPU()
	}
	mm.Active = true
}

// SyncAllToGPU syncs everything to GPU
func (mm *Memory) SyncAllToGPU() {
	mods := mm.Vals.ModRegs()
	if len(mods) == 0 {
		return
	}
	mm.TransferBuffToGPU(mods)
}

// TransferAllToGPU transfers all staging to GPU
func (mm *Memory) TransferAllToGPU() {
	mm.TransferBuffAllToGPU()
}

// TransferBuffAllToGPU transfers entire staging buffer of memory from CPU to GPU
func (mm *Memory) TransferBuffAllToGPU() {
	if mm.BuffSize == 0 || mm.BuffDevMem == nil {
		return
	}
	mm.TransferBuffToGPU([]MemReg{{Offset: 0, Size: mm.BuffSize}})
}

// TransferBuffToGPU transfers buff memory from CPU to GPU for given regs
func (mm *Memory) TransferBuffToGPU(regs []MemReg) {
	if mm.BuffSize == 0 || mm.BuffDevMem == nil {
		return
	}

	cmdBuff := mm.CmdPool.MakeBuff(&mm.Device)

	ret := vk.BeginCommandBuffer(cmdBuff, &vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
		Flags: vk.CommandBufferUsageFlags(vk.CommandBufferUsageOneTimeSubmitBit),
	})
	IfPanic(NewError(ret))

	rg := make([]vk.BufferCopy, len(regs))
	for i, mr := range regs {
		rg[i] = vk.BufferCopy{SrcOffset: vk.DeviceSize(mr.Offset), DstOffset: vk.DeviceSize(mr.Offset), Size: vk.DeviceSize(mr.Size)}
	}

	vk.CmdCopyBuffer(cmdBuff, mm.BuffHost, mm.BuffDev, uint32(len(rg)), rg)
	vk.EndCommandBuffer(cmdBuff)

	cmdBu := []vk.CommandBuffer{cmdBuff}

	ret = vk.QueueSubmit(mm.Device.Queue, 1, []vk.SubmitInfo{{
		SType:              vk.StructureTypeSubmitInfo,
		CommandBufferCount: 1,
		PCommandBuffers:    cmdBu,
	}}, vk.NullFence)
	IfPanic(NewError(ret))

	vk.QueueWaitIdle(mm.Device.Queue)
	vk.FreeCommandBuffers(mm.Device.Device, mm.CmdPool.Pool, 1, cmdBu)
}

func FindRequiredMemoryType(props vk.PhysicalDeviceMemoryProperties,
	deviceRequirements, hostRequirements vk.MemoryPropertyFlagBits) (uint32, bool) {

	for i := uint32(0); i < vk.MaxMemoryTypes; i++ {
		if deviceRequirements&(vk.MemoryPropertyFlagBits(1)<<i) != 0 {
			props.MemoryTypes[i].Deref()
			flags := props.MemoryTypes[i].PropertyFlags
			if flags&vk.MemoryPropertyFlags(hostRequirements) != 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func FindRequiredMemoryTypeFallback(props vk.PhysicalDeviceMemoryProperties,
	deviceRequirements, hostRequirements vk.MemoryPropertyFlagBits) (uint32, bool) {

	for i := uint32(0); i < vk.MaxMemoryTypes; i++ {
		if deviceRequirements&(vk.MemoryPropertyFlagBits(1)<<i) != 0 {
			props.MemoryTypes[i].Deref()
			flags := props.MemoryTypes[i].PropertyFlags
			if flags&vk.MemoryPropertyFlags(hostRequirements) != 0 {
				return i, true
			}
		}
	}
	// Fallback to the first one available.
	if hostRequirements != 0 {
		return FindRequiredMemoryType(props, deviceRequirements, 0)
	}
	return 0, false
}
