// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package world

import (
	"image"
	"log"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/xyz"

	vk "github.com/goki/vulkan"
)

// NoDisplayGPU Initializes the Vulkan GPU (vgpu) and returns that
// and the graphics GPU device, with given name, without connecting
// to the display.
func NoDisplayGPU(nm string) (*gpu.GPU, *gpu.Device, error) {
	if err := gpu.InitNoDisplay(); err != nil {
		return nil, nil, errors.Log(err)
	}
	gp := gpu.NewGPU()
	if err := gp.Config(nm, nil); err != nil {
		log.Println(err)
		return nil, nil, errors.Log(err)
	}
	dev := &gpu.Device{}
	if err := dev.Init(gp, vk.QueueGraphicsBit); err != nil { // todo: add wrapper to vgpu
		return nil, nil, errors.Log(err)
	}
	return gp, dev, nil
}

// NoDisplayScene returns a xyz Scene initialized and ready to use
// in NoGUI offscreen rendering mode, using given GPU and device.
// Must manually call Init3D and Style3D on the Scene prior to
// a RenderOffNode call to grab the image from a specific camera.
func NoDisplayScene(gp *gpu.GPU, dev *gpu.Device) *xyz.Scene {
	sc := xyz.NewScene()
	sc.MultiSample = 4
	sc.Geom.Size = image.Point{1024, 768}
	sc.ConfigFrame(gp, dev)
	return sc
}
