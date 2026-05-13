// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"math"
	"runtime"
	"sync"

	"cogentcore.org/core/base/errors"
	"github.com/cogentcore/webgpu/wgpu"
)

// ComputeSystem manages a system of ComputePipelines that all share
// a common collection of Vars and Values.
type ComputeSystem struct {
	// optional name of this ComputeSystem
	Name string

	// vars represents all the data variables used by the system,
	// with one Var for each resource that is made visible to the shader,
	// indexed by Group (@group) and Binding (@binding).
	// Each Var has Value(s) containing specific instance values.
	// Access through the System.Vars() method.
	vars Vars

	// ComputePipelines by name.
	ComputePipelines map[string]*ComputePipeline

	// ComputeEncoder is the compute specific command encoder for the
	// current [BeginComputePass], and released in [EndComputePass].
	ComputeEncoder *wgpu.ComputePassEncoder

	// CommandEncoder is the command encoder created in
	// [BeginComputePass], and released in [EndComputePass].
	CommandEncoder *wgpu.CommandEncoder

	// logical device for this ComputeSystem, which we own.
	device *Device

	// gpu is our GPU device, which has properties
	// and alignment factors.
	gpu *GPU
}

// NewComputeSystem returns a new ComputeSystem, initialized with
// its own new device that is owned by the system.
func NewComputeSystem(gp *GPU, name string) *ComputeSystem {
	sy := &ComputeSystem{}
	sy.init(gp, name)
	return sy
}

// System interface:

func (sy *ComputeSystem) Vars() *Vars     { return &sy.vars }
func (sy *ComputeSystem) Device() *Device { return sy.device }
func (sy *ComputeSystem) GPU() *GPU       { return sy.gpu }
func (sy *ComputeSystem) Render() *Render { return nil }

// init initializes the ComputeSystem
func (sy *ComputeSystem) init(gp *GPU, name string) {
	sy.gpu = gp
	sy.Name = name
	sy.device = errors.Log1(NewComputeDevice(gp))
	sy.vars.device = *sy.device
	sy.vars.sys = sy
	sy.ComputePipelines = make(map[string]*ComputePipeline)
}

// WaitDone waits until device is done with current processing steps
func (sy *ComputeSystem) WaitDone() {
	sy.device.WaitDone()
}

func (sy *ComputeSystem) Release() {
	sy.WaitDone()
	for _, pl := range sy.ComputePipelines {
		pl.Release()
	}
	sy.ComputePipelines = nil
	sy.vars.Release()
	sy.gpu = nil
}

// AddComputePipeline adds a new ComputePipeline to the system
func (sy *ComputeSystem) AddComputePipeline(name string) *ComputePipeline {
	pl := NewComputePipeline(name, sy)
	sy.ComputePipelines[pl.Name] = pl
	return pl
}

// Config configures the entire system, after Pipelines and Vars
// have been initialized.  After this point, just need to set
// values for the vars, and then do compute passes.  This should
// not need to be called more than once.
func (sy *ComputeSystem) Config() {
	sy.vars.Config(sy.device)
	if Debug {
		fmt.Printf("%s\n", sy.vars.StringDoc())
	}
	for _, pl := range sy.ComputePipelines {
		pl.Config(true)
	}
}

// NewCommandEncoder returns a new CommandEncoder for encoding
// compute commands.  This is automatically called by
// BeginRenderPass and the result maintained in [CommandEncoder].
func (sy *ComputeSystem) NewCommandEncoder() (*wgpu.CommandEncoder, error) {
	cmd, err := sy.device.Device.CreateCommandEncoder(nil)
	if errors.Log(err) != nil {
		return nil, err
	}
	return cmd, nil
}

// BeginComputePass adds commands to the given command buffer
// to start the compute pass, returning the encoder object
// to which further compute commands should be added.
// Call [EndComputePass] when done.
// If an existing [ComputeSystem.ComputeEncoder] is already set from
// a prior BeginComputePass call, then that is returned, so this
// is safe and efficient to call for every compute shader dispatch,
// where the first call will create and the rest add to the ongoing job.
func (sy *ComputeSystem) BeginComputePass() (*wgpu.ComputePassEncoder, error) {
	if sy.ComputeEncoder != nil {
		return sy.ComputeEncoder, nil
	}
	cmd, err := sy.NewCommandEncoder()
	if errors.Log(err) != nil {
		return nil, err
	}
	sy.CommandEncoder = cmd
	sy.ComputeEncoder = cmd.BeginComputePass(nil) // optional name in the encoder
	return sy.ComputeEncoder, nil
}

// EndComputePass submits the current compute commands to the device
// Queue and releases the [ComputeSystem.CommandEncoder] and
// [ComputeSystem.ComputeEncoder].  You must call ce.End prior to calling this.
// Can insert other commands after ce.End, e.g., to copy data back
// from the GPU, prior to calling EndComputePass.
func (sy *ComputeSystem) EndComputePass() error {
	ce := sy.ComputeEncoder
	cmd := sy.CommandEncoder
	sy.ComputeEncoder = nil
	sy.CommandEncoder = nil
	ce.Release() // must happen before Finish
	cmdBuffer, err := cmd.Finish(nil)
	if errors.Log(err) != nil {
		return err
	}
	sy.device.Queue.Submit(cmdBuffer)
	cmdBuffer.Release()
	cmd.Release()
	for _, pl := range sy.ComputePipelines {
		pl.releaseOldBindGroups()
	}
	return nil
}

// NumThreads is the number of threads to use for parallel threading,
// in the [VectorizeFunc] that is used for CPU versions of GPU functions.
// The default of 0 causes the [runtime.GOMAXPROCS] to be used.
var NumThreads = 0

// DefaultNumThreads returns the default number of threads to use:
// NumThreads if non-zero, otherwise [runtime.GOMAXPROCS].
func DefaultNumThreads() int {
	if NumThreads > 0 {
		return NumThreads
	}
	return runtime.GOMAXPROCS(0)
}

// VectorizeFunc runs given GPU kernel function taking a uint32 index
// on the CPU, using given number of threads with goroutines, for n iterations.
// If threads is 0, then GOMAXPROCS is used.
func VectorizeFunc(threads, n int, fun func(idx uint32)) {
	if threads == 0 {
		threads = DefaultNumThreads()
	}
	if threads <= 1 {
		for idx := range n {
			fun(uint32(idx))
		}
		return
	}
	nper := int(math.Ceil(float64(n) / float64(threads)))
	wait := sync.WaitGroup{}
	for start := 0; start < n; start += nper {
		end := start + nper
		if end > n {
			end = n
		}
		wait.Add(1)
		go func() {
			for idx := start; idx < end; idx++ {
				fun(uint32(idx))
			}
			wait.Done()
		}()
	}
	wait.Wait()
}
