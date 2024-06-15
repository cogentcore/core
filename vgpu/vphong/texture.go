// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"fmt"
	"image"
	"log"
	"log/slog"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/vgpu"
)

// Texture has texture image -- stored as image.RGBA for GPU compatibility
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
	tx.Image = vgpu.ImageToRGBA(img)
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
	vars := ph.Sys.Vars()
	txset := vars.SetMap[int(TexSet)]
	txset.ConfigValues(1)
	dimg := image.NewRGBA(image.Rectangle{Max: image.Point{2, 2}})
	_, img, _ := txset.ValueByIndexTry("Tex", 0)
	img.Texture.ConfigGoImage(dimg.Bounds().Size(), 0)
}

// ConfigTextures configures vals for textures -- this is the first
// of two passes -- call Phong.Sys.Config after everything is config'd.
// This automatically allocates images by size so everything fits
// within the MaxTexturesPerStage limit, as texture arrays.
func (ph *Phong) ConfigTextures() {
	ntx := ph.Textures.Len()
	if ntx == 0 {
		ph.configDummyTexture()
		return
	}
	vars := ph.Sys.Vars()
	txset := vars.SetMap[int(TexSet)]
	txset.ConfigValues(ntx)
	for i, kv := range ph.Textures.Order {
		_, img, err := txset.ValueByIndexTry("Tex", i)
		if err != nil {
			slog.Error("vgpu.Phong ConfigTextures: txset Image is nil", "Image", i)
			continue
		}
		if kv.Value.Image == nil {
			slog.Error("vgpu.Phong ConfigTextures: Image is nil", "Image", i)
			continue
		}
		img.Texture.ConfigGoImage(kv.Value.Image.Bounds().Size(), 1)
	}
	ivar := txset.VarMap["Tex"]
	ivar.Values.AllocTexBySize(ph.Sys.GPU, ivar) // organize images by size so all fit
}

// AllocTextures allocates textures that have been previously configured,
// via ConfigTextures(), after Phong.Sys.Config() has been called.
func (ph *Phong) AllocTextures() {
	vars := ph.Sys.Vars()
	ntx := ph.Textures.Len()
	if ntx > 0 {
		txset := vars.SetMap[int(TexSet)]
		ivar := txset.VarMap["Tex"]
		for i, kv := range ph.Textures.Order {
			ivar.Values.SetGoImage(i, kv.Value.Image, vgpu.FlipY)
		}
	}
	vars.BindAllTextureVars(int(TexSet)) // gets images
}

// ResetTextures resets all textures
func (ph *Phong) ResetTextures() {
	vars := ph.Sys.Vars()
	txset := vars.SetMap[int(TexSet)]
	txset.Destroy(ph.Sys.Device.Device)
	ph.Textures.Reset()
}

// UseNoTexture turns off texture rendering
func (ph *Phong) UseNoTexture() {
	ph.Cur.UseTexture = false
}

// UseTextureIndex selects texture by index for current render step
func (ph *Phong) UseTextureIndex(idx int) error {
	ph.Cur.TexIndex = idx // todo: range check
	ph.Cur.UseTexture = true
	return nil
}

// UseTextureName selects texture by name for current render step
func (ph *Phong) UseTextureName(name string) error {
	idx, ok := ph.Textures.IndexByKeyTry(name)
	if !ok {
		err := fmt.Errorf("vphong:UseTextureName -- name not found: %s", name)
		if vgpu.Debug {
			log.Println(err)
		}
	}
	return ph.UseTextureIndex(idx)
}

// UpdateTextureIndex updates texture by index -- call this when
// the underlying image changes.  Assumes the size remains the same.
// Must Sync for the changes to take effect.
func (ph *Phong) UpdateTextureIndex(idx int) error {
	ph.UpdateMu.Lock()
	defer ph.UpdateMu.Unlock()
	if idx >= ph.Textures.Len() {
		return nil
	}
	tx := ph.Textures.Order[idx].Value
	vars := ph.Sys.Vars()
	txset := vars.SetMap[int(TexSet)]
	ivar := txset.VarMap["Tex"]
	ivar.Values.SetGoImage(idx, tx.Image, vgpu.FlipY)
	return nil
}

// UpdateTextureName updates texture by name
func (ph *Phong) UpdateTextureName(name string) error {
	idx, ok := ph.Textures.IndexByKeyTry(name)
	if !ok {
		err := fmt.Errorf("vphong:UpdateTextureName -- name not found: %s", name)
		if vgpu.Debug {
			log.Println(err)
		}
	}
	return ph.UpdateTextureIndex(idx)
}

// UseTexturePars sets the texture parameters for the next render command:
// how often the texture repeats along each dimension, and the offset
func (ph *Phong) UseTexturePars(repeat math32.Vector2, off math32.Vector2) {
	ph.Cur.TexPars.Set(repeat, off)
}

// UseFullTexture sets the texture parameters for the next render command:
// to render the full texture: repeat = 1,1; off = 0,0
func (ph *Phong) UseFullTexture() {
	ph.Cur.TexPars.Set(math32.Vec2(1, 1), math32.Vector2{})
}

// TexPars holds texture parameters: how often to repeat the texture image and offset
type TexPars struct {

	// how often to repeat the texture in each direction
	Repeat math32.Vector2

	// offset for when to start the texure in each direction
	Off math32.Vector2
}

func (tp *TexPars) Set(rpt, off math32.Vector2) {
	tp.Repeat = rpt
	tp.Off = off
}

// RenderTexture renders current settings to texture pipeline
func (ph *Phong) RenderTexture() {
	sy := &ph.Sys
	cmd := sy.CmdPool.Buff
	pl := sy.PipelineMap["texture"]

	vars := ph.Sys.Vars()
	idxs := vars.TextureGroupSizeIndexes(int(TexSet), "Tex", ph.Cur.TexIndex)

	txIndex, _, _, err := sy.CmdBindTextureVarIndex(cmd, int(TexSet), "Tex", idxs.GpIndex)
	if err != nil {
		return
	}
	push := ph.Cur.NewPush()
	push.Tex = ph.Cur.TexPars
	push.ModelMtx[15] = idxs.PctSize.X // packing bits..
	push.Color.Emissive.W = idxs.PctSize.Y
	txIndexP := txIndex*1024 + idxs.ItemIndex // packing index and layer into one
	push.Color.ShinyBright.W = float32(txIndexP)
	ph.Push(pl, push)
	pl.BindDrawVertex(cmd, ph.Cur.DescIndex)
}
