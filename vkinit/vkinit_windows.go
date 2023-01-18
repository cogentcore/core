// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vkinit

import (
	"fmt"
	"syscall"
	"unsafe"

	vk "github.com/goki/vulkan"
)

var DlName = "vulkan-1.dll"

func LoadVulkan() error {
	handle, err := syscall.LoadLibrary(DlName)
	if err != nil {
		return fmt.Errorf("Vulkan library named: %s not found!\n", DlName)
	}
	pAddr, err := syscall.GetProcAddress(handle, "vkGetInstanceProcAddr")
	if err != nil {
		return fmt.Errorf("Vulkan instance proc addr not found!\n")
	}
	vk.SetGetInstanceProcAddr(unsafe.Pointer(pAddr))
	return vk.Init()
}
