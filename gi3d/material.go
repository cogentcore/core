// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"log"

	"github.com/goki/gi/gist"
	"github.com/goki/mat32"
)

// Tiling are the texture tiling parameters
type Tiling struct {
	Repeat mat32.Vec2 `desc:"how often to repeat the texture in each direction"`
	Off    mat32.Vec2 `desc:"offset for when to start the texure in each direction"`
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
type Material struct {
	Color     gist.Color `xml:"color" desc:"prop: color = main color of surface, used for both ambient and diffuse color in standard Phong model -- alpha component determines transparency -- note that transparent objects require more complex rendering"`
	Emissive  gist.Color `xml:"emissive" desc:"prop: emissive = color that surface emits independent of any lighting -- i.e., glow -- can be used for marking lights with an object"`
	Specular  gist.Color `xml:"specular" desc:"prop: specular = shiny reflective color of surface -- set to white for shiny objects and to Color for non-shiny objects"`
	Shiny     float32    `xml:"shiny" desc:"prop: shiny = specular shininess factor -- how focally the surface shines back directional light -- this is an exponential factor, with 0 = very broad diffuse reflection, and higher values (typically max of 128 or so but can go higher) having a smaller more focal specular reflection.  Also set Specular color to affect overall shininess effect."`
	Bright    float32    `xml:"bright" desc:"prop: bright = overall multiplier on final computed color value -- can be used to tune the overall brightness of various surfaces relative to each other for a given set of lighting parameters"`
	Texture   TexName    `xml:"texture" desc:"prop: texture = texture to provide color for the surface"`
	Tiling    Tiling     `view:"inline" viewif:"Texture!=''" desc:"texture tiling parameters -- repeat and offset"`
	CullBack  bool       `xml:"cull-back" desc:"prop: cull-back = cull the back-facing surfaces"`
	CullFront bool       `xml:"cull-front" desc:"prop: cull-front = cull the front-facing surfaces"`
	TexPtr    Texture    `view:"-" desc:"pointer to texture"`
}

// Defaults sets default surface parameters
func (mt *Material) Defaults() {
	mt.Color.SetUInt8(128, 128, 128, 255)
	mt.Emissive.SetUInt8(0, 0, 0, 0)
	mt.Specular.SetUInt8(255, 255, 255, 255)
	mt.Shiny = 30
	mt.Bright = 1
	mt.Tiling.Defaults()
	mt.CullBack = true
}

// Disconnect resets pointers etc
func (mt *Material) Disconnect() {
	mt.TexPtr = nil
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
	tx, ok := sc.Textures[texName]
	if !ok {
		err := fmt.Errorf("gi3d.Material in Scene: %s SetTexture name: %s not found in scene", sc.Path(), texName)
		log.Println(err)
		return err
	}
	mt.Texture = TexName(texName)
	mt.TexPtr = tx
	return nil
}

// SetTexture sets material to use given texture
func (mt *Material) SetTexture(sc *Scene, tex Texture) {
	mt.TexPtr = tex
	if mt.TexPtr != nil {
		mt.Texture = TexName(mt.TexPtr.Name())
	} else {
		mt.Texture = ""
	}
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
