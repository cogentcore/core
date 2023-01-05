// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math/rand"
	"runtime"

	"github.com/goki/vgpu/vgpu"
)

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

var TheGPU *vgpu.GPU

func main() {
	if vgpu.Init() != nil {
		return
	}

	gp := vgpu.NewComputeGPU()
	gp.Debug = true
	gp.Config("compute1")
	TheGPU = gp

	// gp.PropsString(true) // print

	sy := gp.NewComputeSystem("compute1")
	pl := sy.NewPipeline("compute1")
	pl.AddShaderFile("sqvecel", vgpu.ComputeShader, "sqvecel.spv")

	vars := sy.Vars()
	set := vars.AddSet()

	n := 20 // note: not necc to spec up-front, but easier if so
	inv := set.Add("In", vgpu.Float32Vec4, n, vgpu.Storage, vgpu.ComputeShader)
	outv := set.Add("Out", vgpu.Float32Vec4, n, vgpu.Storage, vgpu.ComputeShader)
	_ = outv

	set.ConfigVals(1) // one val per var
	sy.Config()       // configures vars, allocates vals, configs pipelines..

	ivl, _ := inv.Vals.ValByIdxTry(0)
	idat := ivl.Floats32()
	for i := 0; i < n; i++ {
		idat[i*4+0] = rand.Float32()
		idat[i*4+1] = rand.Float32()
		idat[i*4+2] = rand.Float32()
		idat[i*4+3] = rand.Float32()
	}
	ivl.SetMod()

	sy.Mem.SyncToGPU()

	vars.BindDynValIdx(0, "In", 0)
	vars.BindDynValIdx(0, "Out", 0)

	sy.CmdResetBindVars(sy.CmdPool.Buff, 0)
	pl.RunComputeWait(sy.CmdPool.Buff, n, 1, 1)
	// note: could use semaphore here instead of waiting on the compute

	sy.Mem.SyncValIdxFmGPU(0, "Out", 0)
	_, ovl, _ := vars.ValByIdxTry(0, "Out", 0)

	odat := ovl.Floats32()
	for i := 0; i < n; i++ {
		fmt.Printf("In:  %d\tr: %g\tg: %g\tb: %g\ta: %g\n", i, idat[i*4+0], idat[i*4+1], idat[i*4+2], idat[i*4+3])
		fmt.Printf("Out: %d\tr: %g\tg: %g\tb: %g\ta: %g\n", i, odat[i*4+0], odat[i*4+1], odat[i*4+2], odat[i*4+3])
	}
	fmt.Printf("\n")

	sy.Destroy()
	gp.Destroy()
	vgpu.Terminate()
}
