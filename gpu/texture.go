// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/slicesx"
	"github.com/cogentcore/webgpu/wgpu"
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

	// Sampler defines how the texture is sampled on the GPU.
	// Needed for textures used as fragment shader inputs.
	Sampler Sampler

	// indicates that this texture is shared with some other
	// resource, and therefore should not be released when done.
	// Use SetShared method to set texture in this way.
	shared bool

	// WebGPU texture handle, in device memory
	texture *wgpu.Texture `display:"-"`

	// WebGPU texture view -- needed for most textures
	view *wgpu.TextureView `display:"-"`

	// keep track of device for destroying view
	device Device `display:"-"`

	// readBuffer is an optional buffer for reading the contents of a texture
	readBuffer *wgpu.Buffer

	// current size of the readBuffer
	ReadBufferDims TextureBufferDims
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
	rimg := imagex.AsRGBA(img)
	sz := rimg.Rect.Size()

	nfmt := TextureFormat{Size: sz, Format: wgpu.TextureFormatRGBA8UnormSrgb, Layers: 1, Samples: 1}

	if tx.texture == nil || tx.Format != nfmt {
		tx.Format = nfmt
		err := tx.CreateTexture(wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst)
		if err != nil { // already logged
			return err
		}
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
	tx.ReleaseTexture()

	size := tx.Format.Extent3D()
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

// ConfigRenderTexture configures this texture as a render texture
// using format.  Sets multisampling to 1, layers to 1.
func (tx *Texture) ConfigRenderTexture(dev *Device, imgFmt *TextureFormat) error {
	tx.device = *dev
	nfmt := *imgFmt
	nfmt.SetMultisample(1)
	if tx.texture != nil && tx.Format == nfmt {
		return nil
	}
	tx.Format = nfmt
	return tx.CreateTexture(wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageCopySrc | wgpu.TextureUsageTextureBinding)
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
	return tx.CreateTexture(wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageCopySrc)
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

// SetShared sets this texture to point to the given Texture's
// underlying GPU texture, with the shared flag set so that
// it will not be released.
func (tx *Texture) SetShared(ot *Texture) {
	tx.ReleaseTexture()
	tx.texture = ot.texture
	tx.view = ot.view
	tx.shared = true
	tx.Format = ot.Format
}

// ReleaseView destroys any existing view
func (tx *Texture) ReleaseView() {
	if tx.view == nil {
		return
	}
	if !tx.shared {
		tx.view.Release()
	}
	tx.view = nil
}

// ReleaseTexture frees device memory version of texture that we own
func (tx *Texture) ReleaseTexture() {
	tx.ReleaseView()
	if tx.texture == nil {
		return
	}
	if !tx.shared {
		tx.texture.Release()
	}
	tx.shared = false
	tx.texture = nil
}

// Release destroys any existing view, nils fields
func (tx *Texture) Release() {
	tx.ReleaseTexture()
	tx.Sampler.Release()
	if tx.readBuffer != nil {
		tx.readBuffer.Release()
		tx.readBuffer = nil
	}
}

///////////////////////////////////////////////////////////////
// 	ReadBuffer

// ConfigReadBuffer configures the [readBuffer] for this Texture.
// Must have this in place prior to render pass with a
// [CopyToReadBuffer] command added to it.
func (tx *Texture) ConfigReadBuffer() error {
	dims := NewTextureBufferDims(tx.Format.Size)
	buffSize := dims.PaddedSize()

	if tx.readBuffer != nil && tx.ReadBufferDims == *dims {
		return nil
	}
	b, err := tx.device.Device.CreateBuffer(&wgpu.BufferDescriptor{
		Size:  buffSize,
		Usage: wgpu.BufferUsageMapRead | wgpu.BufferUsageCopyDst,
	})
	if errors.Log(err) != nil {
		return err
	}
	tx.readBuffer = b
	tx.ReadBufferDims = *dims
	return nil
}

// CopyToReadBuffer adds a command to the given command encoder
// to copy this texture to its [readBuffer]. Must have called
// [ConfigReadBuffer] prior to start of render pass for this to work.
func (tx *Texture) CopyToReadBuffer(cmd *wgpu.CommandEncoder) error {
	if tx.readBuffer == nil {
		err := errors.New("gpu.Texture.CopyToReadBuffer: must configure readBuffer prior to render pass")
		return errors.Log(err)
	}
	size := tx.Format.Extent3D()
	cmd.CopyTextureToBuffer(
		tx.texture.AsImageCopy(),
		&wgpu.ImageCopyBuffer{
			Buffer: tx.readBuffer,
			Layout: wgpu.TextureDataLayout{
				Offset:       0,
				BytesPerRow:  uint32(tx.ReadBufferDims.PaddedRowSize),
				RowsPerImage: wgpu.CopyStrideUndefined,
			},
		},
		&size,
	)
	return nil
}

// ReadGoImage reads the GPU-resident Texture and returns
// a Go image.NRGBA image of the texture.
func (tx *Texture) ReadGoImage() (*image.NRGBA, error) {
	var data []byte
	err := tx.ReadData(&data, true)
	if err != nil {
		return nil, err
	}
	img := &image.NRGBA{
		Pix:    data,
		Stride: int(tx.ReadBufferDims.PaddedRowSize),
		Rect:   image.Rect(0, 0, int(tx.ReadBufferDims.Width), int(tx.ReadBufferDims.Height)),
	}
	return img, nil
}

// ReadData reads the data from a GPU-resident Texture,
// setting the given data bytes, which will be resized to fit
// the data.  If removePadding is true, then extra padding will be
// removed, if present.
func (tx *Texture) ReadData(data *[]byte, removePadding bool) error {
	ud, err := tx.ReadDataMapped()
	if err != nil {
		return err
	}
	if !removePadding || tx.ReadBufferDims.HasNoPadding() {
		buffSize := tx.ReadBufferDims.PaddedSize()
		*data = slicesx.SetLength(*data, int(buffSize))
		copy(*data, ud)
		tx.UnmapReadData()
		return nil
	}
	dataSize := tx.ReadBufferDims.UnpaddedSize()
	*data = slicesx.SetLength(*data, int(dataSize))
	dims := tx.ReadBufferDims
	for r := range dims.Height {
		dStart := r * dims.UnpaddedRowSize
		dEnd := dStart + dims.UnpaddedRowSize
		sStart := r * dims.PaddedRowSize
		sEnd := sStart + dims.UnpaddedRowSize
		copy((*data)[dStart:dEnd], ud[sStart:sEnd])
	}
	tx.UnmapReadData()
	return nil
}

// ReadDataMapped reads the data from a GPU-resident Texture,
// returning the bytes as mapped from the readBuffer,
// so they must be used immediately, followed by an [UnmapReadData]
// call to unmap the data.  See [ReadData] for a version that copies
// the data into a bytes slice, which is safe for indefinite use.
// There is alignment padding as reflected in the
// [ReadBufferDims] data.
func (tx *Texture) ReadDataMapped() ([]byte, error) {
	dims := NewTextureBufferDims(tx.Format.Size)
	buffSize := dims.PaddedSize()

	if tx.readBuffer == nil || tx.ReadBufferDims != *dims {
		b, err := tx.device.Device.CreateBuffer(&wgpu.BufferDescriptor{
			Size:  buffSize,
			Usage: wgpu.BufferUsageMapRead | wgpu.BufferUsageCopyDst,
		})
		if errors.Log(err) != nil {
			return nil, err
		}
		tx.readBuffer = b
		tx.ReadBufferDims = *dims
	}

	err := BufferReadSync(&tx.device, int(buffSize), tx.readBuffer)
	if errors.Log(err) != nil {
		return nil, err
	}
	return tx.readBuffer.GetMappedRange(0, uint(buffSize)), nil
}

// UnmapReadData unmaps the data from a prior ReadDataMapped call.
func (tx *Texture) UnmapReadData() error {
	if tx.readBuffer == nil {
		return errors.Log(errors.New("gpu.Texture.UnmapReadData: buffer is nil"))
	}
	tx.readBuffer.Unmap()
	return nil
}
