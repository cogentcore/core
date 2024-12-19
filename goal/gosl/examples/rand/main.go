// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"

	"log/slog"

	"cogentcore.org/core/base/timer"
)

//go:generate gosl

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

func main() {
	GPUInit()

	// n := 10
	// n := 16_000_000 // max for macbook M*
	n := 200_000

	UseGPU = false

	Seed = make([]Seeds, 1)

	dataC := make([]Rnds, n)
	dataG := make([]Rnds, n)

	Data = dataC

	cpuTmr := timer.Time{}
	cpuTmr.Start()
	RunOneCompute(n)
	cpuTmr.Stop()

	UseGPU = true
	Data = dataG

	gpuFullTmr := timer.Time{}
	gpuFullTmr.Start()

	ToGPU(SeedVar, DataVar)

	gpuTmr := timer.Time{}
	gpuTmr.Start()

	RunCompute(n)
	gpuTmr.Stop()

	RunDone(DataVar)
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

	GPURelease()
}
