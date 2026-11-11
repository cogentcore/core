// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !novideo

package video

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// frameLog records the frames that a playback shows, and when.
type frameLog struct {
	mu    sync.Mutex
	times []time.Duration // of the frames in the media
	at    []time.Time     // when they were shown
}

func (fl *frameLog) add(f Frame) {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	fl.times = append(fl.times, f.Time)
	fl.at = append(fl.at, time.Now())
}

func (fl *frameLog) len() int {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	return len(fl.times)
}

func newPlayback(t *testing.T, fl *frameLog) *Playback {
	m, err := OpenMedia("../examples/video/countdown.mp4", nil)
	require.NoError(t, err)
	return &Playback{Media: m, Width: 32, Muted: true, OnFrame: fl.add}
}

func TestPlaybackTiming(t *testing.T) {
	fl := &frameLog{}
	p := newPlayback(t, fl)
	start := time.Now()
	require.NoError(t, p.Play())
	assert.True(t, p.IsPlaying())
	time.Sleep(500 * time.Millisecond)
	p.Stop()
	assert.False(t, p.IsPlaying())

	// About 15 frames at 29.97 per second, shown at their times.
	n := fl.len()
	assert.InDelta(t, 15, n, 3)
	for i := 1; i < n; i++ {
		assert.Greater(t, fl.times[i], fl.times[i-1])
		late := fl.at[i].Sub(start) - (fl.times[i] - fl.times[0])
		assert.InDelta(t, 0, late.Seconds(), 0.05, "frame %d", i)
	}

	// No frames after Stop.
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, n, fl.len())
}

func TestPlaybackPause(t *testing.T) {
	fl := &frameLog{}
	p := newPlayback(t, fl)
	require.NoError(t, p.Play())
	time.Sleep(200 * time.Millisecond)
	p.Pause()
	assert.True(t, p.IsPaused())
	n := fl.len()
	time.Sleep(300 * time.Millisecond)
	assert.LessOrEqual(t, fl.len(), n+1, "no frames while paused")

	// After resuming, the frames continue at their times, shifted by the pause.
	p.Resume()
	time.Sleep(200 * time.Millisecond)
	p.Stop()
	m := fl.len()
	assert.InDelta(t, n+6, m, 3)
	var gap time.Duration
	for i := 1; i < m; i++ {
		gap = max(gap, fl.at[i].Sub(fl.at[i-1]))
	}
	assert.Greater(t, gap, 250*time.Millisecond, "the pause")
}

func TestPlaybackEnd(t *testing.T) {
	// testdata/short.mp4 is 1 second: 10 frames at 10 per second.
	m, err := OpenMedia("testdata/short.mp4", nil)
	require.NoError(t, err)
	fl := &frameLog{}
	ended := make(chan struct{})
	p := &Playback{Media: m, OnFrame: fl.add, OnEnd: func() { close(ended) }}
	require.NoError(t, p.Play()) // with audio, if there is a speaker
	select {
	case <-ended:
	case <-time.After(3 * time.Second):
		t.Fatal("no end")
	}
	assert.Equal(t, 10, fl.len())
	assert.Equal(t, 900*time.Millisecond, fl.times[9]-fl.times[0])
}

func TestPlaybackLoop(t *testing.T) {
	m, err := OpenMedia("testdata/short.mp4", nil)
	require.NoError(t, err)
	fl := &frameLog{}
	p := &Playback{Media: m, Muted: true, Loop: true, OnFrame: fl.add}
	start := time.Now()
	require.NoError(t, p.Play())
	time.Sleep(1550 * time.Millisecond)
	p.Stop()

	// The second time through, the frames are shown at their times again,
	// after the first second.
	n := fl.len()
	assert.InDelta(t, 16, n, 2)
	assert.Equal(t, fl.times[0], fl.times[10], "the first frame again")
	assert.InDelta(t, 1.0, fl.at[10].Sub(start).Seconds(), 0.08)
}

func TestFirstFrame(t *testing.T) {
	m, err := OpenMedia("../examples/video/deer.mp4", nil)
	require.NoError(t, err)
	img, err := m.FirstFrame(320, 0)
	require.NoError(t, err)
	assert.Equal(t, 320, img.Bounds().Dx())
	assert.Greater(t, colorful(img), 4)
}

func TestOpenURL(t *testing.T) {
	// A video over http, with the headers of the options (as the cookies of
	// a web page), as the videos of web pages are played.
	var mu sync.Mutex
	var agents, cookies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		agents = append(agents, r.UserAgent())
		cookies = append(cookies, r.Header.Get("Cookie"))
		mu.Unlock()
		http.ServeFile(w, r, "../examples/video/countdown.mp4")
	}))
	defer srv.Close()
	opts := map[string]string{"user_agent": "test-agent", "headers": "Cookie: a=b\r\n"}
	m, err := OpenMedia(srv.URL+"/countdown.mp4", opts)
	require.NoError(t, err)
	assert.Equal(t, 320, m.Width)
	img, err := m.FirstFrame(0, 0)
	require.NoError(t, err)
	assert.Equal(t, 240, img.Bounds().Dy())
	mu.Lock()
	defer mu.Unlock()
	require.NotEmpty(t, agents)
	assert.Equal(t, "test-agent", agents[0])
	assert.Equal(t, "a=b", cookies[0])
}
