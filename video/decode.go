// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package video

import (
	"errors"
	"image"
	"time"
)

// Frame is a decoded video frame.
type Frame struct {
	// Image is the image of the frame.
	Image *image.RGBA

	// Time is the presentation time of the frame in the media, which
	// starts at the time of the first frame (not always 0).
	Time time.Duration
}

// decodeSink receives the decoded data; it returns false to stop decoding.
type decodeSink struct {
	video func(f Frame) bool
	audio func(samples [][2]float64) bool
}

const (
	// sampleRate is the rate of the decoded audio samples.
	sampleRate = 44100

	// frameBufferSize is the number of decoded video frames buffered ahead
	// of playback. Frames can be large (33 MB for 4K RGBA), so this is kept
	// small; the audio buffer covers the interleaving of the streams.
	frameBufferSize = 30

	// sampleBufferSize is the number of audio samples buffered ahead of
	// playback (12 seconds).
	sampleBufferSize = 12 * sampleRate
)

// Decode starts decoding the video, and the audio if audio is true (and
// the media has audio), in a separate goroutine that sends the decoded
// data to the returned channels. Video frames are scaled to the given size,
// where 0 for a dimension means to keep the aspect ratio (both 0 = the
// inherent size). The audio samples are stereo at 44.1 kHz. The decoding
// stops at the end of the media, at an error, which is sent to the error
// channel, or when done is closed; all of the channels are closed then.
func (m *Media) Decode(width, height int, audio bool, done <-chan struct{}) (<-chan Frame, <-chan [2]float64, <-chan error, error) {
	return m.DecodeFrom(0, width, height, audio, done)
}

// DecodeFrom is like [Media.Decode], starting at the given time of the
// media: the frames and audio before it are not sent.
func (m *Media) DecodeFrom(start time.Duration, width, height int, audio bool, done <-chan struct{}) (<-chan Frame, <-chan [2]float64, <-chan error, error) {
	// Open the media again, for this decoding, from its start.
	d, err := openDecoder(m.Path, m.Options, audio)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := d.seek(start); err != nil {
		d.free()
		return nil, nil, nil, err
	}
	d.scaleWidth, d.scaleHeight = width, height
	frames := make(chan Frame, frameBufferSize)
	samples := make(chan [2]float64, sampleBufferSize)
	errs := make(chan error, 1)

	// Decode in a goroutine, which stops when done is closed.
	go func() {
		defer func() {
			d.free()
			close(frames)
			close(samples)
			close(errs)
		}()
		err := d.run(decodeSink{
			video: func(f Frame) bool {
				select {
				case frames <- f:
					return true
				case <-done:
					return false
				}
			},
			audio: func(ss [][2]float64) bool {
				for _, s := range ss {
					select {
					case samples <- s:
					case <-done:
						return false
					}
				}
				return true
			},
		})
		if err != nil {
			errs <- err
		}
	}()
	return frames, samples, errs, nil
}

// FirstFrame returns the first video frame of the media, scaled to the
// given size as for [Media.Decode], as to show before the media plays.
func (m *Media) FirstFrame(width, height int) (*image.RGBA, error) {
	done := make(chan struct{})
	defer close(done)
	frames, _, errs, err := m.Decode(width, height, false, done)
	if err != nil {
		return nil, err
	}
	if f, ok := <-frames; ok {
		return f.Image, nil
	}
	if err := <-errs; err != nil {
		return nil, err
	}
	return nil, errors.New("video: the media has no video frames")
}
