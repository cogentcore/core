// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package video

import (
	"image"
	"image/draw"

	"cogentcore.org/core/system/composer"
	"cogentcore.org/core/system/driver/base"
)

// videoSource implements [composer.Source] for core direct rendering.
type videoSource struct {
	destBBox, srcBBox image.Rectangle
	rotation          float32
	unchanged         bool
	frame             *image.RGBA
}

func (vs *videoSource) Draw(c composer.Composer) {
	cd, ok := c.(*base.ComposerDrawer)
	if !ok {
		return
	}
	cd.Drawer.Scale(vs.destBBox, vs.frame, vs.srcBBox, vs.rotation, draw.Src, vs.unchanged)
}

// RenderSource returns the [composer.Source] for direct rendering.
func (v *Video) RenderSource(op draw.Op) composer.Source {
	frame, unchanged := v.CurrentFrame()
	if frame == nil {
		return nil
	}
	bb, sbb, empty := v.DirectRenderDrawBBoxes(frame.Bounds())
	if empty {
		return nil
	}
	return &videoSource{destBBox: bb, srcBBox: sbb, rotation: v.Rotation, unchanged: unchanged, frame: frame}
}
