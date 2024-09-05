// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"
	"fmt"
	"runtime"
	"unsafe"

	"log/slog"

	"cogentcore.org/core/base/timer"
	"cogentcore.org/core/gpu"
)

// note: standard one to use is plain "gosl" which should be go install'd

//go:generate ../../gosl rand.go rand.wgsl

//go:embed shaders/*.wgsl
var shaders embed.FS

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

func main() {
	gpu.Debug = true
	gp := gpu.NewComputeGPU()
	fmt.Printf("Running on GPU: %s\n", gp.DeviceName)

	// n := 10
	n := 4_000_000 // 5_000_000 is too much -- 256_000_000 -- up against buf size limit
	threads := 64

	dataC := make([]Rnds, n)
	dataG := make([]Rnds, n)

	cpuTmr := timer.Time{}
	cpuTmr.Start()

	seed := uint64(0)
	for i := range dataC {
		d := &dataC[i]
		d.RndGen(seed, uint32(i))
	}
	cpuTmr.Stop()

	sy := gpu.NewComputeSystem(gp, "slrand")
	pl := gpu.NewComputePipelineShaderFS(shaders, "shaders/rand.wgsl", sy)
	vars := sy.Vars()
	sgp := vars.AddGroup(gpu.Storage)

	ctrv := sgp.AddStruct("Counter", int(unsafe.Sizeof(seed)), 1, gpu.ComputeShader)
	datav := sgp.AddStruct("Data", int(unsafe.Sizeof(Rnds{})), n, gpu.ComputeShader)

	sgp.SetNValues(1)
	sy.Config()

	cvl := ctrv.Values.Values[0]
	dvl := datav.Values.Values[0]

	gpuFullTmr := timer.Time{}
	gpuFullTmr.Start()

	gpu.SetValueFrom(cvl, []uint64{seed})
	gpu.SetValueFrom(dvl, dataG)

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
	gpu.ReadToBytes(dvl, dataG)

	gpuFullTmr.Stop()

	anyDiffEx := false
	anyDiffTol := false
	mx := min(n, 5)
	fmt.Printf("Index\tDif(Ex,Tol)\t   CPU   \t  then GPU\n")
	for i := 0; i < n; i++ {
		dc := &dataC[i]
		dg := &dataG[i]
		smEx, smTol := dc.IsSame(dg)
		if !smEx {
			anyDiffEx = true
		}
		if !smTol {
			anyDiffTol = true
		}
		if i > mx {
			continue
		}
		exS := " "
		if !smEx {
			exS = "*"
		}
		tolS := " "
		if !smTol {
			tolS = "*"
		}
		fmt.Printf("%d\t%s %s\t%s\n\t\t%s\n", i, exS, tolS, dc.String(), dg.String())
	}
	fmt.Printf("\n")

	if anyDiffEx {
		slog.Error("Differences between CPU and GPU detected at Exact level (excludes Gauss)")
	}
	if anyDiffTol {
		slog.Error("Differences between CPU and GPU detected at Tolerance level", "tolerance", Tol)
	}

	cpu := cpuTmr.Total
	gpu := gpuTmr.Total
	fmt.Printf("N: %d\t CPU: %v\t GPU: %v\t Full: %v\t CPU/GPU: %6.4g\n", n, cpu, gpu, gpuFullTmr.Total, float64(cpu)/float64(gpu))

	sy.Release()
	gp.Release()
}
