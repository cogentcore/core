// Copyright (c) 2022, Cogent Core. All rights reserved.
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

	"cogentcore.org/core/enums"
	"cogentcore.org/core/grows/images"
	"cogentcore.org/core/mat32"
	vk "github.com/goki/vulkan"
)

// SRGBToLinearComp converts an sRGB rgb component to linear space (removes gamma).
// Used in converting from sRGB to XYZ colors.
func SRGBToLinearComp(srgb float32) float32 {
	if srgb <= 0.04045 {
		return srgb / 12.92
	}
	return mat32.Pow((srgb+0.055)/1.055, 2.4)
}

// SRGBFmLinearComp converts an sRGB rgb linear component
// to non-linear (gamma corrected) sRGB value
// Used in converting from XYZ to sRGB.
func SRGBFmLinearComp(lin float32) float32 {
	if lin <= 0.0031308 {
		return 12.92 * lin
	}
	return (1.055*mat32.Pow(lin, 1/2.4) + 0.055)
}

// SRGBToLinear converts set of sRGB components to linear values,
// removing gamma correction.
func SRGBToLinear(r, g, b float32) (rl, gl, bl float32) {
	rl = SRGBToLinearComp(r)
	gl = SRGBToLinearComp(g)
	bl = SRGBToLinearComp(b)
	return
}

// SRGBFmLinear converts set of sRGB components from linear values,
// adding gamma correction.
func SRGBFmLinear(rl, gl, bl float32) (r, g, b float32) {
	r = SRGBFmLinearComp(rl)
	g = SRGBFmLinearComp(gl)
	b = SRGBFmLinearComp(bl)
	return
}

func ImgCompToUint8(val float32) uint8 {
	if val > 1.0 {
		val = 1.0
	}
	return uint8(val * float32(0xff))
}

// ImageSRGBFmLinear returns a sRGB colorspace version of given linear
// colorspace image
func ImageSRGBFmLinear(img *image.RGBA) *image.RGBA {
	out := image.NewRGBA(img.Rect)
	sz := len(img.Pix)
	tof := 1.0 / float32(0xff)
	for i := 0; i < sz; i += 4 {
		r := float32(img.Pix[i]) * tof
		g := float32(img.Pix[i+1]) * tof
		b := float32(img.Pix[i+2]) * tof
		a := img.Pix[i+3]
		rs, gs, bs := SRGBFmLinear(r, g, b)
		out.Pix[i] = ImgCompToUint8(rs)
		out.Pix[i+1] = ImgCompToUint8(gs)
		out.Pix[i+2] = ImgCompToUint8(bs)
		out.Pix[i+3] = a
	}
	return out
}

// ImageSRGBToLinear returns a linear colorspace version of sRGB
// colorspace image
func ImageSRGBToLinear(img *image.RGBA) *image.RGBA {
	out := image.NewRGBA(img.Rect)
	sz := len(img.Pix)
	tof := 1.0 / float32(0xff)
	for i := 0; i < sz; i += 4 {
		r := float32(img.Pix[i]) * tof
		g := float32(img.Pix[i+1]) * tof
		b := float32(img.Pix[i+2]) * tof
		a := img.Pix[i+3]
		rs, gs, bs := SRGBToLinear(r, g, b)
		out.Pix[i] = ImgCompToUint8(rs)
		out.Pix[i+1] = ImgCompToUint8(gs)
		out.Pix[i+2] = ImgCompToUint8(bs)
		out.Pix[i+3] = a
	}
	return out
}

// SetImageSRGBFmLinear sets in place the pixel values to sRGB colorspace
// version of given linear colorspace image.
// This directly modifies the given image!
func SetImageSRGBFmLinear(img *image.RGBA) {
	sz := len(img.Pix)
	tof := 1.0 / float32(0xff)
	for i := 0; i < sz; i += 4 {
		r := float32(img.Pix[i]) * tof
		g := float32(img.Pix[i+1]) * tof
		b := float32(img.Pix[i+2]) * tof
		a := img.Pix[i+3]
		rs, gs, bs := SRGBFmLinear(r, g, b)
		img.Pix[i] = ImgCompToUint8(rs)
		img.Pix[i+1] = ImgCompToUint8(gs)
		img.Pix[i+2] = ImgCompToUint8(bs)
		img.Pix[i+3] = a
	}
}

