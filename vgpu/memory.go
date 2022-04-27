// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"log"
	"unsafe"

	vk "github.com/vulkan-go/vulkan"
)

// MemSizeAlign returns the size aligned according to align byte increments
// e.g., if align = 16 and size = 12, it returns 16
func MemSizeAlign(size, align int) int {
	if size%align == 0 {
		return size
	}
	nb := size / align
	return (nb + 1) * align
}

// MemReg is a region of memory
type MemReg struct {
	Offset int
	Size   int
}

// Memory manages memory for the GPU, using separate buffers for
// Images (Textures) vs. other values.
type Memory struct {
	GPU     *GPU
	Device  Device               `desc:"logical device that this memory is managed for: a Surface or GPU itself"`
	CmdPool CmdPool              `desc:"command pool for memory transfers"`
	Vals    Vals                 `desc:"values of Vars, each with a unique name -- can be any number of different values per same Var (e.g., different meshes with vertex data) -- up to user code to bind each Var prior to pipeline execution.  Each of these Vals is mapped into GPU memory."`
	Buffs   [BuffTypesN]*MemBuff `desc:"memory buffers"`
}

// Init configures the Memory for use with given gpu, device, and associated queueindex
func (mm *Memory) Init(gp *GPU, device *Device) {
	mm.GPU = gp
	mm.Device = *device
	mm.CmdPool.Init(device, vk.CommandPoolCreateTransientBit)
	for bt := VtxIdxBuff; bt < BuffTypesN; bt++ {
		mm.Buffs[bt] = &MemBuff{Type: bt}
	}
}

func (mm *Memory) Destroy(dev vk.Device) {
	mm.Free()
	mm.CmdPool.Destroy(dev)
	mm.GPU = nil
}

// Config should be called after all Vals have been configured
// and are ready to go with their initial data.
// Does: Alloc(), AllocDev(), CopyToStaging(), TransferAllToGPU()
func (mm *Memory) Config() {
	mm.Alloc()
	mm.AllocDev()
	mm.TransferToGPU()
}

// Alloc allocates memory for all bufers
func (mm *Memory) Alloc() {
	for bt := VtxIdxBuff; bt < BuffTypesN; bt++ {
		mm.AllocBuff(bt)
	}
}

// AllocBuff allocates memory for given buffer
func (mm *Memory) AllocBuff(bt BuffTypes) {
	buff := mm.Buffs[bt]
	bsz := mm.Vals.MemSize(bt)
	if bsz != buff.Size {
		usage := BuffUsages[buff.Type]
		hostUse := usage
		devUse := usage
		if bt.IsReadOnly() {
			hostUse |= vk.BufferUsageTransferSrcBit
			devUse |= vk.BufferUsageTransferDstBit
		} else {
			hostUse |= vk.BufferUsageTransferSrcBit | vk.BufferUsageTransferDstBit
			devUse |= vk.BufferUsageTransferSrcBit | vk.BufferUsageTransferDstBit
		}
		buff.Host = mm.MakeBuffer(bsz, hostUse)
		buff.Dev = mm.MakeBuffer(bsz, devUse)
		buff.HostMem = mm.AllocMem(buff.Host, vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit)
		buff.Size = bsz

		var buffPtr unsafe.Pointer
		ret := vk.MapMemory(mm.Device.Device, buff.HostMem, 0, vk.DeviceSize(buff.Size), 0, &buffPtr)
		if IsError(ret) {
			log.Printf("vulkan Memory:CopyBuffs warning: failed to map device memory for data (len=%d)", buff.Size)
			return
		}
		buff.HostPtr = buffPtr
		buff.AlignBytes = buff.Type.AlignBytes(mm.GPU)
	}
	mm.Vals.Alloc(buff, 0)
}

// AllocDev allocates device memory for all bufers
func (mm *Memory) AllocDev() {
	for bt := VtxIdxBuff; bt < BuffTypesN; bt++ {
		mm.AllocDevBuff(bt)
	}
}

