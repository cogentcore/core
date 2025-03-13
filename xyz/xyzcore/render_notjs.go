// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package xyzcore

import (
	"errors"
	"image"
	"image/draw"

	"cogentcore.org/core/core"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/gpudraw"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/composer"
)

// xyzSource implements [composer.Source] for core direct rendering.
type xyzSource struct {
	destBBox, srcBBox image.Rectangle
	texture           *gpu.Texture
}

func getGPUDrawer(c composer.Composer) *gpudraw.Drawer {
	cd := c.(*composer.ComposerDrawer)
	agd, ok := cd.Drawer.(*gpudraw.Drawer)
	if !ok {
		return nil
	}
	return agd.AsGPUDrawer()
}

func (xr *xyzSource) Draw(c composer.Composer) {
	gdrw := getGPUDrawer(c)
	if gdrw == nil {
		return
	}
	gdrw.UseTexture(xr.texture)
	gdrw.CopyUsed(xr.destBBox.Min, xr.srcBBox, draw.Src, false)
}

// RenderSource returns the [composer.Source] for direct rendering.
func (sw *Scene) RenderSource(op draw.Op) composer.Source {
	if sw.XYZ.Frame == nil || !sw.IsVisible() {
		return nil
	}
	tex, _ := sw.XYZ.Frame.GetCurrentTextureObject()
	bb, sbb, empty := sw.DirectRenderDrawBBoxes(tex.Format.Bounds())
	if empty {
		return nil
	}
	return &xyzSource{destBBox: bb, srcBBox: sbb, texture: tex}
}

// configFrame configures the render frame in a platform-specific manner.
func (sw *Scene) configFrame() {
	win := sw.WidgetBase.Scene.Events.RenderWindow()
	if win == nil {
		return
	}
	gdrw := getGPUDrawer(win.SystemWindow.Composer())
	if gdrw == nil {
		return
	}
	system.TheApp.RunOnMain(func() {
		sf, ok := gdrw.Renderer().(*gpu.Surface)
		if !ok {
			core.ErrorSnackbar(sw, errors.New("WebGPU not available for 3D rendering"))
			return
		}
		sw.XYZ.ConfigFrameFromSurface(sf) // does a full build if Frame == nil, else just new size
	})
}
