// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package video implements a video player widget in Cogent Core.
package video

//go:generate core generate

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/draw"
	"time"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/goosi"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/zergon321/reisen"
)

// Video represents a video playback widget without any controls.
// See [Player] for a version with controls.
//
//gti:add
type Video struct {
	gi.WidgetBase

	// Media is the video media.
	Media *reisen.Media

	// degrees of rotation to apply to the video images
	// 90 = left 90, -90 = right 90
	Rotation float32

	// setting this to true will stop the playing
	Stop bool

	frameBuffer <-chan *image.RGBA

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
	// v.Media.Destroy()
	// todo: frameBuffer
	v.WidgetBase.Destroy()
}

// DirectRenderImage uploads the current view frame directly into the drawer
func (v *Video) DirectRenderImage(drw goosi.Drawer, idx int) {
	if !v.IsVisible() {
		return
	}
	if v.framePlayed >= v.frameTarg {
		return
	}
	frame, ok := <-v.frameBuffer
	if !ok {
		v.Stop = true
		return
	}
	v.framePlayed++
	drw.SetGoImage(idx, 0, frame, goosi.NoFlipY)
}

// DirectRenderDraw draws the current image to RenderWin drawer
func (v *Video) DirectRenderDraw(drw goosi.Drawer, idx int, flipY bool) {
	if !v.IsVisible() {
		return
	}
	bb := v.Geom.TotalBBox
	drw.Scale(idx, 0, bb, image.Rectangle{}, draw.Src, flipY, v.Rotation)
}

// Open opens the video specified by the given filepath.
func (v *Video) Open(fpath string) error {
	// Initialize the audio speaker.
	err := speaker.Init(sampleRate, SpeakerSampleRate.N(time.Second/10))
	if err != nil {
		return err
	}

	media, err := reisen.NewMedia(fpath)
	if err != nil {
		return err
	}
	v.Media = media
	return nil
}

// Play starts playing the video at the specified size. Values of 0
// indicate to use the inherent size of the video for that dimension.
func (v *Video) Play(width, height float32) error {
	videoFPS, _ := v.Media.Streams()[0].FrameRate()

	if videoFPS == 0 || videoFPS > 100 {
		videoFPS = 30
	}
	// seconds per frame for frame ticker
	spf := time.Duration(float64(time.Second) / float64(videoFPS))
	// fmt.Println(videoFPS, spf)

	// Start decoding streams.
	var sampleSource <-chan [2]float64
	frameBuffer, sampleSource, errChan, err := v.ReadVideoAndAudio()

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
	_ = errChan

	go func() {
		for {
			// todo: this is causing everything to stop on my sample video that
			// has an error a ways into it -- maybe need a buffered chan or something?
			// also see commented-out parts where it tries to send the errors
			// or something?
			// select {
			// case err, ok := <-errChan:
			// 	if ok {
			// 		fmt.Println(err)
			// 		// return err
			// 	}
			// default:
			if v.Stop {
				return
			}
			d := time.Now().Sub(start)
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
	frameBufferSize                   = 1024
	sampleRate                        = 44100
	channelCount                      = 2
	bitDepth                          = 8
	sampleBufferSize                  = 32 * channelCount * bitDepth * 1024
	SpeakerSampleRate beep.SampleRate = 44100
)

// ReadVideoAndAudio reads video and audio frames from the opened media and
// sends the decoded data to che channels to be played.
func (v *Video) ReadVideoAndAudio() (<-chan *image.RGBA, <-chan [2]float64, chan error, error) {
	frameBuffer := make(chan *image.RGBA, frameBufferSize)
	sampleBuffer := make(chan [2]float64, sampleBufferSize)
	errs := make(chan error)

	err := v.Media.OpenDecode()

	if err != nil {
		return nil, nil, nil, err
	}

	videoStream := v.Media.VideoStreams()[0]
	err = videoStream.Open()

	if err != nil {
		return nil, nil, nil, err
	}

	audioStream := v.Media.AudioStreams()[0]
	err = audioStream.Open()

	if err != nil {
		return nil, nil, nil, err
	}

	go func() {
		for {
			packet, gotPacket, err := v.Media.ReadPacket()

			if err != nil {
				go func(err error) {
					errs <- err
				}(err)
			}

			if !gotPacket {
				break
			}

			switch packet.Type() {
			case reisen.StreamVideo:
				s := v.Media.Streams()[packet.StreamIndex()].(*reisen.VideoStream)
				videoFrame, gotFrame, err := s.ReadVideoFrame()
				_ = err

				// note: this is causing a panic send on closed channel
				// if err != nil {
				// 	go func(err error) {
				// 		errs <- err
				// 	}(err)
				// }

				if !gotFrame {
					break
				}

				if videoFrame == nil {
					continue
				}

				frameBuffer <- videoFrame.Image()

			case reisen.StreamAudio:
				s := v.Media.Streams()[packet.StreamIndex()].(*reisen.AudioStream)
				audioFrame, gotFrame, err := s.ReadAudioFrame()

				// note: this is causing a panic send on closed channel
				// if err != nil {
				// 	go func(err error) {
				// 		errs <- err
				// 	}(err)
				// }

				if !gotFrame {
					break
				}

				if audioFrame == nil {
					continue
				}

				// Turn the raw byte data into
				// audio samples of type [2]float64.
				reader := bytes.NewReader(audioFrame.Data())

				// See the README.md file for
				// detailed scheme of the sample structure.
				for reader.Len() > 0 {
					sample := [2]float64{0, 0}
					var result float64
					err = binary.Read(reader, binary.LittleEndian, &result)

					if err != nil {
						go func(err error) {
							errs <- err
						}(err)
					}

					sample[0] = result

					err = binary.Read(reader, binary.LittleEndian, &result)

					if err != nil {
						go func(err error) {
							errs <- err
						}(err)
					}

					sample[1] = result
					sampleBuffer <- sample
				}
			}
		}

		videoStream.Close()
		audioStream.Close()
		v.Media.CloseDecode()
		close(frameBuffer)
		close(sampleBuffer)
		close(errs)
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
