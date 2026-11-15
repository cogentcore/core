// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !novideo

package video

import (
	"errors"
	"image/color"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVideoNoMedia tests a video without media, which does nothing.
func TestVideoNoMedia(t *testing.T) {
	v := NewVideo().SetRotation(90).SetLoop(true).SetControls(false).SetPoster("poster.png")
	assert.Equal(t, "poster.png", v.Poster)
	assert.Equal(t, float32(90), v.Rotation)
	assert.True(t, v.Loop)
	assert.Error(t, v.Play(0, 0))
	assert.Equal(t, time.Duration(0), v.CurrentTime())
	assert.Equal(t, time.Duration(0), v.Duration())
	assert.False(t, v.IsPlaying())
	frame, _ := v.CurrentFrame()
	assert.Nil(t, frame)
	v.Seek(time.Second) // logs the error, and does nothing
	v.StepFrame(1)
	assert.False(t, v.IsPlaying())

	assert.Equal(t, float32(1), v.Volume(), "the default volume")
	v.SetVolume(0)
	assert.Greater(t, v.Volume(), float32(0), "a volume of 0 is kept above 0, as 0 is the default")
	v.SetVolume(2)
	assert.Equal(t, float32(1), v.Volume(), "at most 1")
	v.SetMuted(true)
	assert.True(t, v.IsMuted())
	v.release()
}

// TestVideoStartsOnSeek tests that seeking or stepping a video that has
// not played yet starts it.
func TestVideoStartsOnSeek(t *testing.T) {
	m, err := OpenMedia("testdata/short.mp4", nil)
	require.NoError(t, err)
	v := NewVideo().SetMedia(m)
	v.Muted = true
	assert.Greater(t, v.Duration(), time.Duration(0))
	v.StepFrame(1)
	require.NotNil(t, v.playback, "started by stepping")
	assert.Eventually(t, v.hasFrame, 2*time.Second, 10*time.Millisecond)
	v.Stop()

	v = NewVideo().SetMedia(m)
	v.Muted = true
	v.Seek(500 * time.Millisecond)
	require.NotNil(t, v.playback, "started by seeking")
	assert.Eventually(t, func() bool { return v.CurrentTime() >= 400*time.Millisecond }, 2*time.Second, 10*time.Millisecond)
	v.Stop()
}

func TestPlaybackAudioSettings(t *testing.T) {
	p := &Playback{}
	vol, muted := p.Audio()
	assert.Equal(t, float32(1), vol, "the default volume")
	assert.False(t, muted)
	p.SetVolume(0.5)
	p.SetMuted(true)
	vol, muted = p.Audio()
	assert.Equal(t, float32(0.5), vol)
	assert.True(t, muted)
}

func TestPlaybackError(t *testing.T) {
	var got error
	p := &Playback{OnError: func(err error) { got = err }}
	p.error(errors.New("decode failed"))
	assert.EqualError(t, got, "decode failed")
	p.OnError = nil
	assert.NotPanics(t, func() { p.error(errors.New("logged")) })
}

// TestVideoBlackBox tests that the black of the box of a video that is not
// at the origin of the window is drawn in its box (at its place).
func TestVideoBlackBox(t *testing.T) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.X.Dp(320)
		s.Min.Y.Dp(240)
	})
	core.NewText(b).SetText("above the video")
	v := NewVideo(b)
	v.Muted = true
	v.Styler(func(s *styles.Style) {
		s.Min.X.Dp(120)
		s.Min.Y.Dp(120)
	})
	require.NoError(t, v.Open("testdata/short.mp4"))
	img := core.RenderImage(b, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		require.NoError(t, v.Play(0, 0))
		waitUnlocked(t, b, v.hasFrame)
		v.Pause()
	})
	require.NotNil(t, img)
	black := color.RGBA{0, 0, 0, 255}
	assert.NotEqual(t, black, img.RGBAAt(1, 1), "the top left of the window is not the black of the video")
	box := v.Geom.ContentBBox
	require.False(t, box.Empty())
	assert.True(t, box.Min.Y > 1, "the video is below the text")
}

// TestControlsAudio tests the controls of an audio player, which are one
// row with the play button, the time, the position, and the volume.
func TestControlsAudio(t *testing.T) {
	b := core.NewBody()
	v := NewVideo(b)
	v.Muted = true
	require.NoError(t, v.Open("testdata/tone.mp3"))
	c := NewControls(b).SetAudio()
	c.SetPlayer(v, v)
	row := c.play.Parent
	for _, w := range []core.Widget{c.time, c.position, c.mute, c.volume} {
		assert.Equal(t, row, w.AsTree().Parent, "in the row of the play button")
	}
	assert.True(t, c.audio)
	core.RenderImage(b, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.True(t, c.IsShown(), "audio controls are always shown")
	})
}
