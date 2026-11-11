// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !novideo

package video

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeFrames decodes the first n video frames of the media file,
// scaled to the given size, along with the audio samples up to them.
func decodeFrames(t *testing.T, fpath string, n, width, height int) ([]*image.RGBA, [][2]float64) {
	d, err := openDecoder(fpath, nil, true)
	require.NoError(t, err)
	defer d.free()
	d.scaleWidth, d.scaleHeight = width, height
	var frames []*image.RGBA
	var samples [][2]float64
	err = d.run(decodeSink{
		video: func(f Frame) bool {
			frames = append(frames, f.Image)
			return len(frames) < n
		},
		audio: func(s [][2]float64) bool {
			samples = append(samples, s...)
			return true
		},
	})
	require.NoError(t, err)
	return frames, samples
}

// colorful returns the number of distinct colors in a sample of the image,
// to check that it is not blank.
func colorful(img *image.RGBA) int {
	colors := map[[4]uint8]bool{}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += max(1, b.Dy()/32) {
		for x := b.Min.X; x < b.Max.X; x += max(1, b.Dx()/32) {
			i := img.PixOffset(x, y)
			colors[[4]uint8(img.Pix[i:i+4])] = true
		}
	}
	return len(colors)
}

func TestOpenMedia(t *testing.T) {
	m, err := OpenMedia("../examples/video/countdown.mp4", nil)
	require.NoError(t, err)
	assert.Equal(t, 320, m.Width)
	assert.Equal(t, 240, m.Height)
	assert.InDelta(t, 29.97, m.FrameRate, 0.01)
	assert.True(t, m.HasAudio)

	_, err = OpenMedia("nonexistent.mp4", nil)
	assert.Error(t, err)
}

func TestDecode(t *testing.T) {
	frames, samples := decodeFrames(t, "../examples/video/countdown.mp4", 30, 0, 0)
	require.Len(t, frames, 30)
	for _, f := range frames {
		assert.Equal(t, image.Rect(0, 0, 320, 240), f.Bounds())
	}
	assert.Greater(t, colorful(frames[29]), 4)
	// About one second of 44.1 kHz audio is interleaved with 30 frames.
	assert.Greater(t, len(samples), sampleRate/2)
}

func TestDecodeScaled(t *testing.T) {
	// deer.mp4 is 3840x2176; scaling by width keeps the aspect ratio.
	frames, _ := decodeFrames(t, "../examples/video/deer.mp4", 3, 480, 0)
	require.Len(t, frames, 3)
	assert.Equal(t, image.Rect(0, 0, 480, 272), frames[0].Bounds())
	assert.Greater(t, colorful(frames[2]), 4)
}

func TestDecodeAll(t *testing.T) {
	// Decode all of the frames to the end, including the decoder flush.
	m, err := OpenMedia("../examples/video/countdown.mp4", nil)
	require.NoError(t, err)
	frames, samples := decodeFrames(t, m.Path, 1<<30, 32, 24)
	assert.Greater(t, len(frames), 100)
	seconds := float64(len(frames)) / m.FrameRate
	assert.InDelta(t, seconds, float64(len(samples))/sampleRate, 0.5)
}