// SetImageSRGBToLinear sets in place the pixel values to linear colorspace
// version of sRGB colorspace image.
// This directly modifies the given image!
func SetImageSRGBToLinear(img *image.RGBA) {
	sz := len(img.Pix)
	tof := 1.0 / float32(0xff)
	for i := 0; i < sz; i += 4 {
		r := float32(img.Pix[i]) * tof
		g := float32(img.Pix[i+1]) * tof
		b := float32(img.Pix[i+2]) * tof
		a := img.Pix[i+3]
		rs, gs, bs := SRGBToLinear(r, g, b)
		img.Pix[i] = ImgCompToUint8(rs)
		img.Pix[i+1] = ImgCompToUint8(gs)
		img.Pix[i+2] = ImgCompToUint8(bs)
		img.Pix[i+3] = a
	}
}

// ImageToRGBA returns image.RGBA version of given image
// either because it already is one, or by converting it.
func ImageToRGBA(img image.Image) *image.RGBA {
	rimg, ok := img.(*image.RGBA)
	if !ok {
		rimg = images.CloneAsRGBA(img)
	}
	return rimg
}

// ImageFormat describes the size and vulkan format of an Image
// If Layers > 1, all must be the same size.
type ImageFormat struct {

	// Size of image
	Size image.Point

	// Image format -- FormatR8g8b8a8Srgb is a standard default
	Format vk.Format

	// number of samples -- set higher for Framebuffer rendering but otherwise default of SampleCount1Bit
	Samples vk.SampleCountFlagBits

	// number of layers for texture arrays
	Layers int
}

// NewImageFormat returns a new ImageFormat with default format and given size
// and number of layers
func NewImageFormat(width, height, layers int) *ImageFormat {
	im := &ImageFormat{}
	im.Defaults()
	im.Size = image.Point{X: width, Y: height}
	im.Layers = layers
	return im
}

func (im *ImageFormat) Defaults() {
	im.Format = vk.FormatR8g8b8a8Srgb
	im.Samples = vk.SampleCount1Bit
	im.Layers = 1
}

// String returns human-readable version of format
func (im *ImageFormat) String() string {
	return fmt.Sprintf("Size: %v  Format: %s  MultiSample: %d  Layers: %d", im.Size, ImageFormatNames[im.Format], im.Samples, im.Layers)
}

// IsStdRGBA returns true if image format is the standard vk.FormatR8g8b8a8Srgb format
// which is compatible with go image.RGBA format.
func (im *ImageFormat) IsStdRGBA() bool {
	return im.Format == vk.FormatR8g8b8a8Srgb
}

