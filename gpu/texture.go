// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/enums"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// Texture represents a WebGPU Texture with an associated TextureView.
// The WebGPU Texture is in device memory, in an optimized format.
// There can also be an optional host-visible, plain pixel buffer
// which can be a pointer into a larger buffer or owned by the Texture.
type Texture struct {

	// Name of the texture, e.g., same as Value name if used that way.
	// This is helpful for debugging. Is auto-set to filename if loaded from
	// a file and otherwise empty.
	Name string

	// bit flags for texture state, for indicating nature of ownership and state
	Flags TextureFlags

	// Format & size of texture
	Format TextureFormat

	// WebGPU texture handle, in device memory
	texture *wgpu.Texture `display:"-"`

	// WebGPU texture view
	view *wgpu.TextureView `display:"-"`

	// keep track of device for destroying view
	device Device `display:"-"`
}

func NewTexture(dev *Device) *Texture {
	tx := &Texture{}
	tx.device = *dev
	tx.Format.Defaults()
	return tx
}

// HasFlag checks if flag is set
// using atomic, safe for concurrent access
func (tx *Texture) HasFlag(flag TextureFlags) bool {
	return tx.Flags.HasFlag(flag)
}

// SetFlag sets flag(s) using atomic, safe for concurrent access
func (tx *Texture) SetFlag(on bool, flag ...enums.BitFlag) {
	tx.Flags.SetFlag(on, flag...)
}

// IsActive returns true if the texture is set and has a view
func (tx *Texture) IsActive() bool {
	return tx.HasFlag(TextureActive)
}

// IsTextureOwner returns true if the vk.Texture is owned by us
func (tx *Texture) IsTextureOwner() bool {
	return tx.HasFlag(TextureOwnsTexture)
}

