// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !novideo

package video

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The fragmented MP4 files of testdata, as the media that pages append to
// a MediaSource: frag-video.mp4 is the video of short.mp4 (10 frames at
// 10 per second, a key frame every 3), and frag-audio.mp4 its audio, each
// an init segment and fragments.

// slowReader is a stream of a Source that gives the given data in pieces,
// waiting between them, as it is appended, until it is closed.
type slowReader struct {
	data   []byte
	piece  int
	wait   time.Duration
	mu     sync.Mutex
	closed bool
}

func (r *slowReader) Read(b []byte) (int, error) {
	r.mu.Lock()
	closed := r.closed
	r.mu.Unlock()
	if closed {
		return 0, io.EOF
	}
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	time.Sleep(r.wait)
	n := copy(b, r.data[:min(r.piece, len(r.data))])
	r.data = r.data[n:]
	return n, nil
}

func (r *slowReader) Close() error {
	r.mu.Lock()
	r.closed = true
	r.mu.Unlock()
	return nil
}

// fileSource is a Source of the given files, read slowly (in pieces).
type fileSource struct {
	files []string
	piece int
	wait  time.Duration
}

func (fs *fileSource) Streams(start time.Duration) ([]io.ReadCloser, error) {
	var rs []io.ReadCloser
	for _, f := range fs.files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		rs = append(rs, &slowReader{data: bytes.Clone(b), piece: fs.piece, wait: fs.wait})
	}
	return rs, nil
}

func (fs *fileSource) Duration() time.Duration { return time.Second }

func TestSourceDecode(t *testing.T) {
	src := &fileSource{files: []string{"testdata/frag-video.mp4", "testdata/frag-audio.mp4"}, piece: 1000}
	m, err := OpenSource(src)
	require.NoError(t, err)
	assert.Equal(t, 64, m.Width)
	assert.Equal(t, 48, m.Height)
	assert.True(t, m.HasAudio, "from the other stream")
	assert.Equal(t, time.Second, m.Duration)

	// the video and audio, of their two streams
	done := make(chan struct{})
	defer close(done)
	frames, samples, errs, err := m.Decode(0, 0, true, done)
	require.NoError(t, err)
	n, ns := 0, 0
	for frames != nil || samples != nil {
		select {
		case _, ok := <-frames:
			if !ok {
				frames = nil
				continue
			}
			n++
		case _, ok := <-samples:
			if !ok {
				samples = nil
				continue
			}
			ns++
		}
	}
	assert.NoError(t, <-errs)
	assert.Equal(t, 10, n)
	assert.InDelta(t, sampleRate, ns, sampleRate/5, "about a second of audio")
}

func TestSourcePlaybackSlow(t *testing.T) {
	// A Source that is appended to slowly plays, and is stopped while it
	// waits for more (the readers are closed).
	src := &fileSource{files: []string{"testdata/frag-video.mp4", "testdata/frag-audio.mp4"}, piece: 400, wait: 20 * time.Millisecond}
	m, err := OpenSource(src)
	require.NoError(t, err)
	fl := &frameLog{}
	p := &Playback{Media: m, Muted: true, OnFrame: fl.add}
	require.NoError(t, p.Play())
	require.Eventually(t, func() bool { return fl.len() >= 2 }, 3*time.Second, 10*time.Millisecond)
	p.Stop()
	n := fl.len()
	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, n, fl.len(), "stopped")
}