// IsRGBAUnorm returns true if image format is the vk.FormatR8g8b8a8Unorm format
// which is compatible with go image.RGBA format with colorspace conversion.
func (im *ImageFormat) IsRGBAUnorm() bool {
	return im.Format == vk.FormatR8g8b8a8Unorm
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

// NSamples returns the integer number of samples based on Samples flag setting
func (im *ImageFormat) NSamples() int {
	ns := 1
	switch im.Samples {
	case vk.SampleCount1Bit:
		ns = 1
	case vk.SampleCount2Bit:
		ns = 2
	case vk.SampleCount4Bit:
		ns = 4
	case vk.SampleCount8Bit:
		ns = 8
	case vk.SampleCount16Bit:
		ns = 16
	case vk.SampleCount32Bit:
		ns = 32
	case vk.SampleCount64Bit:
		ns = 64
	}
	return ns
}

// Size32 returns size as uint32 values
func (im *ImageFormat) Size32() (width, height uint32) {
	width = uint32(im.Size.X)
	height = uint32(im.Size.Y)
	return
}

// Aspect returns the aspect ratio X / Y
func (im *ImageFormat) Aspect() float32 {
	if im.Size.Y > 0 {
		return float32(im.Size.X) / float32(im.Size.Y)
	}
	return 1.3
}

// Bounds returns the rectangle defining this image: 0,0,w,h
func (im *ImageFormat) Bounds() image.Rectangle {
	return image.Rectangle{Max: im.Size}
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

// LayerByteSize returns number of bytes required to represent one layer of
// image in Host memory.  TODO only works
// for known formats -- need to add more as needed.
func (im *ImageFormat) LayerByteSize() int {
	return im.BytesPerPixel() * im.Size.X * im.Size.Y
}

// TotalByteSize returns total number of bytes required to represent all layers of
// images in Host memory.  TODO only works
// for known formats -- need to add more as needed.
func (im *ImageFormat) TotalByteSize() int {
	return im.LayerByteSize() * im.Layers
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

	// name of the image -- e.g., same as Val name if used that way -- helpful for debugging -- set to filename if loaded from a file and otherwise empty
	Name string

	// bit flags for image state, for indicating nature of ownership and state
	Flags ImageFlags

	// format & size of image
	Format ImageFormat

	// vulkan image handle, in device memory
	Image vk.Image `view:"-"`

	// vulkan image view
	View vk.ImageView `view:"-"`

	// memory for image when we allocate it
	Mem vk.DeviceMemory `view:"-"`

	// keep track of device for destroying view
	Dev vk.Device `view:"-"`

	// host memory buffer representation of the image
	Host HostImage

	// pointer to our GPU
	GPU *GPU
}

// HasFlag checks if flag is set
// using atomic, safe for concurrent access
func (im *Image) HasFlag(flag ImageFlags) bool {
	return im.Flags.HasFlag(flag)
}

// SetFlag sets flag(s) using atomic, safe for concurrent access
func (im *Image) SetFlag(on bool, flag ...enums.BitFlag) {
	im.Flags.SetFlag(on, flag...)
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

// HostPixels returns host staging pixels at given layer
func (im *Image) HostPixels(layer int) []byte {
	lsz := im.Format.LayerByteSize()
	lstart := lsz * layer
	return im.Host.Pixels()[lstart : lstart+lsz]
}

// GoImage returns an *image.RGBA standard Go image, of the Host
// memory representation at given layer.
// Only works if IsHostActive and Format is default vk.FormatR8g8b8a8Srgb
// (strongly recommended in any case)
func (im *Image) GoImage(layer int) (*image.RGBA, error) {
	if !im.IsHostActive() {
		return nil, fmt.Errorf("vgpu.Image: Go image not available because Host not active: %s", im.Name)
	}
	if !im.Format.IsStdRGBA() && !im.Format.IsRGBAUnorm() {
		return nil, fmt.Errorf("vgpu.Image: Go image not standard RGBA format: %s", im.Name)
	}
	rgba := &image.RGBA{}
	rgba.Pix = im.HostPixels(layer)
	rgba.Stride = im.Format.Stride()
	rgba.Rect = image.Rect(0, 0, im.Format.Size.X, im.Format.Size.Y)
	if im.Format.IsRGBAUnorm() {
		return ImageSRGBFmLinear(rgba), nil
	}
	return rgba, nil
}

// DevGoImage returns an image.RGBA standard Go image version of the HostOnly Device
// memory representation, directly pointing to the source memory.
// This will be valid only as long as that memory is valid, and modifications
// will directly write into the source memory.  You MUST call UnmapDev once
// done using that image memory, at which point it will become invalid.
// This is only for immediate, transitory use of the image
// (e.g., saving or then drawing it into another image).
// See [DevGoImageCopy] for a version that copies into an image.RGBA.
// Only works if ImageOnHostOnly and Format is default
// vk.FormatR8g8b8a8Srgb.
func (im *Image) DevGoImage() (*image.RGBA, error) {
	if !im.HasFlag(ImageOnHostOnly) || im.Mem == vk.NullDeviceMemory {
		return nil, fmt.Errorf("vgpu.Image DevGoImage: Image not available because device Image is not HostOnly, or Mem is nil: %s", im.Name)
	}
	if !im.Format.IsStdRGBA() && !im.Format.IsRGBAUnorm() {
		return nil, fmt.Errorf("vgpu.Image DevGoImage: Device image is not standard RGBA format: %s", im.Format.String())
	}
	ptr := MapMemoryAll(im.Dev, im.Mem)
	subrec := vk.ImageSubresource{}
	subrec.AspectMask = vk.ImageAspectFlags(vk.ImageAspectColorBit)
	subrec.ArrayLayer = 0
	sublay := vk.SubresourceLayout{}
	vk.GetImageSubresourceLayout(im.Dev, im.Image, &subrec, &sublay)
	sublay.Deref()
	offset := int(sublay.Offset)
	size := int(sublay.Size) // im.Format.LayerByteSize()
	pix := (*[ByteCopyMemoryLimit]byte)(ptr)[offset : size+offset]

	rgba := &image.RGBA{}
	rgba.Pix = pix
	rgba.Stride = int(sublay.RowPitch) // im.Format.Stride()
	rgba.Rect = image.Rect(0, 0, im.Format.Size.X, im.Format.Size.Y)
	return rgba, nil
}

// DevGoImageCopy sets the given image.RGBA standard Go image to
// a copy of the HostOnly Device memory representation,
// re-sizing the pixel memory as needed.
// If the image pixels are sufficiently sized, no memory allocation occurs.
// Only works if ImageOnHostOnly, and works best if Format is default
// vk.FormatR8g8b8a8Srgb (strongly recommended in any case).
// If format is vk.FormatR8g8b8a8Unorm, it will be converted to srgb.
func (im *Image) DevGoImageCopy(rgba *image.RGBA) error {
	if !im.HasFlag(ImageOnHostOnly) || im.Mem == vk.NullDeviceMemory {
		return fmt.Errorf("vgpu.Image DevGoImage: Image not available because device Image is not HostOnly, or Mem is nil: %s", im.Name)
	}
	if !im.Format.IsStdRGBA() && !im.Format.IsRGBAUnorm() {
		return fmt.Errorf("vgpu.Image DevGoImage: Device image is not standard RGBA or Unorm format: %s", im.Format.String())
	}

	size := im.Format.LayerByteSize()
	subrec := vk.ImageSubresource{}
	subrec.AspectMask = vk.ImageAspectFlags(vk.ImageAspectColorBit)
	sublay := vk.SubresourceLayout{}
	vk.GetImageSubresourceLayout(im.Dev, im.Image, &subrec, &sublay)
	offset := int(sublay.Offset)
	ptr := MapMemoryAll(im.Dev, im.Mem)
	pix := (*[ByteCopyMemoryLimit]byte)(ptr)[offset : size+offset]

	if cap(rgba.Pix) < size {
		rgba.Pix = make([]byte, size)
	} else {
		rgba.Pix = rgba.Pix[:size]
	}
	copy(rgba.Pix, pix)
	vk.UnmapMemory(im.Dev, im.Mem)
	if im.Format.IsRGBAUnorm() {
		fmt.Println("converting to linear")
		SetImageSRGBFmLinear(rgba)
	}
	rgba.Stride = im.Format.Stride()
	rgba.Rect = image.Rect(0, 0, im.Format.Size.X, im.Format.Size.Y)
	return nil
}

// UnmapDev calls UnmapMemory on the mapped memory for this image,
// set by MapMemoryAll.  This must be called after image is used in
// DevGoImage (only if you use it immediately!)
func (im *Image) UnmapDev() {
	vk.UnmapMemory(im.Dev, im.Mem)
}

// ConfigGoImage configures the image for storing an image
// of the given size, for images allocated in a shared host buffer.
// (i.e., not Var.TextureOwns).  Image format will be set to default
// unless format is already set.  Layers is number of separate images
// of given size allocated in a texture array.
// Once memory is allocated then SetGoImage can be called in a
// second pass.
func (im *Image) ConfigGoImage(sz image.Point, layers int) {
	if im.Format.Format != vk.FormatR8g8b8a8Srgb {
		im.Format.Defaults()
	}
	im.Format.Size = sz
	if layers <= 0 {
		layers = 1
	}
	im.Format.Layers = layers
}

const (
	// FlipY used as named arg for flipping the Y axis of images, etc
	FlipY = true

	// NoFlipY used as named arg for not flipping the Y axis of images
	NoFlipY = false
)

// SetGoImage sets staging image data from a standard Go image at given layer.
// This is most efficiently done using an image.RGBA, but other
// formats will be converted as necessary.
// If flipY is true then the Image Y axis is flipped
// when copying into the image data, so that images will appear
// upright in the standard OpenGL Y-is-up coordinate system.
// If using the Y-is-down Vulkan coordinate system, don't flip.
// Only works if IsHostActive and Image Format is default vk.FormatR8g8b8a8Srgb,
// Must still call AllocImage to have image allocated on the device,
// and copy from this host staging data to the device.
func (im *Image) SetGoImage(img image.Image, layer int, flipY bool) error {
	if !im.IsHostActive() {
		return fmt.Errorf("vgpu.Image.SetGoImage: image cannot be set because Host not active: %s", im.Name)
	}
	if !im.Format.IsStdRGBA() {
		return fmt.Errorf("vgpu.Image: Format is not standard RGBA format: %s", im.Name)
	}
	if img == nil {
		return fmt.Errorf("vgpu.Image: input image is nil: %s", im.Name)
	}
	rimg := ImageToRGBA(img)
	sz := rimg.Rect.Size()
	dpix := im.HostPixels(layer)
	sti := rimg.Rect.Min.Y*rimg.Stride + rimg.Rect.Min.X*4
	spix := rimg.Pix[sti:]
	ssz := len(spix)
	dsz := len(dpix)
	mx := min(ssz, dsz)
	str := im.Format.Stride()
	if rimg.Stride == str && !flipY {
		copy(dpix[:mx], spix[:mx])
		return nil
	}
	rows := min(sz.Y, im.Format.Size.Y)
	rsz := min(rimg.Stride, str)
	dmax := str * rows
	if dmax > dsz {
		return fmt.Errorf("vgpu.Image: image named: %s, format size: %d doesn't fit in actual destination size: %d", im.Name, dmax, dsz)
	}
	sidx := 0
	if flipY {
		didx := (rows - 1) * str
		for rw := 0; rw < rows; rw++ {
			copy(dpix[didx:didx+rsz], spix[sidx:sidx+rsz])
			for ii := didx + rsz; ii < didx+str; ii++ { // zero out = transparent any extra mem
				dpix[ii] = 0
			}
			sidx += rimg.Stride
			didx -= str
		}
	} else {
		didx := 0
		for rw := 0; rw < rows; rw++ {
			copy(dpix[didx:didx+rsz], spix[sidx:sidx+rsz])
			for ii := didx + rsz; ii < didx+str; ii++ { // zero out = transparent any extra mem
				dpix[ii] = 0
			}
			sidx += rimg.Stride
			didx += str
		}
	}
	return nil
}

// SetVkImage sets a Vk Image and configures a default 2D view
// based on existing format information (which must be set properly).
// Any exiting view is destroyed first.  Must pass the relevant device.
func (im *Image) SetVkImage(gp *GPU, dev vk.Device, img vk.Image) {
	im.GPU = gp
	im.Image = img
	im.Dev = dev
	im.ConfigStdView()
}

// ConfigFramebuffer configures this image as a framebuffer image
// using format.  Sets multisampling to 1, layers to 1.
// Only makes a device image -- no host rep.
func (im *Image) ConfigFramebuffer(gp *GPU, dev vk.Device, imgFmt *ImageFormat) {
	im.GPU = gp
	im.Dev = dev
	im.Format.Format = imgFmt.Format
	im.Format.SetMultisample(1)
	im.Format.Layers = 1
	im.SetFlag(true, ImageOwnsImage, FramebufferImage)
	if im.SetSize(imgFmt.Size) {
		im.ConfigStdView()
	}
}

// ConfigDepth configures this image as a depth image
// using given depth image format, and other on format information
// from the render image format.
func (im *Image) ConfigDepth(gp *GPU, dev vk.Device, depthType Types, imgFmt *ImageFormat) {
	im.GPU = gp
	im.Dev = dev
	im.Format.Format = depthType.VkFormat()
	im.Format.Samples = imgFmt.Samples
	im.Format.Layers = 1
	im.SetFlag(true, DepthImage)
	if im.SetSize(imgFmt.Size) {
		im.ConfigDepthView()
	}
}

// ConfigMulti configures this image as a mutisampling image
// using format.  Only makes a device image -- no host rep.
func (im *Image) ConfigMulti(gp *GPU, dev vk.Device, imgFmt *ImageFormat) {
	im.GPU = gp
	im.Dev = dev
	im.Format.Format = imgFmt.Format
	im.Format.Samples = imgFmt.Samples
	im.Format.Layers = 1
	im.SetFlag(true, ImageOwnsImage, FramebufferImage)
	if im.SetSize(imgFmt.Size) {
		im.ConfigStdView()
	}
}

// ConfigStdView configures a standard 2D image view, for current image,
// format, and device.
func (im *Image) ConfigStdView() {
	im.DestroyView()
	var view vk.ImageView
	viewtyp := vk.ImageViewType2d
	if !im.HasFlag(DepthImage) && !im.HasFlag(FramebufferImage) {
		viewtyp = vk.ImageViewType2dArray
	}
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
			LayerCount: uint32(im.Format.Layers),
		},
		ViewType: viewtyp,
		Image:    im.Image,
	}, nil, &view)
	IfPanic(NewError(ret))
	im.View = view
	im.SetFlag(true, ImageActive)
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
	im.SetFlag(true, ImageActive)
}