/*
// GoImage returns an *texture.RGBA standard Go texture, of the Host
// memory representation at given layer.
// Only works if IsHostActive and Format is default wgpu.TextureFormatR8g8b8a8Srgb
// (strongly recommended in any case)
func (tx *Texture) GoImage(layer int) (*texture.RGBA, error) {
	if !tx.IsHostActive() {
		return nil, fmt.Errorf("gpu.Texture: Go texture not available because Host not active: %s", tx.Name)
	}
	if !tx.Format.IsStdRGBA() && !tx.Format.IsRGBAUnorm() {
		return nil, fmt.Errorf("gpu.Texture: Go texture not standard RGBA format: %s", tx.Name)
	}
	rgba := &texture.RGBA{}
	rgba.Pix = tx.HostPixels(layer)
	rgba.Stride = tx.Format.Stride()
	rgba.Rect = texture.Rect(0, 0, tx.Format.Size.X, tx.Format.Size.Y)
	if tx.Format.IsRGBAUnorm() {
		return TextureSRGBFromLinear(rgba), nil
	}
	return rgba, nil
}

// DevGoImage returns an texture.RGBA standard Go texture version of the HostOnly Device
// memory representation, directly pointing to the source memory.
// This will be valid only as long as that memory is valid, and modifications
// will directly write into the source memory.  You MUST call UnmapDev once
// done using that texture memory, at which point it will become invalid.
// This is only for immediate, transitory use of the texture
// (e.g., saving or then drawing it into another texture).
// See [DevGoImageCopy] for a version that copies into an texture.RGBA.
// Only works if TextureOnHostOnly and Format is default
// wgpu.TextureFormatR8g8b8a8Srgb.
func (tx *Texture) DevGoImage() (*texture.RGBA, error) {
	if !tx.HasFlag(TextureOnHostOnly) || tx.Mem == vk.NullDeviceMemory {
		return nil, fmt.Errorf("gpu.Texture DevGoImage: Texture not available because device Texture is not HostOnly, or Mem is nil: %s", tx.Name)
	}
	if !tx.Format.IsStdRGBA() && !tx.Format.IsRGBAUnorm() {
		return nil, fmt.Errorf("gpu.Texture DevGoImage: Device texture is not standard RGBA format: %s", tx.Format.String())
	}
	ptr := MapMemoryAll(tx.Dev, tx.Mem)
	subrec := vk.TextureSubresource{}
	subrec.AspectMask = vk.TextureAspectFlags(vk.TextureAspectColorBit)
	subrec.ArrayLayer = 0
	sublay := vk.SubresourceLayout{}
	vk.GetTextureSubresourceLayout(tx.Dev, tx.Texture, &subrec, &sublay)
	sublay.Deref()
	offset := int(sublay.Offset)
	size := int(sublay.Size) // im.Format.LayerByteSize()
	pix := (*[ByteCopyMemoryLimit]byte)(ptr)[offset : size+offset]

	rgba := &texture.RGBA{}
	rgba.Pix = pix
	rgba.Stride = int(sublay.RowPitch) // im.Format.Stride()
	rgba.Rect = texture.Rect(0, 0, tx.Format.Size.X, tx.Format.Size.Y)
	return rgba, nil
}

// DevGoImageCopy sets the given texture.RGBA standard Go texture to
// a copy of the HostOnly Device memory representation,
// re-sizing the pixel memory as needed.
// If the texture pixels are sufficiently sized, no memory allocation occurs.
// Only works if TextureOnHostOnly, and works best if Format is default
// wgpu.TextureFormatR8g8b8a8Srgb (strongly recommended in any case).
// If format is wgpu.TextureFormatR8g8b8a8Unorm, it will be converted to srgb.
func (tx *Texture) DevGoImageCopy(rgba *texture.RGBA) error {
	if !tx.HasFlag(TextureOnHostOnly) || tx.Mem == vk.NullDeviceMemory {
		return fmt.Errorf("gpu.Texture DevGoImage: Texture not available because device Texture is not HostOnly, or Mem is nil: %s", tx.Name)
	}
	if !tx.Format.IsStdRGBA() && !tx.Format.IsRGBAUnorm() {
		return fmt.Errorf("gpu.Texture DevGoImage: Device texture is not standard RGBA or Unorm format: %s", tx.Format.String())
	}

	size := tx.Format.LayerByteSize()
	subrec := vk.TextureSubresource{}
	subrec.AspectMask = vk.TextureAspectFlags(vk.TextureAspectColorBit)
	sublay := vk.SubresourceLayout{}
	vk.GetTextureSubresourceLayout(tx.Dev, tx.Texture, &subrec, &sublay)
	offset := int(sublay.Offset)
	ptr := MapMemoryAll(tx.Dev, tx.Mem)
	pix := (*[ByteCopyMemoryLimit]byte)(ptr)[offset : size+offset]

	rgba.Pix = slicesx.SetLength(rgba.Pix, size)
	copy(rgba.Pix, pix)
	vk.UnmapMemory(tx.Dev, tx.Mem)
	if tx.Format.IsRGBAUnorm() {
		fmt.Println("converting to linear")
		SetTextureSRGBFromLinear(rgba)
	}
	rgba.Stride = tx.Format.Stride()
	rgba.Rect = texture.Rect(0, 0, tx.Format.Size.X, tx.Format.Size.Y)
	return nil
}

// UnmapDev calls UnmapMemory on the mapped memory for this texture,
// set by MapMemoryAll.  This must be called after texture is used in
// DevGoImage (only if you use it immediately!)
func (tx *Texture) UnmapDev() {
	vk.UnmapMemory(tx.Dev, tx.Mem)
}
*/

// ConfigGoImage configures the texture for storing an texture
// of the given size. Texture format will be set to default
// unless format is already set.  Layers is number of separate textures
// of given size allocated in a texture array.
func (tx *Texture) ConfigGoImage(sz image.Point, layers int) {
	if tx.Format.Format != wgpu.TextureFormatRGBA8UnormSrgb {
		tx.Format.Defaults()
	}
	tx.Format.Size = sz
	if layers <= 0 {
		layers = 1
	}
	tx.Format.Layers = layers
}

const (
	// FlipY used as named arg for flipping the Y axis of textures, etc
	FlipY = true

	// NoFlipY used as named arg for not flipping the Y axis of textures
	NoFlipY = false
)

