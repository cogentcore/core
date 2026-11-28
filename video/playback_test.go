// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !novideo

package video

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
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

// timesBase returns the time of the first frame of the media (its start
// time), which the times of its frames are relative to.
func (fl *frameLog) timesBase(m *Media) time.Duration {
	done := make(chan struct{})
	defer close(done)
	frames, _, _, err := m.Decode(16, 0, false, done)
	if err != nil {
		panic(err)
	}
	return (<-frames).Time
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
		// (from the first frame, when the time of the playing starts)
		late := fl.at[i].Sub(fl.at[0]) - (fl.times[i] - fl.times[0])
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
	assert.True(t, p.IsEnded())
	assert.False(t, p.IsPlaying())
	assert.Equal(t, 900*time.Millisecond, fl.times[9]-fl.times[0])
}

func TestPlaybackLoop(t *testing.T) {
	m, err := OpenMedia("testdata/short.mp4", nil)
	require.NoError(t, err)
	fl := &frameLog{}
	p := &Playback{Media: m, Muted: true, Loop: true, OnFrame: fl.add}
	require.NoError(t, p.Play())
	time.Sleep(1550 * time.Millisecond)
	p.Stop()

	// The second time through, the frames are shown at their times again,
	// after the first second.
	n := fl.len()
	assert.InDelta(t, 16, n, 2)
	assert.Equal(t, fl.times[0], fl.times[10], "the first frame again")
	assert.InDelta(t, 1.0, fl.at[10].Sub(fl.at[0]).Seconds(), 0.08)
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

func TestPlaybackSeek(t *testing.T) {
	// testdata/short.mp4 is 1 second: 10 frames at 10 per second.
	m, err := OpenMedia("testdata/short.mp4", nil)
	require.NoError(t, err)
	assert.InDelta(t, 1.0, m.Duration.Seconds(), 0.05)
	fl := &frameLog{}
	p := &Playback{Media: m, Muted: true, OnFrame: fl.add}

	// Played from a time, its frames start there.
	require.NoError(t, p.PlayFrom(500*time.Millisecond))
	require.Eventually(t, func() bool { return fl.len() >= 3 }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, 500*time.Millisecond, fl.times[0]-fl.timesBase(m))
	assert.Equal(t, fl.times[fl.len()-1], p.Position())

	// A seek while paused shows the frame at the time, and stays paused.
	p.Pause()
	n := fl.len()
	require.NoError(t, p.Seek(200*time.Millisecond))
	require.Eventually(t, func() bool { return fl.len() == n+1 }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, 200*time.Millisecond, fl.times[n]-fl.timesBase(m))
	assert.True(t, p.IsPaused())
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, n+1, fl.len(), "no more frames while paused")

	// It plays on from there when resumed, and a seek past the end is to the end.
	p.Resume()
	require.Eventually(t, func() bool { return fl.len() >= n+3 }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, 300*time.Millisecond, fl.times[n+1]-fl.timesBase(m))
	require.NoError(t, p.Seek(time.Hour))
	assert.Equal(t, m.Duration, p.Position())
	p.Stop()

	// The volume and muting can change while it plays.
	p.SetVolume(0.5)
	assert.Equal(t, float32(0.5), p.Volume)
	p.SetMuted(false)
	assert.False(t, p.Muted)
}

func TestPlaybackStepFrame(t *testing.T) {
	// testdata/short.mp4 is 1 second: 10 frames at 10 per second.
	m, err := OpenMedia("testdata/short.mp4", nil)
	require.NoError(t, err)
	fl := &frameLog{}
	p := &Playback{Media: m, Muted: true, OnFrame: fl.add}
	base := fl.timesBase(m)
	require.NoError(t, p.Seek(500*time.Millisecond))
	p.Pause()
	require.Eventually(t, func() bool { return p.Position() == base+500*time.Millisecond && fl.len() > 0 }, 2*time.Second, 10*time.Millisecond)

	// a frame forward and back, paused
	n := fl.len()
	require.NoError(t, p.StepFrame(1))
	require.Eventually(t, func() bool { return fl.len() == n+1 }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, base+600*time.Millisecond, p.Position())
	assert.True(t, p.IsPaused())
	require.NoError(t, p.StepFrame(-1))
	require.Eventually(t, func() bool { return fl.len() == n+2 }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, base+500*time.Millisecond, p.Position())
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, n+2, fl.len(), "one frame each")
	p.Stop()
}

