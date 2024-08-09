// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package gpu

import (
	"image"
	"syscall/js"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/system/driver/web/jsfs"
	"github.com/cogentcore/webgpu/wgpu"
)

// GLFWCreateWindow is a helper function intended only for use in simple examples that makes a
// new window with glfw on platforms that support it and is largely a no-op on other platforms.
func GLFWCreateWindow(gp *GPU, size image.Point, title string, resize *func(size image.Point)) (surface *wgpu.Surface, terminate func(), pollEvents func() bool, actualSize image.Point, err error) {
	errors.Log1(jsfs.Config(js.Global().Get("fs"))) // needed for printing etc to work
	surface = gp.Instance.CreateSurface(&wgpu.SurfaceDescriptor{})
	terminate = func() {}
	pollEvents = func() bool { return true }
	vv := js.Global().Get("visualViewport")
	getSize := func() image.Point {
		w, h := vv.Get("width").Int(), vv.Get("height").Int()
		canvas := js.Global().Get("document").Call("querySelector", "canvas")
		canvas.Set("width", w)
		canvas.Set("height", h)
		return image.Point{w, h}
	}
	vv.Call("addEventListener", "resize", js.FuncOf(func(this js.Value, args []js.Value) any {
		(*resize)(getSize())
		return nil
	}))
	actualSize = getSize()
	return
}
