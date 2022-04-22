// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	vk "github.com/vulkan-go/vulkan"
)

// Compute represents a compute device, with separate queues
type Compute struct {
	GPU    *GPU
	Device Device `desc:"device for this Compute -- has its own queues"`
}

// Init initializes the device for the compute device
func (cp *Compute) Init(gp *GPU) error {
	cp.GPU = gp
	return gp.Device.Init(gp, vk.QueueComputeBit)
}

func (cp *Compute) Destroy() {
	cp.Device.Destroy()
	cp.GPU = nil
}
