// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package world

import (
	"image"

	"cogentcore.org/core/gpu"
	"cogentcore.org/core/xyz"
)

// NoDisplayScene returns a xyz Scene initialized and ready to use
// in NoGUI offscreen rendering mode, using given GPU and device.
// Must manually call Init3D and Style3D on the Scene prior to
// a RenderFromNode call to grab the image from a specific camera.
func NoDisplayScene(gp *gpu.GPU, dev *gpu.Device) *xyz.Scene {
	sc := xyz.NewScene()
	sc.MultiSample = 4
	sc.Geom.Size = image.Point{1024, 768}
	sc.ConfigOffscreen(gp, dev)
	return sc
}
