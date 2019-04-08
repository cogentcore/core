// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import "github.com/g3n/engine/math32"

// Light represents a light that illuminates a scene
// these are stored on the overall scene object and not within the graph
type Light interface {
}

type LightBase struct {
	On      bool
	Clr     math32.Color
	Intense float32
	Stance  Stance // todo: position, orientation -- for all objects
}

// AmbientLight provides diffuse uniform lighting -- there can only be one of these
type AmbientLight struct {
	LightBase
}

// PointLight is an omnidirectional light with a position
type PointLight struct {
	LightBase
}

// DirLight is a positionless, directional light (position is ignored)
type DirLight struct {
	LightBase
}

// Spotlight is a light with a position and direction
type SpotLight struct {
	LightBase
}
