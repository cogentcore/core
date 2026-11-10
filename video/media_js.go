// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package video

import "errors"

// Media is an opened video file. Decoding video with FFmpeg is not
// supported on the web, so [OpenMedia] always returns an error there.
type Media struct {

	// Path is the path or URL of the media file.
	Path string

	// Options are the options of FFmpeg for opening the input.
	Options map[string]string

	// Width and Height are the inherent size of the video, in pixels.
	Width, Height int

	// FrameRate is the average number of video frames per second,
	// or 0 if it is not known.
	FrameRate float64

	// HasAudio is whether the media has an audio stream.
	HasAudio bool
}

var errUnsupported = errors.New("video: decoding video is not supported on the web")

// OpenMedia returns an error, since decoding video is not supported on the web.
func OpenMedia(fpath string, options map[string]string) (*Media, error) { return nil, errUnsupported }

// decoder is not supported on the web.
type decoder struct{ scaleWidth, scaleHeight int }

func openDecoder(fpath string, options map[string]string, audio bool) (*decoder, error) {
	return nil, errUnsupported
}

func (d *decoder) free() {}

func (d *decoder) run(sink decodeSink) error { return errUnsupported }
