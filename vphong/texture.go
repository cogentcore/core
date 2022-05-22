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

func NewTexture(img image.Image, repeat mat32.Vec2, off mat32.Vec2) *Texture {
	tx := &Texture{}
	tx.Set(img, repeat, off)
	return tx
}

// Set sets values
func (tx *Texture) Set(img image.Image, repeat mat32.Vec2, off mat32.Vec2) {
	tx.Image = img
	tx.Repeat = repeat
	tx.Off = off
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
		_, img, _ := txset.ValByIdxTry("Tex", i)
		img.Texture.ConfigGoImage(kv.Val.Image)
	}
}

// ConfigTextures configures the rendering for the textures that have been added.
func (ph *Phong) ConfigTextures() {
	vars := ph.Sys.Vars()
	txset := vars.SetMap[int(TexSet)]
	for i, kv := range ph.Textures.Order {
		_, img, _ := txset.ValByIdxTry("Tex", i)
		img.SetGoImage(kv.Val.Image, 0, vgpu.FlipY)
	}
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

// TexPars holds texture parameters for push constants -- idx is coded in shiny bright
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
	tex := ph.Textures.ValByIdx(ph.Cur.TexIdx)
	txIdx, _, _, err := sy.CmdBindTextureVarIdx(cmd, int(TexSet), "Tex", ph.Cur.TexIdx)
	if err != nil {
		return
	}
	push := ph.Cur.NewPush()
	push.Tex.Set(tex.Repeat, tex.Off)
	push.Color.ShinyBright.W = float32(txIdx)
	ph.Push(pl, push)
	pl.BindDrawVertex(cmd, ph.Cur.DescIdx)
}
