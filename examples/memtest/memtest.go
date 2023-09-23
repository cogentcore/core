// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"

	"goki.dev/mat32/v2"
	"goki.dev/vgpu/v2/vgpu"
)

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

func main() {
	if vgpu.InitNoDisplay() != nil {
		return
	}

	gp := vgpu.NewComputeGPU()
	vgpu.Debug = true
	gp.Config("memtest")
	fmt.Printf("Running on GPU: %s\n", gp.DeviceName)

	// gp.PropsString(true) // print

	sy := gp.NewComputeSystem("memtest")
	sy.StaticVars = true // not working yet
	pl := sy.NewPipeline("memtest")
	pl.AddShaderFile("gpu_memtest", vgpu.ComputeShader, "gpu_memtest.spv")

	vars := sy.Vars()
	set := vars.AddSet()

	n := 64

	threads := 64
	nInt := mat32.IntMultiple(float32(n), float32(threads))
	n = int(nInt)       // enforce optimal n's -- otherwise requires range checking
	nGps := n / threads // dispatch n

	maxBuff := (gp.GPUProps.Limits.MaxStorageBufferRange - 16) / 4
	mem2g := ((1 << 31) - 1) / 4
	mem1g := ((1 << 30) - 1) / 4

	fmt.Printf("Sizes: Max StructuredBuffer: %X   2 GiB: %X  1 GiB: %X\n", maxBuff, mem2g, mem1g)

	ban := maxBuff // this causes the error: writes to a instead of b
	// ban := mem2g // works with this
	bbn := maxBuff

	bav := set.Add("Ba", vgpu.Uint32, int(ban), vgpu.Storage, vgpu.ComputeShader)
	bbv := set.Add("Bb", vgpu.Uint32, int(bbn), vgpu.Storage, vgpu.ComputeShader)

	_, _ = bav, bbv

	set.ConfigVals(1) // one val per var
	sy.Config()       // configures vars, allocates vals, configs pipelines..

	// bahost := make([]float32, bav)
	// bbhost := make([]float32, bbv)

	// vars.BindDynValsAllIdx(0)

	cmd := sy.ComputeCmdBuff()
	sy.ComputeResetBindVars(cmd, 0)
	pl.ComputeDispatch(cmd, nGps, 1, 1)
	sy.ComputeCmdEnd(cmd)
	sy.ComputeSubmitWait(cmd)

	sy.Mem.SyncValIdxFmGPU(0, "Ba", 0)
	_, bavl, _ := vars.ValByIdxTry(0, "Ba", 0)
	sy.Mem.SyncValIdxFmGPU(0, "Bb", 0)
	_, bbvl, _ := vars.ValByIdxTry(0, "Bb", 0)

	bas := bavl.UInts32()
	bbs := bbvl.UInts32()
	for i := 0; i < n; i++ {
		fmt.Printf("%d\ta: %X\tb: %X\n", i, bas[i], bbs[i])
	}
	fmt.Printf("\n")

	sy.Destroy()
	gp.Destroy()
	vgpu.Terminate()
}
