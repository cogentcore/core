// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This example just does some basic calculations on data structures and
// reports the time difference between the CPU and GPU.
package main

import (
	"fmt"
	"math/rand"
	"runtime"

	"cogentcore.org/core/base/timer"
	"cogentcore.org/core/goal/gosl/sltensor"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/tensor"
)

//go:generate gosl

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

func main() {
	gpu.Debug = true
	GPUInit()

	n := 2_000_000 // note: not necc to spec up-front, but easier if so

	Params = make([]ParamStruct, 1)
	Params[0].Defaults()

	Data = tensor.NewFloat32()
	sltensor.SetShapeSizes(Data, n, 3) // critically, makes GPU compatible Header with strides
	nt := Data.Len()

	IntData = tensor.NewInt32()
	sltensor.SetShapeSizes(IntData, n, 3) // critically, makes GPU compatible Header with strides

	for i := range nt {
		Data.Set1D(rand.Float32(), i)
	}

	sid := tensor.NewInt32()
	sltensor.SetShapeSizes(sid, n, 3) // critically, makes GPU compatible Header with strides

	sd := tensor.NewFloat32()
	sltensor.SetShapeSizes(sd, n, 3)
	for i := range nt {
		sd.Set1D(Data.Value1D(i), i)
	}

	cpuTmr := timer.Time{}
	cpuTmr.Start()

	UseGPU = false
	RunOneAtomic(n)
	RunOneCompute(n)

	cpuTmr.Stop()

	cd := Data
	cid := IntData
	Data = sd
	IntData = sid

	gpuFullTmr := timer.Time{}
	gpuFullTmr.Start()

	ToGPU(ParamsVar, DataVar, IntDataVar)

	gpuTmr := timer.Time{}
	gpuTmr.Start()

	UseGPU = true
	RunAtomic(n)
	RunCompute(n)
	gpuTmr.Stop()

	RunDone(DataVar, IntDataVar)
	gpuFullTmr.Stop()

	mx := min(n, 5)
	for i := 0; i < mx; i++ {
		fmt.Printf("%d\t CPU IntData: %d\t GPU: %d\n", i, cid.Value(1, Integ), sid.Value(i, Integ))
	}
	fmt.Println()
	for i := 0; i < mx; i++ {
		d := cd.Value(i, Exp) - sd.Value(i, Exp)
		fmt.Printf("%d\t Raw: %6.4g\t Integ: %6.4g\t Exp: %6.4g\tTrg: %6.4g\tDiff: %g\n", i, sd.Value(i, Raw), sd.Value(i, Integ), sd.Value(i, Exp), cd.Value(i, Exp), d)
	}
	fmt.Printf("\n")

	cpu := cpuTmr.Total
	gpu := gpuTmr.Total
	gpuFull := gpuFullTmr.Total
	fmt.Printf("N: %d\t CPU: %v\t GPU: %v\t Full: %v\t CPU/GPU: %6.4g\n", n, cpu, gpu, gpuFull, float64(cpu)/float64(gpu))

	GPURelease()
}
