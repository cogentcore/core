// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"
	"unsafe"

	"log/slog"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/vgpu"
	"github.com/emer/gosl/v2/sltype"
	"github.com/emer/gosl/v2/timer"
)

// note: standard one to use is plain "gosl" which should be go install'd

//go:generate ../../gosl rand.go rand.hlsl

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
	gp.Config("slrand")

	// gp.PropsString(true) // print

	// n := 10
	n := 10000000
	threads := 64
	nInt := int(math32.IntMultiple(float32(n), float32(threads)))
	n = nInt               // enforce optimal n's -- otherwise requires range checking
	nGps := nInt / threads // dispatch n

	dataC := make([]Rnds, n)
	dataG := make([]Rnds, n)

	cpuTmr := timer.Time{}
	cpuTmr.Start()

	seed := sltype.Uint2{0, 0}

	for i := range dataC {
		d := &dataC[i]
		d.RndGen(seed, uint32(i))
	}
	cpuTmr.Stop()

	sy := gp.NewComputeSystem("slrand")
	pl := sy.NewPipeline("slrand")
	pl.AddShaderFile("slrand", vgpu.ComputeShader, "shaders/rand.spv")

	vars := sy.Vars()
	setc := vars.AddSet()
	setd := vars.AddSet()

	ctrv := setc.AddStruct("Counter", int(unsafe.Sizeof(seed)), 1, vgpu.Storage, vgpu.ComputeShader)
	datav := setd.AddStruct("Data", int(unsafe.Sizeof(Rnds{})), n, vgpu.Storage, vgpu.ComputeShader)

	setc.ConfigValues(1) // one val per var
	setd.ConfigValues(1) // one val per var
	sy.Config()          // configures vars, allocates vals, configs pipelines..

	gpuFullTmr := timer.Time{}
	gpuFullTmr.Start()

	// this copy is pretty fast -- most of time is below
	cvl, _ := ctrv.Values.ValueByIndexTry(0)
	cvl.CopyFromBytes(unsafe.Pointer(&seed))
	dvl, _ := datav.Values.ValueByIndexTry(0)
	dvl.CopyFromBytes(unsafe.Pointer(&dataG[0]))

	// gpuFullTmr := timer.Time{}
	// gpuFullTmr.Start()

	sy.Mem.SyncToGPU()

	vars.BindDynamicValueIndex(0, "Counter", 0)
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
	dvl.CopyToBytes(unsafe.Pointer(&dataG[0]))

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

	cpu := cpuTmr.TotalSecs()
	gpu := gpuTmr.TotalSecs()
	fmt.Printf("N: %d\t CPU: %6.4g\t GPU: %6.4g\t Full: %6.4g\t CPU/GPU: %6.4g\n", n, cpu, gpu, gpuFullTmr.TotalSecs(), cpu/gpu)

	sy.Destroy()
	gp.Destroy()
	vgpu.Terminate()
}
