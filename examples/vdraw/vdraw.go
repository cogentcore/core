// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"runtime"
	"time"

	vk "github.com/goki/vulkan"

	"github.com/go-gl/glfw/v3.3/glfw"
	"goki.dev/vgpu/v2/vdraw"
	"goki.dev/vgpu/v2/vgpu"
	"goki.dev/video"
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
	window, err := glfw.CreateWindow(1024, 768, "vDraw Test", nil, nil)
	vgpu.IfPanic(err)

	// note: for graphics, require these instance extensions before init gpu!
	winext := window.GetRequiredInstanceExtensions()
	gp := vgpu.NewGPU()
	gp.AddInstanceExt(winext...)
	vgpu.Debug = true
	gp.Config("vDraw test")

	// gp.PropsString(true) // print

	surfPtr, err := window.CreateWindowSurface(gp.Instance, nil)
	if err != nil {
		log.Println(err)
		return
	}
	sf := vgpu.NewSurface(gp, vk.SurfaceFromPointer(surfPtr))

	fmt.Printf("format: %s\n", sf.Format.String())

	drw := &vdraw.Drawer{}
	drw.YIsDown = true
	drw.ConfigSurface(sf, 16) // requires 2 NDesc

	drw.SetMaxTextures(32) // test resizing

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		drw.Destroy()
		sf.Destroy()
		gp.Destroy()
		window.Destroy()
		vgpu.Terminate()
	}

	stoff := 15 // causes images to wrap around sets, so this tests that..

	rendImgs := func(idx int) {
		img, err := video.ReadFrame("../videos/countdown.mp4", idx)
		if err != nil {
			panic(err)
		}
		drw.SetGoImage(stoff, 0, img, vgpu.NoFlipY)
		drw.SyncImages()
		descIdx := 0
		if stoff >= vgpu.MaxTexturesPerSet {
			descIdx = 1
		}
		drw.StartDraw(descIdx) // specifically starting with correct descIdx is key..
		drw.Scale(stoff, 0, sf.Format.Bounds(), image.ZR, vdraw.Src, vgpu.NoFlipY)
		drw.Copy(stoff, 0, image.ZP, image.ZR, vdraw.Src, vgpu.NoFlipY)
		drw.EndDraw()
	}

	_ = rendImgs

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		rendImgs(frameCount)
		frameCount++

		eTime := time.Now()
		dur := float64(eTime.Sub(stTime)) / float64(time.Second)
		if dur > 100 {
			fps := float64(frameCount) / dur
			fmt.Printf("fps: %.0f\n", fps)
			frameCount = 0
			stTime = eTime
		}
	}

	glfw.PollEvents()
	renderFrame()
	glfw.PollEvents()

	exitC := make(chan struct{}, 2)

	fpsDelay := time.Second / 1
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
