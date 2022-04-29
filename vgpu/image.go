// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"fmt"
	"image"
	"log"
	"unsafe"

	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/kit"
	vk "github.com/vulkan-go/vulkan"
)

// ImageFormat describes the size and vulkan format of an Image
type ImageFormat struct {
	Size    image.Point            `desc:"Size of image"`
	Format  vk.Format              `desc:"Image format -- FormatR8g8b8a8Srgb is a standard default"`
	Samples vk.SampleCountFlagBits `desc:"number of samples -- set higher for Framebuffer rendering but otherwise default of SampleCount1Bit"`
}

func (im *ImageFormat) Defaults() {
	im.Format = vk.FormatR8g8b8a8Srgb
	im.Samples = vk.SampleCount1Bit
}

// IsStdRGBA returns true if image format is the standard vk.FormatR8g8b8a8Srgb format
// which is compatible with go image.RGBA format.
func (im *ImageFormat) IsStdRGBA() bool {
	return im.Format == vk.FormatR8g8b8a8Srgb
}

// SetSize sets the width, height
func (im *ImageFormat) SetSize(w, h int) {
	im.Size = image.Point{X: w, Y: h}
}

// Set sets width, height and format
func (im *ImageFormat) Set(w, h int, ft vk.Format) {
	im.SetSize(w, h)
	im.Format = ft
}

// SetFormat sets the format using vgpu standard Types
func (im *ImageFormat) SetFormat(ft Types) {
	im.Format = VulkanTypes[ft]
}

// Size32 returns size as uint32 values
func (im *ImageFormat) Size32() (width, height uint32) {
	width = uint32(im.Size.X)
	height = uint32(im.Size.Y)
	return
}

// BytesPerPixel returns number of bytes required to represent
// one Pixel (in Host memory at least).  TODO only works
// for known formats -- need to add more as needed.
func (im *ImageFormat) BytesPerPixel() int {
	bpp := FormatSizes[im.Format]
	if bpp > 0 {
		return bpp
	}
	log.Println("vgpu.ImageFormat:BytesPerPixel() -- format not yet supported!")
	return 0
}

// ByteSize returns number of bytes required to represent
// image in Host memory.  TODO only works
// for known formats -- need to add more as needed.
func (im *ImageFormat) ByteSize() int {
	return im.BytesPerPixel() * im.Size.X * im.Size.Y
}

// Stride returns number of bytes per image row.  TODO only works
// for known formats -- need to add more as needed.
func (im *ImageFormat) Stride() int {
	return im.BytesPerPixel() * im.Size.X
}

// Image represents a vulkan image with an associated ImageView.
// The vulkan Image is in device memory, in an optimized format.
// There can also be an optional host-visible, plain pixel buffer
// which can be a pointer into a larger buffer or owned by the Image.
type Image struct {
	Name   string          `desc:"name of the image -- e.g., same as Val name if used that way -- helpful for debugging -- set to filename if loaded from a file and otherwise empty"`
	Flags  int32           `desc:"bit flags for image state, for indicating nature of ownership and state"`
	Format ImageFormat     `desc:"format & size of image"`
	Image  vk.Image        `view:"-" desc:"vulkan image handle, in device memory"`
	View   vk.ImageView    `view:"-" desc:"vulkan image view"`
	Mem    vk.DeviceMemory `view:"-" desc:"memory for image when we allocate it"`
	Dev    vk.Device       `view:"-" desc:"keep track of device for destroying view"`
	Host   HostImage       `desc:"host representation of the image"`
}

// HasFlag checks if flag is set
// using atomic, safe for concurrent access
func (im *Image) HasFlag(flag ImageFlags) bool {
	return bitflag.HasAtomic32(&im.Flags, int(flag))
}

// SetFlag sets flag(s) using atomic, safe for concurrent access
func (im *Image) SetFlag(flag ...int) {
	bitflag.SetAtomic32(&im.Flags, flag...)
}

// ClearFlag clears flag(s) using atomic, safe for concurrent access
func (im *Image) ClearFlag(flag ...int) {
	bitflag.ClearAtomic32(&im.Flags, flag...)
}

// IsActive returns true if the image is set and has a view
func (im *Image) IsActive() bool {
	return im.HasFlag(ImageActive)
}

// IsHostActive returns true if the Host accessible version of image is
// active and ready to use
func (im *Image) IsHostActive() bool {
	return im.HasFlag(ImageHostActive)
}

// IsImageOwner returns true if the vk.Image is owned by us
func (im *Image) IsImageOwner() bool {
	return im.HasFlag(ImageOwnsImage)
}

// IsHostOwner returns true if the host buffer is owned by us
func (im *Image) IsHostOwner() bool {
	return im.HasFlag(ImageOwnsHost)
}

// IsVal returns true if the image belongs to a Val
func (im *Image) IsVal() bool {
	return im.HasFlag(ImageIsVal)
}

