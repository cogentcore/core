// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

import (
	"fmt"
	"image"
	"log"

	"cogentcore.org/core/gpu"
)

// Texture has texture image and other params.
// Stored as image.RGBA for GPU compatibility.
type Texture struct {
	Image *image.RGBA
}

func NewTexture(img image.Image) *Texture {
	tx := &Texture{}
	tx.Set(img)
	return tx
}

// Set sets values
func (tx *Texture) Set(img image.Image) {
	tx.Image = gpu.ImageToRGBA(img)
}

// AddTexture adds to list of textures
func (ph *Phong) AddTexture(name string, tex *Texture) {
	ph.Textures.Add(name, tex)
}

// DeleteTexture deletes texture with name
func (ph *Phong) DeleteTexture(name string) {
	ph.Textures.DeleteKey(name)
}

// configDummyTexture configures a dummy texture (if none configured)
func (ph *Phong) configDummyTexture() {
	// there must be one texture -- otherwise Mac Molten triggers an error
	tgp := ph.Sys.Vars.Groups[int(TextureGroup)]
	tgp.SetNValues(1)
	dimg := image.NewRGBA(image.Rectangle{Max: image.Point{2, 2}})
	img := tgp.ValueByIndex("TexSampler", 0)
	img.SetFromGoImage(dimg, 0, gpu.NoFlipY)
}

// ConfigTextures configures vals for textures -- this is the first
// of two passes -- call Phong.Sys.Config after everything is config'd.
// This automatically allocates images by size so everything fits
// within the MaxTexturesPerStage limit, as texture arrays.
func (ph *Phong) ConfigTextures() {
	sy := ph.Sys
	ntx := ph.Textures.Len()
	if ntx == 0 {
		ph.configDummyTexture()
		return
	}
	tvr := sy.Vars.VarByName(int(TextureGroup), "TexSampler")
	tvr.SetNValues(&sy.Device, ntx)
	for i, kv := range ph.Textures.Order {
		tvv := tvr.Values.Values[i]
		tvv.SetFromGoImage(kv.Value.Image, 1, gpu.NoFlipY)
	}
}

// ResetTextures resets all textures
func (ph *Phong) ResetTextures() {
	sy := ph.Sys
	tgp := sy.Vars.Groups[int(TextureGroup)]
	tgp.Release()
	ph.Textures.Reset()
}

// UseNoTexture turns off texture rendering
func (ph *Phong) UseNoTexture() {
	ph.UseTexture = false
}

// UseTextureIndex selects texture by index for current render step
func (ph *Phong) UseTextureIndex(idx int) error {
	ph.Sys.Vars.SetCurrentValue(int(TextureGroup), "TexSampler", idx)
	ph.UseTexture = true
	return nil
}

// UseTextureName selects texture by name for current render step
func (ph *Phong) UseTextureName(name string) error {
	idx, ok := ph.Textures.IndexByKeyTry(name)
	if !ok {
		err := fmt.Errorf("phong:UseTextureName -- name not found: %s", name)
		if gpu.Debug {
			log.Println(err)
		}
	}
	return ph.UseTextureIndex(idx)
}

// UpdateTextureIndex updates texture by index -- call this when
// the underlying image changes.  Assumes the size remains the same.
// Must Sync for the changes to take effect.
func (ph *Phong) UpdateTextureIndex(idx int) error {
	sy := ph.Sys
	ph.Lock()
	defer ph.Unlock()
	if idx >= ph.Textures.Len() {
		return nil
	}
	tx := ph.Textures.Order[idx].Value
	tvl := sy.Vars.ValueByIndex(int(TextureGroup), "TexSampler", idx)
	tvl.SetFromGoImage(tx.Image, 1, gpu.FlipY)
	return nil
}

// UpdateTextureName updates texture by name
func (ph *Phong) UpdateTextureName(name string) error {
	idx, ok := ph.Textures.IndexByKeyTry(name)
	if !ok {
		err := fmt.Errorf("vphong:UpdateTextureName -- name not found: %s", name)
		if gpu.Debug {
			log.Println(err)
		}
	}
	return ph.UpdateTextureIndex(idx)
}