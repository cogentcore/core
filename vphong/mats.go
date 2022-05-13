// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

import (
	"image"

	"github.com/goki/vgpu/vgpu"
)

// AddTexImage adds an image to list of textures
func (ph *Phong) AddTexImage(img image.Image) {
	ph.TexImages = append(ph.TexImages, img)
}

// ConfigTextures configures the rendering for the textures that have been added.
func (ph *Phong) ConfigTextures() {
	vars := ph.Sys.Vars()
	txset := vars.SetMap[int(TexSet)]
	txset.ConfigVals(len(ph.TexImages))
	for i, gimg := range ph.TexImages {
		_, img, _ := vars.ValByIdxTry(int(TexSet), "Tex", i)
		img.SetGoImage(gimg, vgpu.FlipY)
	}
}
