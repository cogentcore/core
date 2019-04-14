// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glgpu

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Texture2D manages a texture, including loading from an image file
// and activating on GPU
type Texture2D struct {
	init   bool
	handle uint32
	name   string
	size   image.Point
	img    *image.RGBA // when loaded
	// magFilter uint32 // magnification filter
	// minFilter uint32 // minification filter
	// wrapS     uint32 // wrap mode for s coordinate
	// wrapT     uint32 // wrap mode for t coordinate
}

// Name returns the name of the texture (filename without extension
// by default)
func (tx *Texture2D) Name() string {
	return tx.name
}

// SetName sets the name of the texture
func (tx *Texture2D) SetName(name string) {
	tx.name = name
}

// Open loads texture image from file.
// format inferred from filename -- JPEG and PNG
// supported by default.
func (tx *Texture2D) Open(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	im, _, err := image.Decode(file)
	if err != nil {
		return err
	}
	return tx.SetImage(im)
}

// SaveAs saves texture image to file.
// format inferred from filename -- JPEG and PNG
// supported by default.
func (tx *Texture2D) SaveAs(path string) error {
	im, err := tx.Image()
	if err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".png" {
		return png.Encode(file, im)
	} else if ext == ".jpg" || ext == ".jpeg" {
		return jpeg.Encode(file, im, &jpeg.Options{Quality: 90})
	} else {
		return fmt.Errorf("glgpu Texture2D save image: extention: %s not recognized -- only .png and .jpg / jpeg supported", ext)
	}
}

// Image returns the image -- typically as an image.RGBA
func (tx *Texture2D) Image() (image.Image, error) {
	if !tx.init {
		if tx.img == nil {
			return nil, fmt.Errorf("glgpu Texture2D SaveAs: no image available")
		}
		return tx.img, nil
	}
	// todo: get image from buffer
	return tx.img, nil
}

// SetImage sets the image -- typically as an image.RGBA
// If called after Activate and different than current size,
// then does Delete.
func (tx *Texture2D) SetImage(img image.Image) error {
	if rgba, ok := img.(*image.RGBA); ok {
		tx.img = rgba
		tx.size = rgba.Rect.Size()
		return nil
	}
	// Converts image to RGBA format
	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return fmt.Errorf("glgpu Texture2D: unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)
	tx.img = rgba
	return nil
}

// Size returns the size of the image
func (tx *Texture2D) Size() image.Point {
	return tx.size
}

// SetSize sets the size of the image.
// If called after Activate and different than current size,
// then does Delete.
func (tx *Texture2D) SetSize(size image.Point) {
	if tx.size == size {
		return
	}
	if tx.init {
		tx.Delete()
	}
	tx.size = size
	tx.img = nil
}

// Activate establishes the GPU resources and handle for the
// texture, using the given texture number (0-31 range)
func (tx *Texture2D) Activate(texNo int) {
	if !tx.init {
		gl.GenTextures(1, &tx.handle)
		gl.ActiveTexture(gl.TEXTURE0 + uint32(texNo))
		gl.BindTexture(gl.TEXTURE_2D, tx.handle)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		var dat []uint8
		if tx.img != nil {
			dat = tx.img.Pix
		}
		gl.TexImage2D(
			gl.TEXTURE_2D,
			0,
			gl.RGBA,
			int32(tx.size.X),
			int32(tx.size.Y),
			0,
			gl.RGBA,
			gl.UNSIGNED_BYTE,
			gl.Ptr(dat))
		tx.init = true
	} else {
		gl.ActiveTexture(gl.TEXTURE0 + uint32(texNo))
		gl.BindTexture(gl.TEXTURE_2D, tx.handle)
	}
}

// Handle returns the GPU handle for the texture -- only
// valid after Activate
func (tx *Texture2D) Handle() uint32 {
	return tx.handle
}

// Delete deletes the GPU resources associated with this image
// (requires Activate to re-establish a new one).
// Should be called prior to Go object being deleted
// (ref counting can be done externally).
func (tx *Texture2D) Delete() {
	if !tx.init {
		return
	}
	gl.DeleteTextures(1, &tx.handle)
	tx.init = false
}
