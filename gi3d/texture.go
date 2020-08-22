// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"log"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin/gpu"
	"github.com/goki/ki/kit"
)

// TexName provides a GUI interface for choosing textures
type TexName string

type Texture interface {
	// Name returns name of the texture
	Name() string

	// Init initializes the texture and uploads it to the GPU, so it is ready to use
	// Must be called in context on main thread
	Init(sc *Scene) error

	// Activate activates this texture on the GPU, at given texture number.
	// in preparation for rendering
	// Must be called in context on main thread
	Activate(sc *Scene, texNo int)

	// Delete deletes the texture GPU resources -- must be called in context on main thread
	Delete(sc *Scene)

	// BotZero returns true if this texture has the Y=0 pixels at the bottom
	// of the image.  Otherwise, Y=0 is at the top, which is the default
	// for most images loaded from files.
	BotZero() bool

	// SetBotZero sets whether this texture has the Y=0 pixels at the bottom
	// of the image.  Otherwise, Y=0 is at the top, which is the default
	// for most images loaded from files.
	SetBotZero(botzero bool)

	// IsTransparent returns true if there is any transparency present in the texture
	// This is not auto-detected but rather must be set manually.
	// It affects the rendering order -- transparent items are rendered last.
	IsTransparent() bool

	// SetTransparent sets the transparency flag for this texture.
	SetTransparent(trans bool)
}

//////////////////////////////////////////////////////////////////////////////////////
// TextureBase

// TextureBase is the base texture implementation
type TextureBase struct {
	Nm    string        `desc:"name of the texture -- textures are connected to material by name"`
	Bot0  bool          `desc:"set to true if this texture has Y=0 at the bottom -- otherwise default is Y=0 is at top as is the case in most images loaded from files etc"`
	Trans bool          `desc:"set to true if texture has transparency"`
	Tex   gpu.Texture2D `view:"-" desc:"gpu texture object"`
}

var KiT_TextureBase = kit.Types.AddType(&TextureBase{}, nil)

func (tx *TextureBase) Name() string {
	return tx.Nm
}

func (tx *TextureBase) BotZero() bool {
	return tx.Bot0
}

func (tx *TextureBase) SetBotZero(botzero bool) {
	tx.Bot0 = botzero
	if tx.Tex != nil {
		tx.Tex.SetBotZero(tx.Bot0)
	}
}

func (tx *TextureBase) IsTransparent() bool {
	return tx.Trans
}

func (tx *TextureBase) SetTransparent(trans bool) {
	tx.Trans = trans
}

// makes a new gpu.Texture2D if Tex field is nil, and returns it in any case
func (tx *TextureBase) NewTex() gpu.Texture2D {
	if tx.Tex != nil {
		return tx.Tex
	}
	tx.Tex = gpu.TheGPU.NewTexture2D(tx.Nm)
	tx.Tex.SetBotZero(tx.Bot0)
	return tx.Tex
}

// Init initializes the texture and activates it -- for base case it must be set externally
// prior to this call.
// Must be called in context on main thread
func (tx *TextureBase) Init(sc *Scene) error {
	if tx.Tex != nil {
		tx.Tex.SetBotZero(tx.Bot0)
		tx.Tex.Activate(0)
	}
	return nil
}

// Activate activates this texture on the GPU, in preparation for rendering
// Must be called in context on main thread
func (tx *TextureBase) Activate(sc *Scene, texNo int) {
	if tx.Tex != nil {
		tx.Tex.SetBotZero(tx.Bot0)
		tx.Tex.Activate(texNo)
	}
}

// Delete deletes the texture GPU resources -- must be called in context on main thread
func (tx *TextureBase) Delete(sc *Scene) {
	if tx.Tex != nil {
		tx.Tex.Delete()
	}
	tx.Tex = nil
}

//////////////////////////////////////////////////////////////////////////////////////
// TextureFile

// TextureFile is a texture loaded from a file
type TextureFile struct {
	TextureBase
	File gi.FileName `desc:"filename for the texture"`
}

var KiT_TextureFile = kit.Types.AddType(&TextureFile{}, nil)

// AddNewTextureFile adds a new texture from file of given name and filename
func AddNewTextureFile(sc *Scene, name string, filename string) *TextureFile {
	tx := &TextureFile{}
	tx.Nm = name
	tx.File = gi.FileName(filename)
	sc.AddTexture(tx)
	return tx
}

// Init initializes the texture, opens the file, and uploads it to the GPU
// Must be called in context on main thread
func (tx *TextureFile) Init(sc *Scene) error {
	if tx.Tex != nil {
		tx.Tex.SetBotZero(tx.Bot0)
		tx.Tex.Activate(0)
		return nil
	}
	if tx.File == "" {
		err := fmt.Errorf("gi3d.Texture: %v File must be set to a filename to load texture from", tx.Nm)
		log.Println(err)
		return err
	}
	tx.Tex = gpu.TheGPU.NewTexture2D(tx.Nm)
	tx.Tex.SetBotZero(tx.Bot0)
	err := tx.Tex.Open(string(tx.File))
	if err != nil {
		log.Println(err)
		return err
	}
	tx.Tex.Activate(0)
	return nil
}

// Activate activates this texture on the GPU, in preparation for rendering
// Must be called in context on main thread
func (tx *TextureFile) Activate(sc *Scene, texNo int) {
	if tx.Tex == nil {
		tx.Init(sc)
	}
	tx.Tex.SetBotZero(tx.Bot0)
	tx.Tex.Activate(texNo)
}

// TextureGi2D is a dynamic texture material driven by a gi.Viewport2D viewport
// anything rendered to the viewport will be projected onto the surface of any
// solid using this texture.
type TextureGi2D struct {
	TextureBase
	Viewport *gi.Viewport2D
}
