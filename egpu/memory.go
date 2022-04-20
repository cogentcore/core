// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package egpu

import (
	"fmt"
	"log"
	"unsafe"

	vk "github.com/vulkan-go/vulkan"
)

// BuffAlloc is one memory allocation into a Vulkan Buffer.
// Used to generate vk.CmdBindVertexBuffers calls for Vectors,
// vkCmdBindIndexBuffer for Indexes
type BuffAlloc struct {
	Mem    *Memory `desc:"memory controller with "`
	Offset int     `desc:"starting offset"`
	Size   int     `desc:"size"`
}

func (ba *BuffAlloc) Set(mem *Memory, off, size int) {
	ba.Mem = mem
	ba.Offset = off
	ba.Size = size
}

func (ba *BuffAlloc) Free() {
	ba.Set(nil, 0, 0)
}

// BuffMgrAlloc is one memory allocation for a given buffer manager.
// This is the level at which the Memory object manages buffer memory.
type BuffMgrAlloc struct {
	Buff   *BufferMgr
	Offset int `desc:"starting offset"`
	Size   int `desc:"size"`
}

// Memory manages the memory for the GPU, in terms of BufferMgr allocations.
// You configure all the BufferMgrs and then allocate them here.
type Memory struct {
	GPU     *GPU
	Device  Device  `desc:"logical device that this memory is managed for: a Surface or GPU itself"`
	CmdPool CmdPool `desc:"command pool for memory transfers"`

	BuffAllocs   []*BuffMgrAlloc              `desc:"allocations for buffermgrs, in order added"`
	BuffAllocMap map[*BufferMgr]*BuffMgrAlloc `desc:"allocations for buffermgrs, as map"`
	BuffSize     int                          `desc:"total size in bytes allocated"`

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

// AddBuff adds BufferMgr to be managed by this memory manager
func (mm *Memory) AddBuff(buff *BufferMgr) {
	if mm.BuffAllocMap == nil {
		mm.BuffAllocMap = make(map[*BufferMgr]*BuffMgrAlloc)
	}
	ba := &BuffMgrAlloc{Buff: buff}
	mm.BuffAllocMap[buff] = ba
	mm.BuffAllocs = append(mm.BuffAllocs, ba)
}

// BuffMemSize returns the total size needs in bytes for buffers
func (mm *Memory) BuffMemSize() int {
	sz := 0
	for _, ba := range mm.BuffAllocs {
		sz += ba.Buff.MemSize()
	}
	return sz
}

func (mm *Memory) Destroy() {
	mm.Free()
	mm.GPU = nil
}

// Config should be called after all memory elements have been configured
// and are ready to go with their initial data.
// Does: Alloc(), AllocDev(), CopyToStaging(), TransferAllToGPU()
func (mm *Memory) Config() {
	mm.Alloc()
	mm.AllocDev()
	mm.CopyBuffsToStaging()
	mm.TransferBuffAllToGPU()
	mm.Active = true
}

// Alloc allocates memory for all buffs (if size changed)
func (mm *Memory) Alloc() {
	nsz := mm.BuffMemSize()
	if nsz != mm.BuffSize {
		mm.Free()

		mm.BuffSize = nsz
		if mm.BuffSize == 0 {
			return
		}

		mm.BuffHost = mm.MakeBuffer(mm.BuffSize, vk.BufferUsageTransferSrcBit)
		mm.BuffDev = mm.MakeBuffer(mm.BuffSize, vk.BufferUsageTransferDstBit|vk.BufferUsageVertexBufferBit|vk.BufferUsageIndexBufferBit)
		mm.BuffHostMem = mm.AllocMem(mm.BuffHost, vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit)
	}

	// now allocate views into main buffer memory
	off := 0
	for _, ba := range mm.BuffAllocs {
		sz := ba.Buff.Alloc(mm, off)
		off += sz
	}
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
	for _, ba := range mm.BuffAllocs {
		ba.Buff.Free()
	}
	mm.FreeBuffMem(&mm.BuffDevMem)
	vk.DestroyBuffer(mm.Device.Device, mm.BuffDev, nil)
	mm.FreeBuffMem(&mm.BuffHostMem)
	vk.DestroyBuffer(mm.Device.Device, mm.BuffHost, nil)
	mm.BuffSize = 0
	mm.BuffHost = nil
	mm.BuffDev = nil
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

// BuffMemCopy copies source data into destination buffer represented by
// a dst unsafe.Pointer to start of buffer.
// call with unsafe.Pointer(&buff[0]) to first element in source buffer.
func BuffMemCopy(ba *BuffAlloc, dst unsafe.Pointer, src unsafe.Pointer) error {
	if ba.Size == 0 {
		return nil
	}
	const m = 0x7fffffff
	dstView := (*[m]byte)(unsafe.Pointer(uintptr(dst) + uintptr(ba.Offset)))
	srcView := (*[m]byte)(src)
	n := copy(dstView[:ba.Size], srcView[:ba.Size])
	if n != ba.Size {
		err := fmt.Errorf("vulkan BuffMemCopy warning: failed to copy data, %d != %d", n, ba.Size)
		log.Println(err)
		return err
	}
	return nil
}

// CopyBuffsToStaging copies all of the buffer source data into the CPU side staging buffer.
// this does not check for changes -- use for initial configuration.
// See SyncBuffsToStaging for a only updates.
func (mm *Memory) CopyBuffsToStaging() {
	if mm.BuffSize == 0 || mm.BuffHostMem == nil {
		return
	}
	var bufPtr unsafe.Pointer
	ret := vk.MapMemory(mm.Device.Device, mm.BuffHostMem, 0, vk.DeviceSize(mm.BuffSize), 0, &bufPtr)
	if IsError(ret) {
		log.Printf("vulkan Memory:CopyBuffs warning: failed to map device memory for data (len=%d)", mm.BuffSize)
		return
	}
	for _, ba := range mm.BuffAllocs {
		ba.Buff.CopyBuffsToStaging(bufPtr)
	}
	vk.UnmapMemory(mm.Device.Device, mm.BuffHostMem)
}

// SyncBuffsToStaging copies only changed buffer source data to CPU side staging buffer.
// returns a list of allocs that were updated
func (mm *Memory) SyncBuffsToStaging() []*BuffAlloc {
	if mm.BuffSize == 0 || mm.BuffDevMem == nil {
		return nil
	}
	var bufPtr unsafe.Pointer
	ret := vk.MapMemory(mm.Device.Device, mm.BuffHostMem, 0, vk.DeviceSize(mm.BuffSize), 0, &bufPtr)
	if IsError(ret) {
		log.Printf("vulkan Memory:CopyBuffs warning: failed to map device memory for data (len=%d)", mm.BuffSize)
		return nil
	}
	var as []*BuffAlloc
	for _, ba := range mm.BuffAllocs {
		aa := ba.Buff.SyncBuffsToStaging(bufPtr)
		if aa != nil {
			as = append(as, aa)
		}
	}
	vk.UnmapMemory(mm.Device.Device, mm.BuffHostMem)
	return as
}

// SyncAllToGPU syncs everything to GPU
func (mm *Memory) SyncAllToGPU() {
	as := mm.SyncBuffsToStaging()
	mm.TransferBuffToGPU(as)
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
	mm.TransferBuffToGPU([]*BuffAlloc{&BuffAlloc{Offset: 0, Size: mm.BuffSize}})
}

// TransferBuffToGPU transfers buff memory from CPU to GPU for given allocs
func (mm *Memory) TransferBuffToGPU(allocs []*BuffAlloc) {
	if mm.BuffSize == 0 || mm.BuffDevMem == nil {
		return
	}

	cmdBuff := mm.CmdPool.MakeBuff(&mm.Device)

	ret := vk.BeginCommandBuffer(cmdBuff, &vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
		Flags: vk.CommandBufferUsageFlags(vk.CommandBufferUsageOneTimeSubmitBit),
	})
	IfPanic(NewError(ret))

	regs := make([]vk.BufferCopy, len(allocs))
	for i, ba := range allocs {
		regs[i] = vk.BufferCopy{SrcOffset: vk.DeviceSize(ba.Offset), DstOffset: vk.DeviceSize(ba.Offset), Size: vk.DeviceSize(ba.Size)}
	}

	vk.CmdCopyBuffer(cmdBuff, mm.BuffHost, mm.BuffDev, uint32(len(regs)), regs)
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
