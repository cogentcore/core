// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/g3n/engine/math32"
	"github.com/goki/gi"
	"github.com/goki/gi/oswin/gpu"
)

// https://learnopengl.com/Lighting/Basic-Lighting
// https://en.wikipedia.org/wiki/Blinn%E2%80%93Phong_shading_model

// todo: add shadows..

// Surface describes the properties of a surface (colors, shininess)
// i.e., phong lighting parameters.
// Main color is used for both ambient and diffuse color, and alpha component
// is used for opacity.  The Emissive color is only for glowing objects.
// The Specular color is always white (multiplied by light color).
type Surface struct {
	Color     gi.Color `desc:"main color of surface, used for both ambient and diffuse color in standard Phong model -- alpha component determines transparency -- note that transparent objects require more complex rendering"`
	Emissive  gi.Color `desc:"color that surface emits independent of any lighting -- i.e., glow -- can be used for marking lights with an object"`
	Shininess float32  `desc:"specular shininess factor -- how strongly the surface shines back directional light -- this is an exponential factor -- 0 = not at all shiny, and 128 is a typical maximum"`
}

// Defaults sets default surface parameters
func (sf *Surface) Defaults() {
	sf.Color.Set(128, 128, 128, 255)
	sf.Emissive.Set(0, 0, 0, 0)
	sf.Shininess = 1
}

// Transparent returns true if color has alpha < 255
func (sf *Surface) Transparent() bool {
	return sf.Color.A < 255
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
}

// Design note: all vertex data must be in ONE gpu.VectorsBuffer, and the cost of
// consolidating a bunch of vector data dynamically on the CPU is going to be
// way higher than buffer switching on the GPU and re-running the same program
// so basically the gpu.VectorsBuffer must be assembled at the Object level
// and, yeah, each object rendered separately -- no aggragation is sensible.

// Base material type
type MaterialBase struct {
	Nm   string
	Phng Phong
	Pipe gpu.Pipeline // todo: program abstraction, has multiple gpu.Program's
}

func (mb *MaterialBase) Name() string {
	return mb.Nm
}

func (mb *MaterialBase) TypeOrder() int {
	return 0
}

func (mb *MaterialBase) ItemOrder() int {
	return 0
}

func (mb *MaterialBase) Phong() Phong {
	return mb.Phng
}

func (mb *MaterialBase) SetShininess(shininess float32) {
	mb.Phng.Shininess = shininess
}

// ColorOpaqueVertex is a material with opaque color parameters per vertex.
// This uses the standard Phong color model, with color computed in the
// fragment shader (more accurate, more expensive).
type ColorOpaqueVertex struct {
	MaterialBase
}

// ColorTransVertex is a material with transparent color parameters per vertex.
// This uses the standard Phong color model, with color computed in the
// fragment shader (more accurate, more expensive).
// Verticies are automatically depth-sorted using GPU-computed depth map.
type ColorTransVertex struct {
	MaterialBase
}

// ColorOpaqueUniform is a material with one set of opaque color parameters
// for entire object.  There is one of these per color.
// This uses the standard Phong color model, with color computed in the
// fragment shader (more accurate, more expensive).
type ColorOpaqueUniform struct {
	MaterialBase
	Color math32.Color
}

// ColorTransUniform is a material with one set of transparent color parameters
// for entire object. There is one of these per color.
// This uses the standard Phong color model, with color computed in the
// fragment shader (more accurate, more expensive).
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
