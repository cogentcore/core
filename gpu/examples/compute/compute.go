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

	"cogentcore.org/core/gpu"
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
	gpu.Debug = true
	gp := gpu.NewComputeGPU()
	fmt.Printf("Running on GPU: %s\n", gp.DeviceName)

	// gp.PropertiesString(true) // print

	sy := gpu.NewComputeSystem(gp, "compute")
	pl := gpu.NewComputePipelineShaderFS(shaders, "squares.wgsl", sy)

	vars := sy.Vars()
	sgp := vars.AddGroup(gpu.Storage)

	n := 20 // note: not necc to spec up-front, but easier if so
	threads := 64

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

	ce, _ := sy.BeginComputePass()
	pl.Dispatch1D(ce, n, threads)
	ce.End()
	dvl.GPUToRead(sy.CommandEncoder)
	sy.EndComputePass()

	dvl.ReadSync()
	gpu.ReadToBytes(dvl, sd)

	for i := 0; i < n; i++ {
		tc := sd[i].A + sd[i].B
		td := tc * tc
		dc := sd[i].C - tc
		dd := sd[i].D - td
		fmt.Printf("%d\t A: %g\t B: %g\t C: %g\t trg: %g\t D: %g \t trg: %g \t difC: %g \t difD: %g\n", i, sd[i].A, sd[i].B, sd[i].C, tc, sd[i].D, td, dc, dd)
	}
	fmt.Printf("\n")

	sy.Release()
	gp.Release()
}
