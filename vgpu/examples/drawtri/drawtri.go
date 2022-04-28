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

	vk "github.com/vulkan-go/vulkan"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/vgpu/vgpu"
	"github.com/xlab/closer"
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
	defer closer.Close()

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(1024, 768, "Draw Triangle", nil, nil)
	vgpu.IfPanic(err)

	// note: for graphics, require these instance extensions before init gpu!
	winext := window.GetRequiredInstanceExtensions()
	fmt.Println(winext)
	gp := vgpu.NewGPU()
	gp.AddInstanceExt(winext...)
	gp.Init("drawtri", true)
	TheGPU = gp

	// gp.PropsString(true) // print

	surfPtr, err := window.CreateWindowSurface(gp.Instance, nil)
	if err != nil {
		log.Println(err)
		return
	}
	vks := vk.SurfaceFromPointer(surfPtr)

	sf := &vgpu.Surface{}
	sf.Defaults()
	sf.Init(gp, vks)

	// some sync logic
	doneC := make(chan struct{}, 2)
	exitC := make(chan struct{}, 2)
	defer closer.Bind(func() {
		exitC <- struct{}{}
		<-doneC
		log.Println("Bye!")
	})

	fpsDelay := time.Second / 60
	fpsTicker := time.NewTicker(fpsDelay)
	for {
		select {
		case <-exitC:
			gp.Destroy()
			window.Destroy()
			glfw.Terminate()
			fpsTicker.Stop()
			doneC <- struct{}{}
			return
		case <-fpsTicker.C:
			if window.ShouldClose() {
				exitC <- struct{}{}
				continue
			}
			glfw.PollEvents()
			// app.NextFrame()

			/*
				imageIdx, outdated, err := app.Context().AcquireNextImage()
				orPanic(err)
				if outdated {
					imageIdx, _, err = app.Context().AcquireNextImage()
					orPanic(err)
				}
				_, err = app.Context().PresentImage(imageIdx)
				orPanic(err)
			*/
		}
	}

}
