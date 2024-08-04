// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import "github.com/rajveermalviya/go-webgpu/wgpu"

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
	if err != nil {
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

// NewGraphicsDevice returns a new Graphics Device, on given GPU.
// TODO: WebGPU does not appear to make any distinction between
// graphics and compute devices, so probably remove this.
func NewGraphicsDevice(gp *GPU) (*Device, error) {
	return NewDevice(gp)
}
