// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package egpu

import (
	"log"
	"unsafe"

	vk "github.com/vulkan-go/vulkan"
)

// BuffAlloc is one memory allocation for a given buffer manager
type BuffAlloc struct {
	Offset int `desc:"starting offset"`
	Size   int `desc:"size"`
	Buff   *BufferMgr
}

// Memory manages the memory for the GPU, in terms of BufferMgr allocations.
// You configure all the BufferMgrs and then allocate them here.
type Memory struct {
	GPU    *GPU
	Device vk.Device `desc:"logical device that this memory is managed for: a Surface or GPU itself"`

	Usage        vk.BufferUsageFlagBits    `desc:"buffer usage"`
	Buffer       vk.Buffer                 `desc:"the mega buffer holding everything, as bytes"`
	BuffAllocs   []*BuffAlloc              `desc:"allocations for buffermgrs, in order added"`
	BuffAllocMap map[*BufferMgr]*BuffAlloc `desc:"allocations for buffermgrs, as map"`
	BuffSize     int                       `desc:"total size in bytes allocated"`
	BuffMem      []byte                    `desc:"allocated local CPU memory"`
	BuffMemory   vk.DeviceMemory           `desc:"device memory corresponding to BuffMem"`
}

//
func (mm *Memory) Init(gp *GPU, device vk.Device, usage vk.BufferUsageFlagBits) {
	mm.GPU = gp
	mm.Device = device
	mm.Usage = usage
}

// AddBuff adds BufferMgr to be managed by this memory manager
func (mm *Memory) AddBuff(buff *BufferMgr) {
	if mm.BuffAllocMap == nil {
		mm.BuffAllocMap = make(map[*BufferMgr]*BuffAlloc)
	}
	ba := &BuffAlloc{Buff: buff}
	mm.BuffAllocMap[buff] = ba
	mm.BuffAllocs = append(mm.BuffAllocs, ba)
}

// Free frees any allocated memory -- returns true if freed
func (mm *Memory) Free() bool {
	if mm.BuffSize == 0 {
		return false
	}
	for _, ba := range mm.BuffAllocs {
		ba.Buff.Free(mm)
	}
	vk.FreeMemory(mm.Device, mm.BuffMemory, nil)
	vk.DestroyBuffer(mm.Device, mm.Buffer, nil)
	mm.BuffSize = 0
	mm.BuffMem = mm.BuffMem[:0]
	mm.BuffMemory = nil
	return true
}

func (mm *Memory) Destroy() {
	mm.Free()
	mm.GPU = nil
	mm.Device = nil
	mm.BuffMem = nil
}

// Alloc allocates memory for all buffs, freeing any existing
func (mm *Memory) Alloc() {
	mm.Free()

	mm.BuffSize = mm.BuffMemSize()
	if mm.BuffSize == 0 {
		return
	}

	if cap(mm.BuffMem) >= mm.BuffSize {
		mm.BuffMem = mm.BuffMem[:mm.BuffSize]
	} else {
		mm.BuffMem = make([]byte, mm.BuffSize)
	}

	var buffer vk.Buffer
	var memory vk.DeviceMemory
	ret := vk.CreateBuffer(mm.Device, &vk.BufferCreateInfo{
		SType: vk.StructureTypeBufferCreateInfo,
		Usage: vk.BufferUsageFlags(mm.Usage),
		Size:  vk.DeviceSize(mm.BuffSize),
	}, nil, &buffer)
	IfPanic(NewError(ret))

	// Ask device about its memory requirements.
	var memReqs vk.MemoryRequirements
	vk.GetBufferMemoryRequirements(mm.Device, buffer, &memReqs)
	memReqs.Deref()

	memProps := mm.GPU.MemoryProps
	memType, ok := FindRequiredMemoryType(memProps, vk.MemoryPropertyFlagBits(memReqs.MemoryTypeBits),
		vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit)
	if !ok {
		log.Println("vulkan warning: failed to find required memory type")
	}

	// Allocate device memory and bind to the buffer.
	ret = vk.AllocateMemory(mm.Device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memType,
	}, nil, &memory)
	IfPanic(NewError(ret), func() {
		vk.DestroyBuffer(mm.Device, buffer, nil)
	})
	vk.BindBufferMemory(mm.Device, buffer, memory, 0)

	mm.Buffer = buffer
	mm.BuffMemory = memory

	// now allocate views into main buffer memory
	off := 0
	for _, ba := range mm.BuffAllocs {
		sz := ba.Buff.Alloc(mm, off)
		off += sz
	}
}

// Activate maps memory and copies CPU to GPU
func (mm *Memory) Activate() {
	if mm.BuffSize == 0 {
		return
	}

	// todo: what about coherent memory?

	var pData unsafe.Pointer
	ret := vk.MapMemory(mm.Device, mm.BuffMemory, 0, vk.DeviceSize(len(mm.BuffMem)), 0, &pData)
	if IsError(ret) {
		log.Printf("vulkan warning: failed to map device memory for data (len=%d)", len(mm.BuffMem))
		return
	}
	n := vk.Memcopy(pData, mm.BuffMem)
	if n != len(mm.BuffMem) {
		log.Printf("vulkan warning: failed to copy data, %d != %d", n, len(mm.BuffMem))
	}

	// todo: update alloc pointers to memview

	vk.UnmapMemory(mm.Device, mm.BuffMemory)
}

// BuffMemSize returns the total size needs in bytes for buffers
func (mm *Memory) BuffMemSize() int {
	sz := 0
	for _, ba := range mm.BuffAllocs {
		sz += ba.Buff.MemSize()
	}
	return sz
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
