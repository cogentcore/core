// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package video

// Media is a video file. On desktop platforms, it is decoded with FFmpeg
// via go-astiav: OpenMedia records the properties of the file, and each call
// to [Media.Decode] opens the file again, so that the media can be played
// any number of times. On the web, the browser plays it in a video element
// (see [Video]), so it is not decoded here, and its size is set once the
// browser has loaded it. With the novideo build tag, FFmpeg is not used,
// and OpenMedia returns an error (for apps that do not need video, built
// without the FFmpeg libraries).
type Media struct {

	// Path is the path or URL of the media file (any URL that FFmpeg
	// reads, as http and https).
	Path string

	// Options are the options of FFmpeg for opening the input, as
	// user_agent and headers (as "Cookie: a=b\r\n") for http URLs.
	Options map[string]string

	// Width and Height are the inherent size of the video, in pixels.
	Width, Height int

	// FrameRate is the average number of video frames per second,
	// or 0 if it is not known.
	FrameRate float64

	// HasAudio is whether the media has an audio stream.
	HasAudio bool
}
