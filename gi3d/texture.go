// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"image"
	"log"

	"github.com/goki/gi/gi"
	"github.com/goki/ki/kit"
	"github.com/goki/vgpu/vphong"
)

// TexName provides a GUI interface for choosing textures
type TexName string

type Texture interface {
	// Name returns name of the texture
	Name() string

	// IsTransparent returns true if there is any transparency present in the texture
	// This is not auto-detected but rather must be set manually.
	// It affects the rendering order -- transparent items are rendered last.
	IsTransparent() bool

	// SetTransparent sets the transparency flag for this texture.
	SetTransparent(trans bool)

	// Image returns image for the texture
	Image() image.Image

	// SetImage sets image for the texture
	SetImage(img image.Image)
}

//////////////////////////////////////////////////////////////////////////////////////
// TextureBase

// TextureBase is the base texture implementation
type TextureBase struct {
	Nm    string      `desc:"name of the texture -- textures are connected to material by name"`
	Trans bool        `desc:"set to true if texture has transparency"`
	Img   image.Image `desc:"cached image"`
}

var KiT_TextureBase = kit.Types.AddType(&TextureBase{}, nil)

func (tx *TextureBase) Name() string {
	return tx.Nm
}

func (tx *TextureBase) IsTransparent() bool {
	return tx.Trans
}

func (tx *TextureBase) SetTransparent(trans bool) {
	tx.Trans = trans
}

func (tx *TextureBase) Image() image.Image {
	return tx.Img
}

func (tx *TextureBase) SetImage(img image.Image) {
	tx.Img = img
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

func (tx *TextureFile) Image() image.Image {
	if tx.Img == nil {
		if tx.File == "" {
			err := fmt.Errorf("gi3d.Texture: %v File must be set to a filename to load texture from", tx.Nm)
			log.Println(err)
			return nil
		}
		var err error
		tx.Img, err = gi.OpenImage(string(tx.File))
		if err != nil {
			log.Println(err)
			return nil
		}
	}
	return tx.Img
}

// TextureGi2D is a dynamic texture material driven by a gi.Viewport2D viewport
// anything rendered to the viewport will be projected onto the surface of any
// solid using this texture.
type TextureGi2D struct {
	TextureBase
	Viewport *gi.Viewport2D
}

///////////////////////////////////////////////////////////////////////////
// Scene code

// AddTexture adds given texture to texture collection
// see AddNewTextureFile to add a texture that loads from file
func (sc *Scene) AddTexture(tx Texture) {
	sc.Textures.Add(tx.Name(), tx)
}

// TextureByName looks for texture by name -- returns nil if not found
func (sc *Scene) TextureByName(nm string) Texture {
	tx, ok := sc.Textures.ValByKey(nm)
	if ok {
		return tx
	}
	return nil
}

// TextureByNameTry looks for texture by name -- returns error if not found
func (sc *Scene) TextureByNameTry(nm string) (Texture, error) {
	tx, ok := sc.Textures.ValByKey(nm)
	if ok {
		return tx, nil
	}
	return nil, fmt.Errorf("Texture named: %v not found in Scene: %v", nm, sc.Nm)
}

// TextureList returns a list of available textures (e.g., for chooser)
func (sc *Scene) TextureList() []string {
	return sc.Textures.Keys()
}

// DeleteTexture deletes texture of given name -- returns error if not found
// must call ConfigTextures or Init3D to reset after deleting
func (sc *Scene) DeleteTexture(nm string) {
	sc.Textures.DeleteKey(nm)
}

// DeleteTextures removes all textures
// must call ConfigTextures or Init3D to reset after deleting
func (sc *Scene) DeleteTextures() {
	sc.Textures.Reset()
}

// must be called after adding or deleting any meshes or altering
// the number of verticies.
func (sc *Scene) ConfigTextures() {
	ph := &sc.Phong
	ph.ResetTextures()
	for _, kv := range sc.Textures.Order {
		tx := kv.Val
		// todo: remove repeat from texture, move to color.
		ph.AddTexture(kv.Key, vphong.NewTexture(tx.Image()))
	}
}

// ReconfigTextures reconfigures textures on the Phong renderer
// if there has been a change to the mesh structure
// Init3D does a full configure of everything -- this is optimized
// just for texture changes.
func (sc *Scene) ReconfigTextures() {
	sc.ConfigTextures()
	sc.Phong.Config()
}
