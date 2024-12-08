// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"
	"fmt"
	"math/rand"
	"runtime"
	"unsafe"

	"cogentcore.org/core/base/timer"
	"cogentcore.org/core/gpu"
	// "cogentcore.org/core/system/driver/web/jsfs"
)

//go:embed squares.wgsl
var shaders embed.FS

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

type Data struct {
	A float32
	B float32
	C float32
	D float32
}

func main() {
	// errors.Log1(jsfs.Config(js.Global().Get("fs"))) // needed for printing etc to work
	// time.Sleep(1 * time.Second)
	// b := core.NewBody()
	// bt := core.NewButton(b).SetText("Run Compute")
	// bt.OnClick(func(e events.Event) {
	compute()
	// })
	// b.RunMainWindow()
	// select {}
}

func compute() {
	// gpu.SetDebug(true)
	gp := gpu.NewComputeGPU()
	fmt.Printf("Running on GPU: %s\n", gp.DeviceName)

	// gp.PropertiesString(true) // print

	sy := gpu.NewComputeSystem(gp, "compute")
	pl := gpu.NewComputePipelineShaderFS(shaders, "squares.wgsl", sy)

	vars := sy.Vars()
	sgp := vars.AddGroup(gpu.Storage)

	// n := 16_000_000 // near max capacity on Mac M*
	n := 200_000 // should fit in any webgpu
	threads := 64
	nx, ny := gpu.NumWorkgroups1D(n, threads)
	fmt.Printf("workgroup sizes: %d, %d  storage mem bytes: %X\n", nx, ny, n*int(unsafe.Sizeof(Data{})))

	dv := sgp.AddStruct("Data", int(unsafe.Sizeof(Data{})), n, gpu.ComputeShader)

	sgp.SetNValues(1)
	sy.Config()

	dvl := dv.Values.Values[0]

	sd := make([]Data, n)
	for i := range sd {
		sd[i].A = rand.Float32()
		sd[i].B = rand.Float32()
	}
	gpu.SetValueFrom(dvl, sd)

	gpuTmr := timer.Time{}
	cpyTmr := timer.Time{}
	gpuTmr.Start()
	nItr := 1

	for range nItr {
		ce, _ := sy.BeginComputePass()
		pl.Dispatch1D(ce, n, threads)
		ce.End()
		dvl.GPUToRead(sy.CommandEncoder)
		sy.EndComputePass()

		cpyTmr.Start()
		dvl.ReadSync()
		cpyTmr.Stop()
		gpu.ReadToBytes(dvl, sd)
	}

	gpuTmr.Stop()

	mx := min(n, 10)
	for i := 0; i < mx; i++ {
		tc := sd[i].A + sd[i].B
		td := tc * tc
		dc := sd[i].C - tc
		dd := sd[i].D - td
		fmt.Printf("%d\t A: %g\t B: %g\t C: %g\t trg: %g\t D: %g \t trg: %g \t difC: %g \t difD: %g\n", i, sd[i].A, sd[i].B, sd[i].C, tc, sd[i].D, td, dc, dd)
	}
	fmt.Printf("\n")
	fmt.Println("total:", gpuTmr.Total, "copy:", cpyTmr.Total)

	sy.Release()
	gp.Release()
}
