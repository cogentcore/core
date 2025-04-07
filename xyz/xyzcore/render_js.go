// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package xyzcore

import (
	"image"
	"image/draw"

	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/phong"
	"cogentcore.org/core/system/composer"
	"github.com/cogentcore/webgpu/wgpu"
)

// xyzSource implements [composer.Source] for core direct rendering.
type xyzSource struct {
	sc *Scene
}

func (xr *xyzSource) Draw(c composer.Composer) {
	// nop?
}

// RenderSource returns the [composer.Source] for direct rendering.
func (sw *Scene) RenderSource(op draw.Op) composer.Source {
	if sw.XYZ.Frame == nil || !sw.IsVisible() {
		return nil
	}
	return &xyzSource{sc: sw}
}

// configFrame configures the render frame in a platform-specific manner.
func (sw *Scene) configFrame(sz image.Point) {
	wsurf := gpu.Instance().CreateSurface(&wgpu.SurfaceDescriptor{})
	gp := gpu.NewGPU(nil)
	sc := sw.XYZ
	sc.Frame = gpu.NewSurface(gp, wsurf, sz, 4, gpu.Depth32)
	sc.Phong = phong.NewPhong(gp, sc.Frame)
	sc.ConfigNewPhong()
}
