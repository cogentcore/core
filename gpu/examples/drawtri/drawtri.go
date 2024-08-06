// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"
	"fmt"
	"image"
	"runtime"
	"time"

	"cogentcore.org/core/gpu"
)

//go:embed trianglelit.wgsl
var trianglelit string

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

func main() {
	gp := gpu.NewGPU()
	gpu.Debug = true
	gp.Config("drawtri")

	var resize func(size image.Point)
	size := image.Point{1024, 768}
	sp, terminate, pollEvents, size, err := gpu.GLFWCreateWindow(gp, size, "Draw Triangle", &resize)
	if err != nil {
		return
	}

	sf := gpu.NewSurface(gp, sp, size, 1, gpu.UndefinedType)
	sy := gpu.NewGraphicsSystem(gp, "drawtri", sf)
	fmt.Printf("format: %s\n", sf.Format.String())

	resize = func(size image.Point) { sf.SetSize(size) }
	destroy := func() {
		sy.Release()
		sf.Release()
		gp.Release()
		terminate()
	}

	pl := sy.AddGraphicsPipeline("drawtri")
	// pl.SetFrontFace(wgpu.FrontFaceCCW)
	// pl.SetCullMode(wgpu.CullModeNone)
	// pl.SetAlphaBlend(false)

	sh := pl.AddShader("trianglelit")
	sh.OpenCode(trianglelit)
	pl.AddEntry(sh, gpu.VertexShader, "vs_main")
	pl.AddEntry(sh, gpu.FragmentShader, "fs_main")

	sy.Config()
	// no vars..

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		// fmt.Printf("frame: %d\n", frameCount)
		// rt := time.Now()

		rp, err := sy.BeginRenderPass()
		if err != nil {
			return
		}
		pl.BindPipeline(rp)
		rp.Draw(3, 1, 0, 0)
		sy.EndRenderPass(rp)

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

	fpsDelay := time.Second / 6
	fpsTicker := time.NewTicker(fpsDelay)
	for {
		select {
		case <-exitC:
			fpsTicker.Stop()
			destroy()
			return
		case <-fpsTicker.C:
			if !pollEvents() {
				exitC <- struct{}{}
				continue
			}
			renderFrame()
		}
	}
}
