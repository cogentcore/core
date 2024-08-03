// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"

	"cogentcore.org/core/base/errors"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// Texture represents a WebGPU Texture with an associated TextureView.
// The WebGPU Texture is in device memory, in an optimized format.
type Texture struct {

	// Name of the texture, e.g., same as Value name if used that way.
	// This is helpful for debugging. Is auto-set to filename if loaded from
	// a file and otherwise empty.
	Name string

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

// SetFromGoImage sets texture data from a standard Go texture at given layer.
// This is most efficiently done using an texture.RGBA, but other
// formats will be converted as necessary.
// This starts the full WriteTexture call to upload to device.
func (tx *Texture) SetFromGoImage(img image.Image, layer int) error {
	rimg := ImageToRGBA(img)
	sz := rimg.Rect.Size()

	tx.Format.Size = sz
	tx.Format.Format = wgpu.TextureFormatRGBA8UnormSrgb
	tx.Format.Layers = 1

	err := tx.CreateTexture(wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst)
	if err != nil { // already logged
		return err
	}

	size := tx.Format.Extent3D()

	// https://www.w3.org/TR/webgpu/#gpuimagecopytexture
	tx.device.Queue.WriteTexture(
		&wgpu.ImageCopyTexture{
			Aspect:   wgpu.TextureAspectAll,
			Texture:  tx.texture,
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
	return nil
}

// CreateTexture creates the texture based on current settings,
// and a view of that texture.  Calls release first.
func (tx *Texture) CreateTexture(usage wgpu.TextureUsage) error {
	tx.Release()

	sz := tx.Format.Size
	size := wgpu.Extent3D{
		Width:              uint32(sz.X),
		Height:             uint32(sz.Y),
		DepthOrArrayLayers: uint32(tx.Format.Layers),
	}
	t, err := tx.device.Device.CreateTexture(&wgpu.TextureDescriptor{
		Label:         tx.Name,
		Size:          size,
		MipLevelCount: 1,
		SampleCount:   uint32(tx.Format.Samples),
		Dimension:     wgpu.TextureDimension2D,
		Format:        tx.Format.Format,
		Usage:         usage,
	})
	if errors.Log(err) != nil {
		return err
	}
	tx.texture = t
	vw, err := t.CreateView(nil)
	if errors.Log(err) != nil {
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
	// tx.device = *dev
	// tx.Format.Format = imgFmt.Format
	// tx.Format.SetMultisample(1)
	// tx.Format.Layers = 1
	// if tx.SetSize(imgFmt.Size) {
	// 	tx.ConfigStdView()
	// }
}

// ConfigDepth configures this texture as a depth texture
// using given depth texture format, and other format information
// from the given render texture format.
// If current texture is identical format, does not recreate.
func (tx *Texture) ConfigDepth(dev *Device, depthType Types, imgFmt *TextureFormat) error {
	tx.device = *dev
	nfmt := *imgFmt
	nfmt.Format = depthType.TextureFormat()
	if tx.texture != nil && tx.Format == nfmt {
		return nil
	}
	tx.Format = nfmt
	return tx.CreateTexture(wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageTextureBinding)
}

// ConfigMulti configures this texture as a mutisampling texture
// using format.
func (tx *Texture) ConfigMulti(dev *Device, imgFmt *TextureFormat) error {
	tx.device = *dev
	if tx.texture != nil && tx.Format == *imgFmt {
		return nil
	}
	tx.Format = *imgFmt
	return tx.CreateTexture(wgpu.TextureUsageRenderAttachment)
}

// ReleaseView destroys any existing view
func (tx *Texture) ReleaseView() {
	if tx.view == nil {
		return
	}
	tx.view.Release()
	tx.view = nil
}

// ReleaseTexture frees device memory version of texture that we own
func (tx *Texture) ReleaseTexture() {
	tx.ReleaseView()
	if tx.texture == nil {
		return
	}
	tx.texture.Release()
	tx.texture = nil
}

// Release destroys any existing view, nils fields
func (tx *Texture) Release() {
	tx.ReleaseTexture()
}

// SetNil sets everything to nil, for shared texture
func (tx *Texture) SetNil() {
	tx.view = nil
	tx.texture = nil
}
