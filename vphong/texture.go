// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"fmt"
	"image"
	"log"

	"github.com/goki/mat32"
	"github.com/goki/vgpu/vgpu"
)

// Texture has texture values: image, tiling
type Texture struct {
	Image  image.Image
	Repeat mat32.Vec2 `desc:"how often to repeat the texture in each direction"`
	Off    mat32.Vec2 `desc:"offset for when to start the texure in each direction"`
}

// AddTexture adds to list of textures
func (ph *Phong) AddTexture(name string, tex *Texture) {
	ph.Textures.Add(name, tex)
}

// AllocTextures allocates vals for textures
func (ph *Phong) AllocTextures() {
	vars := ph.Sys.Vars()
	txset := vars.SetMap[int(TexSet)]
	txset.ConfigVals(ph.Textures.Len())
	for i, kv := range ph.Textures.Order {
		_, img, _ := vars.ValByIdxTry(int(TexSet), "Tex", i)
		img.SetGoImage(kv.Val.Image, vgpu.FlipY)
	}
}

// ConfigTextures configures the rendering for the textures that have been added.
func (ph *Phong) ConfigTextures() {
	vars.BindVarsStart(0)          // only one set of bindings
	vars.BindStatVars(int(TexSet)) // gets images
	vars.BindVarsEnd()
}

// UseNoTexture turns off texture rendering
func (ph *Phong) UseNoTexture() {
	ph.Cur.UseTexture = false
}

// UseTextureIdx selects texture by index for current render step
func (ph *Phong) UseTextureIdx(idx int) error {
	tex := ph.Textures.ValByIdx(idx)
	vars := ph.Sys.Vars()
	ph.Cur.TexIdx = idx // todo: range check
	ph.Cur.UseTexture = true
}

// UseTextureName selects texture by name for current render step
func (ph *Phong) UseTextureName(name string) error {
	idx, ok := ph.Textures.IdxByKey(name)
	if !ok {
		err := fmt.Errorf("vphong:UseTextureName -- name not found: %s", name)
		if vgpu.TheGPU.Debug {
			log.Println(err)
		}
	}
	return ph.UseTextureIdx(idx)
}

// TexPush holds vals for texture push constants
type TexPush struct {
	Repeat mat32.Vec2 `desc:"how often to repeat the texture in each direction"`
	Off    mat32.Vec2 `desc:"offset for when to start the texure in each direction"`
	Idx    int32      `desc:"index"`
}
