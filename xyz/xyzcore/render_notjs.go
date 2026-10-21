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
	sw                *Scene
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
	if xr.sw.XYZ == nil {
		return
	}
	xr.sw.XYZ.Lock()
	gdrw.UseTexture(xr.texture)
	gdrw.CopyUsed(xr.destBBox.Min, xr.srcBBox, draw.Src, false)
	xr.sw.XYZ.Unlock()
}

// RenderSource returns the [composer.Source] for direct rendering.
func (sw *Scene) RenderSource(op draw.Op) composer.Source {
	if sw.XYZ == nil || sw.XYZ.Frame == nil || !sw.IsVisible() {
		return nil
	}
	sw.XYZ.Lock()
	defer sw.XYZ.Unlock()
	rt := sw.XYZ.Frame.(*gpu.RenderTexture)
	tex := rt.CurrentFrame()
	// The frame has the size of the whole scene, of which only the visible
	// part, Geom.ContentBBox, is drawn, when it is scrolled out of its parent.
	visible := sw.Geom.ContentBBox
	sbb := visible.Sub(sw.frameBBox().Min).Intersect(tex.Format.Bounds())
	if sbb.Empty() {
		return nil
	}
	bb := image.Rectangle{Max: sbb.Size()}.Add(visible.Min.Add(sw.Scene.SceneGeom.Pos))
	return &xyzSource{destBBox: bb, srcBBox: sbb, texture: tex, sw: sw}
}

// configFrame configures the render frame in a platform-specific manner.
func (sw *Scene) configFrame(sz image.Point) {
	win := sw.WidgetBase.Scene.Events.RenderWindow()
	if win == nil {
		return
	}
	system.TheApp.RunOnMain(func() {
		gdrw := getGPUDrawer(win.SystemWindow.Composer())
		if gdrw == nil {
			return
		}
		sf, ok := gdrw.Renderer().(*gpu.Surface)
		if !ok {
			core.ErrorSnackbar(sw, errors.New("WebGPU not available for 3D rendering"))
			return
		}
		sw.XYZ.ConfigOffscreenFromSurface(sf) // does a full build if Frame == nil, else just new size
	})
}

// destroyFrame releases platform-specific resources of the render frame.
func (sw *Scene) destroyFrame() {}
