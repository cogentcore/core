// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !offscreen && ((darwin && !ios) || windows || (linux && !android) || dragonfly || openbsd)

package gpu

import (
	"image"

	"cogentcore.org/core/base/errors"
	"github.com/cogentcore/webgpu/wgpu"
	"github.com/cogentcore/webgpu/wgpuglfw"
	"github.com/go-gl/glfw/v3.3/glfw"
)

// note: this file contains the glfw dependencies, for desktop platform builds
// other platforms (mobile, web) need to provide their own Init() and Terminate()
// methods.

// Init initializes WebGPU system for Display-enabled use, using glfw.
// Must call before doing any vgpu stuff.
// Calls glfw.Init and sets the Vulkan instance proc addr and calls Init.
// IMPORTANT: must be called on the main initial thread!
func Init() error {
	err := glfw.Init()
	if err != nil {
		return errors.Log(err)
	}
	// vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	// return errors.Log(vk.Init())
	return nil
}

// Terminate shuts down the WebGPU system -- call as last thing before quitting.
// IMPORTANT: must be called on the main initial thread!
func Terminate() {
	glfw.Terminate()
}

// GLFWCreateWindow is a helper function intended only for use in simple examples that makes a
// new window with glfw on platforms that support it and is largely a no-op on other platforms.
func GLFWCreateWindow(size image.Point, title string, resize *func(size image.Point)) (surface *wgpu.Surface, terminate func(), pollEvents func() bool, actualSize image.Point, err error) {
	if err = Init(); err != nil {
		return
	}
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(size.X, size.Y, title, nil, nil)
	if err != nil {
		return
	}
	inst := Instance()
	surface = inst.CreateSurface(wgpuglfw.GetSurfaceDescriptor(window))
	terminate = func() {
		window.Destroy()
		Terminate()
	}
	pollEvents = func() bool {
		if window.ShouldClose() {
			return false
		}
		glfw.PollEvents()
		return true
	}
	window.SetSizeCallback(func(w *glfw.Window, width, height int) {
		if resize != nil {
			(*resize)(image.Point{width, height})
		}
	})
	actualSize = size
	return
}
