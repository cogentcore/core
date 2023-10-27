// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"image/color"
	"log"

	"goki.dev/colors"
	"goki.dev/laser"
	"goki.dev/mat32/v2"
)

// Tiling are the texture tiling parameters
type Tiling struct {

	// how often to repeat the texture in each direction
	Repeat mat32.Vec2

	// offset for when to start the texure in each direction
	Off mat32.Vec2
}

// Defaults sets default tiling params if not yet initialized
func (tl *Tiling) Defaults() {
	if tl.Repeat.IsNil() {
		tl.Repeat.Set(1, 1)
	}
}

// Material describes the material properties of a surface (colors, shininess, texture)
// i.e., phong lighting parameters.
// Main color is used for both ambient and diffuse color, and alpha component
// is used for opacity.  The Emissive color is only for glowing objects.
// The Specular color is always white (multiplied by light color).
// Textures are stored on the Scene and accessed by name
type Material struct { //gti:add -setters

	// prop: color = main color of surface, used for both ambient and diffuse color in standard Phong model -- alpha component determines transparency -- note that transparent objects require more complex rendering
	Color color.RGBA

	// prop: emissive = color that surface emits independent of any lighting -- i.e., glow -- can be used for marking lights with an object
	Emissive color.RGBA

	// prop: shiny = specular shininess factor -- how focally vs. broad the surface shines back directional light -- this is an exponential factor, with 0 = very broad diffuse reflection, and higher values (typically max of 128 or so but can go higher) having a smaller more focal specular reflection.  Also set Reflective factor to change overall shininess effect.
	Shiny float32

	// prop: reflective = specular reflectiveness factor -- how much it shines back directional light.  The specular reflection color is always white * the incoming light.
	Reflective float32

	// prop: bright = overall multiplier on final computed color value -- can be used to tune the overall brightness of various surfaces relative to each other for a given set of lighting parameters
	Bright float32

	// prop: texture = texture to provide color for the surface
	Texture TexName `set:"-"`

	// texture tiling parameters -- repeat and offset
	Tiling Tiling `view:"inline" viewif:"Texture!=''"`

	// prop: cull-back = cull the back-facing surfaces
	CullBack bool

	// prop: cull-front = cull the front-facing surfaces
	CullFront bool

	// pointer to texture
	TexPtr Texture `set:"-" view:"-"`
}

// Defaults sets default surface parameters
func (mt *Material) Defaults() {
	mt.Color = colors.FromRGB(128, 128, 128)
	mt.Emissive = color.RGBA{0, 0, 0, 0}
	mt.Shiny = 30
	mt.Reflective = 1
	mt.Bright = 1
	mt.Tiling.Defaults()
	mt.CullBack = true
}

// Disconnect resets pointers etc
func (mt *Material) Disconnect() {
	mt.TexPtr = nil
}

func (mt Material) String() string {
	return laser.StringJSON(mt)
}

// IsTransparent returns true if texture says it is, or if color has alpha < 255
func (mt *Material) IsTransparent() bool {
	if mt.TexPtr != nil {
		return mt.TexPtr.IsTransparent()
	}
	return mt.Color.A < 255
}

// NoTexture resets any texture setting that might have been set
func (mt *Material) NoTexture() {
	mt.Texture = ""
	mt.TexPtr = nil
}

// SetTextureName sets material to use given texture name
// (textures are accessed by name on Scene).
// If name is empty, then texture is reset
func (mt *Material) SetTextureName(sc *Scene, texName string) error {
	if texName == "" {
		mt.NoTexture()
		return nil
	}
	tx, err := sc.TextureByNameTry(texName)
	if err != nil {
		log.Println(err)
		return err
	}
	mt.Texture = TexName(texName)
	mt.TexPtr = tx
	return nil
}

// SetTexture sets material to use given texture
func (mt *Material) SetTexture(tex Texture) *Material {
	mt.TexPtr = tex
	if mt.TexPtr != nil {
		mt.Texture = TexName(mt.TexPtr.Name())
	} else {
		mt.Texture = ""
	}
	return mt
}

// Validate does overall material validation, including checking that material
// texture is valid if set
func (mt *Material) Validate(sc *Scene) error {
	if mt.Bright == 0 {
		mt.Bright = 1
	}
	mt.Tiling.Defaults()
	if mt.Texture == "" {
		mt.TexPtr = nil
	} else if mt.TexPtr == nil || mt.TexPtr.Name() != string(mt.Texture) {
		err := mt.SetTextureName(sc, string(mt.Texture))
		if err != nil {
			return err
		}
	}
	return nil
}

func (mt *Material) Render(sc *Scene) {
	sc.Phong.UseColor(mt.Color, mt.Emissive, mt.Shiny, mt.Reflective, mt.Bright)
	sc.Phong.UseTexturePars(mt.Tiling.Repeat, mt.Tiling.Off)
	if mt.Texture != "" {
		sc.Phong.UseTextureName(string(mt.Texture))
	} else {
		sc.Phong.UseNoTexture()
	}
}