// GoImage returns an *image.RGBA standard Go image, of the Host
// memory representation.  Only works if IsHostActive and Format
// is default vk.FormatR8g8b8a8Srgb (strongly recommended in any case)
func (im *Image) GoImage() (*image.RGBA, error) {
	if !im.IsHostActive() {
		return nil, fmt.Errorf("vgpu.Image: Go image not available because Host not active: %s", im.Name)
	}
	if !im.Format.IsStdRGBA() {
		return nil, fmt.Errorf("vgpu.Image: Go image not standard RGBA format: %s", im.Name)
	}
	rgba := &image.RGBA{}
	rgba.Pix = im.Host.Pixels()
	rgba.Stride = im.Format.Stride()
	rgba.Rect = image.Rect(0, 0, im.Format.Size.X, im.Format.Size.Y)
	return rgba, nil
}

// SetGoImage sets staging image data from an *image.RGBA standard Go image,
// Only works if IsHostActive and Format is default vk.FormatR8g8b8a8Srgb,
// Uses very efficient direct copy of bytes -- most efficiently if the
// size and stride is the same, but also works row-by-row if not.
func (im *Image) SetGoImage(img *image.RGBA) error {
	if !im.IsHostActive() {
		return fmt.Errorf("vgpu.Image: Go image not available because Host not active: %s", im.Name)
	}
	if !im.Format.IsStdRGBA() {
		return fmt.Errorf("vgpu.Image: Go image not standard RGBA format: %s", im.Name)
	}
	sz := img.Rect.Size()
	dpix := im.Host.Pixels()
	sti := img.Rect.Min.Y*img.Stride + img.Rect.Min.X*4
	spix := img.Pix[sti:]
	str := im.Format.Stride()
	if img.Stride == str {
		mx := ints.MinInt(len(spix), len(dpix))
		copy(dpix[:mx], spix[:mx])
	}
	rows := ints.MinInt(sz.Y, im.Format.Size.Y)
	rsz := ints.MinInt(img.Stride, str)
	sidx := 0
	didx := 0
	for rw := 0; rw < rows; rw++ {
		copy(dpix[didx:didx+rsz], spix[sidx:sidx+rsz])
		didx += str
		sidx += img.Stride
	}
	return nil
}

// SetVkImage sets a Vk Image and generates a default 2D view
// based on existing format information (which must be set properly).
// Any exiting view is destroyed first.  Must pass the relevant device.
func (im *Image) SetVkImage(dev vk.Device, img vk.Image) {
	im.Image = img
	im.Dev = dev
	im.MakeStdView()
}

// MakeStdView makes a standard 2D image view, for current image,
// format, and device.
func (im *Image) MakeStdView() {
	im.DestroyView()
	var view vk.ImageView
	ret := vk.CreateImageView(im.Dev, &vk.ImageViewCreateInfo{
		SType:  vk.StructureTypeImageViewCreateInfo,
		Format: im.Format.Format,
		Components: vk.ComponentMapping{ // this is the default anyway
			R: vk.ComponentSwizzleIdentity,
			G: vk.ComponentSwizzleIdentity,
			B: vk.ComponentSwizzleIdentity,
			A: vk.ComponentSwizzleIdentity,
		},
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit),
			LevelCount: 1,
			LayerCount: 1,
		},
		ViewType: vk.ImageViewType2d,
		Image:    im.Image,
	}, nil, &view)
	IfPanic(NewError(ret))
	im.View = view
	im.SetFlag(int(ImageActive))
}

// DestroyView destroys any existing view
func (im *Image) DestroyView() {
	if im.View == nil {
		return
	}
	vk.DestroyImageView(im.Dev, im.View, nil)
	im.View = nil
	im.ClearFlag(int(ImageActive))
}

// DestroyImage destroys image that we own
func (im *Image) DestroyImage() {
	im.DestroyView()
	if im.Image == nil || !im.IsImageOwner() {
		return
	}
	vk.FreeMemory(im.Dev, im.Mem, nil)
	vk.DestroyImage(im.Dev, im.Image, nil)
	im.Mem = nil
	im.Image = nil
	im.ClearFlag(int(ImageOwnsImage))
}

// DestroyHost destroys host buffer
func (im *Image) DestroyHost() {
	if im.Host.Size == 0 || !im.IsHostOwner() {
		return
	}
	vk.UnmapMemory(im.Dev, im.Host.Mem)
	FreeBuffMem(im.Dev, &im.Host.Mem)
	DestroyBuffer(im.Dev, &im.Host.Buff)
	im.Host.Size = 0
	im.Host.Ptr = nil
	im.ClearFlag(int(ImageOwnsHost))
}

// Destroy destroys any existing view, nils fields
func (im *Image) Destroy() {
	im.DestroyView()
	im.DestroyImage()
	im.DestroyHost()
	im.Image = nil
	im.Dev = nil
}

// SetNil sets everything to nil, for shared image
func (im *Image) SetNil() {
	im.View = nil
	im.Image = nil
	im.Dev = nil
	im.ClearFlag(int(ImageActive))
}

// SetSize sets the size. If the size is not the same as current,
// and Image owns the Host and / or Image, then those are resized.
// returns true if resized.
func (im *Image) SetSize(size image.Point) bool {
	if im.Format.Size == size {
		return false
	}
	im.Format.Size = size
	if im.IsHostOwner() {
		im.AllocHost()
	}
	if im.IsImageOwner() {
		im.AllocImage()
	}
	return true
}

