// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/goki/ki/ints"
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
	gp.Config("exptest")
	fmt.Printf("Running on GPU: %s\n", gp.DeviceName)

	fmt.Printf("Max StructuredBuffer Size: %X\n", gp.GPUProps.Limits.MaxStorageBufferRange)

	// gp.PropsString(true) // print

	sy := gp.NewComputeSystem("exptest")
	pl := sy.NewPipeline("exptest")
	pl.AddShaderFile("gpu_exptest", vgpu.ComputeShader, "gpu_exptest.spv")

	vars := sy.Vars()
	set := vars.AddSet()

	n := 64

	threads := 64
	nInt := ints.IntMultiple(n, threads)
	n = nInt               // enforce optimal n's -- otherwise requires range checking
	nGps := nInt / threads // dispatch n
	fmt.Printf("n: %d\n", n)

	inv := set.Add("In", vgpu.Float32, n, vgpu.Storage, vgpu.ComputeShader)
	outv := set.Add("Out", vgpu.Float32, n, vgpu.Storage, vgpu.ComputeShader)
	_ = outv

	set.ConfigVals(1) // one val per var
	sy.Config()       // configures vars, allocates vals, configs pipelines..

	ivals := make([]float32, n)
	cpuVals := make([]float32, n)

	// st := float32(-89)
	// st := float32(3)
	st := float32(-70)
	inc := float32(1.0e-01)
	cur := st
	for i := 0; i < n; i++ {
		ivals[i] = cur
		// cpuVals[i] = mat32.FastExp(ivals[i]) // 0 diffs
		vbio := ivals[i]
		eval := 0.1 * ((vbio + 90.0) + 10.0)
		// cpuVals[i] = (vbio + 90.0) / (1.0 + mat32.FastExp(eval)) // lots of diffs
		// cpuVals[i] = eval // 0 diff
		cpuVals[i] = float32(1.0) / eval // no diff from casting
		// cpuVals[i] = 1.0 / mat32.FastExp(eval)
		cur += inc
	}

	ivl, _ := inv.Vals.ValByIdxTry(0)
	ivl.CopyFromBytes(unsafe.Pointer(&(ivals[0])))
	sy.Mem.SyncToGPU()

	vars.BindDynValsAllIdx(0)

	cmd := sy.ComputeCmdBuff()

	sy.ComputeResetBindVars(cmd, 0)
	pl.ComputeDispatch(cmd, nGps, 1, 1)
	sy.ComputeCmdEnd(cmd)
	sy.ComputeSubmitWait(cmd)

	sy.Mem.SyncValIdxFmGPU(0, "Out", 0)
	_, ovl, _ := vars.ValByIdxTry(0, "Out", 0)

	odat := ovl.Floats32()
	for i := 0; i < n; i++ {
		diff := odat[i] - cpuVals[i]
		fmt.Printf("In:  %d\tival: %g\tcpu: %g\tgpu: %g\tdiff: %g\n", i, ivals[i], cpuVals[i], odat[i], diff)
	}
	fmt.Printf("\n")

	sy.Destroy()
	gp.Destroy()
	vgpu.Terminate()
}
