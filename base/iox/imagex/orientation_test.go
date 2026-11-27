// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagex

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exifJPEG returns a JPEG of a 64x32 image, red on the left and blue on
// the right, with an EXIF segment with the given orientation in the given
// byte order, if it is not 0.
func exifJPEG(t *testing.T, orientation int, bo binary.ByteOrder) []byte {
	im := image.NewRGBA(image.Rect(0, 0, 64, 32))
	for y := range 32 {
		for x := range 64 {
			c := color.RGBA{255, 0, 0, 255}
			if x >= 32 {
				c = color.RGBA{0, 0, 255, 255}
			}
			im.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, im, &jpeg.Options{Quality: 95}))
	b := buf.Bytes()
	if orientation == 0 {
		return b
	}
	tiff := make([]byte, 8+2+12+4)
	if bo == binary.LittleEndian {
		copy(tiff, "II")
	} else {
		copy(tiff, "MM")
	}
	bo.PutUint16(tiff[2:], 42)
	bo.PutUint32(tiff[4:], 8)
	bo.PutUint16(tiff[8:], 1)       // one entry
	bo.PutUint16(tiff[10:], 0x0112) // Orientation
	bo.PutUint16(tiff[12:], 3)      // SHORT
	bo.PutUint32(tiff[14:], 1)
	bo.PutUint16(tiff[18:], uint16(orientation))
	seg := append([]byte("Exif\x00\x00"), tiff...)
	app1 := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(app1[2:], uint16(len(seg)+2))
	app1 = append(app1, seg...)
	return append(append([]byte{0xFF, 0xD8}, app1...), b[2:]...)
}

// isRed returns whether the given color is mostly red (else blue).
func isRed(c color.Color) bool {
	r, _, b, _ := c.RGBA()
	return r > b
}

// TestJPEGOrientation tests that JPEG images are turned upright as their
// EXIF orientation says, as photos of phones taken sideways.
func TestJPEGOrientation(t *testing.T) {
	for _, bo := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		b := exifJPEG(t, 6, bo)
		assert.Equal(t, 6, jpegOrientation(b))
		im, f, err := Read(bytes.NewReader(b))
		require.NoError(t, err)
		assert.Equal(t, JPEG, f)
		assert.Equal(t, image.Pt(32, 64), im.Bounds().Size(), "turned 90° clockwise")
		assert.True(t, isRed(im.At(16, 8)), "the left side on the top")
		assert.False(t, isRed(im.At(16, 56)))
	}
	cases := []struct {
		o             int
		size          image.Point
		redAt, blueAt image.Point
	}{
		{1, image.Pt(64, 32), image.Pt(8, 16), image.Pt(56, 16)},
		{2, image.Pt(64, 32), image.Pt(56, 16), image.Pt(8, 16)},
		{3, image.Pt(64, 32), image.Pt(56, 16), image.Pt(8, 16)},
		{4, image.Pt(64, 32), image.Pt(8, 16), image.Pt(56, 16)},
		{5, image.Pt(32, 64), image.Pt(16, 8), image.Pt(16, 56)},
		{7, image.Pt(32, 64), image.Pt(16, 56), image.Pt(16, 8)},
		{8, image.Pt(32, 64), image.Pt(16, 56), image.Pt(16, 8)},
	}
	for _, c := range cases {
		im, _, err := Read(bytes.NewReader(exifJPEG(t, c.o, binary.BigEndian)))
		require.NoError(t, err)
		assert.Equal(t, c.size, im.Bounds().Size(), "orientation %d", c.o)
		assert.True(t, isRed(im.At(c.redAt.X, c.redAt.Y)), "orientation %d", c.o)
		assert.False(t, isRed(im.At(c.blueAt.X, c.blueAt.Y)), "orientation %d", c.o)
	}
	assert.Equal(t, 1, jpegOrientation(exifJPEG(t, 0, nil)), "no EXIF")
	assert.Equal(t, 1, jpegOrientation([]byte("not a jpeg")))
}
