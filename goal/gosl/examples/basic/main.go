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
	"cogentcore.org/core/gpu"
)

//go:generate gosl .

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

func main() {
	gpu.Debug = true
	GPUInit()

	n := 2000000 // note: not necc to spec up-front, but easier if so

	Params = make([]ParamStruct, 1)
	Params[0].Defaults()

	Data = make([]DataStruct, n)
	for i := range Data {
		Data[i].Raw = rand.Float32()
	}

	sd := make([]DataStruct, n)
	for i := range sd {
		sd[i].Raw = Data[i].Raw
	}

	cpuTmr := timer.Time{}
	cpuTmr.Start()

	UseGPU = false
	RunOneCompute(n)

	cpuTmr.Stop()

	cd := Data
	Data = sd

	gpuFullTmr := timer.Time{}
	gpuFullTmr.Start()

	ToGPU(ParamsVar, DataVar)

	gpuTmr := timer.Time{}
	gpuTmr.Start()

	UseGPU = true
	RunOneCompute(n, DataVar)

	gpuTmr.Stop()

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

	GPURelease()
}
