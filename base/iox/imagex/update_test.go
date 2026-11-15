// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagex

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAssertUpdateKeepsSame tests that updating the test images keeps an
// existing image file with the same pixels, whatever its encoding, and
// replaces one with different pixels.
func TestAssertUpdateKeepsSame(t *testing.T) {
	t.Chdir(t.TempDir())
	update := UpdateTestImages
	UpdateTestImages = true
	defer func() { UpdateTestImages = update }()

	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	img.Set(2, 3, color.RGBA{200, 10, 10, 255})
	require.NoError(t, os.MkdirAll("testdata", 0750))
	fname := filepath.Join("testdata", "same.png")
	f, err := os.Create(fname)
	require.NoError(t, err)
	// an encoding that Save does not use
	require.NoError(t, (&png.Encoder{CompressionLevel: png.NoCompression}).Encode(f, img))
	require.NoError(t, f.Close())
	orig, err := os.ReadFile(fname)
	require.NoError(t, err)

	Assert(t, img, "same")
	got, err := os.ReadFile(fname)
	require.NoError(t, err)
	assert.Equal(t, orig, got, "the same pixels keep the file")

	img.Set(4, 4, color.RGBA{0, 0, 255, 255})
	Assert(t, img, "same")
	got, err = os.ReadFile(fname)
	require.NoError(t, err)
	assert.NotEqual(t, orig, got, "different pixels are saved")
	saved, _, err := Open(fname)
	require.NoError(t, err)
	_, content, _, _, _, _ := ImagesEqual(saved, img, 0)
	assert.True(t, content)
}
