// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"
	"image"
	"io/fs"
	"log"
	"log/slog"

	"cogentcore.org/core/glop/dirs"
	"cogentcore.org/core/grows/images"
	"cogentcore.org/core/vgpu"
	"cogentcore.org/core/vgpu/vphong"
)

// TexName provides a GUI interface for choosing textures
type TexName string

// Texture is the interface for all textures
type Texture interface {
	// Name returns name of the texture
	Name() string

	// IsTransparent returns true if there is any transparency present in the texture
	// This is not auto-detected but rather must be set manually.
	// It affects the rendering order -- transparent items are rendered last.
	IsTransparent() bool

	// SetTransparent sets the transparency flag for this texture.
	SetTransparent(trans bool)

	// Image returns image for the texture, in image.RGBA format used internally
	Image() *image.RGBA

	// SetImage sets image for the texture
	SetImage(img image.Image)
}

//////////////////////////////////////////////////////////////////////////////////////
// TextureBase

// TextureBase is the base texture implementation
// it uses an image.RGBA as underlying image storage to facilitate interface with GPU
type TextureBase struct {

	// name of the texture -- textures are connected to material by name
	Nm string

	// set to true if texture has transparency
	Trans bool

	// cached image
	Img *image.RGBA
}

func (tx *TextureBase) Name() string {
	return tx.Nm
}

func (tx *TextureBase) IsTransparent() bool {
	return tx.Trans
}

func (tx *TextureBase) SetTransparent(trans bool) {
	tx.Trans = trans
}

func (tx *TextureBase) Image() *image.RGBA {
	return tx.Img
}

func (tx *TextureBase) SetImage(img image.Image) {
	if img == nil {
		tx.Img = nil
	} else {
		tx.Img = vgpu.ImageToRGBA(img)
	}
}

//////////////////////////////////////////////////////////////////////////////////////
// TextureFile

// TextureFile is a texture loaded from a file
type TextureFile struct {
	TextureBase

	// filesystem for embedded etc
	FSys fs.FS

	// filename for the texture
	File string
}

// NewTextureFile adds a new texture from file of given name and filename
func NewTextureFile(sc *Scene, name string, filename string) *TextureFile {
	tx := &TextureFile{}
	tx.Nm = name
	dfs, fnm, err := dirs.DirFS(filename)
	if err != nil {
		slog.Error("xyz.NewTextureFile: Image not found error", "file:", filename, "error", err)
		return nil
	}
	tx.FSys = dfs
	tx.File = fnm
	sc.AddTexture(tx)
	return tx
}

// NewTextureFileFS adds a new texture from file of given name and filename
func NewTextureFileFS(fsys fs.FS, sc *Scene, name string, filename string) *TextureFile {
	tx := &TextureFile{}
	tx.Nm = name
	tx.FSys = fsys
	tx.File = filename
	sc.AddTexture(tx)
	return tx
}

func (tx *TextureFile) Image() *image.RGBA {
	if tx.Img != nil {
		return tx.Img
	}
	if tx.File == "" {
		err := fmt.Errorf("xyz.Texture: %v File must be set to a filename to load texture from", tx.Nm)
		log.Println(err)
		return nil
	}
	img, _, err := images.OpenFS(tx.FSys, tx.File)
	if err != nil {
		slog.Error("xyz.TextureFile: Image load error", "file:", tx.File, "error", err)
		return nil
	}
	tx.Img = vgpu.ImageToRGBA(img)
	return tx.Img
}

// TextureGi2D is a dynamic texture material driven by a gi.Viewport2D viewport
// anything rendered to the viewport will be projected onto the surface of any
// solid using this texture.
type TextureGi2D struct {
	TextureBase
	// Scene2D *gi.Scene
}

///////////////////////////////////////////////////////////////////////////
// Scene code

// AddTexture adds given texture to texture collection
// see NewTextureFile to add a texture that loads from file
func (sc *Scene) AddTexture(tx Texture) {
	sc.Textures.Add(tx.Name(), tx)
}

// TextureByName looks for texture by name -- returns nil if not found
func (sc *Scene) TextureByName(nm string) Texture {
	tx, ok := sc.Textures.ValueByKeyTry(nm)
	if ok {
		return tx
	}
	return nil
}

// TextureByNameTry looks for texture by name -- returns error if not found
func (sc *Scene) TextureByNameTry(nm string) (Texture, error) {
	tx, ok := sc.Textures.ValueByKeyTry(nm)
	if ok {
		return tx, nil
	}
	return nil, fmt.Errorf("texture named: %v not found in Scene: %v", nm, sc.Nm)
}

// TextureList returns a list of available textures (e.g., for chooser)
func (sc *Scene) TextureList() []string {
	return sc.Textures.Keys()
}

// DeleteTexture deletes texture of given name -- returns error if not found
// must call ConfigTextures or Config to reset after deleting
func (sc *Scene) DeleteTexture(nm string) {
	sc.Textures.DeleteKey(nm)
}

// DeleteTextures removes all textures
// must call ConfigTextures or Config to reset after deleting
func (sc *Scene) DeleteTextures() {
	sc.Textures.Reset()
}

// ConfigTextures must be called after adding or deleting any meshes or altering
// the number of verticies.
func (sc *Scene) ConfigTextures() {
	ph := &sc.Phong
	ph.ResetTextures()
	for _, kv := range sc.Textures.Order {
		tx := kv.Value
		// todo: remove repeat from texture, move to color.
		ph.AddTexture(kv.Key, vphong.NewTexture(tx.Image()))
	}
}

// ReconfigTextures reconfigures textures on the Phong renderer
// if there has been a change to the mesh structure
// Config does a full configure of everything -- this is optimized
// just for texture changes.
func (sc *Scene) ReconfigTextures() {
	sc.ConfigTextures()
	sc.Phong.Config()
}
