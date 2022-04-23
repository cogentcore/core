// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	vk "github.com/vulkan-go/vulkan"
)

type Texture struct {
	sampler vk.Sampler

	image       vk.Image
	imageLayout vk.ImageLayout

	memAlloc *vk.MemoryAllocateInfo
	mem      vk.DeviceMemory
	view     vk.ImageView

	texWidth  int32
	texHeight int32
}

func (t *Texture) Destroy(dev vk.Device) {
	vk.DestroyImageView(dev, t.view, nil)
	vk.FreeMemory(dev, t.mem, nil)
	vk.DestroyImage(dev, t.image, nil)
	vk.DestroySampler(dev, t.sampler, nil)
}

func (t *Texture) DestroyImage(dev vk.Device) {
	vk.FreeMemory(dev, t.mem, nil)
	vk.DestroyImage(dev, t.image, nil)
}

type Depth struct {
	format   vk.Format
	image    vk.Image
	memAlloc *vk.MemoryAllocateInfo
	mem      vk.DeviceMemory
	view     vk.ImageView
}

func (d *Depth) Destroy(dev vk.Device) {
	vk.DestroyImageView(dev, d.view, nil)
	vk.DestroyImage(dev, d.image, nil)
	vk.FreeMemory(dev, d.mem, nil)
}

// func loadTextureSize(name string) (w int, h int, err error) {
// 	data := MustAsset(name)
// 	r := bytes.NewReader(data)
// 	ppmCfg, err := ppm.DecodeConfig(r)
// 	if err != nil {
// 		return 0, 0, err
// 	}
// 	return ppmCfg.Width, ppmCfg.Height, nil
// }

// func loadTextureData(name string, layout vk.SubresourceLayout) ([]byte, error) {
// 	data := MustAsset(name)
// 	r := bytes.NewReader(data)
// 	img, err := ppm.Decode(r)
// 	if err != nil {
// 		return nil, err
// 	}
// 	newImg := image.NewRGBA(img.Bounds())
// 	newImg.Stride = int(layout.RowPitch)
// 	draw.Draw(newImg, newImg.Bounds(), img, image.ZP, draw.Src)
// 	return []byte(newImg.Pix), nil
// }

/*
func loadTextureData(name string, rowPitch int) ([]byte, int, int, error) {
	// data := MustAsset(name)
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, err
	}
	newImg := image.NewRGBA(img.Bounds())
	if rowPitch <= 4*img.Bounds().Dy() {
		// apply the proposed row pitch only if supported,
		// as we're using only optimal textures.
		newImg.Stride = rowPitch
	}
	draw.Draw(newImg, newImg.Bounds(), img, image.ZP, draw.Src)
	size := newImg.Bounds().Size()
	return []byte(newImg.Pix), size.X, size.Y, nil
}
*/
