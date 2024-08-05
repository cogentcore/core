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

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/gpu"

	"github.com/rajveermalviya/go-webgpu/wgpu"
)

//go:embed trianglelit.wgsl
var trianglelit string

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

func main() {
	gp := gpu.NewGPU()
	gp.Config("drawtri")

	var resize func(width, height int)
	width, height := 1024, 768
	sp, terminate, pollEvents, width, height, err := gpu.GLFWCreateWindow(gp, width, height, "Draw Triangle", &resize)
	if err != nil {
		return
	}

	sf := gpu.NewSurface(gp, sp, width, height)

	fmt.Printf("format: %s\n", sf.Format.String())

	sy := gp.NewGraphicsSystem("drawtri", sf.Device)
	sy.ConfigRender(&sf.Format, gpu.UndefType, sf)

	resize = func(width, height int) {
		sf.Resized(image.Point{width, height})
	}
	destroy := func() {
		sy.Release()
		sf.Release()
		gp.Release()
		terminate()
	}

	pl := sy.AddGraphicsPipeline("drawtri")
	pl.SetFrontFace(wgpu.FrontFaceCW)

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
		view, err := sf.AcquireNextTexture()
		if errors.Log(err) != nil {
			return
		}
		// fmt.Printf("\nacq: %v\n", time.Now().Sub(rt))
		cmd := sy.NewCommandEncoder()
		rp := sy.BeginRenderPass(cmd, view)
		// fmt.Printf("rp: %v\n", time.Now().Sub(rt))
		pl.BindPipeline(rp)
		rp.Draw(3, 1, 0, 0)
		rp.End()
		sf.SubmitRender(rp, cmd) // this is where it waits for the 16 msec
		// fmt.Printf("submit %v\n", time.Now().Sub(rt))
		sf.Present()
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
