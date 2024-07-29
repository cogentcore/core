// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"cogentcore.org/core/gpu"

	"github.com/go-gl/glfw/v3.3/glfw"
)

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

func main() {
	if gpu.Init() != nil {
		return
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(1024, 768, "Draw Triangle", nil, nil)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	gp := gpu.NewGPU()
	gp.Config("drawtri")

	sp := gp.Instance.CreateSurface(wgpuext_glfw.GetSurfaceDescriptor(window))
	width, height := window.GetSize()
	sf := gpu.NewSurface(gp, sp, width, height)

	fmt.Printf("format: %s\n", sf.Format.String())

	sy := gp.NewGraphicsSystem("drawtri", &sf.Device)
	pl := sy.AddGraphicPipeline("drawtri")
	sy.ConfigRender(gpu.UndefType)
	// sf.SetRender(&sy.Render)

	sh := pl.AddShader("trianglelit").OpenFile("trianglelit.wgl")
	pl.AddEntry(sh, gpu.VertexShader, "vs_main")
	pl.AddEntry(sh, gpu.FragmentShader, "fs_main")

	sy.Config()
	// no vars..

	destroy := func() {
		// vk.DeviceWaitIdle(sf.Device.Device)
		// todo: poll
		sy.Release()
		sf.Release()
		gp.Release()
		window.Release()
		gpu.Terminate()
	}

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()
		view, ok := sf.AcquireNextTexture()
		if !ok {
			return
		}
		// fmt.Printf("\nacq: %v\n", time.Now().Sub(rt))
		cmd := sy.NewCommandEncoder()
		rp := sy.BeginRenderPass(cmd, view)
		// fmt.Printf("rp: %v\n", time.Now().Sub(rt))
		rp.SetPipeline(pl.Pipeline())
		rp.Draw(3, 1, 0, 0)
		sy.EndRenderPass(cmd)
		sf.SubmitRender(cmd) // this is where it waits for the 16 msec
		// fmt.Printf("submit %v\n", time.Now().Sub(rt))
		sf.PresentTexture(idx)
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
