// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !novideo

package video

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"math"
	"time"

	"github.com/asticode/go-astiav"
)

// OpenMedia opens the media file at the given path or URL, with the given
// options of FFmpeg for the input (see [Media.Options]), which can be nil,
// and reads its properties.
func OpenMedia(fpath string, options map[string]string) (*Media, error) {
	// Open the file and find its streams, to read their properties.
	d, err := openDecoder(fpath, options, true)
	if err != nil {
		return nil, err
	}
	defer d.free()
	m := &Media{Path: fpath, Options: options, HasAudio: d.audio != nil}
	par := d.video.stream.CodecParameters()
	m.Width, m.Height = par.Width(), par.Height()
	if fr := d.video.stream.AvgFrameRate(); fr.Num() > 0 && fr.Den() > 0 {
		m.FrameRate = fr.Float64()
	}
	return m, nil
}

// decodeStream is an opened decoder for one stream of a media file.
type decodeStream struct {
	stream *astiav.Stream
	codec  *astiav.CodecContext
}

// decoder decodes the video and audio streams of a media file.
type decoder struct {
	input        *astiav.FormatContext
	video, audio *decodeStream
	packet       *astiav.Packet
	frame        *astiav.Frame

	// scaled is the decoded video frame converted to RGBA by scaler,
	// at the size scaleWidth x scaleHeight (0 = inherent size).
	scaled                  *astiav.Frame
	scaler                  *astiav.SoftwareScaleContext
	scaleWidth, scaleHeight int

	// resampled is the decoded audio converted by resampler
	// to interleaved float64 stereo samples at [sampleRate].
	resampled *astiav.Frame
	resampler *astiav.SoftwareResampleContext

	// next is the time of the next video frame, for frames without
	// a timestamp, and frameTime is the duration of a frame.
	next, frameTime time.Duration
}

// openDecoder opens the media file at the given path with the given
// options, and opens a decoder for its first video stream, and for its
// first audio stream, if any and audio.
func openDecoder(fpath string, options map[string]string, audio bool) (*decoder, error) {
	// Open the file and read its stream info.
	d := &decoder{input: astiav.AllocFormatContext()}
	if d.input == nil {
		return nil, errors.New("video: allocating format context failed")
	}
	var dict *astiav.Dictionary
	if len(options) > 0 {
		dict = astiav.NewDictionary()
		defer dict.Free()
		for k, v := range options {
			if err := dict.Set(k, v, 0); err != nil {
				d.input.Free()
				return nil, fmt.Errorf("video: setting option %s: %w", k, err)
			}
		}
	}
	if err := d.input.OpenInput(fpath, nil, dict); err != nil {
		d.input.Free()
		return nil, fmt.Errorf("video: opening %q: %w", fpath, err)
	}
	if err := d.input.FindStreamInfo(nil); err != nil {
		d.free()
		return nil, fmt.Errorf("video: finding stream info of %q: %w", fpath, err)
	}

	// Open a decoder for the first video and first audio stream.
	for _, s := range d.input.Streams() {
		var target **decodeStream
		switch s.CodecParameters().MediaType() {
		case astiav.MediaTypeVideo:
			target = &d.video
		case astiav.MediaTypeAudio:
			if audio {
				target = &d.audio
			}
		}
		if target == nil || *target != nil {
			continue
		}
		ds, err := openDecodeStream(s)
		if err != nil {
			d.free()
			return nil, err
		}
		*target = ds
	}
	if d.video == nil {
		d.free()
		return nil, fmt.Errorf("video: no video stream in %q", fpath)
	}
	d.frameTime = time.Second / 30
	if fr := d.video.stream.AvgFrameRate(); fr.Num() > 0 && fr.Den() > 0 {
		d.frameTime = time.Duration(float64(time.Second) / fr.Float64())
	}

	// Allocate the packet and frames used for decoding.
	d.packet = astiav.AllocPacket()
	d.frame = astiav.AllocFrame()
	d.scaled = astiav.AllocFrame()
	if d.audio != nil {
		d.resampled = astiav.AllocFrame()
		d.resampler = astiav.AllocSoftwareResampleContext()
	}
	return d, nil
}

// openDecodeStream opens a decoder for the given stream.
func openDecodeStream(s *astiav.Stream) (*decodeStream, error) {
	par := s.CodecParameters()
	codec := astiav.FindDecoder(par.CodecID())
	if codec == nil {
		return nil, fmt.Errorf("video: no decoder for codec %s", par.CodecID())
	}
	cc := astiav.AllocCodecContext(codec)
	if cc == nil {
		return nil, errors.New("video: allocating codec context failed")
	}
	if err := par.ToCodecContext(cc); err != nil {
		cc.Free()
		return nil, fmt.Errorf("video: setting codec parameters: %w", err)
	}
	if err := cc.Open(codec, nil); err != nil {
		cc.Free()
		return nil, fmt.Errorf("video: opening codec %s: %w", par.CodecID(), err)
	}
	return &decodeStream{stream: s, codec: cc}, nil
}

// free frees all of the resources of the decoder.
func (d *decoder) free() {
	for _, f := range []*astiav.Frame{d.frame, d.scaled, d.resampled} {
		if f != nil {
			f.Free()
		}
	}
	if d.packet != nil {
		d.packet.Free()
	}
	if d.scaler != nil {
		d.scaler.Free()
	}
	if d.resampler != nil {
		d.resampler.Free()
	}
	for _, s := range []*decodeStream{d.video, d.audio} {
		if s != nil {
			s.codec.Free()
		}
	}
	d.input.CloseInput()
	d.input.Free()
}