// DestroyView destroys any existing view
func (im *Image) DestroyView() {
	if im.View == vk.NullImageView {
		return
	}
	im.SetFlag(false, ImageActive)
	vk.DestroyImageView(im.Dev, im.View, nil)
	im.View = vk.NullImageView
}

// FreeImage frees device memory version of image that we own
func (im *Image) FreeImage() {
	if im.Dev == nil {
		return
	}
	vk.DeviceWaitIdle(im.Dev)
	im.DestroyView()
	if im.Image == vk.NullImage || !im.IsImageOwner() {
		return
	}
	im.SetFlag(false, ImageOwnsImage)
	vk.FreeMemory(im.Dev, im.Mem, nil)
	vk.DestroyImage(im.Dev, im.Image, nil)
	im.Mem = vk.NullDeviceMemory
	im.Image = vk.NullImage
}

// FreeHost frees memory in host buffer representation of image
// Only if we own the host buffer.
func (im *Image) FreeHost() {
	if im.Host.Size == 0 || !im.IsHostOwner() {
		return
	}
	im.SetFlag(false, ImageOwnsHost)
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
	im.Image = vk.NullImage
	im.Dev = nil
}

// SetNil sets everything to nil, for shared image
func (im *Image) SetNil() {
	im.View = vk.NullImageView
	im.Image = vk.NullImage
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
	// var imgFlags vk.ImageCreateFlags
	imgType := vk.ImageType2d
	switch {
	case im.HasFlag(DepthImage):
		usage |= vk.ImageUsageDepthStencilAttachmentBit
	case im.HasFlag(FramebufferImage):
		usage |= vk.ImageUsageColorAttachmentBit
		usage |= vk.ImageUsageTransferSrcBit // todo: extra bit to qualify
	default:
		usage |= vk.ImageUsageSampledBit // default is sampled texture
		usage |= vk.ImageUsageTransferDstBit
	}
	if im.IsHostActive() && !im.HasFlag(FramebufferImage) {
		usage |= vk.ImageUsageTransferDstBit
	}
	if im.HasFlag(ImageOnHostOnly) {
		usage |= vk.ImageUsageTransferDstBit
	}

	if im.Format.Layers == 0 {
		im.Format.Layers = 1
	}

	var image vk.Image
	w, h := im.Format.Size32()
	imgcfg := &vk.ImageCreateInfo{
		SType: vk.StructureTypeImageCreateInfo,
		// Flags:     imgFlags,
		ImageType: imgType,
		Format:    im.Format.Format,
		Extent: vk.Extent3D{
			Width:  w,
			Height: h,
			Depth:  1,
		},
		MipLevels:     1,
		ArrayLayers:   uint32(im.Format.Layers),
		Samples:       im.Format.Samples,
		Tiling:        vk.ImageTilingOptimal,
		Usage:         vk.ImageUsageFlags(usage),
		InitialLayout: vk.ImageLayoutUndefined,
		SharingMode:   vk.SharingModeExclusive,
	}

	props := vk.MemoryPropertyDeviceLocalBit
	if im.HasFlag(ImageOnHostOnly) {
		props = vk.MemoryPropertyHostVisibleBit | vk.MemoryPropertyHostCoherentBit
		imgcfg.Tiling = vk.ImageTilingLinear // essential for grabbing
	}

	ret := vk.CreateImage(im.Dev, imgcfg, nil, &image)
	IfPanic(NewError(ret))
	im.Image = image

	var memReqs vk.MemoryRequirements
	vk.GetImageMemoryRequirements(im.Dev, im.Image, &memReqs)
	memReqs.Deref()
	sz := memReqs.Size

	memProps := im.GPU.MemoryProps
	memTypeIndex, _ := FindRequiredMemoryTypeFallback(memProps,
		vk.MemoryPropertyFlagBits(memReqs.MemoryTypeBits), props)
	ma := &vk.MemoryAllocateInfo{
		SType:           vk.StructureTypeMemoryAllocateInfo,
		AllocationSize:  sz,
		MemoryTypeIndex: memTypeIndex,
	}
	var mem vk.DeviceMemory
	ret = vk.AllocateMemory(im.Dev, ma, nil, &mem)
	IfPanic(NewError(ret))

	im.Mem = mem
	ret = vk.BindImageMemory(im.Dev, im.Image, im.Mem, 0)
	IfPanic(NewError(ret))

	im.SetFlag(true, ImageOwnsImage)
}

