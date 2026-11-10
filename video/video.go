// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package video implements a video player widget in Cogent Core.
package video

//go:generate core generate

import (
	"image"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

// Video represents a video playback widget without any controls.
// See [Player] for a version with controls.
type Video struct { //types:add
	core.WidgetBase

	// Media is the video media.
	Media *Media

	// degrees of rotation to apply to the video images
	// 90 = left 90, -90 = right 90
	Rotation float32

	// setting this to true will stop the playing
	Stop bool

	frameBuffer <-chan *image.RGBA

	// done is closed to stop the decoding of the current playing.
	done chan struct{}

	// last frame we have rendered
	lastFrame *image.RGBA

	// target frame number to be played
	frameTarg int

	// actual frame number displayed
	framePlayed int

	// frame number to stop playing at, if > 0
	frameStop int
}

func (v *Video) OnAdd() {
	v.WidgetBase.OnAdd()
	v.Scene.AddDirectRender(v)
}

func (v *Video) Destroy() {
	v.Scene.DeleteDirectRender(v)
	v.Stop = true
	v.stopDecoding()
	v.WidgetBase.Destroy()
}

// stopDecoding stops the decoding of the current playing, if any.
func (v *Video) stopDecoding() {
	if v.done != nil {
		close(v.done)
		v.done = nil
	}
}

// CurrentFrame returns the current frame (for rendering),
// and whether it is unchanged or not.
func (v *Video) CurrentFrame() (*image.RGBA, bool) {
	if !v.IsVisible() {
		return nil, true
	}
	frame := v.lastFrame
	unchanged := true
	if v.framePlayed >= v.frameTarg && frame == nil {
		return nil, true
	}
	newFrame, ok := <-v.frameBuffer
	if !ok && frame == nil {
		v.Stop = true
		return nil, true
	}
	if ok {
		frame = newFrame
		v.lastFrame = frame
		unchanged = false
	}
	v.framePlayed++
	return frame, unchanged
}

// Open opens the video specified by the given filepath.
func (v *Video) Open(fpath string) error {
	// Initialize the audio speaker.
	err := speaker.Init(sampleRate, SpeakerSampleRate.N(time.Second/10))
	if err != nil {
		return err
	}

	media, err := OpenMedia(fpath)
	if err != nil {
		return err
	}
	v.Media = media
	return nil
}

// Play starts playing the video at the specified size. Values of 0
// indicate to use the inherent size of the video for that dimension.
func (v *Video) Play(width, height float32) error {
	if v.Media == nil {
		return errors.New("video: no media opened")
	}
	videoFPS := v.Media.FrameRate
	if videoFPS == 0 || videoFPS > 100 {
		videoFPS = 30
	}
	// seconds per frame for frame ticker
	spf := time.Duration(float64(time.Second) / float64(videoFPS))
	// fmt.Println(videoFPS, spf)

	// Start decoding streams.
	v.stopDecoding()
	v.done = make(chan struct{})
	frameBuffer, sampleSource, errChan, err := v.ReadVideoAndAudio(int(width), int(height))

	if err != nil {
		return err
	}

	v.frameBuffer = frameBuffer

	start := time.Now()
	v.Stop = false
	// todo: should set a v.frameStop target, and also get this as an arg
	// to set for playing just a snippet.  probably need more general timestamp etc stuff there.
	v.frameTarg = 0
	v.framePlayed = 0
	v.NeedsRender()
	// Start playing audio samples.
	speaker.Play(v.StreamSamples(sampleSource))
	go func() {
		for err := range errChan {
			errors.Log(err)
		}
	}()

	go func() {
		for {
			if v.Stop {
				return
			}
			d := time.Since(start)
			td := time.Duration(v.frameTarg) * spf
			shouldStop := v.frameStop > 0 && v.frameTarg >= v.frameStop
			if d > td && !shouldStop {
				v.AsyncLock()
				v.frameTarg++
				v.NeedsRender()
				v.AsyncUnlock()
			} else if v.frameTarg > v.framePlayed {
				v.AsyncLock()
				v.NeedsRender()
				v.AsyncUnlock()
			} else if shouldStop {
				return
			} else {
				time.Sleep(td - d)
			}
		}
	}()
	return nil
}

const (
	// frameBufferSize is the number of decoded video frames buffered ahead
	// of playback. Frames can be large (33 MB for 4K RGBA), so this is kept
	// small; the audio buffer covers the interleaving of the streams.
	frameBufferSize = 30

	sampleRate                        = 44100
	channelCount                      = 2
	bitDepth                          = 8
	sampleBufferSize                  = 32 * channelCount * bitDepth * 1024
	SpeakerSampleRate beep.SampleRate = 44100
)

// ReadVideoAndAudio starts decoding the video and audio of the opened media,
// in a separate goroutine that sends the decoded data to the returned channels
// to be played. Video frames are scaled to the given size, where 0 for
// a dimension means to keep the aspect ratio (both 0 = the inherent size).
// Decoding errors are sent to the error channel, which is closed with the
// others at the end of the media, or when the playing is stopped.
func (v *Video) ReadVideoAndAudio(width, height int) (<-chan *image.RGBA, <-chan [2]float64, chan error, error) {
	if v.Media == nil {
		return nil, nil, nil, errors.New("video: no media opened")
	}
	d, err := openDecoder(v.Media.Path)
	if err != nil {
		return nil, nil, nil, err
	}
	d.scaleWidth, d.scaleHeight = width, height
	frameBuffer := make(chan *image.RGBA, frameBufferSize)
	sampleBuffer := make(chan [2]float64, sampleBufferSize)
	errs := make(chan error, 1)
	done := v.done

	go func() {
		defer func() {
			d.free()
			close(frameBuffer)
			close(sampleBuffer)
			close(errs)
		}()
		err := d.run(decodeSink{
			video: func(img *image.RGBA) bool {
				select {
				case frameBuffer <- img:
					return true
				case <-done:
					return false
				}
			},
			audio: func(samples [][2]float64) bool {
				for _, s := range samples {
					select {
					case sampleBuffer <- s:
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

	return frameBuffer, sampleBuffer, errs, nil
}

// StreamSamples creates a new custom streamer for
// playing audio samples provided by the source channel.
//
// See https://github.com/faiface/beep/wiki/Making-own-streamers
// for reference.
func (v *Video) StreamSamples(sampleSource <-chan [2]float64) beep.Streamer {
	return beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		numRead := 0

		for i := 0; i < len(samples); i++ {
			sample, ok := <-sampleSource

			if !ok {
				numRead = i + 1
				break
			}

			samples[i] = sample
			numRead++
		}

		if numRead < len(samples) {
			return numRead, false
		}

		return numRead, true
	})
}
