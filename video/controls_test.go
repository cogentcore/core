// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !novideo

package video

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoControls(t *testing.T) {
	b := core.NewBody()
	v := NewVideo(b)
	v.Controls = true
	v.Muted = true
	v.Styler(func(s *styles.Style) {
		s.Min.X.Dp(320)
		s.Min.Y.Dp(240)
	})
	require.NoError(t, v.Open("testdata/short.mp4"))
	b.RenderImage(func() {}, func() {
		c := v.controls
		require.NotNil(t, c, "the controls")
		assert.Equal(t, icons.PlayArrow, c.play.Icon)
		assert.False(t, c.bar.StateIs(states.Invisible), "shown before it plays")
		assert.True(t, v.RenderGroup())

		// it is decoded at the size that fits in its box, up to its own
		// size (as of the 4K deer video)
		deer, err := OpenMedia("../examples/video/deer.mp4", nil)
		require.NoError(t, err)
		short := v.Media
		v.Media = deer
		assert.InDelta(t, v.Geom.Size.Actual.Content.X, float32(v.fitWidth()), 1)
		v.Media = short
		require.NoError(t, v.Play(0, 0))
		assert.Equal(t, 64, v.playback.Width)
		assert.Equal(t, icons.Pause, c.play.Icon)
		require.Eventually(t, v.hasFrame, 2*time.Second, 10*time.Millisecond)

		// the play button pauses and resumes it
		c.play.Send(events.Click)
		assert.True(t, v.isPaused())
		assert.Equal(t, icons.PlayArrow, c.play.Icon)
		c.play.Send(events.Click)
		assert.True(t, v.IsPlaying())

		// the time, after a seek while paused
		v.Pause()
		v.Seek(600 * time.Millisecond)
		require.Eventually(t, func() bool { return v.CurrentTime() >= 600*time.Millisecond }, 2*time.Second, 10*time.Millisecond)
		v.updatePosition(v.CurrentTime())
		assert.Equal(t, "0:00 / 0:01", c.time.Text)
		assert.InDelta(t, 1.0, c.position.Max, 0.01)

		// muting and the volume
		c.mute.Send(events.Click)
		assert.False(t, v.Muted)
		assert.Equal(t, icons.VolumeUp, c.mute.Icon)
		v.SetVolume(0.25)
		assert.Equal(t, float32(0.25), v.Volume())
		assert.Equal(t, float32(0.25), c.volume.Value)
		assert.Equal(t, float32(0.25), v.playback.Volume)

		// the controls hide after the mouse stops, while it plays
		v.Resume()
		v.setControlsShown(true)
		v.setControlsShown(false)
		assert.True(t, c.bar.StateIs(states.Invisible))
		v.Pause()
		assert.False(t, c.bar.StateIs(states.Invisible), "shown when paused")
		v.Stop()
	})
}

func TestFitRect(t *testing.T) {
	r := image.Rect(0, 0, 400, 300)
	assert.Equal(t, image.Rect(0, 37, 400, 262), fitRect(r, image.Pt(16, 9)), "letterbox")
	assert.Equal(t, image.Rect(87, 0, 312, 300), fitRect(r, image.Pt(3, 4)), "pillarbox")
	assert.Equal(t, r, fitRect(r, image.Pt(4, 3)))
}

func TestFormatTime(t *testing.T) {
	assert.Equal(t, "0:00", formatTime(0))
	assert.Equal(t, "1:05", formatTime(65*time.Second))
	assert.Equal(t, "1:01:05", formatTime(time.Hour+65*time.Second))
}
