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

// SetMultisample sets the number of multisampling to decrease aliasing
// 4 is typically sufficient.  Values must be power of 2.
func (im *ImageFormat) SetMultisample(nsamp int) {
	ns := vk.SampleCount1Bit
	switch nsamp {
	case 2:
		ns = vk.SampleCount2Bit
	case 4:
		ns = vk.SampleCount4Bit
	case 8:
		ns = vk.SampleCount8Bit
	case 16:
		ns = vk.SampleCount16Bit
	case 32:
		ns = vk.SampleCount32Bit
	case 64:
		ns = vk.SampleCount64Bit
	}
	im.Samples = ns
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

/////////////////////////////////////////////////////////////////////
// Image

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
	Host   HostImage       `desc:"host memory buffer representation of the image"`
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

// ConfigGoImage configures the image for storing the given GoImage.
// Does not call SetGoImage -- this is for configuring a Val for
// an image prior to allocating memory. Once memory is allocated
// then SetGoImage can be called.
func (im *Image) ConfigGoImage(img *image.RGBA) {
	im.Format.Defaults()
	im.Format.Size = img.Rect.Size()
}

// SetGoImage sets staging image data from an *image.RGBA standard Go image,
// Only works if IsHostActive and Format is default vk.FormatR8g8b8a8Srgb,
// Uses very efficient direct copy of bytes -- most efficiently if the
// size and stride is the same, but also works row-by-row if not.
// Must still call AllocImage to have image allocated on the host.
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

// SetVkImage sets a Vk Image and configures a default 2D view
// based on existing format information (which must be set properly).
// Any exiting view is destroyed first.  Must pass the relevant device.
func (im *Image) SetVkImage(dev vk.Device, img vk.Image) {
	im.Image = img
	im.Dev = dev
	im.ConfigStdView()
}

// ConfigDepthImage configures this image as a depth image
// using given depth image format, and other on format information
// from the render image format.
func (im *Image) ConfigDepthImage(dev vk.Device, depthType Types, imgFmt *ImageFormat) {
	im.Dev = dev
	im.Format.Format = depthType.VkType()
	im.Format.Samples = imgFmt.Samples
	im.SetFlag(int(DepthImage))
	im.SetSize(imgFmt.Size)
	im.ConfigDepthView()
}

// ConfigStdView configures a standard 2D image view, for current image,
// format, and device.
func (im *Image) ConfigStdView() {
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

// ConfigDepthView configures a depth view image
func (im *Image) ConfigDepthView() {
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
			AspectMask: vk.ImageAspectFlags(vk.ImageAspectDepthBit),
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
	im.ClearFlag(int(ImageActive))
	vk.DestroyImageView(im.Dev, im.View, nil)
	im.View = nil
}

// FreeImage frees device memory version of image that we own
func (im *Image) FreeImage() {
	im.DestroyView()
	if im.Image == nil || !im.IsImageOwner() {
		return
	}
	im.ClearFlag(int(ImageOwnsImage))
	vk.FreeMemory(im.Dev, im.Mem, nil)
	vk.DestroyImage(im.Dev, im.Image, nil)
	im.Mem = nil
	im.Image = nil
}

// FreeHost frees memory in host buffer representation of image
// Only if we own the host buffer.
func (im *Image) FreeHost() {
	if im.Host.Size == 0 || !im.IsHostOwner() {
		return
	}
	im.ClearFlag(int(ImageOwnsHost))
	vk.UnmapMemory(im.Dev, im.Host.Mem)
	FreeBuffMem(im.Dev, &im.Host.Mem)
	DestroyBuffer(im.Dev, &im.Host.Buff)
	im.Host.SetNil()
}

// Destroy destroys any existing view, nils fields
func (im *Image) Destroy() {
	im.FreeImage()
	im.FreeHost()
	im.DestroyView()
	im.Image = nil
	im.Dev = nil
}

// SetNil sets everything to nil, for shared image
func (im *Image) SetNil() {
	im.View = nil
	im.Image = nil
	im.Dev = nil
	im.Host.SetNil()
	im.Flags = 0
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
	if im.IsImageOwner() || im.HasFlag(DepthImage) {
		im.AllocImage()
	}
	return true
}

// AllocImage allocates the VkImage on the device (must set first),
// based on the current Format info, and other flags.
func (im *Image) AllocImage() {
	im.FreeImage()
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
		SharingMode:   vk.SharingModeExclusive,
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
		im.FreeHost()
	}
	im.Host.Buff = NewBuffer(im.Dev, imsz, vk.BufferUsageTransferSrcBit|vk.BufferUsageTransferDstBit)
	im.Host.Mem = AllocBuffMem(im.Dev, im.Host.Buff, vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit)
	im.Host.Size = imsz
	im.Host.Ptr = MapMemory(im.Dev, im.Host.Mem, im.Host.Size)
	im.Host.Offset = 0
	im.SetFlag(int(ImageOwnsHost), int(ImageHostActive))
}

// ConfigValHost configures host staging buffer from memory buffer for val-owned image
func (im *Image) ConfigValHost(buff *MemBuff, buffPtr unsafe.Pointer, offset int) {
	imsz := im.Format.ByteSize()
	im.Host.Buff = buff.Host
	im.Host.Mem = nil
	im.Host.Size = imsz
	im.Host.Ptr = buffPtr
	im.Host.Offset = offset
	im.SetFlag(int(ImageIsVal), int(ImageHostActive))
}

// CopyRec returns info for this Image for the BufferImageCopy operations
func (im *Image) CopyRec() vk.BufferImageCopy {
	w, h := im.Format.Size32()
	reg := vk.BufferImageCopy{
		BufferOffset:      vk.DeviceSize(im.Host.Offset),
		BufferRowLength:   0, // packed default
		BufferImageHeight: 0,
	}
	reg.ImageSubresource.AspectMask = vk.ImageAspectFlags(vk.ImageAspectColorBit)
	reg.ImageSubresource.LayerCount = 1
	reg.ImageExtent.Width = w
	reg.ImageExtent.Height = h
	return reg
}

/////////////////////////////////////////////////////////////////////
// Transition

// Transition transitions image to new layout
func (im *Image) Transition(cmd *CmdPool, aspectMask vk.ImageAspectFlagBits,
	oldImageLayout, newImageLayout vk.ImageLayout,
	srcAccessMask vk.AccessFlagBits,
	srcStages, dstStages vk.PipelineStageFlagBits) {

	imageMemoryBarrier := vk.ImageMemoryBarrier{
		SType:         vk.StructureTypeImageMemoryBarrier,
		SrcAccessMask: vk.AccessFlags(srcAccessMask),
		DstAccessMask: 0,
		OldLayout:     oldImageLayout,
		NewLayout:     newImageLayout,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask: vk.ImageAspectFlags(aspectMask),
			LayerCount: 1,
			LevelCount: 1,
		},
		Image: im.Image,
	}
	switch newImageLayout {
	case vk.ImageLayoutTransferDstOptimal:
		// make sure anything that was copying from this image has completed
		imageMemoryBarrier.DstAccessMask = vk.AccessFlags(vk.AccessTransferWriteBit)
	case vk.ImageLayoutColorAttachmentOptimal:
		imageMemoryBarrier.DstAccessMask = vk.AccessFlags(vk.AccessColorAttachmentWriteBit)
	case vk.ImageLayoutDepthStencilAttachmentOptimal:
		imageMemoryBarrier.DstAccessMask = vk.AccessFlags(vk.AccessDepthStencilAttachmentWriteBit)
	case vk.ImageLayoutShaderReadOnlyOptimal:
		imageMemoryBarrier.DstAccessMask =
			vk.AccessFlags(vk.AccessShaderReadBit) | vk.AccessFlags(vk.AccessInputAttachmentReadBit)
	case vk.ImageLayoutTransferSrcOptimal:
		imageMemoryBarrier.DstAccessMask = vk.AccessFlags(vk.AccessTransferReadBit)
	case vk.ImageLayoutPresentSrc:
		imageMemoryBarrier.DstAccessMask = vk.AccessFlags(vk.AccessMemoryReadBit)
	default:
		imageMemoryBarrier.DstAccessMask = 0
	}

	vk.CmdPipelineBarrier(cmd.Buff,
		vk.PipelineStageFlags(srcStages), vk.PipelineStageFlags(dstStages),
		0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{imageMemoryBarrier})
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

func (hi *HostImage) SetNil() {
	hi.Size = 0
	hi.Buff = nil
	hi.Offset = 0
	hi.Mem = nil
	hi.Ptr = nil
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
