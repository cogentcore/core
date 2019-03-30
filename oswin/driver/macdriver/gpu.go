// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin
// +build 386 amd64
// +build !ios
// +build !3d

package macdriver

import (
	"github.com/goki/gi/oswin/gpu"
	"github.com/vulkan-go/vulkan"
)

type gpuImpl struct {
	gpu.GPUBase
}

var theGPU = &gpuImpl{}

func initGPU() error {
	gp := theGPU

	vulkan.SetDefaultGetInstanceProcAddr()
	vulkan.Init()

	gp.SetReqDeviceExts([]string{
		"VK_KHR_swapchain",
		"VK_MVK_macos_surface",
	})
	// todo: SetReqInstanceExts

	gp.SetReqValidationLayers([]string{
		"VK_LAYER_LUNARG_standard_validation",
		// "VK_LAYER_GOOGLE_threading",
		// "VK_LAYER_LUNARG_parameter_validation",
		// "VK_LAYER_LUNARG_object_tracker",
		// "VK_LAYER_LUNARG_core_validation",
		// "VK_LAYER_LUNARG_api_dump",
		// "VK_LAYER_LUNARG_swapchain",
		// "VK_LAYER_GOOGLE_unique_objects",
	})

	return gp.InitBase()
}