// SetFromGoImage sets texture data from a standard Go texture at given layer.
// This is most efficiently done using an texture.RGBA, but other
// formats will be converted as necessary.
// If flipY is true then the Texture Y axis is flipped
// when copying into the texture data, so that textures will appear
// upright in the standard OpenGL Y-is-up coordinate system.
// If using the Y-is-down Vulkan coordinate system, don't flip.
// This starts the full WriteTexture call to upload to device.
func (tx *Texture) SetFromGoImage(img image.Image, layer int, flipY bool) error {
	rimg := ImageToRGBA(img)
	sz := rimg.Rect.Size()

	// todo: deal with layer / array
	// flipY has to be managed using a Draw command presumably --
	// doesn't read through image interface so can't do something easy there.
	tx.Format.Size = sz
	tx.Format.Format = wgpu.TextureFormatRGBA8UnormSrgb
	tx.Format.Layers = 1

	size := wgpu.Extent3D{
		Width:              uint32(sz.X),
		Height:             uint32(sz.Y),
		DepthOrArrayLayers: 1,
	}
	t, err := tx.device.Device.CreateTexture(&wgpu.TextureDescriptor{
		Label:         tx.Name,
		Size:          size,
		MipLevelCount: 1,
		SampleCount:   1,
		Dimension:     wgpu.TextureDimension2D,
		Format:        wgpu.TextureFormatRGBA8UnormSrgb,
		Usage:         wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst,
	})
	if errors.Log(err) != nil {
		return err
	}
	tx.texture = t

	// https://www.w3.org/TR/webgpu/#gpuimagecopytexture
	tx.device.Queue.WriteTexture(
		&wgpu.ImageCopyTexture{
			Aspect:   wgpu.TextureAspectAll,
			Texture:  t,
			MipLevel: 0,
			Origin:   wgpu.Origin3D{X: 0, Y: 0, Z: 0},
		},
		rimg.Pix,
		&wgpu.TextureDataLayout{
			Offset:       0,
			BytesPerRow:  4 * uint32(sz.X),
			RowsPerImage: uint32(sz.Y),
		},
		&size,
	)

	vw, err := t.CreateView(nil)
	if err != nil {
		return err
	}
	tx.view = vw
	return nil
}

// https://eliemichel.github.io/LearnWebGPU/advanced-techniques/headless.html

// ConfigFramebuffer configures this texture as a framebuffer texture
// using format.  Sets multisampling to 1, layers to 1.
// Only makes a device texture -- no host rep.
func (tx *Texture) ConfigFramebuffer(dev *Device, imgFmt *TextureFormat) {
	tx.device = *dev
	tx.Format.Format = imgFmt.Format
	tx.Format.SetMultisample(1)
	tx.Format.Layers = 1
	tx.SetFlag(true, TextureOwnsTexture, FramebufferTexture)
	if tx.SetSize(imgFmt.Size) {
		tx.ConfigStdView()
	}
}

// ConfigDepth configures this texture as a depth texture
// using given depth texture format, and other on format information
// from the render texture format.
func (tx *Texture) ConfigDepth(dev *Device, depthType Types, imgFmt *TextureFormat) {
	tx.device = *dev
	tx.Format.Format = depthType.TextureFormat()
	tx.Format.Samples = imgFmt.Samples
	tx.Format.Layers = 1
	tx.SetFlag(true, DepthTexture)
	if tx.SetSize(imgFmt.Size) {
		tx.ConfigDepthView()
	}
}

// ConfigMulti configures this texture as a mutisampling texture
// using format.  Only makes a device texture -- no host rep.
func (tx *Texture) ConfigMulti(dev *Device, imgFmt *TextureFormat) {
	tx.device = *dev
	tx.Format.Format = imgFmt.Format
	tx.Format.Samples = imgFmt.Samples
	tx.Format.Layers = 1
	tx.SetFlag(true, TextureOwnsTexture, FramebufferTexture)
	if tx.SetSize(imgFmt.Size) {
		tx.ConfigStdView()
	}
}