// AllocHost allocates a staging buffer on the host for the image
// on the device (must set first), based on the current Format info,
// and other flags.  If the existing host buffer is sufficient to hold
// the image, then nothing happens.
func (im *Image) AllocHost() {
	imsz := im.Format.TotalByteSize()
	if im.Host.Size >= imsz {
		return
	}
	if im.Host.Size > 0 {
		im.FreeHost()
	}
	im.Host.Buff = NewBuffer(im.Dev, imsz, vk.BufferUsageTransferSrcBit|vk.BufferUsageTransferDstBit)
	im.Host.Mem = AllocBuffMem(im.GPU, im.Dev, im.Host.Buff, vk.MemoryPropertyHostVisibleBit|vk.MemoryPropertyHostCoherentBit)
	im.Host.Size = imsz
	im.Host.Ptr = MapMemory(im.Dev, im.Host.Mem, im.Host.Size)
	im.Host.Offset = 0
	im.SetFlag(true, ImageOwnsHost, ImageHostActive)
}

// ConfigValHost configures host staging buffer from memory buffer for val-owned image
func (im *Image) ConfigValHost(buff *MemBuff, buffPtr unsafe.Pointer, offset int) {
	if im.IsHostOwner() {
		return
	}
	imsz := im.Format.TotalByteSize()
	im.Host.Buff = buff.Host
	im.Host.Mem = vk.NullDeviceMemory
	im.Host.Size = imsz
	im.Host.Ptr = unsafe.Pointer(uintptr(buffPtr) + uintptr(offset))
	im.Host.Offset = offset
	im.SetFlag(true, ImageIsVal, ImageHostActive)
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
	reg.ImageSubresource.LayerCount = uint32(im.Format.Layers)
	reg.ImageExtent.Width = w
	reg.ImageExtent.Height = h
	reg.ImageExtent.Depth = 1
	return reg
}

