// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This example just does some basic calculations on data structures and
// reports the time difference between the CPU and GPU.
package main

import (
	"embed"
	"fmt"
	"math/rand"
	"runtime"
	"unsafe"

	"cogentcore.org/core/base/timer"
	"cogentcore.org/core/gpu"
)

//go:generate ../../gosl cogentcore.org/core/math32/fastexp.go compute.go

//go:embed shaders/basic.wgsl shaders/fastexp.wgsl
var shaders embed.FS

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

func main() {
	gpu.Debug = true
	gp := gpu.NewComputeGPU()
	fmt.Printf("Running on GPU: %s\n", gp.DeviceName)

	// gp.PropertiesString(true) // print

	sy := gpu.NewComputeSystem(gp, "compute")
	pl := gpu.NewComputePipelineShaderFS(shaders, "shaders/basic.wgsl", sy)
	vars := sy.Vars()
	sgp := vars.AddGroup(gpu.Storage)

	n := 2000000 // note: not necc to spec up-front, but easier if so
	threads := 64

	pv := sgp.AddStruct("Params", int(unsafe.Sizeof(ParamStruct{})), 1, gpu.ComputeShader)
	dv := sgp.AddStruct("Data", int(unsafe.Sizeof(DataStruct{})), n, gpu.ComputeShader)

	sgp.SetNValues(1)
	sy.Config()

	pvl := pv.Values.Values[0]
	dvl := dv.Values.Values[0]

	pars := make([]ParamStruct, 1)
	pars[0].Defaults()

	cd := make([]DataStruct, n)
	for i := range cd {
		cd[i].Raw = rand.Float32()
	}

	sd := make([]DataStruct, n)
	for i := range sd {
		sd[i].Raw = cd[i].Raw
	}

	cpuTmr := timer.Time{}
	cpuTmr.Start()
	for i := range cd {
		pars[0].IntegFromRaw(&cd[i])
	}
	cpuTmr.Stop()

	gpuFullTmr := timer.Time{}
	gpuFullTmr.Start()

	gpu.SetValueFrom(pvl, pars)
	gpu.SetValueFrom(dvl, sd)

	sgp.CreateReadBuffers()

	gpuTmr := timer.Time{}
	gpuTmr.Start()

	ce, _ := sy.BeginComputePass()
	pl.Dispatch1D(ce, n, threads)
	ce.End()
	dvl.GPUToRead(sy.CommandEncoder)
	sy.EndComputePass(ce)

	gpuTmr.Stop()

	dvl.ReadSync()
	gpu.ReadToBytes(dvl, sd)

	gpuFullTmr.Stop()

	mx := min(n, 5)
	for i := 0; i < mx; i++ {
		d := cd[i].Exp - sd[i].Exp
		fmt.Printf("%d\t Raw: %g\t Integ: %g\t Exp: %6.4g\tTrg: %6.4g\tDiff: %g\n", i, sd[i].Raw, sd[i].Integ, sd[i].Exp, cd[i].Exp, d)
	}
	fmt.Printf("\n")

	cpu := cpuTmr.Total
	gpu := gpuTmr.Total
	gpuFull := gpuFullTmr.Total
	fmt.Printf("N: %d\t CPU: %v\t GPU: %v\t Full: %v\t CPU/GPU: %6.4g\n", n, cpu, gpu, gpuFull, float64(cpu)/float64(gpu))

	sy.Release()
	gp.Release()
}
