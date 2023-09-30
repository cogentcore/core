package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"runtime"
	"time"

	vk "github.com/goki/vulkan"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/zergon321/reisen"
	"goki.dev/vgpu/v2/vdraw"
	"goki.dev/vgpu/v2/vgpu"
)

const (
	width                             = 320
	height                            = 240
	frameBufferSize                   = 1024
	sampleRate                        = 44100
	channelCount                      = 2
	bitDepth                          = 8
	sampleBufferSize                  = 32 * channelCount * bitDepth * 1024
	SpeakerSampleRate beep.SampleRate = 44100
)

// readVideoAndAudio reads video and audio frames
// from the opened media and sends the decoded
// data to che channels to be played.
func readVideoAndAudio(media *reisen.Media) (<-chan *image.RGBA, <-chan [2]float64, chan error, error) {
	frameBuffer := make(chan *image.RGBA,
		frameBufferSize)
	sampleBuffer := make(chan [2]float64, sampleBufferSize)
	errs := make(chan error)

	err := media.OpenDecode()

	if err != nil {
		return nil, nil, nil, err
	}

	videoStream := media.VideoStreams()[0]
	err = videoStream.Open()

	if err != nil {
		return nil, nil, nil, err
	}

	audioStream := media.AudioStreams()[0]
	err = audioStream.Open()

	if err != nil {
		return nil, nil, nil, err
	}

	/*err = media.Streams()[0].Rewind(60 * time.Second)

	if err != nil {
		return nil, nil, nil, err
	}*/

	/*err = media.Streams()[0].ApplyFilter("h264_mp4toannexb")

	if err != nil {
		return nil, nil, nil, err
	}*/

	go func() {
		for {
			packet, gotPacket, err := media.ReadPacket()

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
				s := media.Streams()[packet.StreamIndex()].(*reisen.VideoStream)
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
				s := media.Streams()[packet.StreamIndex()].(*reisen.AudioStream)
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
		media.CloseDecode()
		close(frameBuffer)
		close(sampleBuffer)
		close(errs)
	}()

	return frameBuffer, sampleBuffer, errs, nil
}

// streamSamples creates a new custom streamer for
// playing audio samples provided by the source channel.
//
// See https://github.com/faiface/beep/wiki/Making-own-streamers
// for reference.
func streamSamples(sampleSource <-chan [2]float64) beep.Streamer {
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

func init() {
	// must lock main thread for gpu!
	runtime.LockOSThread()
}

func main() {
	if vgpu.Init() != nil {
		return
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(1024, 768, "vDraw Test", nil, nil)
	vgpu.IfPanic(err)

	// note: for graphics, require these instance extensions before init gpu!
	winext := window.GetRequiredInstanceExtensions()
	gp := vgpu.NewGPU()
	gp.AddInstanceExt(winext...)
	vgpu.Debug = true
	gp.Config("vDraw test")

	// gp.PropsString(true) // print

	surfPtr, err := window.CreateWindowSurface(gp.Instance, nil)
	if err != nil {
		panic(err)
	}
	sf := vgpu.NewSurface(gp, vk.SurfaceFromPointer(surfPtr))

	fmt.Printf("format: %s\n", sf.Format.String())

	// Audio and video logic

	// Initialize the audio speaker.
	err = speaker.Init(sampleRate,
		SpeakerSampleRate.N(time.Second/10))

	if err != nil {
		panic(err)
	}

	// Open the media file.
	media, err := reisen.NewMedia("../videos/deer.mp4")

	if err != nil {
		panic(err)

	}

	// Get the FPS for playing
	// video frames.
	videoFPS, _ := media.Streams()[0].FrameRate()

	if err != nil {
		panic(err)

	}

	// SPF for frame ticker.
	spf := 1.0 / float64(videoFPS)
	frameDuration, err := time.ParseDuration(fmt.Sprintf("%fs", spf))

	if err != nil {
		panic(err)
	}

	// Start decoding streams.
	var sampleSource <-chan [2]float64
	frameBuffer, sampleSource, errChan, err := readVideoAndAudio(media)

	if err != nil {
		panic(err)
	}

	// Start playing audio samples.
	speaker.Play(streamSamples(sampleSource))

	drw := &vdraw.Drawer{}
	drw.YIsDown = true
	drw.ConfigSurface(sf, 16) // requires 2 NDesc

	drw.SetMaxTextures(32) // test resizing

	destroy := func() {
		vk.DeviceWaitIdle(sf.Device.Device)
		drw.Destroy()
		sf.Destroy()
		gp.Destroy()
		window.Destroy()
		vgpu.Terminate()
	}

	stoff := 15 // causes images to wrap around sets, so this tests that..

	update := func(idx int) {
		// Check for incoming errors.
		select {
		case err, ok := <-errChan:
			if ok {
				panic(err)
			}
		default:
		}

		// Read video frames and draw them.
		frame, ok := <-frameBuffer

		if ok {
			drw.SetGoImage(stoff, 0, frame, vgpu.NoFlipY)
			drw.SyncImages()
		}

		descIdx := 0
		if stoff >= vgpu.MaxTexturesPerSet {
			descIdx = 1
		}
		drw.StartDraw(descIdx) // specifically starting with correct descIdx is key..
		drw.Scale(stoff, 0, sf.Format.Bounds(), image.ZR, vdraw.Src, vgpu.NoFlipY)
		// drw.Copy(stoff, 0, image.ZP, image.ZR, vdraw.Src, vgpu.NoFlipY)
		drw.EndDraw()
	}

	frameCount := 0
	stTime := time.Now()

	renderFrame := func() {
		update(frameCount)
		frameCount++

		eTime := time.Now()
		dur := float64(eTime.Sub(stTime)) / float64(time.Second)
		if dur > 100 {
			fps := float64(frameCount) / dur
			fmt.Printf("fps: %.0f\n", fps)
			frameCount = 0
			stTime = eTime
		}
	}

	glfw.PollEvents()
	renderFrame()
	glfw.PollEvents()

	exitC := make(chan struct{}, 2)

	fpsTicker := time.NewTicker(frameDuration)
	for {
		select {
		case <-exitC:
			fpsTicker.Stop()
			destroy()
			return
		case <-fpsTicker.C:
			if window.ShouldClose() {
				exitC <- struct{}{}
				continue
			}
			glfw.PollEvents()
			renderFrame()
		}
	}
}
