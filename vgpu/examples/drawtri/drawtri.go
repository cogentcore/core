// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	vk "github.com/goki/vulkan"

	"github.com/go-gl/glfw/v3.3/glfw"
	"goki.dev/vgpu/v2/vgpu"
)

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

func main() {
	if vgpu.Init() != nil {
		return
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(1024, 768, "Draw Triangle", nil, nil)
	vgpu.IfPanic(err)

	// note: for graphics, require these instance extensions before init gpu!
	winext := window.GetRequiredInstanceExtensions()
	gp := vgpu.NewGPU()
	gp.AddInstanceExt(winext...)
	vgpu.Debug = true
	gp.Config("drawtri")

	// gp.PropsString(true) // print

	surfPtr, err := window.CreateWindowSurface(gp.Instance, nil)
	if err != nil {
		log.Println(err)
		return
	}
	sf := vgpu.NewSurface(gp, vk.SurfaceFromPointer(surfPtr))

	fmt.Printf("format: %s\n", sf.Format.String())

	sy := gp.NewGraphicsSystem("drawtri", &sf.Device)
	pl := sy.NewPipeline("drawtri")
	sy.ConfigRender(&sf.Format, vgpu.UndefType)
	sf.SetRender(&sy.Render)

	pl.AddShaderFile("trianglelit", vgpu.VertexShader, "trianglelit.spv")
	pl.AddShaderFile("vtxcolor", vgpu.FragmentShader, "vtxcolor.spv")

	sy.Config()

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		sy.Destroy()
		sf.Destroy()
		gp.Destroy()
		window.Destroy()
		vgpu.Terminate()
	}

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()
		idx := sf.AcquireNextImage()
		// fmt.Printf("\nacq: %v\n", time.Now().Sub(rt))
		descIdx := 0 // if running multiple frames in parallel, need diff sets
		cmd := sy.CmdPool.Buff
		sy.ResetBeginRenderPass(cmd, sf.Frames[idx], descIdx)
		// fmt.Printf("rp: %v\n", time.Now().Sub(rt))
		pl.BindPipeline(cmd)
		pl.Draw(cmd, 3, 1, 0, 0)
		sy.EndRenderPass(cmd)
		sf.SubmitRender(cmd) // this is where it waits for the 16 msec
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
