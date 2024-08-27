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

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/phong"
)

// TextureName provides a GUI interface for choosing textures.
type TextureName string

// Texture is the interface for all textures.
type Texture interface {
	// AsTextureBase returns the [TextureBase] for this texture,
	// which contains the core data and functionality.
	AsTextureBase() *TextureBase

	// Image returns the image for the texture in the [image.RGBA] format used internally.
	Image() *image.RGBA
}

//////////////////////////////////////////////////////////////////////////////////////
// TextureBase

// TextureBase is the base texture implementation.
// It uses an [image.RGBA] as the underlying image storage
// to facilitate interface with GPU.
type TextureBase struct { //types:add --setters
	// Name is the name of the texture;
	// textures are connected to [Material]s by name.
	Name string

	// Transprent is whether the texture has transparency.
	Transparent bool

	// RGBA is the cached internal representation of the image.
	RGBA *image.RGBA
}

func (tx *TextureBase) AsTextureBase() *TextureBase {
	return tx
}

func (tx *TextureBase) Image() *image.RGBA {
	return tx.RGBA
}

//////////////////////////////////////////////////////////////////////////////////////
// TextureFile

// TextureFile is a texture loaded from a file
type TextureFile struct {
	TextureBase

	// filesystem for embedded etc
	FS fs.FS

	// filename for the texture
	File string
}

// NewTextureFile adds a new texture from file of given name and filename
func NewTextureFile(sc *Scene, name string, filename string) *TextureFile {
	tx := &TextureFile{}
	tx.Name = name
	dfs, fnm, err := fsx.DirFS(filename)
	if err != nil {
		slog.Error("xyz.NewTextureFile: Image not found error", "file:", filename, "error", err)
		return nil
	}
	tx.FS = dfs
	tx.File = fnm
	sc.SetTexture(tx)
	return tx
}

// NewTextureFileFS adds a new texture from file of given name and filename
func NewTextureFileFS(fsys fs.FS, sc *Scene, name string, filename string) *TextureFile {
	tx := &TextureFile{}
	tx.Name = name
	tx.FS = fsys
	tx.File = filename
	sc.SetTexture(tx)
	return tx
}

func (tx *TextureFile) Image() *image.RGBA {
	if tx.RGBA != nil {
		return tx.RGBA
	}
	if tx.File == "" {
		err := fmt.Errorf("xyz.Texture: %v File must be set to a filename to load texture from", tx.Name)
		log.Println(err)
		return nil
	}
	img, _, err := imagex.OpenFS(tx.FS, tx.File)
	if err != nil {
		slog.Error("xyz.TextureFile: Image load error", "file:", tx.File, "error", err)
		return nil
	}
	tx.RGBA = gpu.ImageToRGBA(img)
	return tx.RGBA
}

// TextureCore is a dynamic texture material driven by a core.Scene.
// Anything rendered to the scene will be projected onto the surface of any
// solid using this texture. TODO: update this along with embed2d
type TextureCore struct {
	TextureBase
	// Scene2D *core.Scene
}

///////////////////////////////////////////////////////////////////////////
// Scene code

// SetTexture adds given texture to texture collection
// see NewTextureFile to add a texture that loads from file
func (sc *Scene) SetTexture(tx Texture) {
	name := tx.AsTextureBase().Name
	sc.Textures.Add(name, tx) // does replease
	if sc.IsLive() {
		sc.Phong.SetTexture(name, phong.NewTexture(tx.Image()))
	}
}

func (sc *Scene) setAllTextures() {
	ph := sc.Phong
	for _, kv := range sc.Textures.Order {
		tx := kv.Value
		ph.SetTexture(kv.Key, phong.NewTexture(tx.Image()))
	}
}

// TextureByName looks for texture by name, returning error if not found.
func (sc *Scene) TextureByName(nm string) (Texture, error) {
	tx, ok := sc.Textures.ValueByKeyTry(nm)
	if ok {
		return tx, nil
	}
	return nil, fmt.Errorf("Texture named: %v not found in Scene: %v", nm, sc.Name)
}

// TextureList returns a list of available textures (e.g., for chooser)
func (sc *Scene) TextureList() []string {
	return sc.Textures.Keys()
}
