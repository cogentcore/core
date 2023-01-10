// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (linux && cgo) || (darwin && cgo) || (freebsd && cgo)

package vkinit

// #cgo LDFLAGS: -ldl
// #include <stdlib.h>
// #include <dlfcn.h>
import "C"
import (
	"fmt"
	"unsafe"

	vk "github.com/goki/vulkan"
)

func LoadVulkan() error {
	clibnm := C.CString(DlName)
	defer C.free(unsafe.Pointer(clibnm))
	handle := C.dlopen(clibnm, C.RTLD_LAZY)
	if handle == nil {
		return fmt.Errorf("Vulkan library named: %s not found!\n", DlName)
	}
	cpAddr := C.CString("vkGetInstanceProcAddr")
	defer C.free(unsafe.Pointer(cpAddr))
	pAddr := C.dlsym(handle, cpAddr)
	if pAddr == nil {
		return fmt.Errorf("Vulkan instance proc addr not found!\n")
	}
	vk.SetGetInstanceProcAddr(pAddr)
	return vk.Init()
}
