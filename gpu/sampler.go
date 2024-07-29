// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"log/slog"

	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// SampledTexture supplies a Texture and a Sampler
type SampledTexture struct {
	Texture
	Sampler
}

func (tx *SampledTexture) Defaults() {
	tx.Texture.Format.Defaults()
	tx.Sampler.Defaults()
}

func (tx *SampledTexture) Release() {
	tx.Sampler.Release()
	tx.Texture.Release()
}

// AllocSampledTexture allocates texture device image, stdview, and sampler
// func (tx *SampledTexture) AllocSampledTexture() {
// 	tx.AllocTexture()
// 	if tx.Sampler.VkSampler == vk.NullSampler {
// 		tx.Sampler.Config(tx.GPU, tx.Dev)
// 	}
// 	tx.ConfigStdView()
// }

///////////////////////////////////////////////////

// Sampler represents a WebGPU image sampler
type Sampler struct {
	Name string

	// for U (horizontal) axis -- what to do when going off the edge
	UMode SamplerModes

	// for V (vertical) axis -- what to do when going off the edge
	VMode SamplerModes

	// for W (horizontal) axis -- what to do when going off the edge
	WMode SamplerModes

	// border color for Clamp modes
	Border BorderColors

	// the WebGPU sampler
	Sampler *wgpu.Sampler
}

func (sm *Sampler) Defaults() {
	sm.UMode = Repeat
	sm.VMode = Repeat
	sm.WMode = Repeat
	sm.Border = BorderTrans
}

// Config configures sampler on device
func (sm *Sampler) Config(dev *Device) error {
	sm.Release()
	samp, err = dev.Device.CreateSampler(&wgpu.SamplerDescriptor{
		AddressModeU:   sm.UMode.Mode(),
		AddressModeV:   sm.VMode.Mode(),
		AddressModeW:   sm.WMode.Mode(),
		MagFilter:      wgpu.FilterMode_Linear, // nearest?
		MinFilter:      wgpu.FilterMode_Linear,
		MipmapFilter:   wgpu.MipmapFilterMode_Linear,
		Compare:        wgpu.CompareFunction_LessEqual,
		LodMinClamp:    0,
		LodMaxClamp:    32,
		MaxAnisotrophy: 1,
	})
	// MaxAnisotropy:           gp.GPUProperties.Limits.MaxSamplerAnisotropy,
	// BorderColor:             sm.Border.VkColor(),
	// UnnormalizedCoordinates: vk.False,
	// CompareEnable:           vk.False,
	if err != nil {
		slog.Error(err)
		return err
	}
	sm.Sampler = samp
	return nil
}

func (sm *Sampler) Release() {
	if sm.Sampler != nil {
		sm.Sampler.Release()
		sm.Sampler = nil
	}
}

// SampledTexture image sampler modes
type SamplerModes int32 //enums:enum

const (
	// Repeat the texture when going beyond the image dimensions.
	Repeat SamplerModes = iota

	// Like repeat, but inverts the coordinates to mirror the image when going beyond the dimensions.
	MirroredRepeat

	// Take the color of the edge closest to the coordinate beyond the image dimensions.
	ClampToEdge

	// Return a solid color when sampling beyond the dimensions of the image.
	// ClampToBorder

	// Like clamp to edge, but instead uses the edge opposite to the closest edge.
	// MirrorClampToEdge
)

func (sm SamplerModes) Mode() wgpu.AddressMode {
	return WebGPUSamplerModes[sm]
}

var WebGPUSamplerModes = map[SamplerModes]wgpu.AddressMode{
	Repeat:         wgpu.AddressMode_Repeat,
	MirroredRepeat: wgpu.AddressMode_MirroredRepeat,
	ClampToEdge:    wgpu.AddressMode_ClampToEdge,
}

//////////////////////////////////////////////////////

// SampledTexture image sampler modes
type BorderColors int32 //enums:enum -trim-prefix Border

const (
	// Repeat the texture when going beyond the image dimensions.
	BorderTrans BorderColors = iota
	BorderBlack
	BorderWhite
)

// todo: not (yet) supported
//
// func (bc BorderColors) VkColor() vk.BorderColor {
// 	return VulkanBorderColors[bc]
// }
//
// var VulkanBorderColors = map[BorderColors]vk.BorderColor{
// 	BorderTrans: vk.BorderColorIntTransparentBlack,
// 	BorderBlack: vk.BorderColorIntOpaqueBlack,
// 	BorderWhite: vk.BorderColorIntOpaqueWhite,
// }
