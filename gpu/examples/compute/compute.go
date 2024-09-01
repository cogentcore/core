// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"
	"fmt"
	"math/rand"
	"runtime"

	"cogentcore.org/core/gpu"
)

//go:embed squares.wgsl
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
	pl := gpu.NewComputePipelineShaderFS(shaders, "squares.wgsl", sy)

	vars := sy.Vars()
	sgp := vars.AddGroup(gpu.Storage)

	n := 20 // note: not necc to spec up-front, but easier if so
	threads := 64

	inv := sgp.Add("In", gpu.Float32, n, gpu.ComputeShader)
	outv := sgp.Add("Out", gpu.Float32, n, gpu.ComputeShader)
	_ = outv

	sgp.SetNValues(1)
	sy.Config()

	ivl := inv.Values.Values[0]
	ovl := outv.Values.Values[0]

	ivals := make([]float32, n)
	for i := range ivals {
		ivals[i] = rand.Float32()
	}
	gpu.SetValueFrom(ivl, ivals)
	ovl.CreateBuffer()

	sgp.CreateReadBuffers()

	ce, _ := sy.BeginComputePass()
	pl.Dispatch1D(ce, n, threads)
	ce.End()
	ovl.GPUToRead(sy.CommandEncoder)
	sy.EndComputePass(ce)

	ovl.ReadSync()
	ovals := make([]float32, n)
	gpu.ReadToBytes(ovl, ovals)

	for i := 0; i < n; i++ {
		trg := ivals[i] * ivals[i]
		diff := ovals[i] - trg
		fmt.Printf("In:  %d\t in: %g\t out: %g\t trg: %g diff: %g\n", i, ivals[i], ovals[i], trg, diff)
	}
	fmt.Printf("\n")

	sy.Release()
	gp.Release()
}
