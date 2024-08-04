// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package gpu

import (
	"syscall/js"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/system/driver/web/jsfs"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// GLFWCreateWindow is a helper function intended only for use in simple examples that makes a
// new window with glfw on platforms that support it and is largely a no-op on other platforms.
func GLFWCreateWindow(gp *GPU, width, height int, title string, resize *func(width, height int)) (surface *wgpu.Surface, terminate func(), pollEvents func() bool, err error) {
	errors.Log1(jsfs.Config(js.Global().Get("fs"))) // needed for printing etc to work
	surface = gp.Instance.CreateSurface(&wgpu.SurfaceDescriptor{})
	terminate = func() {}
	pollEvents = func() bool { return true }
	return
}
