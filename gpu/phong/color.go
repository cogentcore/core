// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

import (
	"image/color"

	"cogentcore.org/core/math32"
)

// Colors are the material colors with padding for direct uploading to shader.
type Colors struct {
	// main color of surface, used for both ambient and diffuse color
	// in standard Phong model. Alpha component determines transparency.
	// Note that transparent objects require more complex rendering.
	Color math32.Vector4

	// ShinyBright has shininess and brightness factors:
	//
	// X = shininess spread factor (30 default), which determines
	// 	how focally the surface shines back directional light, which
	// 	is an exponential factor, where 0 = very broad diffuse reflection,
	// 	and higher values (typically max of 128) have a smaller more
	// 	focal specular reflection.
	//
	// Y = shine reflection factor (1 default), which determines how much
	//		of the incident light is reflected back (0 = no shine).
	//
	// Z = brightness factor (1 default), which is an overall multiplier
	// 	on final computed color value, which can be used to tune the
	//		overall brightness of various surfaces relative to each other
	//		for a given set of lighting parameters.
	ShinyBright math32.Vector4

	// color that surface emits independent of any lighting,
	// i.e., glow. Can be used for marking lights with an object.
	Emissive math32.Vector4

	// texture repeat and offset factors.
	// X,Y = how often to repeat the texture in each direction
	// Z,W = offset for where to start the texture in each direction
	TextureRepeatOff math32.Vector4
}

// NewColors returns a new Colors with given values.
// Texture defaults to using full texture with 0 offset.
func NewColors(clr, emis color.Color, shiny, reflect, bright float32) *Colors {
	cl := &Colors{}
	cl.SetColors(clr, emis, shiny, reflect, bright)
	cl.TextureRepeatOff.Set(1, 1, 0, 0)
	return cl
}

func (cl *Colors) Defaults() {
	cl.ShinyBright.X = 30
	cl.ShinyBright.Y = 1
	cl.ShinyBright.Z = 1
}

// SetColors sets the colors from standard Go colors.
func (cl *Colors) SetColors(clr, emis color.Color, shiny, reflect, bright float32) *Colors {
	cl.Color = math32.NewVector4Color(clr).SRGBToLinear()
	cl.Emissive = math32.NewVector4Color(emis).SRGBToLinear()
	cl.ShinyBright.X = shiny
	cl.ShinyBright.Y = reflect
	cl.ShinyBright.Z = bright
	return cl
}

// UseFullTexture sets the texture parameters
// to render the full texture: repeat = 1,1; off = 0,0
func (cl *Colors) UseFullTexture() *Colors {
	cl.TextureRepeatOff.Set(1, 1, 0, 0)
	return cl
}

// SetTextureRepeat sets how often to repeat the texture in each direction
func (cl *Colors) SetTextureRepeat(repeat math32.Vector2) *Colors {
	cl.TextureRepeatOff.X = repeat.X
	cl.TextureRepeatOff.Y = repeat.Y
	return cl
}

// SetTextureOffset sets texture start offsets in each direction
func (cl *Colors) SetTextureOffset(offset math32.Vector2) *Colors {
	cl.TextureRepeatOff.Z = offset.X
	cl.TextureRepeatOff.W = offset.Y
	return cl
}
