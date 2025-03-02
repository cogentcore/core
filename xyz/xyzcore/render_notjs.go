// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package xyzcore

import (
	"image"
	"image/draw"

	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/gpudraw"
	"cogentcore.org/core/system/composer"
	"cogentcore.org/core/system/driver/base"
)

// xyzSource implements [composer.Source] for core direct rendering.
type xyzSource struct {
	destBBox, srcBBox image.Rectangle
	texture           *gpu.Texture
}

func (xr *xyzSource) Draw(c composer.Composer) {
	cd, ok := c.(*base.ComposerDrawer)
	if !ok {
		return
	}
	agd, ok := cd.Drawer.(gpudraw.AsGPUDrawer)
	if !ok {
		return
	}
	gdrw := agd.AsGPUDrawer()
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
