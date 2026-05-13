// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"

	"cogentcore.org/core/base/errors"
	"github.com/cogentcore/webgpu/wgpu"
)

// note: WriteBuffer is the preferred method for writing, so we only need to manage Read

// BufferMapAsyncError returns an error message if the status is not success.
func BufferMapAsyncError(status wgpu.BufferMapAsyncStatus) error {
	if status != wgpu.BufferMapAsyncStatusSuccess {
		return errors.New("gpu BufferMapAsync was not successful")
	}
	return nil
}

// BufferReadSync does a MapAsync on given buffer, waiting on the device
// until the sync is complete, and returning error if any issues.
func BufferReadSync(device *Device, size int, buffer *wgpu.Buffer) error {
	var status wgpu.BufferMapAsyncStatus
	err := buffer.MapAsync(wgpu.MapModeRead, 0, uint64(size), func(s wgpu.BufferMapAsyncStatus) {
		status = s
	})
	if errors.Log(err) != nil {
		return err
	}
	device.WaitDone()
	return BufferMapAsyncError(status)
}

// ValueReadSync does a MapAsync on given Values, waiting on the device
// until the sync is complete, and returning error if any issues.
// It is more efficient to get all relevant buffers at the same time.
func ValueReadSync(device *Device, values ...*Value) error {
	nv := len(values)
	if nv == 0 {
		return nil
	}
	var errs []error
	status := make([]wgpu.BufferMapAsyncStatus, nv)
	for i, vl := range values {
		err := vl.readBuffer.MapAsync(wgpu.MapModeRead, 0, uint64(vl.AllocSize), func(s wgpu.BufferMapAsyncStatus) {
			status[i] = s
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	device.WaitDone()
	for _, s := range status {
		err := BufferMapAsyncError(s)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// ValueGroups provides named lists of value groups that can be
// used to simplify the calling of the multiple Read functions
// needed to read data back from the GPU.
type ValueGroups map[string][]*Value

func (vg *ValueGroups) init() {
	if vg == nil {
		*vg = make(map[string][]*Value)
	}
}

// Add Adds named group of values
func (vg *ValueGroups) Add(name string, vals ...*Value) {
	vg.init()
	(*vg)[name] = vals
}

func (vg ValueGroups) ValuesByName(name string) ([]*Value, error) {
	vls, ok := vg[name]
	if !ok {
		return nil, fmt.Errorf("gpu.ValueGroups: %q not found", name)
	}
	return vls, nil
}

// GPUToRead adds commands to given command encoder to read values
// from the GPU to its read buffer.  Next step is ReadSync after
// command has been submitted (EndComputePass).
func (vg ValueGroups) GPUToRead(name string, cmd *wgpu.CommandEncoder) error {
	vls, err := vg.ValuesByName(name)
	if errors.Log(err) != nil {
		return err
	}
	var errs []error
	for _, vl := range vls {
		err := vl.GPUToRead(cmd)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Log(errors.Join(errs...))
}

// ReadSync does a MapAsync on values.
func (vg ValueGroups) ReadSync(name string) error {
	vls, err := vg.ValuesByName(name)
	if errors.Log(err) != nil {
		return err
	}
	if len(vls) == 0 {
		return nil
	}
	dev := &vls[0].device
	return ValueReadSync(dev, vls...)
}
