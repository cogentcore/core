// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package egpu

import (
	"errors"

	vk "github.com/vulkan-go/vulkan"
)

// Compute represents a compute device, with separate queues
type Compute struct {
	GPU        *GPU
	Device     vk.Device `desc:"device for this Compute -- has its own queues"`
	QueueIndex uint32    `desc:"queue index for compute device"`
	Queue      vk.Queue  `desc:"queue for compute device"`
}

// Init initializes the device for the compute device
func (cp *Compute) Init(gp *GPU) error {
	cp.GPU = gp
	// Get queue family properties
	var queueCount uint32
	vk.GetPhysicalDeviceQueueFamilyProperties(cp.GPU.GPU, &queueCount, nil)
	queueProperties := make([]vk.QueueFamilyProperties, queueCount)
	vk.GetPhysicalDeviceQueueFamilyProperties(cp.GPU.GPU, &queueCount, queueProperties)
	if queueCount == 0 { // probably should try another GPU
		return errors.New("vulkan error: no queue families found on GPU 0")
	}

	// Find a suitable queue family for the target Vulkan mode
	found := false
	required := vk.QueueFlags(vk.QueueComputeBit)
	for i := uint32(0); i < queueCount; i++ {
		queueProperties[i].Deref()
		if queueProperties[i].QueueFlags&required != 0 {
			cp.QueueIndex = i
			found = true
			break
		}
	}
	if !found {
		err := errors.New("GPU vulkan error: could not found queue with compute capabilities")
		return err
	}

	queueInfos := []vk.DeviceQueueCreateInfo{{
		SType:            vk.StructureTypeDeviceQueueCreateInfo,
		QueueFamilyIndex: cp.QueueIndex,
		QueueCount:       1,
		PQueuePriorities: []float32{1.0},
	}}

	var device vk.Device
	ret := vk.CreateDevice(cp.GPU.GPU, &vk.DeviceCreateInfo{
		SType:                   vk.StructureTypeDeviceCreateInfo,
		QueueCreateInfoCount:    uint32(len(queueInfos)),
		PQueueCreateInfos:       queueInfos,
		EnabledExtensionCount:   uint32(len(cp.GPU.DeviceExts)),
		PpEnabledExtensionNames: cp.GPU.DeviceExts,
		EnabledLayerCount:       uint32(len(cp.GPU.ValidationLayers)),
		PpEnabledLayerNames:     cp.GPU.ValidationLayers,
	}, nil, &device)
	IfPanic(NewError(ret))
	cp.Device = device

	var queue vk.Queue
	vk.GetDeviceQueue(cp.Device, cp.QueueIndex, 0, &queue)
	cp.Queue = queue
	return nil
}

func (cp *Compute) Destroy() {
	if cp.Device != nil {
		vk.DeviceWaitIdle(cp.Device)
		vk.DestroyDevice(cp.Device, nil)
		cp.Device = nil
	}
	cp.GPU = nil
}
