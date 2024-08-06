// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"cogentcore.org/core/base/errors"
	"github.com/cogentcore/webgpu/wgpu"
)

// TextureSample supplies a Texture and a Sampler
type TextureSample struct {
	Texture
	Sampler
}

func NewTextureSample(dev *Device) *TextureSample {
	tx := &TextureSample{}
	tx.device = *dev
	tx.Format.Defaults()
	return tx
}

func (tx *TextureSample) Defaults() {
	tx.Texture.Format.Defaults()
	tx.Sampler.Defaults()
}

func (tx *TextureSample) Release() {
	tx.Sampler.Release()
	tx.Texture.Release()
}

// AllocTextureSample allocates texture device image, stdview, and sampler
// func (tx *TextureSample) AllocTextureSample() {
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
	sampler *wgpu.Sampler
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
	samp, err := dev.Device.CreateSampler(&wgpu.SamplerDescriptor{
		AddressModeU:  sm.UMode.Mode(),
		AddressModeV:  sm.VMode.Mode(),
		AddressModeW:  sm.WMode.Mode(),
		MagFilter:     wgpu.FilterModeLinear, // nearest?
		MinFilter:     wgpu.FilterModeLinear,
		MipmapFilter:  wgpu.MipmapFilterModeLinear,
		LodMinClamp:   0,
		LodMaxClamp:   32,
		Compare:       wgpu.CompareFunctionUndefined,
		MaxAnisotropy: 1,
	})
	// MaxAnisotropy:           gp.GPUProperties.Limits.MaxSamplerAnisotropy,
	// BorderColor:             sm.Border.VkColor(),
	// UnnormalizedCoordinates: vk.False,
	// CompareEnable:           vk.False,
	if errors.Log(err) != nil {
		return err
	}
	sm.sampler = samp
	return nil
}

func (sm *Sampler) Release() {
	if sm.sampler != nil {
		sm.sampler.Release()
		sm.sampler = nil
	}
}

// TextureSample image sampler modes
type SamplerModes int32 //enums:enum

const (
	// Repeat the texture when going beyond the image dimensions.
	Repeat SamplerModes = iota

	// Like repeat, but inverts the coordinates to mirror the image when going beyond the dimensions.
	MirrorRepeat

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
	Repeat:       wgpu.AddressModeRepeat,
	MirrorRepeat: wgpu.AddressModeMirrorRepeat,
	ClampToEdge:  wgpu.AddressModeClampToEdge,
}

//////////////////////////////////////////////////////

// TextureSample image sampler modes
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
