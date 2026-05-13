// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"

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
	wdev, err := gpu.GPU.RequestDevice(nil)
	if errors.Log(err) != nil {
		return nil, err
	}
	dev := &Device{Device: wdev}
	dev.Queue = wdev.GetQueue()
	return dev, nil
}

// NewComputeDevice returns a new device for given GPU,
// for compute functionality, which requests maximum buffer sizes.
// It gets the Queue for this device.
func NewComputeDevice(gpu *GPU) (*Device, error) {
	// we only request max buffer sizes so compute can go as big as it needs to
	limits := wgpu.DefaultLimits()
	// Per https://github.com/cogentcore/core/issues/1362 -- this may cause issues on "downlevel"
	// hardware, so we may need to detect that. OTOH it probably won't be useful for compute anyway,
	// but we can just sort that out later
	// note: on web / chromium / dawn, limited to 10: https://issues.chromium.org/issues/366151398?pli=1
	limits.MaxStorageBuffersPerShaderStage = gpu.Limits.Limits.MaxStorageBuffersPerShaderStage
	// fmt.Println("MaxStorageBuffersPerShaderStage:", gpu.Limits.Limits.MaxStorageBuffersPerShaderStage)
	// note: these limits are being processed and allow the MaxBufferSize to be the
	// controlling factor -- if we don't set these, then the slrand example doesn't
	// work above a smaller limit.
	// TODO: converting these limits to int may cause issues on 32-bit systems
	limits.MaxUniformBufferBindingSize = uint64(MemSizeAlignDown(int(gpu.Limits.Limits.MaxUniformBufferBindingSize), int(gpu.Limits.Limits.MinUniformBufferOffsetAlignment)))

	limits.MaxStorageBufferBindingSize = uint64(MemSizeAlignDown(int(gpu.Limits.Limits.MaxStorageBufferBindingSize), int(gpu.Limits.Limits.MinStorageBufferOffsetAlignment)))
	// note: this limit is not working properly:
	g4 := uint64(0xFFFFFF00)
	limits.MaxBufferSize = uint64(MemSizeAlignDown(int(min(gpu.Limits.Limits.MaxBufferSize, g4)), int(gpu.Limits.Limits.MinStorageBufferOffsetAlignment)))
	if limits.MaxBufferSize == 0 {
		limits.MaxBufferSize = g4
	}
	// limits.MaxBindGroups = gpu.Limits.Limits.MaxBindGroups // note: no point in changing -- web constraint

	if Debug {
		fmt.Printf("Requesting sizes: MaxStorageBufferBindingSize: %X  MaxBufferSize: %X\n", limits.MaxStorageBufferBindingSize, limits.MaxBufferSize)
	}
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
