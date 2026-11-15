// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"
	"image/color"
	"testing"

	"github.com/oliverbestmann/webgpu/wgpu"
	"github.com/stretchr/testify/assert"
)

// TestTextureFormat tests the sizes of texture formats.
func TestTextureFormat(t *testing.T) {
	tf := NewTextureFormat(10, 4, 3)
	assert.True(t, tf.IsStdRGBA())
	assert.False(t, tf.IsRGBAUnorm())
	assert.Equal(t, 4, tf.BytesPerPixel())
	assert.Equal(t, 40, tf.Stride())
	assert.Equal(t, 160, tf.LayerByteSize())
	assert.Equal(t, 480, tf.TotalByteSize())
	assert.Equal(t, float32(2.5), tf.Aspect())
	assert.Equal(t, image.Rect(0, 0, 10, 4), tf.Bounds())
	w, h := tf.Size32()
	assert.Equal(t, uint32(10), w)
	assert.Equal(t, uint32(4), h)
	assert.Equal(t, uint32(3), tf.Extent3D().DepthOrArrayLayers)
	assert.Contains(t, tf.String(), "Layers: 3")

	tf.Set(2, 0, wgpu.TextureFormatRGBA8Unorm)
	assert.True(t, tf.IsRGBAUnorm())
	assert.Equal(t, float32(1.3), tf.Aspect(), "a default for a zero height")
	tf.SetFormat(Depth32)
	assert.Equal(t, wgpu.TextureFormatDepth32Float, tf.Format)
	tf.SetMultisample(4)
	assert.Equal(t, 4, tf.Samples)
}

// TestTypes tests the sizes and formats of types.
func TestTypes(t *testing.T) {
	assert.Equal(t, 4, Float32.Bytes())
	assert.Equal(t, 12, Float32Vector3.Bytes())
	assert.Equal(t, 64, Float32Matrix4.Bytes())
	assert.Equal(t, wgpu.IndexFormatUint16, Uint16.IndexType())
	assert.Equal(t, wgpu.IndexFormatUint32, Uint32.IndexType())
	assert.Equal(t, wgpu.VertexFormatFloat32x3, Float32Vector3.VertexFormat())
	assert.Equal(t, wgpu.TextureFormatRGBA8UnormSrgb, TextureRGBA32.TextureFormat())
}

// TestMemSizeAlign tests aligning memory sizes up and down.
func TestMemSizeAlign(t *testing.T) {
	assert.Equal(t, 16, MemSizeAlign(12, 16))
	assert.Equal(t, 32, MemSizeAlign(32, 16))
	assert.Equal(t, 48, MemSizeAlign(33, 16))
	assert.Equal(t, 0, MemSizeAlignDown(12, 16))
	assert.Equal(t, 32, MemSizeAlignDown(47, 16))
	assert.Equal(t, 32, MemSizeAlignDown(32, 16))
}

// TestSRGB tests converting between sRGB and linear colors.
func TestSRGB(t *testing.T) {
	for _, v := range []float32{0, 0.001, 0.04, 0.2, 0.5, 0.9, 1} {
		assert.InDelta(t, v, SRGBFromLinearComp(SRGBToLinearComp(v)), 1e-5, "round trip of %g", v)
	}
	assert.InDelta(t, 0.214, SRGBToLinearComp(0.5), 0.001)
	r, g, b := SRGBToLinear(1, 0.5, 0)
	assert.InDelta(t, 1, r, 1e-6)
	assert.InDelta(t, 0.214, g, 0.001)
	assert.Equal(t, float32(0), b)
	_, g, _ = SRGBFromLinear(r, g, b)
	assert.InDelta(t, 0.5, g, 1e-4)
	assert.Equal(t, uint8(255), ImgCompToUint8(1.2))
	assert.Equal(t, uint8(0), ImgCompToUint8(-1))
	assert.Equal(t, uint8(128), ImgCompToUint8(0.5))

	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.SetRGBA(0, 0, color.RGBA{128, 255, 0, 200})
	lin := TextureSRGBToLinear(img)
	assert.Less(t, lin.RGBAAt(0, 0).R, uint8(128), "linear is darker")
	assert.Equal(t, uint8(200), lin.RGBAAt(0, 0).A, "alpha is kept")
	back := TextureSRGBFromLinear(lin)
	assert.InDelta(t, 128, int(back.RGBAAt(0, 0).R), 2)
	SetTextureSRGBToLinear(img)
	assert.Equal(t, lin.RGBAAt(0, 0), img.RGBAAt(0, 0), "in place")
	SetTextureSRGBFromLinear(img)
	assert.InDelta(t, 128, int(img.RGBAAt(0, 0).R), 2)
}