// audioLog records the times that a playback of media with only audio
// gives to OnTime, and when it ends.
type audioLog struct {
	mu    sync.Mutex
	times []time.Duration
	ended chan struct{}
}

func newAudioPlayback(t *testing.T, path string) (*Playback, *audioLog) {
	m, err := OpenMedia(path, nil)
	require.NoError(t, err)
	al := &audioLog{ended: make(chan struct{})}
	p := &Playback{Media: m, Muted: true,
		OnFrame: func(Frame) { t.Error("a frame of media with only audio") },
		OnTime: func(tm time.Duration) {
			al.mu.Lock()
			al.times = append(al.times, tm)
			al.mu.Unlock()
		},
		OnEnd: func() { close(al.ended) },
	}
	return p, al
}

func (al *audioLog) last() time.Duration {
	al.mu.Lock()
	defer al.mu.Unlock()
	if len(al.times) == 0 {
		return 0
	}
	return al.times[len(al.times)-1]
}

func TestPlaybackAudioOnly(t *testing.T) {
	p, al := newAudioPlayback(t, "testdata/tone.mp3")
	m := p.Media
	assert.True(t, m.HasAudio)
	assert.False(t, m.HasVideo)
	assert.Zero(t, m.Width)
	assert.InDelta(t, 1.5, m.Duration.Seconds(), 0.1)
	_, err := m.FirstFrame(16, 0)
	assert.Error(t, err, "media with only audio has no frames")

	// it plays for the time of its audio, which it is timed by
	start := time.Now()
	require.NoError(t, p.Play())
	assert.True(t, p.IsPlaying())
	select {
	case <-al.ended:
	case <-time.After(5 * time.Second):
		t.Fatal("the audio did not end")
	}
	elapsed := time.Since(start)
	assert.Greater(t, elapsed, 1200*time.Millisecond)
	assert.Less(t, elapsed, 3500*time.Millisecond)
	assert.True(t, p.IsEnded())
	assert.False(t, p.IsPlaying())
	assert.InDelta(t, 1.5, p.Position().Seconds(), 0.15)
	al.mu.Lock()
	defer al.mu.Unlock()
	assert.Greater(t, len(al.times), 5)
	for i := 1; i < len(al.times); i++ {
		assert.GreaterOrEqual(t, al.times[i], al.times[i-1])
	}
}

func TestPlaybackAudioOnlyPause(t *testing.T) {
	p, al := newAudioPlayback(t, "testdata/tone.mp3")
	require.NoError(t, p.Play())
	defer p.Stop()
	time.Sleep(500 * time.Millisecond)
	p.Pause()
	time.Sleep(150 * time.Millisecond) // (the last time given)
	paused := p.Position()
	assert.Greater(t, paused, 200*time.Millisecond)
	time.Sleep(400 * time.Millisecond)
	assert.Equal(t, paused, p.Position(), "the time does not move while paused")
	assert.True(t, p.IsPaused())

	// a seek while paused stays paused, at the time
	require.NoError(t, p.Seek(time.Second))
	time.Sleep(250 * time.Millisecond)
	assert.True(t, p.IsPaused())
	assert.InDelta(t, 1.0, p.Position().Seconds(), 0.05)
	p.Resume()
	select {
	case <-al.ended:
	case <-time.After(3 * time.Second):
		t.Fatal("the audio did not end after it was resumed")
	}
	assert.InDelta(t, 1.5, al.last().Seconds(), 0.15)
}

