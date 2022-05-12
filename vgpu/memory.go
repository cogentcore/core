// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"log"

	vk "github.com/goki/vulkan"
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
// Memory is organized by Vars with associated Vals.
type Memory struct {
	GPU     *GPU
	Device  Device               `desc:"logical device that this memory is managed for -- set from System"`
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
	mm.Vars.Mem = mm
}

// Destroy destroys all vulkan allocations, using given dev
func (mm *Memory) Destroy(dev vk.Device) {
	mm.Free()
	mm.Vars.Destroy(dev)
	mm.CmdPool.Destroy(dev)
	mm.GPU = nil
}

// Config should be called after all Vals have been configured
// and are ready to go with their initial data.
// Does: AllocHost(), AllocDev()
func (mm *Memory) Config(dev vk.Device) {
	mm.Vars.Config(dev)
	mm.AllocHost()
	mm.AllocDev()
	mm.Vars.BindDynVarsAll()
}

// AllocHost allocates memory for all buffers
func (mm *Memory) AllocHost() {
	for bt := VtxIdxBuff; bt < BuffTypesN; bt++ {
		mm.AllocHostBuff(bt)
	}
}

// AllocHostBuff allocates host memory for given buffer
func (mm *Memory) AllocHostBuff(bt BuffTypes) {
	buff := mm.Buffs[bt]
	buff.AlignBytes = buff.Type.AlignBytes(mm.GPU)
	bsz := mm.Vars.MemSize(buff)
	buff.AllocHost(mm.Device.Device, bsz)
	mm.Vars.AllocHost(buff, 0)
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
		mm.Vars.AllocTextures(mm)
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
	mm.Vars.Free(buff)
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
	mods := mm.Vars.ModRegs(bt)
	if len(mods) == 0 {
		return
	}
	mm.TransferRegsToGPU(buff, mods)
}

// SyncValNameFmGPU syncs given value from GPU device memory to CPU host memory,
// specifying value by name for given named variable in given set.
// Variable can only only be Storage memory -- otherwise an error is returned.
func (mm *Memory) SyncValNameFmGPU(set int, varNm, valNm string) error {
	vr, vl, err := mm.Vars.ValByNameTry(set, varNm, valNm)
	if err != nil {
		return err
	}
	if vr.BuffType() != StorageBuff {
		err = fmt.Errorf("SyncValFmGPU: Variable must be in Storage buffer, not: %s", vr.BuffType().String())
		if mm.GPU.Debug {
			log.Println(err)
			return err
		}
	}
	mm.SyncValFmGPU(vl)
	return nil
}

// SyncValIdxFmGPU syncs given value from GPU device memory to CPU host memory,
// specifying value by index for given named variable, in given set.
// Variable can only only be Storage memory -- otherwise an error is returned.
func (mm *Memory) SyncValIdxFmGPU(set int, varNm string, valIdx int) error {
	vr, vl, err := mm.Vars.ValByIdxTry(set, varNm, valIdx)
	if err != nil {
		return err
	}
	if vr.BuffType() != StorageBuff {
		err = fmt.Errorf("SyncValFmGPU: Variable must be in Storage buffer, not: %s", vr.BuffType().String())
		if mm.GPU.Debug {
			log.Println(err)
			return err
		}
	}
	mm.SyncValFmGPU(vl)
	return nil
}

// SyncValFmGPU syncs given value from GPU device memory to CPU host memory.
// Must be in Storage memory -- otherwise an error will be printed and returned.
func (mm *Memory) SyncValFmGPU(vl *Val) {
	mods := vl.MemReg()
	mm.TransferRegsFmGPU(mm.Buffs[StorageBuff], []MemReg{mods})
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
	if buff.Size == 0 || buff.DevMem == nil || len(regs) == 0 {
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
	if buff.Size == 0 || buff.DevMem == nil || len(regs) == 0 {
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
	if len(imgs) == 0 {
		return
	}
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
	vs := &mm.Vars
	ns := vs.NSets()
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
		if st == nil || st.Set == PushConstSet {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role != TextureRole {
				continue
			}
			for _, vl := range vr.Vals.Vals {
				if vl.Texture == nil {
					continue
				}
				imgs = append(imgs, &vl.Texture.Image)
			}
		}
	}
	if len(imgs) > 0 {
		mm.TransferImagesToGPU(buff.Host, imgs...)
	}
}

// SyncValsTextures syncs all changed vals images from host buffer to device memory
func (mm *Memory) SyncValsTextures(buff *MemBuff) {
	var imgs []*Image
	vs := &mm.Vars
	ns := vs.NSets()
	for si := vs.StartSet(); si < ns; si++ {
		st := vs.SetMap[si]
		if st == nil || st.Set == PushConstSet {
			continue
		}
		for _, vr := range st.Vars {
			if vr.Role != TextureRole {
				continue
			}
			for _, vl := range vr.Vals.Vals {
				if vl.Texture == nil || !vl.IsMod() {
					continue
				}
				imgs = append(imgs, &vl.Texture.Image)
				vl.ClearMod()
			}
		}
	}
	if len(imgs) > 0 {
		mm.TransferImagesToGPU(buff.Host, imgs...)
	}
}
