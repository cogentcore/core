// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package video

import (
	"io"
	"time"
)

// Debug is whether to show the log messages of FFmpeg (warnings and errors
// about the media, as "Could not update timestamps for skipped samples"),
// which are not shown otherwise, as the errors are returned.
var Debug = false

// Media is a video file. On desktop platforms, it is decoded with FFmpeg
// via go-astiav: OpenMedia records the properties of the file, and each call
// to [Media.Decode] opens the file again, so that the media can be played
// any number of times. On the web, the browser plays it in a video element
// (see [Video]), so it is not decoded here, and its size is set once the
// browser has loaded it. With the novideo build tag, FFmpeg is not used,
// and OpenMedia returns an error (for apps that do not need video, built
// without the FFmpeg libraries).
type Media struct {

	// Source is the source of the media, instead of its Path, if set
	// (as the MediaSource of a web page).
	Source Source

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

	// Rotation is the rotation of the video for display, in degrees
	// clockwise (0, 90, 180 or 270), from its metadata (as of the videos of
	// phones). Its frames are turned by it, and Width and Height are of the
	// turned frames.
	Rotation int

	// Duration is the duration of the media, or 0 if it is not known.
	Duration time.Duration

	// HasAudio is whether the media has an audio stream.
	HasAudio bool
}

// Source is a source of the media of a [Media] other than a file or URL,
// as the MediaSource of a web page (Media Source Extensions), whose media
// is appended to it by the scripts of the page.
type Source interface {
	// Streams returns new readers of the byte streams of the media (as one
	// for its video and one for its audio, each a fragmented MP4), from the
	// given time: they block until more of the media is available, return
	// io.EOF at its end, and are closed when the decoding stops (which
	// must return a blocked read).
	Streams(start time.Duration) ([]io.ReadCloser, error)

	// Duration returns the duration of the media, or 0 if it is not known.
	Duration() time.Duration
}
