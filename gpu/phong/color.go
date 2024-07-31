// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

import (
	"fmt"
	"image/color"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32"
)

// Colors are the material colors with padding for direct uploading to shader.
type Colors struct {
	// main color of surface, used for both ambient and diffuse color
	// in standard Phong model. Alpha component determines transparency.
	// Note that transparent objects require more complex rendering.
	Color math32.Vector4

	// X = shininess spread factor, Y = shine reflection factor,
	// Z = brightness factor:  shiny = specular shininess factor:
	// how focally the surface shines back directional light, which
	// is an exponential factor, where 0 = very broad diffuse reflection,
	// and higher values (typically max of 128) have a smaller more
	// focal specular reflection.
	// Shine reflect = 1 for full shine white reflection (specular) color,
	// 0 = no shine reflection.
	// Bright = overall multiplier on final computed color value,
	// which can be used to tune the overall brightness of various surfaces
	// relative to each other for a given set of lighting parameters.
	ShinyBright math32.Vector4

	// color that surface emits independent of any lighting,
	// i.e., glow. Can be used for marking lights with an object.
	Emissive math32.Vector4

	// texture repeat and offset factors
	// X,Y = how often to repeat the texture in each direction
	// Z,W = offset for where to start the texture in each direction
	TexRepeatOff math32.Vector4
}

// NewGoColor sets the colors from standard Go colors.
func NewColors(clr, emis color.Color, shiny, reflect, bright float32) *Colors {
	cl := &Colors{}
	cl.SetColors(clr, emis, shiny, reflect, bright)
	cl.TexRepeatOff.Set(1, 1, 0, 0)
	return cl
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
	cl.TexRepeatOff.Set(1, 1, 0, 0)
	return cl
}

// SetTextureRepeat sets how often to repeat the texture in each direction
func (cl *Colors) SetTextureRepeat(repeat math32.Vector2) *Colors {
	cl.TexRepeatOff.X = repeat.X
	cl.TexRepeatOff.Y = repeat.Y
	return cl
}

// SetTextureOffset sets texture start offsets in each direction
func (cl *Colors) SetTextureOffset(offset math32.Vector2) *Colors {
	cl.TexRepeatOff.Z = offset.X
	cl.TexRepeatOff.W = offset.Y
	return cl
}

// AddColor adds to list of colors, which can be used
// for a materials library.
func (ph *Phong) AddColor(name string, clr *Colors) {
	ph.Colors.Add(name, clr)
}

// UseColorIndex selects color by index for current render step.
func (ph *Phong) UseColorIndex(idx int) error {
	if err := ph.Colors.IndexIsValid(idx); err != nil {
		return errors.Log(err)
	}
	ph.Sys.Vars.SetDynamicIndex(int(ColorGroup), "Colors", idx)
	return nil
}

// UseColorName selects color by name for current render step.
func (ph *Phong) UseColorName(name string) error {
	idx, ok := ph.Colors.IndexByKeyTry(name)
	if !ok {
		err := fmt.Errorf("vphong:UseColorName -- name not found: %s", name)
		return errors.Log(err)
	}
	ph.Sys.Vars.SetDynamicIndex(int(ColorGroup), "Colors", idx)
	return nil
}
