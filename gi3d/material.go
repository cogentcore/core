// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/g3n/engine/math32"
	"github.com/goki/gi"
)

// https://learnopengl.com/Lighting/Basic-Lighting
// https://en.wikipedia.org/wiki/Blinn%E2%80%93Phong_shading_model

// todo: add shadows..

// Phong lighting parameters -- all color comes from color of light
// and color of object -- otherwise it is just the shininess of the object
// that needs to be specified.  The specular color is always white multiplied
// by the light color and object color.
type Phong struct {
	Shininess float32 // Specular shininess factor
}

// MatName is a material name -- provides an automatic gui chooser for materials
type MatName string

// Material is the overall interface for all materials.
// The material controls everything about rendering, including the
// order (transparent after solid, etc), and has different shader programs
// for each type of material.
type Material interface {
	// Name returns the material's name -- all materials must have unique names
	// and are stored as central resources in the scene
	Name() string

	// TypeOrder represents the outer-loop material type ordering.
	// It is fixed and determined by the type of material (e.g., transparent
	// comes after opaque)
	TypeOrder() int

	// ItemOrder represents the inner-loop ordering of material items within
	// the type.  For Transparent items, this depends on overall object
	// distances.
	ItemOrder() int

	// Phong returns the phong lighting parameters
	Phong() Phong
}

// Base material type
type MaterialBase struct {
	Nm   string
	Phng Phong
	Pipe gpu.Pipeline // todo: program abstraction, has multiple gpu.Program's
}

// ColorOpaqueVertex is a material with opaque color parameters per vertex.
// All verticies with this type of material, which has no individual material
// parameters at all, can be rendered together at the same time -- this
// material aggregates all of those verticies (subject to other potential
// optimizations about what is rendered in the scene) and does them all at once.
// This uses the standard Phong color model, with color computed in the
// fragment shader (more accurate, more expensive).
type ColorOpaqueVertex struct {
	MaterialBase
}

// ColorTransVertex is a material with transparent color parameters per vertex.
// All verticies with this type of material, which has no individual material
// colors, can be rendered together at the same time -- this
// material aggregates all of those verticies (subject to other potential
// optimizations about what is rendered in the scene) and does them all at once.
// This uses the standard Phong color model, with color computed in the
// fragment shader (more accurate, more expensive).
// Verticies are automatically depth-sorted using GPU-computed depth map.
type ColorTransVertex struct {
	MaterialBase
}

// ColorOpaqueUniform is a material with one set of opaque color parameters
// for entire object.  There is one of these per color, and it renders all verticies
// of that color in one pass.  This uses the standard Phong color model, with
// color computed in the fragment shader (more accurate, more expensive).
type ColorOpaqueUniform struct {
	MaterialBase
	Color math32.Color
}

// ColorTransUniform is a material with one set of transparent color parameters
// for entire object. There is one of these per color per distance, and it renders
// all verticies of that color in one pass.  This uses the standard Phong color model, with
// color computed in the fragment shader (more accurate, more expensive).
type ColorTransUniform struct {
	MaterialBase
	Color math32.Color
}

// Texture is a texture material -- any objects using the same texture can be rendered
// at the same time.  This is a static texture.
type Texture struct {
	MaterialBase
	TextureFile string
}

// TextureGi2D is a dynamic texture material driven by a gi.Viewport2D viewport
// anything rendered to the viewport will be projected onto the surface of any
// object using this texture.
type TextureGi2D struct {
	MaterialBase
	Viewport *gi.Viewport2D
}