// CopyImageRec returns info for this Image for the ImageCopy operations
func (im *Image) CopyImageRec() vk.ImageCopy {
	w, h := im.Format.Size32()
	reg := vk.ImageCopy{}
	reg.SrcSubresource.AspectMask = vk.ImageAspectFlags(vk.ImageAspectColorBit)
	reg.SrcSubresource.LayerCount = uint32(im.Format.Layers)
	reg.DstSubresource.AspectMask = vk.ImageAspectFlags(vk.ImageAspectColorBit)
	reg.DstSubresource.LayerCount = uint32(im.Format.Layers)
	reg.Extent.Width = w
	reg.Extent.Height = h
	reg.Extent.Depth = 1
	return reg
}

/////////////////////////////////////////////////////////////////////
// Transition -- prepare device images for different roles

// https://gpuopen.com/learn/vulkan-barriers-explained/

// TransitionForDst transitions to TransferDstOptimal to prepare
// device image to be copied to.  source stage is as specified.
func (im *Image) TransitionForDst(cmd vk.CommandBuffer, srcStage vk.PipelineStageFlagBits) {
	im.Transition(cmd, im.Format.Format, vk.ImageLayoutUndefined, vk.ImageLayoutTransferDstOptimal, srcStage, vk.PipelineStageTransferBit)
}

