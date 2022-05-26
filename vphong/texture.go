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
	Image image.Image
}

func NewTexture(img image.Image) *Texture {
	tx := &Texture{}
	tx.Set(img)
	return tx
}

// Set sets values
func (tx *Texture) Set(img image.Image) {
	tx.Image = img
}

// AddTexture adds to list of textures
func (ph *Phong) AddTexture(name string, tex *Texture) {
	ph.Textures.Add(name, tex)
}

// DeleteTexture deletes texture with name
func (ph *Phong) DeleteTexture(name string) {
	ph.Textures.DeleteKey(name)
}

// allocDummyTexture allocates a dummy texture (if none configured)
func (ph *Phong) allocDummyTexture() {
	// there must be one texture -- otherwise Mac Molten triggers an error
	vars := ph.Sys.Vars()
	txset := vars.SetMap[int(TexSet)]
	txset.ConfigVals(1)
	dimg := image.NewRGBA(image.Rectangle{Max: image.Point{2, 2}})
	_, img, _ := txset.ValByIdxTry("Tex", 0)
	img.Texture.ConfigGoImage(dimg)
}

// configDummyTexture configures a dummy texture (if none configured)
func (ph *Phong) configDummyTexture() {
	vars := ph.Sys.Vars()
	txset := vars.SetMap[int(TexSet)]
	dimg := image.NewRGBA(image.Rectangle{Max: image.Point{2, 2}})
	_, img, _ := txset.ValByIdxTry("Tex", 0)
	img.Texture.ConfigGoImage(dimg)
}

// AllocTextures allocates vals for textures
func (ph *Phong) AllocTextures() {
	ntx := ph.Textures.Len()
	if ntx == 0 {
		ph.allocDummyTexture()
		return
	}
	vars := ph.Sys.Vars()
	txset := vars.SetMap[int(TexSet)]
	txset.ConfigVals(ntx)
	for i, kv := range ph.Textures.Order {
		_, img, _ := txset.ValByIdxTry("Tex", i)
		img.Texture.ConfigGoImage(kv.Val.Image)
	}
}

// ConfigTextures configures the rendering for the textures that have been added.
func (ph *Phong) ConfigTextures() {
	vars := ph.Sys.Vars()
	ntx := ph.Textures.Len()
	if ntx == 0 {
		ph.configDummyTexture()
	} else {
		txset := vars.SetMap[int(TexSet)]
		for i, kv := range ph.Textures.Order {
			_, img, _ := txset.ValByIdxTry("Tex", i)
			img.SetGoImage(kv.Val.Image, 0, vgpu.FlipY)
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

// UseTextureIdx selects texture by index for current render step
func (ph *Phong) UseTextureIdx(idx int) error {
	ph.Cur.TexIdx = idx // todo: range check
	ph.Cur.UseTexture = true
	return nil
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

// UpdateTextureIdx updates texture by index
func (ph *Phong) UpdateTextureIdx(idx int) error {
	vars := ph.Sys.Vars()
	txset := vars.SetMap[int(TexSet)]
	tx := ph.Textures.Order[idx].Val
	_, img, _ := txset.ValByIdxTry("Tex", idx)
	img.SetGoImage(tx.Image, 0, vgpu.FlipY)
	return nil
}

// UpdateTextureName updates texture by name
func (ph *Phong) UpdateTextureName(name string) error {
	idx, ok := ph.Textures.IdxByKey(name)
	if !ok {
		err := fmt.Errorf("vphong:UpdateTextureName -- name not found: %s", name)
		if vgpu.TheGPU.Debug {
			log.Println(err)
		}
	}
	return ph.UpdateTextureIdx(idx)
}

// UseTexturePars sets the texture parameters for the next render command:
// how often the texture repeats along each dimension, and the offset
func (ph *Phong) UseTexturePars(repeat mat32.Vec2, off mat32.Vec2) {
	ph.Cur.TexPars.Set(repeat, off)
}

// UseFullTexture sets the texture parameters for the next render command:
// to render the full texture: repeat = 1,1; off = 0,0
func (ph *Phong) UseFullTexture() {
	ph.Cur.TexPars.Set(mat32.Vec2{1, 1}, mat32.Vec2{})
}

// TexPars holds texture parameters: how often to repeat the texture image and offset
type TexPars struct {
	Repeat mat32.Vec2 `desc:"how often to repeat the texture in each direction"`
	Off    mat32.Vec2 `desc:"offset for when to start the texure in each direction"`
}

func (tp *TexPars) Set(rpt, off mat32.Vec2) {
	tp.Repeat = rpt
	tp.Off = off
}

// RenderTexture renders current settings to texture pipeline
func (ph *Phong) RenderTexture() {
	sy := &ph.Sys
	cmd := sy.CmdPool.Buff
	pl := sy.PipelineMap["texture"]
	txIdx, _, _, err := sy.CmdBindTextureVarIdx(cmd, int(TexSet), "Tex", ph.Cur.TexIdx)
	if err != nil {
		return
	}
	push := ph.Cur.NewPush()
	push.Tex = ph.Cur.TexPars
	push.Color.ShinyBright.W = float32(txIdx)
	ph.Push(pl, push)
	pl.BindDrawVertex(cmd, ph.Cur.DescIdx)
}
