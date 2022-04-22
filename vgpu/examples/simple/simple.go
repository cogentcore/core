// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/demos
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License
// and https://bakedbits.dev/posts/vulkan-compute-example/

package main

import (
	"log"
	"runtime"

	"github.com/emer/egpu/egpu"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/xlab/closer"
)

func init() {
	// must lock main thread for gpu!  this also means that vulkan must be used
	// for gogi/oswin eventually if we want gui and compute
	runtime.LockOSThread()
}

func main() {
	egpu.IfPanic(glfw.Init())
	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	egpu.IfPanic(vk.Init())
	defer closer.Close()

	app := NewApplication(true)
	reqDim := app.VulkanSwapchainDimensions()
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(int(reqDim.Width), int(reqDim.Height), "VulkanCube (GLFW)", nil, nil)
	orPanic(err)
	app.windowHandle = window

	// creates a new platform, also initializes Vulkan context in the app
	platform, err := as.NewGPU(app)
	orPanic(err)

	dim := app.Pipeline().SwapchainDimensions()
	log.Printf("Initialized %s with %+v swapchain", app.VulkanAppName(), dim)
}
