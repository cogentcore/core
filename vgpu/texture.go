// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"github.com/goki/ki/kit"
	vk "github.com/goki/vulkan"
)

// Texture supplies an Image and a Sampler
type Texture struct {
	Image
	Sampler `desc:"sampler for image"`
}

func (tx *Texture) Defaults() {
	tx.Image.Format.Defaults()
	tx.Sampler.Defaults()
}

func (tx *Texture) Destroy() {
	tx.Sampler.Destroy(tx.Image.Dev)
	tx.Image.Destroy()
}

// AllocTexture allocates texture device image, stdview, and sampler
func (tx *Texture) AllocTexture() {
	tx.AllocImage()
	if tx.Sampler.VkSampler == nil {
		tx.Sampler.Config(tx.Dev)
	}
	tx.ConfigStdView()
}

///////////////////////////////////////////////////

// Sampler represents a vulkan image sampler
type Sampler struct {
	Name      string
	UMode     SamplerModes `desc:"for U (horizontal) axis -- what to do when going off the edge"`
	VMode     SamplerModes `desc:"for V (vertical) axis -- what to do when going off the edge"`
	WMode     SamplerModes `desc:"for W (horizontal) axis -- what to do when going off the edge"`
	Border    BorderColors `desc:"border color for Clamp modes"`
	VkSampler vk.Sampler   `desc:"the vulkan sampler"`
}

func (sm *Sampler) Defaults() {
	sm.UMode = Repeat
	sm.VMode = Repeat
	sm.WMode = Repeat
	sm.Border = BorderTrans
}

// Config configures sampler on device
func (sm *Sampler) Config(dev vk.Device) {
	sm.Destroy(dev)
	var samp vk.Sampler
	ret := vk.CreateSampler(dev, &vk.SamplerCreateInfo{
		SType:                   vk.StructureTypeSamplerCreateInfo,
		MagFilter:               vk.FilterLinear,
		MinFilter:               vk.FilterLinear,
		AddressModeU:            sm.UMode.VkMode(),
		AddressModeV:            sm.VMode.VkMode(),
		AddressModeW:            sm.WMode.VkMode(),
		AnisotropyEnable:        vk.True,
		MaxAnisotropy:           TheGPU.GPUProps.Limits.MaxSamplerAnisotropy,
		BorderColor:             sm.Border.VkColor(),
		UnnormalizedCoordinates: vk.False,
		CompareEnable:           vk.False,
		MipmapMode:              vk.SamplerMipmapModeLinear,
	}, nil, &samp)
	IfPanic(NewError(ret))
	sm.VkSampler = samp
}

func (sm *Sampler) Destroy(dev vk.Device) {
	if sm.VkSampler != nil {
		vk.DestroySampler(dev, sm.VkSampler, nil)
		sm.VkSampler = nil
	}
}

// Texture image sampler modes
type SamplerModes int32

const (
	// Repeat the texture when going beyond the image dimensions.
	Repeat SamplerModes = iota

	// Like repeat, but inverts the coordinates to mirror the image when going beyond the dimensions.
	MirroredRepeat

	// Take the color of the edge closest to the coordinate beyond the image dimensions.
	ClampToEdge

	// Return a solid color when sampling beyond the dimensions of the image.
	ClampToBorder

	// Like clamp to edge, but instead uses the edge opposite to the closest edge.
	MirrorClampToEdge

	SamplerModesN
)

//go:generate stringer -type=SamplerModes

var KiT_SamplerModes = kit.Enums.AddEnum(SamplerModesN, kit.NotBitFlag, nil)

func (sm SamplerModes) VkMode() vk.SamplerAddressMode {
	return VulkanSamplerModes[sm]
}

var VulkanSamplerModes = map[SamplerModes]vk.SamplerAddressMode{
	Repeat:            vk.SamplerAddressModeRepeat,
	MirroredRepeat:    vk.SamplerAddressModeMirroredRepeat,
	ClampToEdge:       vk.SamplerAddressModeClampToEdge,
	ClampToBorder:     vk.SamplerAddressModeClampToBorder,
	MirrorClampToEdge: vk.SamplerAddressModeMirrorClampToEdge,
}

//////////////////////////////////////////////////////

// Texture image sampler modes
type BorderColors int32

const (
	// Repeat the texture when going beyond the image dimensions.
	BorderTrans BorderColors = iota
	BorderBlack
	BorderWhite

	BorderColorsN
)

//go:generate stringer -type=BorderColors

var KiT_BorderColors = kit.Enums.AddEnum(BorderColorsN, kit.NotBitFlag, nil)

func (bc BorderColors) VkColor() vk.BorderColor {
	return VulkanBorderColors[bc]
}

var VulkanBorderColors = map[BorderColors]vk.BorderColor{
	BorderTrans: vk.BorderColorIntTransparentBlack,
	BorderBlack: vk.BorderColorIntOpaqueBlack,
	BorderWhite: vk.BorderColorIntOpaqueWhite,
}
