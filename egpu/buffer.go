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

// InstanceExts gets a list of instance extensions available on the platform.
func InstanceExts() (names []string, err error) {
	defer CheckErr(&err)

	var count uint32
	ret := vk.EnumerateInstanceExtensionProperties("", &count, nil)
	IfPanic(NewError(ret))
	list := make([]vk.ExtensionProperties, count)
	ret = vk.EnumerateInstanceExtensionProperties("", &count, list)
	IfPanic(NewError(ret))
	for _, ext := range list {
		ext.Deref()
		names = append(names, vk.ToString(ext.ExtensionName[:]))
	}
	return names, err
}

// DeviceExts gets a list of instance extensions available on the provided physical device.
func DeviceExts(gpu vk.PhysicalDevice) (names []string, err error) {
	defer CheckErr(&err)

	var count uint32
	ret := vk.EnumerateDeviceExtensionProperties(gpu, "", &count, nil)
	IfPanic(NewError(ret))
	list := make([]vk.ExtensionProperties, count)
	ret = vk.EnumerateDeviceExtensionProperties(gpu, "", &count, list)
	IfPanic(NewError(ret))
	for _, ext := range list {
		ext.Deref()
		names = append(names, vk.ToString(ext.ExtensionName[:]))
	}
	return names, err
}

// ValidationLayers gets a list of validation layers available on the platform.
func ValidationLayers() (names []string, err error) {
	defer CheckErr(&err)

	var count uint32
	ret := vk.EnumerateInstanceLayerProperties(&count, nil)
	IfPanic(NewError(ret))
	list := make([]vk.LayerProperties, count)
	ret = vk.EnumerateInstanceLayerProperties(&count, list)
	IfPanic(NewError(ret))
	for _, layer := range list {
		layer.Deref()
		names = append(names, vk.ToString(layer.LayerName[:]))
	}
	return names, err
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

type Buffer struct {
	// device for destroy purposes.
	device vk.Device
	// Buffer is the buffer object.
	Buffer vk.Buffer
	// Memory is the device memory backing buffer object.
	Memory vk.DeviceMemory
}

func (b *Buffer) Destroy() {
	vk.FreeMemory(b.device, b.Memory, nil)
	vk.DestroyBuffer(b.device, b.Buffer, nil)
	b.device = nil
}

func CreateBuffer(device vk.Device, memProps vk.PhysicalDeviceMemoryProperties,
	data []byte, usage vk.BufferUsageFlagBits) *Buffer {

	var buffer vk.Buffer
	var memory vk.DeviceMemory
	ret := vk.CreateBuffer(device, &vk.BufferCreateInfo{
		SType: vk.StructureTypeBufferCreateInfo,
		Usage: vk.BufferUsageFlags(usage),
		Size:  vk.DeviceSize(len(data)),
	}, nil, &buffer)
	IfPanic(NewError(ret))

	// Ask device about its memory requirements.
	var memReqs vk.MemoryRequirements
	vk.GetBufferMemoryRequirements(device, buffer, &memReqs)
	memReqs.Deref()

	memType, ok := FindRequiredMemoryType(memProps, vk.MemoryPropertyFlagBits(memReqs.MemoryTypeBits),
		vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit)
	if !ok {
		log.Println("vulkan warning: failed to find required memory type")
	}

	// Allocate device memory and bind to the buffer.
	ret = vk.AllocateMemory(device, &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memType,
	}, nil, &memory)
	IfPanic(NewError(ret), func() {
		vk.DestroyBuffer(device, buffer, nil)
	})
	vk.BindBufferMemory(device, buffer, memory, 0)
	b := &Buffer{
		device: device,
		Buffer: buffer,
		Memory: memory,
	}

	// Map the memory and dump data in there.
	if len(data) > 0 {
		var pData unsafe.Pointer
		ret := vk.MapMemory(device, memory, 0, vk.DeviceSize(len(data)), 0, &pData)
		if IsError(ret) {
			log.Printf("vulkan warning: failed to map device memory for data (len=%d)", len(data))
			return b
		}
		n := vk.Memcopy(pData, data)
		if n != len(data) {
			log.Printf("vulkan warning: failed to copy data, %d != %d", n, len(data))
		}
		vk.UnmapMemory(device, memory)
	}
	return b
}

func sliceUint32(data []byte) []uint32 {
	const m = 0x7fffffff
	return (*[m / 4]uint32)(unsafe.Pointer((*sliceHeader)(unsafe.Pointer(&data)).Data))[:len(data)/4]
}

type sliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
