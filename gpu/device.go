// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"cogentcore.org/core/base/errors"
	"github.com/cogentcore/webgpu/wgpu"
)

// Device holds Device and associated Queue info.
// A Device is a usable instance of the GPU Adapter hardware.
// Each device has one Queue.
type Device struct {
	// logical device
	Device *wgpu.Device

	// queue for device
	Queue *wgpu.Queue
}

// NewDevice returns a new device for given GPU.
// It gets the Queue for this device.
func NewDevice(gpu *GPU) (*Device, error) {
	// we only request max buffer sizes so compute can go as big as it needs to
	limits := wgpu.DefaultLimits()
	const maxv = 0xFFFFFFFF
	// note: these limits are being processed and allow the MaxBufferSize to be the
	// controlling factor -- if we don't set these, then the slrand example doesn't
	// work above a smaller limit.
	limits.MaxUniformBufferBindingSize = min(gpu.Limits.Limits.MaxUniformBufferBindingSize, maxv)
	limits.MaxStorageBufferBindingSize = min(gpu.Limits.Limits.MaxStorageBufferBindingSize, maxv)
	// note: this limit is not working properly:
	limits.MaxBufferSize = min(gpu.Limits.Limits.MaxBufferSize, maxv)
	// limits.MaxBindGroups = gpu.Limits.Limits.MaxBindGroups // note: no point in changing -- web constraint

	desc := wgpu.DeviceDescriptor{
		RequiredLimits: &wgpu.RequiredLimits{
			Limits: limits,
		},
	}
	wdev, err := gpu.GPU.RequestDevice(&desc)
	if errors.Log(err) != nil {
		return nil, err
	}
	dev := &Device{Device: wdev}
	dev.Queue = wdev.GetQueue()
	return dev, nil
}

func (dv *Device) Release() {
	if dv.Device == nil {
		return
	}
	dv.Queue.OnSubmittedWorkDone(func(wgpu.QueueWorkDoneStatus) {
		dv.Device.Release()
		dv.Device = nil
		dv.Queue = nil
	})
}

// WaitDoneFunc waits until the device is idle and then calls
// given function, if the device is ready. If it is in some other
// bad state, that generates a panic.
func (dv *Device) WaitDoneFunc(fun func()) {
	dv.Queue.OnSubmittedWorkDone(func(stat wgpu.QueueWorkDoneStatus) {
		if stat == wgpu.QueueWorkDoneStatusSuccess {
			fun()
			return
		}
		panic("Device.WaitDoneFunc: bad queue status: " + stat.String())
	})
}

// WaitDone does a blocking wait until the device is done with current work.
func (dv *Device) WaitDone() {
	if dv.Device == nil {
		return
	}
	dv.Device.Poll(true, nil)
}
