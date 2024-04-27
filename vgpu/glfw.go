// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !offscreen && ((darwin && !ios) || windows || (linux && !android) || dragonfly || openbsd)

package vgpu

import (
	"cogentcore.org/core/base/errors"
	"github.com/go-gl/glfw/v3.3/glfw"
	vk "github.com/goki/vulkan"
)

// note: this file contains the glfw dependencies, for desktop platform builds
// other platforms (mobile, web) need to provide their own Init() and Terminate()
// methods.

// Init initializes vulkan system for Display-enabled use, using glfw.
// Must call before doing any vgpu stuff.
// Calls glfw.Init and sets the Vulkan instance proc addr and calls Init.
// IMPORTANT: must be called on the main initial thread!
func Init() error {
	err := glfw.Init()
	if err != nil {
		return errors.Log(err)
	}
	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	return errors.Log(vk.Init())
}

// Terminate shuts down the vulkan system -- call as last thing before quitting.
// IMPORTANT: must be called on the main initial thread!
func Terminate() {
	glfw.Terminate()
}
