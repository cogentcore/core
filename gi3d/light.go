// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/mat32"
)

// Light represents a light that illuminates a scene
// these are stored on the overall scene object and not within the graph
type Light interface {
}

// Lights is the container (map of names) for the lights
type Lights map[string]Light

// LightBase provides the base implementation for Light interface
type LightBase struct {
	On      bool
	Color   gi.Color
	Intense float32
}

// AmbientLight provides diffuse uniform lighting -- there can only be one of these
type AmbientLight struct {
	LightBase
}

// DirLight is a positionless, directional light (no position)
type DirLight struct {
	LightBase
	Pose Pose // position and orientation (position is ignored)
}

// PointLight is an omnidirectional light with a position
// and associated decay factors
type PointLight struct {
	LightBase
	Pos            mat32.Vector3 // position of light
	LinearDecay    float32       // Distance linear decay factor
	QuadraticDecay float32       // Distance quadratic decay factor
}

// Spotlight is a light with a position and direction
// and associated decay factors and angles
type SpotLight struct {
	LightBase
	Pose           Pose    // position and orientation
	LinearDecay    float32 // Distance linear decay factor
	QuadraticDecay float32 // Distance quadratic decay factor
	AngularDecay   float32 // Angular decay factor
	CutoffAngle    float32 // Cut off angle (in radians?)
}
