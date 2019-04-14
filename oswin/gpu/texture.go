// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import "image"

// Texture2D manages a 2D texture, including loading from an image file
// and activating on GPU
type Texture2D interface {
	// Name returns the name of the texture (filename without extension
	// by default)
	Name() string

	// SetName sets the name of the texture
	SetName(name string)

	// Open loads texture image from file.
	// format inferred from filename -- JPEG and PNG
	// supported by default.
	Open(path string) error

	// SaveAs saves texture image to file.
	// format inferred from filename -- JPEG and PNG
	// supported by default.
	SaveAs(path string) error

	// Image returns the image -- typically as an image.RGBA
	Image() image.Image

	// SetImage sets the image -- typically as an image.RGBA
	// If called after Activate and different than current size,
	// then does Delete.
	SetImage(img image.Image) error

	// Size returns the size of the image
	Size() image.Point

	// SetSize sets the size of the image.
	// If called after Activate and different than current size,
	// then does Delete.
	SetSize(size image.Point)

	// Activate establishes the GPU resources and handle for the
	// texture, using the given texture number (0-31 range)
	Activate(texNo int)

	// Handle returns the GPU handle for the texture -- only
	// valid after Activate
	Handle() uint32

	// Delete deletes the GPU resources associated with this image
	// (requires Activate to re-establish a new one).
	// Should be called prior to Go object being deleted
	// (ref counting can be done externally).
	Delete()
}