// ConfigStdView configures a standard 2D texture view, for current texture,
// format, and device.
func (tx *Texture) ConfigStdView() {
	// tx.ReleaseView()
	// var view vk.TextureView
	// viewtyp := vk.TextureViewType2d
	// if !tx.HasFlag(DepthTexture) && !tx.HasFlag(FramebufferTexture) {
	// 	viewtyp = vk.TextureViewType2dArray
	// }
	// ret := vk.CreateTextureView(tx.Dev, &vk.TextureViewCreateInfo{
	// 	SType:  vk.StructureTypeTextureViewCreateInfo,
	// 	Format: tx.Format.Format,
	// 	Components: vk.ComponentMapping{ // this is the default anyway
	// 		R: vk.ComponentSwizzleIdentity,
	// 		G: vk.ComponentSwizzleIdentity,
	// 		B: vk.ComponentSwizzleIdentity,
	// 		A: vk.ComponentSwizzleIdentity,
	// 	},
	// 	SubresourceRange: vk.TextureSubresourceRange{
	// 		AspectMask: vk.TextureAspectFlags(vk.TextureAspectColorBit),
	// 		LevelCount: 1,
	// 		LayerCount: uint32(tx.Format.Layers),
	// 	},
	// 	ViewType: viewtyp,
	// 	Texture:  tx.Texture,
	// }, nil, &view)
	// IfPanic(NewError(ret))
	// tx.View = view
	// tx.SetFlag(true, TextureActive)
}

// ConfigDepthView configures a depth view texture
func (tx *Texture) ConfigDepthView() {
	// tx.ReleaseView()
	// var view vk.TextureView
	// ret := vk.CreateTextureView(tx.Dev, &vk.TextureViewCreateInfo{
	// 	SType:  vk.StructureTypeTextureViewCreateInfo,
	// 	Format: tx.Format.Format,
	// 	Components: vk.ComponentMapping{ // this is the default anyway
	// 		R: vk.ComponentSwizzleIdentity,
	// 		G: vk.ComponentSwizzleIdentity,
	// 		B: vk.ComponentSwizzleIdentity,
	// 		A: vk.ComponentSwizzleIdentity,
	// 	},
	// 	SubresourceRange: vk.TextureSubresourceRange{
	// 		AspectMask: vk.TextureAspectFlags(vk.TextureAspectDepthBit),
	// 		LevelCount: 1,
	// 		LayerCount: 1,
	// 	},
	// 	ViewType: vk.TextureViewType2d,
	// 	Texture:  tx.Texture,
	// }, nil, &view)
	// IfPanic(NewError(ret))
	// tx.View = view
	// tx.SetFlag(true, TextureActive)
}

// ReleaseView destroys any existing view
func (tx *Texture) ReleaseView() {
	if tx.view != nil {
		tx.SetFlag(false, TextureActive)
		tx.view.Release()
		tx.view = nil
	}
}

// ReleaseTexture frees device memory version of texture that we own
func (tx *Texture) ReleaseTexture() {
	tx.ReleaseView()
	if tx.texture == nil || !tx.IsTextureOwner() {
		return
	}
	tx.SetFlag(false, TextureOwnsTexture)
	tx.texture.Release()
	tx.texture = nil
}

// Release destroys any existing view, nils fields
func (tx *Texture) Release() {
	tx.ReleaseTexture()
	tx.ReleaseView()
}

// SetNil sets everything to nil, for shared texture
func (tx *Texture) SetNil() {
	tx.view = nil
	tx.texture = nil
	tx.Flags = 0
}

// SetSize sets the size. If the size is not the same as current,
// and Texture owns the Host and / or Texture, then those are resized.
// returns true if resized.
func (tx *Texture) SetSize(size image.Point) bool {
	if tx.Format.Size == size {
		return false
	}
	tx.Format.Size = size
	// todo: update texture on device!
	return true
}

