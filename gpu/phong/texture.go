// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phong

import (
	"fmt"
	"image"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/gpu"
)

// Texture has texture image and other params.
// Stored as image.RGBA for GPU compatibility.
type Texture struct {
	Image *image.RGBA
}

func NewTexture(img image.Image) *Texture {
	tx := &Texture{}
	return tx.Set(img)
}

// Set sets texture from Go image.
func (tx *Texture) Set(img image.Image) *Texture {
	tx.Image = gpu.ImageToRGBA(img)
	return tx
}

// SetTexture sets given [Texture] to be available for
// [UseTexture] call during render.  If it already exists, it
// is updated, otherwise added.
func (ph *Phong) SetTexture(name string, tx *Texture) {
	ph.Lock()
	defer ph.Unlock()

	tgp := ph.System.Vars().Groups[int(TextureGroup)]
	tvr, _ := tgp.VarByName("TexSampler")
	idx, ok := ph.textures.Map[name]
	if !ok {
		idx = ph.textures.Len()
		ph.textures.Add(name, tx)
		tgp.SetNValues(ph.textures.Len())
	} else {
		ph.textures.Order[idx].Value = tx
	}
	tvv := tvr.Values.Values[idx]
	tvv.SetFromGoImage(tx.Image, 1)
}

// configDummyTextureIfNone configures a dummy texture, if none configured.
func (ph *Phong) configDummyTexture() {
	if ph.textures.Len() > 0 {
		return
	}
	// there must be one texture -- otherwise get gpu error
	tgp := ph.System.Vars().Groups[int(TextureGroup)]
	tgp.SetNValues(1)
	dimg := image.NewRGBA(image.Rectangle{Max: image.Point{2, 2}})
	tvv, _ := tgp.ValueByIndex("TexSampler", 0)
	tvv.SetFromGoImage(dimg, 0)
}

// ResetTextures resets all textures
func (ph *Phong) ResetTextures() {
	ph.Lock()
	defer ph.Unlock()

	ph.textures.Reset()
	tgp := ph.System.Vars().Groups[int(TextureGroup)]
	tgp.SetNValues(1)
}

// UseNoTexture turns off texture rendering
func (ph *Phong) UseNoTexture() {
	ph.UseCurTexture = false
}

// UseTexture selects texture by name for current render step.
func (ph *Phong) UseTexture(name string) error {
	idx, ok := ph.textures.IndexByKeyTry(name)
	if !ok {
		return errors.Log(fmt.Errorf("phong:UseTexture: name not found: %s", name))
	}
	ph.System.Vars().SetCurrentValue(int(TextureGroup), "TexSampler", idx)
	ph.UseCurTexture = true
	return nil
}