// TestPlaybackAudioStream plays audio that is streamed over HTTP without a
// length, as a live radio stream (as an Icecast MP3 stream).
func TestPlaybackAudioStream(t *testing.T) {
	data, err := os.ReadFile("testdata/tone.mp3")
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		for i := 0; i < len(data); i += 1024 { // (chunked, without a length)
			w.Write(data[i:min(i+1024, len(data))])
			w.(http.Flusher).Flush()
		}
	}))
	defer srv.Close()
	p, al := newAudioPlayback(t, srv.URL+"/live.mp3")
	assert.True(t, p.Media.HasAudio)
	assert.False(t, p.Media.HasVideo)
	require.NoError(t, p.Play())
	select {
	case <-al.ended:
	case <-time.After(5 * time.Second):
		t.Fatal("the stream did not end")
	}
	assert.InDelta(t, 1.5, al.last().Seconds(), 0.15)
}

// TestPlaybackStreamOpenedOnce tests that a stream that is played as soon
// as it is opened is requested once: the decoder that opened it to read its
// properties decodes it (the live stream of npr.org, opened twice, took 2.8s
// to start).
func TestPlaybackStreamOpenedOnce(t *testing.T) {
	data, err := os.ReadFile("testdata/tone.mp3")
	require.NoError(t, err)
	var reqs atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs.Add(1)
		w.Header().Set("Content-Type", "audio/mpeg")
		for i := 0; i < len(data); i += 1024 {
			w.Write(data[i:min(i+1024, len(data))])
			w.(http.Flusher).Flush()
		}
	}))
	defer srv.Close()
	p, al := newAudioPlayback(t, srv.URL+"/live.mp3")
	require.NoError(t, p.Play())
	select {
	case <-al.ended:
	case <-time.After(5 * time.Second):
		t.Fatal("the stream did not end")
	}
	assert.Equal(t, int32(1), reqs.Load())
}

// TestPlaybackLiveResume tests that a live stream (of no duration) that is
// resumed after a long pause is requested again, to play from its live
// point, and plays.
func TestPlaybackLiveResume(t *testing.T) {
	old := liveStale
	liveStale = 100 * time.Millisecond
	defer func() { liveStale = old }()
	data, err := os.ReadFile("testdata/tone.mp3")
	require.NoError(t, err)
	var reqs atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs.Add(1)
		w.Header().Set("Content-Type", "audio/mpeg")
		for range 40 { // (a long stream, sent as it plays)
			for i := 0; i < len(data); i += 1024 {
				if _, err := w.Write(data[i:min(i+1024, len(data))]); err != nil {
					return
				}
				w.(http.Flusher).Flush()
				time.Sleep(30 * time.Millisecond)
			}
		}
	}))
	defer srv.Close()
	p, al := newAudioPlayback(t, srv.URL+"/live.mp3")
	defer p.Stop()
	require.NoError(t, p.Play())
	require.Eventually(t, func() bool { return al.last() > 0 }, 5*time.Second, 20*time.Millisecond)
	p.Pause()
	time.Sleep(300 * time.Millisecond)
	p.Resume()
	require.Eventually(t, func() bool { return reqs.Load() == 2 && p.IsPlaying() }, 5*time.Second, 20*time.Millisecond, "requested again, and playing")
}

func TestPlaybackAudioChanged(t *testing.T) {
	// testdata/u8mono.avi is 1 second, of 5 frames per second, with audio
	// of unsigned 8-bit mono samples at 11024 Hz, as the AVI files of old
	// cameras, whose frames are not of the format of the first one for the
	// resampler (Input changed).
	m, err := OpenMedia("testdata/u8mono.avi", nil)
	require.NoError(t, err)
	fl := &frameLog{}
	ended := make(chan struct{})
	var errs []error
	p := &Playback{Media: m, Muted: true, OnFrame: fl.add, OnEnd: func() { close(ended) },
		OnError: func(err error) { errs = append(errs, err) }}
	require.NoError(t, p.Play())
	select {
	case <-ended:
	case <-time.After(3 * time.Second):
		t.Fatal("no end")
	}
	assert.Empty(t, errs)
	assert.Equal(t, 5, fl.len(), "all of the frames")
}