// TransitionDstToShader transitions from TransferDstOptimal to TransferShaderReadOnly
func (im *Image) TransitionDstToShader(cmd vk.CommandBuffer) {
	im.Transition(cmd, im.Format.Format, vk.ImageLayoutTransferDstOptimal, vk.ImageLayoutShaderReadOnlyOptimal, vk.PipelineStageTransferBit, vk.PipelineStageFragmentShaderBit)
}

// TransitionDstToGeneral transitions from Dst to General, in prep for copy from dev to host
func (im *Image) TransitionDstToGeneral(cmd vk.CommandBuffer) {
	im.Transition(cmd, im.Format.Format, vk.ImageLayoutTransferDstOptimal, vk.ImageLayoutGeneral, vk.PipelineStageTransferBit, vk.PipelineStageTransferBit)
}

// Transition transitions image to new layout
func (im *Image) Transition(cmd vk.CommandBuffer, format vk.Format, oldLayout, newLayout vk.ImageLayout, srcStage, dstStage vk.PipelineStageFlagBits) {

	imgMemBar := vk.ImageMemoryBarrier{
		SType:               vk.StructureTypeImageMemoryBarrier,
		SrcQueueFamilyIndex: vk.QueueFamilyIgnored,
		DstQueueFamilyIndex: vk.QueueFamilyIgnored,
		OldLayout:           oldLayout,
		NewLayout:           newLayout,
		Image:               im.Image,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit),
			LayerCount: uint32(im.Format.Layers),
			LevelCount: 1,
		},
	}

	switch newLayout {
	case vk.ImageLayoutTransferDstOptimal:
		// make sure anything that was copying from this image has completed
		imgMemBar.DstAccessMask = vk.AccessFlags(vk.AccessTransferWriteBit)

	case vk.ImageLayoutColorAttachmentOptimal:
		imgMemBar.DstAccessMask = vk.AccessFlags(vk.AccessColorAttachmentWriteBit)

	case vk.ImageLayoutDepthStencilAttachmentOptimal:
		imgMemBar.DstAccessMask = vk.AccessFlags(vk.AccessDepthStencilAttachmentWriteBit)

	case vk.ImageLayoutShaderReadOnlyOptimal:
		imgMemBar.DstAccessMask = vk.AccessFlags(vk.AccessShaderReadBit)
		//  | vk.AccessFlags(vk.AccessInputAttachmentReadBit)
		if oldLayout == vk.ImageLayoutTransferDstOptimal {
			imgMemBar.SrcAccessMask = vk.AccessFlags(vk.AccessTransferWriteBit)
		}

	case vk.ImageLayoutTransferSrcOptimal:
		imgMemBar.DstAccessMask = vk.AccessFlags(vk.AccessTransferReadBit)

	case vk.ImageLayoutPresentSrc:
		imgMemBar.DstAccessMask = vk.AccessFlags(vk.AccessMemoryReadBit)

	case vk.ImageLayoutGeneral:
		if oldLayout == vk.ImageLayoutTransferDstOptimal {
			imgMemBar.SrcAccessMask = vk.AccessFlags(vk.AccessTransferWriteBit)
		}
		imgMemBar.DstAccessMask = vk.AccessFlags(vk.AccessMemoryReadBit)

	default:
		imgMemBar.DstAccessMask = 0
	}

	vk.CmdPipelineBarrier(cmd, vk.PipelineStageFlags(srcStage), vk.PipelineStageFlags(dstStage),
		0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{imgMemBar})
}