// run decodes the whole media, sending the results to the sink,
// until the end of the media, an error, or the sink returns false.
func (d *decoder) run(sink decodeSink) error {
	// Read packets and send them to the decoder of their stream.
	for {
		if err := d.input.ReadFrame(d.packet); err != nil {
			if errors.Is(err, astiav.ErrEof) {
				break
			}
			return fmt.Errorf("video: reading packet: %w", err)
		}
		var s *decodeStream
		switch idx := d.packet.StreamIndex(); {
		case idx == d.video.stream.Index():
			s = d.video
		case d.audio != nil && idx == d.audio.stream.Index():
			s = d.audio
		}
		if s == nil {
			d.packet.Unref()
			continue
		}
		err := s.codec.SendPacket(d.packet)
		d.packet.Unref()
		if err != nil {
			return fmt.Errorf("video: sending packet: %w", err)
		}
		if ok, err := d.receive(s, sink); !ok || err != nil {
			return err
		}
	}

	// Flush the frames remaining in the decoders.
	for _, s := range []*decodeStream{d.video, d.audio} {
		if s == nil {
			continue
		}
		if err := s.codec.SendPacket(nil); err != nil {
			return fmt.Errorf("video: flushing decoder: %w", err)
		}
		if ok, err := d.receive(s, sink); !ok || err != nil {
			return err
		}
	}
	return nil
}

// receive receives all of the frames available from the decoder of
// the given stream, and sends them to the sink. It returns false
// if the sink asked to stop.
func (d *decoder) receive(s *decodeStream, sink decodeSink) (bool, error) {
	for {
		if err := s.codec.ReceiveFrame(d.frame); err != nil {
			if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
				return true, nil
			}
			return false, fmt.Errorf("video: receiving frame: %w", err)
		}
		var ok bool
		var err error
		if s == d.video {
			ok, err = d.sendVideo(sink)
		} else {
			ok, err = d.sendAudio(sink)
		}
		d.frame.Unref()
		if !ok || err != nil {
			return false, err
		}
	}
}

// sendVideo converts the decoded video frame to an RGBA image and sends it.
func (d *decoder) sendVideo(sink decodeSink) (bool, error) {
	// (Re)create the scaler when the source format or size changes.
	f := d.frame
	w, h := d.scaleWidth, d.scaleHeight
	switch {
	case w <= 0 && h <= 0:
		w, h = f.Width(), f.Height()
	case w <= 0:
		w = max(1, f.Width()*h/f.Height())
	case h <= 0:
		h = max(1, f.Height()*w/f.Width())
	}
	if d.scaler == nil || d.scaler.SourceWidth() != f.Width() || d.scaler.SourceHeight() != f.Height() || d.scaler.SourcePixelFormat() != f.PixelFormat() {
		if d.scaler != nil {
			d.scaler.Free()
		}
		var err error
		d.scaler, err = astiav.CreateSoftwareScaleContext(f.Width(), f.Height(), f.PixelFormat(), w, h, astiav.PixelFormatRgba, astiav.NewSoftwareScaleContextFlags(astiav.SoftwareScaleContextFlagBilinear))
		if err != nil {
			d.scaler = nil
			return false, fmt.Errorf("video: creating scaler: %w", err)
		}
		d.scaled.Unref()
	}

	// Convert the frame and copy it into a new image, which the sink keeps.
	if err := d.scaler.ScaleFrame(f, d.scaled); err != nil {
		return false, fmt.Errorf("video: scaling frame: %w", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if err := d.scaled.Data().ToImage(img); err != nil {
		return false, fmt.Errorf("video: copying frame: %w", err)
	}
	t := d.next
	if pts := f.Pts(); pts != astiav.NoPtsValue {
		tb := d.video.stream.TimeBase()
		t = time.Duration(float64(pts) * float64(time.Second) * float64(tb.Num()) / float64(tb.Den()))
	}
	d.next = t + d.frameTime
	return sink.video(Frame{Image: img, Time: t}), nil
}

// sendAudio converts the decoded audio frame to stereo samples and sends them.
func (d *decoder) sendAudio(sink decodeSink) (bool, error) {
	// Resample into a new buffer of interleaved float64 stereo samples.
	r := d.resampled
	r.Unref()
	r.SetChannelLayout(astiav.ChannelLayoutStereo)
	r.SetSampleFormat(astiav.SampleFormatDbl)
	r.SetSampleRate(sampleRate)
	if err := d.resampler.ConvertFrame(d.frame, r); err != nil {
		return false, fmt.Errorf("video: resampling audio: %w", err)
	}
	n := r.NbSamples()
	if n == 0 {
		return true, nil
	}
	b, err := r.Data().Bytes(1)
	if err != nil {
		return false, fmt.Errorf("video: copying audio: %w", err)
	}

	// Turn the raw little-endian bytes into samples.
	samples := make([][2]float64, min(n, len(b)/16))
	for i := range samples {
		samples[i][0] = math.Float64frombits(binary.LittleEndian.Uint64(b[16*i:]))
		samples[i][1] = math.Float64frombits(binary.LittleEndian.Uint64(b[16*i+8:]))
	}
	return sink.audio(samples), nil
}
