// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

/*
// New Event creates a new Event, setting the DeviceOnly bit flag
// because we typically only use events for within-device communication.
// and presumably this is faster if so-scoped.
func NewEvent(dev *Device) vk.Event {
	var sem vk.Event
	ret := vk.CreateEvent(dev, &vk.EventCreateInfo{
		SType: vk.StructureTypeEventCreateInfo,
		Flags: vk.EventCreateFlags(vk.EventCreateDeviceOnlyBit),
	}, nil, &sem)
	IfPanic(NewError(ret))
	return sem
}

func NewSemaphore(dev *Device) vk.Semaphore {
	var sem vk.Semaphore
	ret := vk.CreateSemaphore(dev, &vk.SemaphoreCreateInfo{
		SType: vk.StructureTypeSemaphoreCreateInfo,
	}, nil, &sem)
	IfPanic(NewError(ret))
	return sem
}

func NewFence(dev *Device) vk.Fence {
	var fence vk.Fence
	ret := vk.CreateFence(dev, &vk.FenceCreateInfo{
		SType: vk.StructureTypeFenceCreateInfo,
		Flags: vk.FenceCreateFlags(vk.FenceCreateSignaledBit),
	}, nil, &fence)
	IfPanic(NewError(ret))
	return fence
}

*/