// AllocDevBuff allocates memory on the device for given buffer
func (mm *Memory) AllocDevBuff(bt BuffTypes) {
	buff := mm.Buffs[bt]
	if buff.Size == 0 {
		return
	}
	buff.DevMem = mm.AllocMem(buff.Dev, vk.MemoryPropertyDeviceLocalBit)
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

// Free frees memory for all buffers -- returns true if any freed
func (mm *Memory) Free() bool {
	freed := false
	for bt := VtxIdxBuff; bt < BuffTypesN; bt++ {
		fr := mm.FreeBuff(bt)
		if fr {
			freed = true
		}
	}
	return freed
}

// FreeBuff frees any allocated memory in buffer -- returns true if freed
func (mm *Memory) FreeBuff(bt BuffTypes) bool {
	buff := mm.Buffs[bt]
	if buff.Size == 0 {
		return false
	}
	vk.UnmapMemory(mm.Device.Device, buff.HostMem)
	mm.Vals.Free(buff)
	mm.FreeBuffMem(&buff.DevMem)
	vk.DestroyBuffer(mm.Device.Device, buff.Dev, nil)
	mm.FreeBuffMem(&buff.HostMem)
	vk.DestroyBuffer(mm.Device.Device, buff.Host, nil)
	buff.Size = 0
	buff.Host = nil
	buff.Dev = nil
	buff.Active = false
	return true
}

// Deactivate deactivates device memory for all buffs
func (mm *Memory) Deactivate() {
	for bt := VtxIdxBuff; bt < BuffTypesN; bt++ {
		mm.DeactivateBuff(bt)
	}
}

// DeactivateBuff deactivates device memory in given buffer
func (mm *Memory) DeactivateBuff(bt BuffTypes) {
	buff := mm.Buffs[bt]
	mm.FreeBuffMem(&buff.DevMem)
	buff.Active = false
}

// Activate activates device memory for all buffs
func (mm *Memory) Activate() {
	for bt := VtxIdxBuff; bt < BuffTypesN; bt++ {
		mm.ActivateBuff(bt)
	}
}

// ActivateBuff ensures device memory is ready to use
// assumes the staging memory is configured.
// Call Sync after this if needed.
func (mm *Memory) ActivateBuff(bt BuffTypes) {
	buff := mm.Buffs[bt]
	if buff.Active {
		return
	}
	if buff.DevMem == nil {
		mm.AllocDevBuff(bt)
		mm.TransferToGPUBuff(bt)
	}
	buff.Active = true
}

// SyncToGPU syncs all modified Val regions from CPU to GPU device memory, for all buffs
func (mm *Memory) SyncToGPU() {
	for bt := VtxIdxBuff; bt < BuffTypesN; bt++ {
		mm.SyncToGPUBuff(bt)
	}
}

// SyncToGPUBuff syncs all modified Val regions from CPU to GPU device memory, for given buff
func (mm *Memory) SyncToGPUBuff(bt BuffTypes) {
	buff := mm.Buffs[bt]
	mods := mm.Vals.ModRegs()
	if len(mods) == 0 {
		return
	}
	mm.TransferRegsToGPU(buff, mods)
}

// SyncVarsFmGPU syncs given variables from GPU device memory
// to CPU host memory.
// These variables can only only be Storage memory -- otherwise
// an error will be printed and returned.
func (mm *Memory) SyncVarsFmGPU(vals ...string) error {
	nv := len(vals)
	mods := make([]MemReg, nv)
	var rerr error
	for i, vnm := range vals {
		vl, err := mm.Vals.ValByNameTry(vnm)
		if err != nil {
			log.Println(err)
			rerr = err
			continue
		}
		if vl.BuffType() != StorageBuff {
			err = fmt.Errorf("SyncVarsFmGPU: Variable must be in Storage buffer, not: %s", vl.BuffType)
			log.Println(err)
			rerr = err
			continue
		}
		mods[i] = vl.MemReg()
	}
	mm.TransferRegsFmGPU(mm.Buffs[StorageBuff], mods)
	return rerr
}

// TransferToGPU transfers entire staging to GPU for all buffs
func (mm *Memory) TransferToGPU() {
	for bt := VtxIdxBuff; bt < BuffTypesN; bt++ {
		mm.TransferToGPUBuff(bt)
	}
}

// TransferToGPUBuff transfers entire staging to GPU for given buffer
func (mm *Memory) TransferToGPUBuff(bt BuffTypes) {
	buff := mm.Buffs[bt]
	if buff.Size == 0 || buff.DevMem == nil {
		return
	}
	mm.TransferRegsToGPU(buff, []MemReg{{Offset: 0, Size: buff.Size}})
}

// TransferRegsToGPU transfers memory from CPU to GPU for given regions
func (mm *Memory) TransferRegsToGPU(buff *MemBuff, regs []MemReg) {
	if buff.Size == 0 || buff.DevMem == nil {
		return
	}

	cmdBuff := mm.CmdPool.MakeBuff(&mm.Device)
	mm.CmdPool.BeginCmdOneTime()

	rg := make([]vk.BufferCopy, len(regs))
	for i, mr := range regs {
		rg[i] = vk.BufferCopy{SrcOffset: vk.DeviceSize(mr.Offset), DstOffset: vk.DeviceSize(mr.Offset), Size: vk.DeviceSize(mr.Size)}
	}

	vk.CmdCopyBuffer(cmdBuff, buff.Host, buff.Dev, uint32(len(rg)), rg)

	mm.CmdPool.SubmitWaitFree(&mm.Device)
}

// TransferRegsFmGPU transfers memory from GPU to CPU for given regions
func (mm *Memory) TransferRegsFmGPU(buff *MemBuff, regs []MemReg) {
	if buff.Size == 0 || buff.DevMem == nil {
		return
	}

	cmdBuff := mm.CmdPool.MakeBuff(&mm.Device)
	mm.CmdPool.BeginCmdOneTime()

	rg := make([]vk.BufferCopy, len(regs))
	for i, mr := range regs {
		rg[i] = vk.BufferCopy{SrcOffset: vk.DeviceSize(mr.Offset), DstOffset: vk.DeviceSize(mr.Offset), Size: vk.DeviceSize(mr.Size)}
	}

	vk.CmdCopyBuffer(cmdBuff, buff.Dev, buff.Host, uint32(len(rg)), rg)

	mm.CmdPool.SubmitWaitFree(&mm.Device)
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
