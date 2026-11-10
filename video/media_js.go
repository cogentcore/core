// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package video

import "errors"

// Media is a video file. On the web, the browser plays it in a video
// element (see [Video]), so it is not decoded here, and its size is set
// once the browser has loaded it.
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

var errUnsupported = errors.New("video: decoding video is not supported on the web, where the browser plays it")

// OpenMedia returns the media at the given path or URL, which the browser
// plays; its properties are not known until it has loaded it, and the
// options are not used.
func OpenMedia(fpath string, options map[string]string) (*Media, error) {
	return &Media{Path: fpath, Options: options}, nil
}

// decoder is not supported on the web.
type decoder struct{ scaleWidth, scaleHeight int }

func openDecoder(fpath string, options map[string]string, audio bool) (*decoder, error) {
	return nil, errUnsupported
}

func (d *decoder) free() {}

func (d *decoder) run(sink decodeSink) error { return errUnsupported }
