// Adapted from github.com/oliverbestmann/webgpu/wgpuglfw, which is
// licensed under the Apache License, Version 2.0, to use the Cogent Core
// fork of glfw, github.com/cogentcore/glfw4.

//go:build windows

package wgpuglfw

import (
	"unsafe"

	"github.com/cogentcore/glfw4/v3.4/glfw"
	"github.com/oliverbestmann/webgpu/wgpu"
)

/*

#include <windows.h>

*/
import "C"

func GetSurfaceDescriptor(w *glfw.Window) *wgpu.SurfaceDescriptor {
	return &wgpu.SurfaceDescriptor{
		WindowsHWND: &wgpu.SurfaceSourceWindowsHWND{
			Hwnd:      unsafe.Pointer(w.GetWin32Window()),
			Hinstance: unsafe.Pointer(C.GetModuleHandle(nil)),
		},
	}
}
