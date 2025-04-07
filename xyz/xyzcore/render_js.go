// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package xyzcore

import (
	"image"
	"image/draw"

	"cogentcore.org/core/core"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/phong"
	"cogentcore.org/core/system/composer"
	"github.com/cogentcore/webgpu/wgpu"
)

// xyzSource implements [composer.Source] for core direct rendering.
type xyzSource struct {
	sw *Scene
}

func (xs *xyzSource) Draw(c composer.Composer) {
	cw := c.(*composer.ComposerWeb)

	elem := cw.Element(xs, "canvas")
	cw.SetElementGeom(elem, xs.sw.Geom.ContentBBox.Min, xs.sw.Geom.ContentBBox.Size())
}

// RenderSource returns the [composer.Source] for direct rendering.
func (sw *Scene) RenderSource(op draw.Op) composer.Source {
	if sw.XYZ.Frame == nil || !sw.IsVisible() {
		return nil
	}
	return &xyzSource{sw: sw}
}

// configFrame configures the render frame in a platform-specific manner.
func (sw *Scene) configFrame(sz image.Point) {
	// Even though we are not in [xyzSource.Draw], we can get the Composer
	// and use it to make the canvas element, which we may need before we
	// get to [xyzSource.Draw]. Because we pass sw to ElementContext, we
	// will have the same element as in [xyzSource.Draw], and so no duplicate
	// elements will be created.
	cw := core.TheApp.Window(0).Composer().(*composer.ComposerWeb)
	elem := cw.ElementContext(sw.This, "canvas")

	wsurf := gpu.Instance().CreateSurface(&wgpu.SurfaceDescriptor{Canvas: elem})
	gp := gpu.NewGPU(nil)
	sc := sw.XYZ
	sc.Frame = gpu.NewSurface(gp, wsurf, sz, 4, gpu.Depth32)
	sc.Phong = phong.NewPhong(gp, sc.Frame)
	sc.ConfigNewPhong()
}
