// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/demos
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License
// and https://bakedbits.dev/posts/vulkan-compute-example/

package main

import (
	"fmt"
	"log"
	"runtime"
	"time"
	"unsafe"

	vk "github.com/vulkan-go/vulkan"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/mat32"
	"github.com/goki/vgpu/vgpu"
)

func init() {
	// must lock main thread for gpu!  this also means that vulkan must be used
	// for gogi/oswin eventually if we want gui and compute
	runtime.LockOSThread()
}

var TheGPU *vgpu.GPU

type CamView struct {
	Model mat32.Mat4
	View  mat32.Mat4
	Prjn  mat32.Mat4
}

func main() {
	glfw.Init()
	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	vk.Init()

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(1024, 768, "Draw Triangle", nil, nil)
	vgpu.IfPanic(err)

	// note: for graphics, require these instance extensions before init gpu!
	winext := window.GetRequiredInstanceExtensions()
	gp := vgpu.NewGPU()
	gp.AddInstanceExt(winext...)
	gp.Debug = true
	gp.Config("drawidx")
	TheGPU = gp

	// gp.PropsString(true) // print

	surfPtr, err := window.CreateWindowSurface(gp.Instance, nil)
	if err != nil {
		log.Println(err)
		return
	}
	sf := vgpu.NewSurface(gp, vk.SurfaceFromPointer(surfPtr))

	fmt.Printf("format: %#v\n", sf.Format)

	sy := gp.NewGraphicsSystem("drawidx", &sf.Device)
	pl := sy.NewPipeline("drawidx")
	sy.ConfigRenderPass(&sf.Format, vk.FormatUndefined)
	sf.SetRenderPass(&sy.RenderPass)
	pl.SetGraphicsDefaults()

	pl.AddShaderFile("indexed", vgpu.VertexShader, "indexed.spv")
	pl.AddShaderFile("vtxcolor", vgpu.FragmentShader, "vtxcolor.spv")

	posv := sy.Vars.Add("Pos", vgpu.Float32Vec3, vgpu.Vertex, 0, vgpu.VertexShader)
	clrv := sy.Vars.Add("Color", vgpu.Float32Vec3, vgpu.Vertex, 0, vgpu.VertexShader)

	camv := sy.Vars.Add("Camera", vgpu.Struct, vgpu.Uniform, 0, vgpu.VertexShader)
	camv.SizeOf = vgpu.Float32Mat4.Bytes() * 3 // no padding for these

	nPts := 3

	triPos := sy.Mem.Vals.Add("TriPos", posv, nPts)
	triClr := sy.Mem.Vals.Add("TriClr", clrv, nPts)
	cam := sy.Mem.Vals.Add("Camera", camv, 1)

	sy.Config()
	sy.Mem.Config()

	triPosA := triPos.Floats32()
	triPosA.Set(0,
		-0.5, 0.5, 0.0,
		0.5, 0.5, 0.0,
		0.0, -0.5, 0.0)

	triClrA := triClr.Floats32()
	triClrA.Set(0,
		1.0, 0.0, 0.0,
		0.0, 1.0, 0.0,
		0.0, 0.0, 1.0)

	var camo CamView
	camo.Model.SetIdentity()
	camo.View.SetIdentity()
	camo.Prjn.SetPerspective(30, 1.5, 0.01, 1000)

	cam.CopyBytes(unsafe.Pointer(&camo))

	sy.Mem.SyncToGPU()
	sy.SetVals(0, "TriPos", "TriClr", "Camera")

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		vk.DeviceWaitIdle(sy.Device.Device)
		sy.Destroy()
		sf.Destroy()
		gp.Destroy()
		window.Destroy()
		glfw.Terminate()
	}

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()
		idx := sf.AcquireNextImage()
		// fmt.Printf("\nacq: %v\n", time.Now().Sub(rt))
		pl.FullStdRender(pl.CmdPool.Buff, sf.Frames[idx])
		// fmt.Printf("cmd %v\n", time.Now().Sub(rt))
		sf.SubmitRender(pl.CmdPool.Buff) // this is where it waits for the 16 msec
		// fmt.Printf("submit %v\n", time.Now().Sub(rt))
		sf.PresentImage(idx)
		// fmt.Printf("present %v\n\n", time.Now().Sub(rt))
		frameCount++
		eTime := time.Now()
		dur := float64(eTime.Sub(stTime)) / float64(time.Second)
		if dur > 10 {
			fps := float64(frameCount) / dur
			fmt.Printf("fps: %.0f\n", fps)
			frameCount = 0
			stTime = eTime
		}
	}

	exitC := make(chan struct{}, 2)

	fpsDelay := time.Second / 600
	fpsTicker := time.NewTicker(fpsDelay)
	for {
		select {
		case <-exitC:
			fpsTicker.Stop()
			destroy()
			return
		case <-fpsTicker.C:
			if window.ShouldClose() {
				exitC <- struct{}{}
				continue
			}
			glfw.PollEvents()
			renderFrame()
		}
	}
}
