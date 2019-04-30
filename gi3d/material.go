// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"log"

	"github.com/goki/gi/gi"
)

// Material describes the material properties of a surface (colors, shininess, texture)
// i.e., phong lighting parameters.
// Main color is used for both ambient and diffuse color, and alpha component
// is used for opacity.  The Emissive color is only for glowing objects.
// The Specular color is always white (multiplied by light color).
// Textures are stored on the Scene and accessed by name
type Material struct {
	Color    gi.Color `desc:"main color of surface, used for both ambient and diffuse color in standard Phong model -- alpha component determines transparency -- note that transparent objects require more complex rendering"`
	Emissive gi.Color `desc:"color that surface emits independent of any lighting -- i.e., glow -- can be used for marking lights with an object"`
	Specular gi.Color `desc:"shiny reflective color of surface -- set to white for shiny objects and to Color for non-shiny objects"`
	Shiny    float32  `desc:"specular shininess factor -- how strongly the surface shines back directional light -- this is an exponential factor -- 0 = not at all shiny, and 128 is a typical maximum"`
	Texture  TexName  `desc:"texture to provide color for the surface"`
	TexPtr   *Texture `view:"-" desc:"pointer to texture"`
}

// Defaults sets default surface parameters
func (mt *Material) Defaults() {
	mt.Color.SetUInt8(128, 128, 128, 255)
	mt.Emissive.SetUInt8(0, 0, 0, 0)
	mt.Specular.SetUInt8(255, 255, 255, 255)
	mt.Shiny = 30
}

// IsTransparent returns true if color has alpha < 255
func (mt *Material) IsTransparent() bool {
	return mt.Color.A < 255
}

// NoTexture resets any texture setting that might have been set
func (mt *Material) NoTexture() {
	mt.Texture = ""
	mt.TexPtr = nil
}

// SetTexture sets material to use given texture name (textures are accessed by name on Scene)
// if name is empty, then texture is reset
func (mt *Material) SetTexture(sc *Scene, texName string) error {
	if texName == "" {
		mt.NoTexture()
		return nil
	}
	tex, ok := sc.Textures[texName]
	if !ok {
		err := fmt.Errorf("gi3d.Material in Scene: %s SetTexture name: %s not found in scene", sc.PathUnique(), texName)
		log.Println(err)
		return err
	}
	mt.TexPtr = tex
	return nil
}

// Validate checks that material texture is valid if set
func (mt *Material) Validate(sc *Scene) error {
	if mt.Texture == "" {
		mt.TexPtr = nil
		return nil
	}
	if mt.TexPtr == nil || mt.TexPtr.Name != string(mt.Texture) {
		err := mt.SetTexture(sc, string(mt.Texture))
		if err != nil {
			return err
		}
	}
	return nil
}
