// Adapted from github.com/oliverbestmann/webgpu/wgpuglfw, which is
// licensed under the Apache License, Version 2.0, to use the Cogent Core
// fork of glfw, github.com/cogentcore/glfw4.

//go:build darwin

package wgpuglfw

import (
	"unsafe"

	"github.com/cogentcore/glfw4/v3.4/glfw"
	"github.com/oliverbestmann/webgpu/wgpu"
)

/*

#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework QuartzCore

#import <Cocoa/Cocoa.h>
#import <QuartzCore/CAMetalLayer.h>

CFTypeRef metalLayerFromNSWindow(CFTypeRef nsWindowRef) {
	NSWindow *ns_window = (__bridge NSWindow *)nsWindowRef;
	id metal_layer = NULL;
	[ns_window.contentView setWantsLayer:YES];
	metal_layer = [CAMetalLayer layer];
	[ns_window.contentView setLayer:metal_layer];
	return metal_layer;
}

*/
import "C"

func GetSurfaceDescriptor(w *glfw.Window) *wgpu.SurfaceDescriptor {
	return &wgpu.SurfaceDescriptor{
		MetalLayer: &wgpu.SurfaceSourceMetalLayer{
			Layer: unsafe.Pointer(C.metalLayerFromNSWindow((C.CFTypeRef)(unsafe.Pointer(w.GetCocoaWindow())))),
		},
	}
}
