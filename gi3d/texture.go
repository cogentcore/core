// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"log"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin/gpu"
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
}

// TextureFile is a texture loaded from a file
type TextureFile struct {
	Nm   string        `desc:"name of the texture -- textures are connected to material / objects by name"`
	File gi.FileName   `desc:"filename for the texture"`
	Tex  gpu.Texture2D `view:"-" desc:"gpu texture object"`
}

// AddNewTextureFile adds a new texture from file of given name and filename
func AddNewTextureFile(sc *Scene, name string, filename string) *TextureFile {
	tx := &TextureFile{Nm: name, File: gi.FileName(filename)}
	sc.AddTexture(tx)
	return tx
}

func (tx *TextureFile) Name() string {
	return tx.Nm
}

// Init initializes the texture, opens the file, and uploads it to the GPU
// Must be called in context on main thread
func (tx *TextureFile) Init(sc *Scene) error {
	if tx.Tex != nil {
		return nil
	}
	if tx.File == "" {
		err := fmt.Errorf("gi3d.Texture: %v File must be set to a filename to load texture from", tx.Name)
		log.Println(err)
		return err
	}
	tx.Tex = gpu.TheGPU.NewTexture2D(tx.Nm)
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
	tx.Tex.Activate(texNo)
}

// TextureGi2D is a dynamic texture material driven by a gi.Viewport2D viewport
// anything rendered to the viewport will be projected onto the surface of any
// object using this texture.
type TextureGi2D struct {
	Texture
	Viewport *gi.Viewport2D
}
