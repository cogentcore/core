// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !novideo

package video

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// errSource is a Source whose streams can not be opened.
type errSource struct{}

func (errSource) Streams(start time.Duration) ([]io.ReadCloser, error) {
	return nil, errors.New("no streams")
}

func (errSource) Duration() time.Duration { return 0 }

// TestOpenSourceErrors tests opening sources that fail: one whose streams
// can not be opened, one whose stream is not media, and one without a
// video stream when audio is not wanted.
func TestOpenSourceErrors(t *testing.T) {
	_, err := OpenSource(errSource{})
	assert.ErrorContains(t, err, "no streams")

	_, err = openSourceDecoder(&junkSource{}, 0, true)
	assert.ErrorContains(t, err, "opening stream 0")

	audio := &fileSource{files: []string{"testdata/frag-audio.mp4"}, piece: 1 << 20}
	_, err = openSourceDecoder(audio, 0, false)
	assert.ErrorContains(t, err, "no video stream")
	d, err := openSourceDecoder(audio, 0, true)
	if assert.NoError(t, err, "only audio, as for an audio element") {
		assert.Nil(t, d.video)
		assert.NotNil(t, d.audio)
		assert.Equal(t, time.Second/30, d.frameTime, "the default")
		d.free()
	}
}

// junkSource is a Source with one stream of bytes that are not media.
type junkSource struct{}

func (*junkSource) Streams(start time.Duration) ([]io.ReadCloser, error) {
	return []io.ReadCloser{&slowReader{data: []byte("this is not a media file at all, just some text"), piece: 1 << 20}}, nil
}

func (*junkSource) Duration() time.Duration { return 0 }

// TestFrameTimeDefault tests the duration of a frame of media without a
// frame rate.
func TestFrameTimeDefault(t *testing.T) {
	p := &Playback{Media: &Media{}}
	assert.Equal(t, time.Second/30, p.frameTime())
	p.Media.FrameRate = 10
	assert.Equal(t, time.Second/10, p.frameTime())
}