// AllocTexture allocates the VkTexture on the device (must set first),
// based on the current Format info, and other flags.
func (tx *Texture) AllocTexture() {
	// tx.ReleaseTexture()
	// var usage vk.TextureUsageFlagBits
	// // var imgFlags vk.TextureCreateFlags
	// imgType := vk.TextureType2d
	// switch {
	// case tx.HasFlag(DepthTexture):
	// 	usage |= vk.TextureUsageDepthStencilAttachmentBit
	// case tx.HasFlag(FramebufferTexture):
	// 	usage |= vk.TextureUsageColorAttachmentBit
	// 	usage |= vk.TextureUsageTransferSrcBit // todo: extra bit to qualify
	// default:
	// 	usage |= vk.TextureUsageSampledBit // default is sampled texture
	// 	usage |= vk.TextureUsageTransferDstBit
	// }
	// if tx.IsHostActive() && !tx.HasFlag(FramebufferTexture) {
	// 	usage |= vk.TextureUsageTransferDstBit
	// }
	// if tx.HasFlag(TextureOnHostOnly) {
	// 	usage |= vk.TextureUsageTransferDstBit
	// }
	//
	// if tx.Format.Layers == 0 {
	// 	tx.Format.Layers = 1
	// }
	//
	// var texture *wgpu.Texture
	// w, h := tx.Format.Size32()
	// imgcfg := &vk.TextureCreateInfo{
	// 	SType: vk.StructureTypeTextureCreateInfo,
	// 	// Flags:     imgFlags,
	// 	TextureType: imgType,
	// 	Format:      tx.Format.Format,
	// 	Extent: vk.Extent3D{
	// 		Width:  w,
	// 		Height: h,
	// 		Depth:  1,
	// 	},
	// 	MipLevels:     1,
	// 	ArrayLayers:   uint32(tx.Format.Layers),
	// 	Samples:       tx.Format.Samples,
	// 	Tiling:        vk.TextureTilingOptimal,
	// 	Usage:         vk.TextureUsageFlags(usage),
	// 	InitialLayout: vk.TextureLayoutUndefined,
	// 	SharingMode:   vk.SharingModeExclusive,
	// }
	//
	// properties := vk.MemoryPropertyDeviceLocalBit
	// if tx.HasFlag(TextureOnHostOnly) {
	// 	properties = vk.MemoryPropertyHostVisibleBit | vk.MemoryPropertyHostCoherentBit
	// 	imgcfg.Tiling = vk.TextureTilingLinear // essential for grabbing
	// }
	//
	// ret := vk.CreateTexture(tx.Dev, imgcfg, nil, &texture)
	// IfPanic(NewError(ret))
	// tx.Texture = texture
	//
	// var memReqs vk.MemoryRequirements
	// vk.GetTextureMemoryRequirements(tx.Dev, tx.Texture, &memReqs)
	// memReqs.Deref()
	// sz := memReqs.Size
	//
	// memProperties := tx.GPU.MemoryProperties
	// memTypeIndex, _ := FindRequiredMemoryTypeFallback(memProperties,
	// 	vk.MemoryPropertyFlagBits(memReqs.MemoryTypeBits), properties)
	// ma := &vk.MemoryAllocateInfo{
	// 	SType:           vk.StructureTypeMemoryAllocateInfo,
	// 	AllocationSize:  sz,
	// 	MemoryTypeIndex: memTypeIndex,
	// }
	// var mem vk.DeviceMemory
	// ret = vk.AllocateMemory(tx.Dev, ma, nil, &mem)
	// IfPanic(NewError(ret))
	//
	// tx.Mem = mem
	// ret = vk.BindTextureMemory(tx.Dev, tx.Texture, tx.Mem, 0)
	// IfPanic(NewError(ret))
	//
	// tx.SetFlag(true, TextureOwnsTexture)
}

/////////////////////////////////////////////////////////////////////
// TextureFlags

// TextureFlags are bitflags for Texture state
type TextureFlags int64 //enums:bitflag -trim-prefix Texture

const (
	// TextureActive: the Texture and TextureView are configured and ready to use
	TextureActive TextureFlags = iota

	// TextureOwnsTexture: we own the Texture
	TextureOwnsTexture

	// DepthTexture indicates that this is a Depth buffer texture
	DepthTexture

	// FramebufferTexture indicates that this is a Framebuffer texture
	FramebufferTexture
)