/////////////////////////////////////////////////////////////////////
// HostImage

// HostImage is the host representation of an Image
type HostImage struct {

	// size in bytes allocated for host representation of image
	Size int

	// buffer for host CPU-visible memory, for staging -- can be owned by us or managed by Memory (for Val)
	Buff vk.Buffer `view:"-"`

	// offset into host buffer, when Buff is Memory managed
	Offset int

	// host CPU-visible memory, for staging, when we manage our own memory
	Mem vk.DeviceMemory `view:"-"`

	// memory mapped pointer into host memory -- remains mapped
	Ptr unsafe.Pointer `view:"-"`
}

func (hi *HostImage) SetNil() {
	hi.Size = 0
	hi.Buff = vk.NullBuffer
	hi.Offset = 0
	hi.Mem = vk.NullDeviceMemory
	hi.Ptr = nil
}

// Pixels returns the byte slice of the pixels for host image
// Only valid if host is active!  No error checking is performed here.
func (hi *HostImage) Pixels() []byte {
	return (*[ByteCopyMemoryLimit]byte)(hi.Ptr)[:hi.Size]
}

/////////////////////////////////////////////////////////////////////
// ImageFlags

// ImageFlags are bitflags for Image state
type ImageFlags int64 //enums:bitflag -trim-prefix Image

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

	// ImageOnHostOnly causes the image to be created only on host visible
	// memory, not on device memory -- no additional host buffer should be created.
	// this is for an ImageGrab image.  layout is LINEAR
	ImageOnHostOnly
)
