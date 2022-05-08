// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"log"

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
// different roles, defined in the BuffTypes and managed by a MemBuff.
type Memory struct {
	GPU     *GPU
	Device  Device               `desc:"logical device that this memory is managed for: a Surface or GPU itself"`
	CmdPool CmdPool              `desc:"command pool for memory transfers"`
	Vars    Vars                 `desc:"Vars variables used in shaders, which manage associated Vals containing specific value instances of each var"`
	Buffs   [BuffTypesN]*MemBuff `desc:"memory buffers, organized by different Roles of vars"`
}

// Init configures the Memory for use with given gpu, device, and associated queueindex
func (mm *Memory) Init(gp *GPU, device *Device) {
	mm.GPU = gp
	mm.Device = *device
	mm.CmdPool.ConfigTransient(device)
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
// Does: Alloc(), AllocDev()
func (mm *Memory) Config() {
	mm.Alloc()
	mm.AllocDev()
}

// Alloc allocates memory for all bufers
func (mm *Memory) Alloc() {
	for bt := VtxIdxBuff; bt < BuffTypesN; bt++ {
		mm.AllocBuff(bt)
	}
}

// AllocBuff allocates host memory for given buffer
func (mm *Memory) AllocBuff(bt BuffTypes) {
	buff := mm.Buffs[bt]
	buff.AlignBytes = buff.Type.AlignBytes(mm.GPU)
	bsz := mm.Vals.MemSize(buff)
	buff.Alloc(mm.Device.Device, bsz)
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
	if bt == ImageBuff {
		mm.Vals.AllocTextures(mm)
	} else {
		buff.AllocDev(mm.Device.Device)
	}
}

// NewBuffer makes a buffer of given size, usage
func (mm *Memory) NewBuffer(size int, usage vk.BufferUsageFlagBits) vk.Buffer {
	return NewBuffer(mm.Device.Device, size, usage)
}

// AllocBuffMem allocates memory for given buffer, with given properties
func (mm *Memory) AllocBuffMem(buffer vk.Buffer, props vk.MemoryPropertyFlagBits) vk.DeviceMemory {
	return AllocBuffMem(mm.Device.Device, buffer, props)
}

// FreeBuffMem frees given device memory to nil
func (mm *Memory) FreeBuffMem(memory *vk.DeviceMemory) {
	FreeBuffMem(mm.Device.Device, memory)
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
	mm.Vals.Free(buff)
	if buff.Size == 0 {
		return false
	}
	buff.Free(mm.Device.Device)
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

// todo: activate construct is to vague -- just use Dev and Host terminology.

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
	if bt == ImageBuff {
		mm.Vals.AllocTextures(mm)
		mm.TransferAllValsTextures(buff)
	} else {
		if buff.DevMem == nil {
			mm.AllocDevBuff(bt)
			mm.TransferToGPUBuff(bt)
		}
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
	if bt == ImageBuff {
		mm.SyncValsTextures(buff)
		return
	}
	mods := mm.Vals.ModRegs(bt)
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
	if bt == ImageBuff {
		mm.TransferAllValsTextures(buff)
		return
	}
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

	cmdBuff := mm.CmdPool.NewBuffer(&mm.Device)
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

	cmdBuff := mm.CmdPool.NewBuffer(&mm.Device)
	mm.CmdPool.BeginCmdOneTime()

	rg := make([]vk.BufferCopy, len(regs))
	for i, mr := range regs {
		rg[i] = vk.BufferCopy{SrcOffset: vk.DeviceSize(mr.Offset), DstOffset: vk.DeviceSize(mr.Offset), Size: vk.DeviceSize(mr.Size)}
	}

	vk.CmdCopyBuffer(cmdBuff, buff.Dev, buff.Host, uint32(len(rg)), rg)

	mm.CmdPool.SubmitWaitFree(&mm.Device)
}

////////////////////////////////////////////////////////////////////////////
// Texture functions

// TransferImagesToGPU transfers image memory from CPU to GPU for given images.
// The image Host.Offset *must* be accurate for the given buffer, whether its own
// individual buffer or the shared memory-managed buffer.
func (mm *Memory) TransferImagesToGPU(buff vk.Buffer, imgs ...*Image) {
	cmdBuff := mm.CmdPool.NewBuffer(&mm.Device)
	mm.CmdPool.BeginCmdOneTime()

	for _, im := range imgs {
		im.TransitionForDst(cmdBuff)
		if im.IsHostOwner() {
			vk.CmdCopyBufferToImage(cmdBuff, im.Host.Buff, im.Image, vk.ImageLayoutTransferDstOptimal, 1, []vk.BufferImageCopy{im.CopyRec()})
		} else {
			vk.CmdCopyBufferToImage(cmdBuff, buff, im.Image, vk.ImageLayoutTransferDstOptimal, 1, []vk.BufferImageCopy{im.CopyRec()})
		}
		// if im.IsVal() {
		im.TransitionDstToShader(cmdBuff)
		// }
	}
	mm.CmdPool.SubmitWaitFree(&mm.Device)
}

// TransferImagesFmGPU transfers image memory from GPU to CPU for given images.
// the image Host.Offset *must* be accurate for the given buffer, whether its own
// individual buffer or the shared memory-managed buffer.
func (mm *Memory) TransferImagesFmGPU(buff vk.Buffer, imgs ...*Image) {
	cmdBuff := mm.CmdPool.NewBuffer(&mm.Device)
	mm.CmdPool.BeginCmdOneTime()

	for _, im := range imgs {
		vk.CmdCopyImageToBuffer(cmdBuff, im.Image, vk.ImageLayoutTransferDstOptimal, buff, 1, []vk.BufferImageCopy{im.CopyRec()})
	}
	mm.CmdPool.SubmitWaitFree(&mm.Device)
}

// TransferAllValsTextures copies all vals images from host buffer to device memory
func (mm *Memory) TransferAllValsTextures(buff *MemBuff) {
	var imgs []*Image
	for _, vl := range mm.Vals.Vals {
		if vl.BuffType() != ImageBuff || vl.Texture == nil {
			continue
		}
		imgs = append(imgs, &vl.Texture.Image)
	}
	mm.TransferImagesToGPU(buff.Host, imgs...)
}

// SyncValsTextures syncs all changed vals images from host buffer to device memory
func (mm *Memory) SyncValsTextures(buff *MemBuff) {
	var imgs []*Image
	for _, vl := range mm.Vals.Vals {
		if vl.BuffType() != ImageBuff || vl.Texture == nil || !vl.Mod {
			continue
		}
		imgs = append(imgs, &vl.Texture.Image)
	}
	if len(imgs) > 0 {
		mm.TransferImagesToGPU(buff.Host, imgs...)
	}
}
