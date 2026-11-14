// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package video

import (
	"io"
	"sync"
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

	// HasVideo is whether the media has a video stream: media with only
	// audio (as an MP3 file or stream) has no size nor frames, and its
	// playing is timed by its audio (see [Playback]).
	HasVideo bool

	// probe is the decoder that opened the media to read its properties,
	// kept for its first decoding from its start (see takeProbe), so that
	// it is not opened again (as a stream of a URL, which is read again
	// from the network: the live stream of npr.org took 2.8s to start).
	probe   *decoder
	probeMu sync.Mutex
}

// probeKeep is how long the decoder that opened a media is kept for its
// first decoding (see [Media.probe]); it is freed if none starts by then.
const probeKeep = 10 * time.Second

// keepProbe keeps the given decoder that opened the media, for its first
// decoding (see takeProbe), until [probeKeep] has passed.
func (m *Media) keepProbe(d *decoder) {
	m.probeMu.Lock()
	m.probe = d
	m.probeMu.Unlock()
	time.AfterFunc(probeKeep, func() {
		if d := m.takeProbe(); d != nil {
			d.free()
		}
	})
}

// takeProbe returns the decoder that opened the media, if it is still
// kept, which the caller then owns, or nil.
func (m *Media) takeProbe() *decoder {
	m.probeMu.Lock()
	defer m.probeMu.Unlock()
	d := m.probe
	m.probe = nil
	return d
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
