// Adapted from github.com/oliverbestmann/webgpu/wgpuglfw, which is
// licensed under the Apache License, Version 2.0, to use the Cogent Core
// fork of glfw, github.com/cogentcore/glfw4.

//go:build (linux || freebsd || netbsd || openbsd) && !android && ((!x11 && !wayland) || (x11 && wayland))

package wgpuglfw

import (
	"unsafe"

	"github.com/cogentcore/glfw4/v3.4/glfw"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func GetSurfaceDescriptor(w *glfw.Window) *wgpu.SurfaceDescriptor {
	switch glfw.GetPlatform() {
	case glfw.PlatformX11:
		return &wgpu.SurfaceDescriptor{
			XlibWindow: &wgpu.SurfaceSourceXlibWindow{
				Display: unsafe.Pointer(glfw.GetX11Display()),
				Window:  uint32(w.GetX11Window()),
			},
		}

	case glfw.PlatformWayland:
		return &wgpu.SurfaceDescriptor{
			WaylandSurface: &wgpu.SurfaceSourceWaylandSurface{
				Display: unsafe.Pointer(glfw.GetWaylandDisplay()),
				Surface: unsafe.Pointer(w.GetWaylandWindow()),
			},
		}

	default:
		panic("unsupported glfw platform")
	}
}
