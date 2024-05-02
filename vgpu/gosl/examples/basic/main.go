// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"unsafe"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/vgpu"
	"github.com/emer/gosl/v2/timer"
)

// note: standard one to use is plain "gosl" which should be go install'd

//go:generate ../../gosl cogentcore.org/core/math32/fastexp.go compute.go

func init() {
	// must lock main thread for gpu!  this also means that vulkan must be used
	// for gogi/oswin eventually if we want gui and compute
	runtime.LockOSThread()
}

func main() {
	if vgpu.InitNoDisplay() != nil {
		return
	}

	gp := vgpu.NewComputeGPU()
	// vgpu.Debug = true
	gp.Config("basic")

	// gp.PropsString(true) // print

	n := 100000000 // get 80x with 100m, 50x with 10m
	threads := 64
	nInt := int(math32.IntMultiple(float32(n), float32(threads)))
	n = nInt               // enforce optimal n's -- otherwise requires range checking
	nGps := nInt / threads // dispatch n

	pars := &ParamStruct{}
	pars.Defaults()

	data := make([]DataStruct, n)
	for i := range data {
		d := &data[i]
		d.Raw = rand.Float32()
		d.Integ = 0
	}

	cpuTmr := timer.Time{}
	cpuTmr.Start()
	for i := range data {
		d := &data[i]
		pars.IntegFromRaw(d)
	}
	cpuTmr.Stop()

	sy := gp.NewComputeSystem("basic")
	pl := sy.NewPipeline("basic")
	pl.AddShaderFile("basic", vgpu.ComputeShader, "shaders/basic.spv")

	vars := sy.Vars()
	setp := vars.AddSet()
	setd := vars.AddSet()

	parsv := setp.AddStruct("Params", int(unsafe.Sizeof(ParamStruct{})), 1, vgpu.Storage, vgpu.ComputeShader)
	datav := setd.AddStruct("Data", int(unsafe.Sizeof(DataStruct{})), n, vgpu.Storage, vgpu.ComputeShader)

	setp.ConfigValues(1) // one val per var
	setd.ConfigValues(1) // one val per var
	sy.Config()          // configures vars, allocates vals, configs pipelines..

	gpuFullTmr := timer.Time{}
	gpuFullTmr.Start()

	// this copy is pretty fast -- most of time is below
	pvl, _ := parsv.Values.ValueByIndexTry(0)
	pvl.CopyFromBytes(unsafe.Pointer(pars))
	dvl, _ := datav.Values.ValueByIndexTry(0)
	dvl.CopyFromBytes(unsafe.Pointer(&data[0]))

	// gpuFullTmr := timer.Time{}
	// gpuFullTmr.Start()

	sy.Mem.SyncToGPU()

	vars.BindDynamicValueIndex(0, "Params", 0)
	vars.BindDynamicValueIndex(1, "Data", 0)

	cmd := sy.ComputeCmdBuff()
	sy.CmdResetBindVars(cmd, 0)

	// gpuFullTmr := timer.Time{}
	// gpuFullTmr.Start()

	gpuTmr := timer.Time{}
	gpuTmr.Start()

	pl.ComputeDispatch(cmd, nGps, 1, 1)
	sy.ComputeCmdEnd(cmd)
	sy.ComputeSubmitWait(cmd)

	gpuTmr.Stop()

	sy.Mem.SyncValueIndexFromGPU(1, "Data", 0) // this is about same as SyncToGPU
	dvl.CopyToBytes(unsafe.Pointer(&data[0]))

	gpuFullTmr.Stop()

	mx := min(n, 5)
	for i := 0; i < mx; i++ {
		d := &data[i]
		fmt.Printf("%d\tRaw: %g\tInteg: %g\tExp: %g\n", i, d.Raw, d.Integ, d.Exp)
	}
	fmt.Printf("\n")

	cpu := cpuTmr.TotalSecs()
	gpu := gpuTmr.TotalSecs()
	fmt.Printf("N: %d\t CPU: %6.4g\t GPU: %6.4g\t Full: %6.4g\t CPU/GPU: %6.4g\n", n, cpu, gpu, gpuFullTmr.TotalSecs(), cpu/gpu)

	sy.Destroy()
	gp.Destroy()
	vgpu.Terminate()
}
