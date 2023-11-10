// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package video implements a video player widget in GoGi.
package video

import (
	"bytes"
	"encoding/binary"
	"image"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/zergon321/reisen"
	"goki.dev/gi/v2/gi"
	"goki.dev/ki/v2"
)

// Video represents a video playback widget without any controls.
// See [Player] for a version with controls.
type Video struct {
	gi.Image

	// Media is the video media.
	Media *reisen.Media
}

var _ ki.Ki = (*Video)(nil)

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

// Play starts playing the video.
func (v *Video) Play() error {
	videoFPS, _ := v.Media.Streams()[0].FrameRate()

	// seconds per frame for frame ticker
	spf := 1.0 / float64(videoFPS)
	frameDuration := time.Duration(float64(time.Second) * spf)

	// Start decoding streams.
	var sampleSource <-chan [2]float64
	frameBuffer, sampleSource, errChan, err := v.ReadVideoAndAudio()

	if err != nil {
		return err
	}

	// Start playing audio samples.
	speaker.Play(v.StreamSamples(sampleSource))

	tick := time.NewTicker(frameDuration)
	for range tick.C {
		// Check for incoming errors.
		select {
		case err, ok := <-errChan:
			if ok {
				return err
			}
		default:
		}

		frame, ok := <-frameBuffer
		if ok {
			v.SetImage(frame, 0, 0)
		}
	}
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

	/*err = v.Media.Streams()[0].Rewind(60 * time.Second)

	if err != nil {
		return nil, nil, nil, err
	}*/

	/*err = v.Media.Streams()[0].ApplyFilter("h264_mp4toannexb")

	if err != nil {
		return nil, nil, nil, err
	}*/

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

			/*hash := sha256.Sum256(packet.Data())
			fmt.Println(base58.Encode(hash[:]))*/

			switch packet.Type() {
			case reisen.StreamVideo:
				s := v.Media.Streams()[packet.StreamIndex()].(*reisen.VideoStream)
				videoFrame, gotFrame, err := s.ReadVideoFrame()

				if err != nil {
					go func(err error) {
						errs <- err
					}(err)
				}

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

				if err != nil {
					go func(err error) {
						errs <- err
					}(err)
				}

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
