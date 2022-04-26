// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/demos
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License
// and https://bakedbits.dev/posts/vulkan-compute-example/

package main

import (
	"fmt"
	"math/rand"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/vgpu/vgpu"

	vk "github.com/vulkan-go/vulkan"
)

func init() {
	// must lock main thread for gpu!  this also means that vulkan must be used
	// for gogi/oswin eventually if we want gui and compute
	runtime.LockOSThread()
}

var TheGPU *vgpu.GPU

func main() {
	glfw.Init()
	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	vk.Init()

	gp := vgpu.NewGPU()
	gp.Init("compute1", true)
	TheGPU = gp

	// gp.PropsString(true) // print

	sy := gp.NewSystem("compute1", true)
	pl := sy.AddNewPipeline("compute1")
	pl.AddShaderFile("sqvecel", vgpu.ComputeShader, "sqvecel.spv")
	_ = pl

	inv := sy.Vars.Add("In", vgpu.Float32Vec4, vgpu.Storage, 0, vgpu.ComputeShader)
	outv := sy.Vars.Add("Out", vgpu.Float32Vec4, vgpu.Storage, 0, vgpu.ComputeShader)

	n := 20
	ivl := sy.Mem.Vals.Add("In", inv, n)
	ovl := sy.Mem.Vals.Add("Out", outv, n)
	_ = ovl

	sy.Config()
	sy.Mem.Config()

	idat := ivl.Floats32()
	for i := 0; i < n; i++ {
		idat[i*4+0] = rand.Float32()
		idat[i*4+1] = rand.Float32()
		idat[i*4+2] = rand.Float32()
		idat[i*4+3] = rand.Float32()
		ivl.Mod = true
	}

	sy.Mem.SyncAllToGPU()

	sy.SetVals(0, "In", "Out")

	pl.RunCompute(n, 1, 1)

	sy.Mem.SyncVarsFmGPU("Out")

	odat := ovl.Floats32()
	for i := 0; i < n; i++ {
		fmt.Printf("In:  %d\tr: %g\tg: %g\tb: %g\ta: %g\n", i, idat[i*4+0], idat[i*4+1], idat[i*4+2], idat[i*4+3])
		fmt.Printf("Out: %d\tr: %g\tg: %g\tb: %g\ta: %g\n", i, odat[i*4+0], odat[i*4+1], odat[i*4+2], odat[i*4+3])
	}
	fmt.Printf("\n")

	sy.Destroy()
	gp.Destroy()
}
