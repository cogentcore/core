// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package world

import (
	"image"

	"cogentcore.org/core/math32"
)

// Camera defines the properties of a camera needed for rendering from a node.
type Camera struct {

	// size of image to record
	Size image.Point

	// field of view in degrees
	FOV float32

	// near plane z coordinate
	Near float32 `default:"0.01"`

	// far plane z coordinate
	Far float32 `default:"1000"`

	// maximum distance for depth maps. Anything above is 1.
	// This is independent of Near / Far rendering (though must be < Far)
	// and is for normalized depth maps.
	MaxD float32 `default:"20"`

	// use the natural log of 1 + depth for normalized depth values in display etc.
	LogD bool `default:"true"`

	// number of multi-samples to use for antialising -- 4 is best and default.
	MSample int `default:"4"`

	// up direction for camera. Defaults to positive Y axis,
	// and is reset by call to LookAt method.
	UpDir math32.Vector3
}

func (cm *Camera) Defaults() {
	cm.Size = image.Point{320, 180}
	cm.FOV = 30
	cm.Near = .01
	cm.Far = 1000
	cm.MaxD = 20
	cm.LogD = true
	cm.MSample = 4
	cm.UpDir = math32.Vec3(0, 1, 0)
}