// AllocImage allocates the VkImage on the device (must set first),
// based on the current Format info, and other flags.
func (im *Image) AllocImage() {
	im.DestroyImage()
	var usage vk.ImageUsageFlagBits
	switch {
	case im.HasFlag(DepthImage):
		usage |= vk.ImageUsageDepthStencilAttachmentBit
	case im.HasFlag(FramebufferImage):
		usage |= vk.ImageUsageColorAttachmentBit
	default:
		usage |= vk.ImageUsageSampledBit // default is sampled texture
	}
	if im.IsHostActive() {
		usage |= vk.ImageUsageTransferSrcBit | vk.ImageUsageTransferDstBit
	}

	var image vk.Image
	w, h := im.Format.Size32()
	ret := vk.CreateImage(im.Dev, &vk.ImageCreateInfo{
		SType:     vk.StructureTypeImageCreateInfo,
		ImageType: vk.ImageType2d,
		Format:    im.Format.Format,
		Extent: vk.Extent3D{
			Width:  w,
			Height: h,
			Depth:  1,
		},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       im.Format.Samples,
		Tiling:        vk.ImageTilingOptimal,
		Usage:         vk.ImageUsageFlags(usage),
		InitialLayout: vk.ImageLayoutUndefined,
	}, nil, &image)
	IfPanic(NewError(ret))
	im.Image = image

	props := vk.MemoryPropertyDeviceLocalBit

	var memReqs vk.MemoryRequirements
	vk.GetImageMemoryRequirements(im.Dev, im.Image, &memReqs)
	memReqs.Deref()

	memProps := TheGPU.MemoryProps
	memTypeIndex, _ := FindRequiredMemoryTypeFallback(memProps,
		vk.MemoryPropertyFlagBits(memReqs.MemoryTypeBits), props)
	ma := &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memTypeIndex,
	}
	var mem vk.DeviceMemory
	ret = vk.AllocateMemory(im.Dev, ma, nil, &mem)
	IfPanic(NewError(ret))

	im.Mem = mem
	ret = vk.BindImageMemory(im.Dev, im.Image, im.Mem, 0)
	IfPanic(NewError(ret))

	im.SetFlag(int(ImageOwnsImage))
}

// AllocHost allocates a staging buffer on the host for the image
// on the device (must set first), based on the current Format info,
// and other flags.  If the existing host buffer is sufficient to hold
// the image, then nothing happens.
func (im *Image) AllocHost() {
	imsz := im.Format.ByteSize()
	if im.Host.Size >= imsz {
		return
	}
	if im.Host.Size > 0 {
		im.DestroyHost()
	}
	im.Host.Buff = MakeBuffer(im.Dev, imsz, vk.BufferUsageTransferSrcBit|vk.BufferUsageTransferDstBit)
	im.Host.Mem = AllocBuffMem(im.Dev, im.Host.Buff, vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit)
	im.Host.Size = imsz
	im.Host.Ptr = MapMemory(im.Dev, im.Host.Mem, im.Host.Size)
	im.SetFlag(int(ImageOwnsHost))
}

/////////////////////////////////////////////////////////////////////
// HostImage

// HostImage is the host representation of an Image
type HostImage struct {
	Size   int             `desc:"size in bytes allocated for host representation of image"`
	Buff   vk.Buffer       `view:"-" desc:"buffer for host CPU-visible memory, for staging -- can be owned by us or managed by Memory (for Val)"`
	Offset int             `desc:"offset into host buffer, when Buff is Memory managed"`
	Mem    vk.DeviceMemory `view:"-" desc:"host CPU-visible memory, for staging, when we manage our own memory"`
	Ptr    unsafe.Pointer  `view:"-" desc:"memory mapped pointer into host memory -- remains mapped"`
}

// Pixels returns the byte slice of the pixels for host image
// Only valid if host is active!  No error checking is performed here.
func (hi *HostImage) Pixels() []byte {
	const m = 0x7fffffff
	return (*[m]byte)(hi.Ptr)[:hi.Size]
}

/////////////////////////////////////////////////////////////////////
// ImageFlags

// ImageFlags are bitflags for Image state
type ImageFlags int32

const (
	// ImageActive: the Image and ImageView are configured and ready to use
	ImageActive ImageFlags = iota

	// ImageHostActive: the Host representation of the image is present and
	// ready to be accessed
	ImageHostActive

	// ImageOwnsImage: we own the Vk.Image
	ImageOwnsImage

	// ImageOwnsHost: we own the Host buffer (and it is initialized)
	ImageOwnsHost

	// ImageIsVal: we are a Val image and our Host buffer is shared, with offset.
	// this is incompatible with ImageOwnsHost
	ImageIsVal

	// DepthImage indicates that this is a Depth buffer image
	DepthImage

	// FramebufferImage indicates that this is a Framebuffer image
	FramebufferImage

	ImageFlagsN
)

//go:generate stringer -type=ImageFlags

var KiT_ImageFlags = kit.Enums.AddEnum(ImageFlagsN, kit.BitFlag, nil)
