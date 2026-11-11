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
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
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
		// with the lock of the widgets, as in the event loop, released
		// while waiting for the playback
		b.AsyncLock()
		defer b.AsyncUnlock()
		c := v.controls
		require.NotNil(t, c, "the controls")
		assert.Equal(t, icons.PlayArrow, c.play.Icon)
		assert.True(t, c.IsShown(), "shown before it plays")
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
		waitUnlocked(t, b, v.hasFrame)

		// the play button pauses and resumes it
		c.play.Send(events.Click)
		assert.True(t, v.isPaused())
		assert.Equal(t, icons.PlayArrow, c.play.Icon)
		c.play.Send(events.Click)
		assert.True(t, v.IsPlaying())

		// the time, after a seek while paused
		v.Pause()
		v.Seek(600 * time.Millisecond)
		waitUnlocked(t, b, func() bool { return v.CurrentTime() >= 600*time.Millisecond })
		c.SetTime(v.CurrentTime())
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
		c.SetShown(true)
		c.SetShown(false)
		assert.False(t, c.IsShown())
		v.Pause()
		assert.True(t, c.IsShown(), "shown when paused")
		v.Stop()
	})
}

// waitUnlocked waits for the given condition with the lock of the widgets
// of the given body released, as the playback shows its frames with it.
func waitUnlocked(t *testing.T, b *core.Body, cond func() bool) {
	t.Helper()
	b.AsyncUnlock()
	defer b.AsyncLock()
	require.Eventually(t, cond, 2*time.Second, 10*time.Millisecond)
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

func TestControlsZone(t *testing.T) {
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
		b.AsyncLock()
		defer b.AsyncUnlock()
		c := v.controls
		vb := v.Geom.TotalBBox
		mid := image.Pt((vb.Min.X+vb.Max.X)/2, 0)
		at := func(y int) image.Point { return image.Pt(mid.X, y) }
		assert.False(t, c.inZone(at(vb.Min.Y+10)), "the top")
		assert.False(t, c.inZone(at((vb.Min.Y+vb.Max.Y)/2)), "the middle")
		assert.True(t, c.inZone(at(vb.Max.Y-5)), "the bottom")
		cb := c.Geom.TotalBBox
		assert.True(t, c.inZone(at(cb.Min.Y+1)), "the top of the controls")
		assert.False(t, c.inZone(at(cb.Min.Y-2)), "just above the controls")
		assert.False(t, c.inZone(at(vb.Max.Y+5)), "below the video")

		// moving over the middle of a playing video does not show them, and
		// hides them soon after the mouse moves out of the bottom
		v.Loop = true
		require.NoError(t, v.Play(0, 0))
		c.SetShown(false)
		c.mouseMoved(at((vb.Min.Y + vb.Max.Y) / 2))
		assert.False(t, c.IsShown())
		c.mouseMoved(at(vb.Max.Y - 5))
		assert.True(t, c.IsShown())
		c.mouseMoved(at(vb.Min.Y + 10))
		assert.True(t, c.IsShown(), "until the delay")
		b.AsyncUnlock()
		time.Sleep(controlsLeaveDelay + 200*time.Millisecond)
		b.AsyncLock()
		assert.False(t, c.IsShown(), "after the delay")
		v.Stop()
	})
}

func TestControlsKeys(t *testing.T) {
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
		b.AsyncLock()
		defer b.AsyncUnlock()
		c := v.controls
		assert.True(t, v.AbilityIs(abilities.Focusable), "for the keys")
		press := func(code key.Codes) bool {
			return c.handleKey(events.NewKey(events.KeyChord, 0, code, 0))
		}
		base := time.Duration(0)

		// space plays and pauses
		require.True(t, press(key.CodeSpacebar))
		assert.True(t, v.IsPlaying())
		waitUnlocked(t, b, v.hasFrame)
		require.True(t, press(key.CodeSpacebar))
		assert.True(t, v.isPaused())

		// the arrows step frames and skip 10 seconds (to the end and back to
		// the start of this 1 second video)
		v.Seek(300 * time.Millisecond)
		waitUnlocked(t, b, func() bool { return v.CurrentTime() == base+300*time.Millisecond })
		require.True(t, press(key.CodeRightArrow))
		waitUnlocked(t, b, func() bool { return v.CurrentTime() == base+400*time.Millisecond })
		require.True(t, press(key.CodeLeftArrow))
		waitUnlocked(t, b, func() bool { return v.CurrentTime() == base+300*time.Millisecond })
		assert.True(t, v.isPaused())
		require.True(t, press(key.CodeDownArrow))
		waitUnlocked(t, b, func() bool { return v.CurrentTime() == 0 })
		assert.True(t, c.IsShown(), "shown by the keys")

		// other keys and modifiers are not those of players
		assert.False(t, press(key.CodeA))
		assert.False(t, c.handleKey(events.NewKey(events.KeyChord, 0, key.CodeRightArrow, key.Modifiers(1)<<key.Shift)))

		// the keys also work when a control (as a slider) has the focus,
		// as pressed in the window
		v.Seek(300 * time.Millisecond)
		waitUnlocked(t, b, func() bool { return v.CurrentTime() == base+300*time.Millisecond })
		c.volume.SetFocus()
		value := c.volume.Value
		b.AsyncUnlock()
		src := b.Scene.RenderWindow().SystemWindow.Events()
		src.Key(events.KeyDown, 0, key.CodeLeftArrow, 0)
		src.Key(events.KeyUp, 0, key.CodeLeftArrow, 0)
		time.Sleep(300 * time.Millisecond)
		b.AsyncLock()
		assert.Equal(t, value, c.volume.Value, "not the volume")
		assert.Equal(t, base+200*time.Millisecond, v.CurrentTime(), "a frame back")
		v.Stop()
	})
}

func TestControlsKeysHidden(t *testing.T) {
	b := core.NewBody()
	v := NewVideo(b)
	v.Controls = true
	v.Muted = true
	v.Loop = true
	v.Styler(func(s *styles.Style) {
		s.Min.X.Dp(320)
		s.Min.Y.Dp(240)
	})
	require.NoError(t, v.Open("testdata/short.mp4"))
	b.RenderImage(func() {}, func() {
		b.AsyncLock()
		c := v.controls

		// the video gets the focus when it is shown (or when the mouse moves
		// over it), as nothing has it, so the keys work at once
		assert.True(t, v.ContainsFocus(), "shown")

		// a control that was clicked has the focus, and the controls hide:
		// the video gets the focus, and space still plays and pauses it,
		// and shows the controls
		require.NoError(t, v.Play(0, 0))
		c.play.SetFocus()
		c.SetShown(false)
		assert.False(t, c.ContainsFocus())
		assert.True(t, v.ContainsFocus())
		b.AsyncUnlock()
		src := b.Scene.RenderWindow().SystemWindow.Events()
		// (as the desktop driver sends it: the character, of an unknown code)
		src.KeyChord(' ', key.CodeUnknown, 0)
		time.Sleep(300 * time.Millisecond)
		b.AsyncLock()
		assert.True(t, v.isPaused(), "space paused it")
		assert.True(t, c.IsShown(), "and showed the controls")
		v.Stop()
		b.AsyncUnlock()
	})
}

func TestControlsHideAtStart(t *testing.T) {
	// The controls, shown before the video plays, hide soon after it
	// starts playing, without the mouse.
	b := core.NewBody()
	v := NewVideo(b)
	v.Controls = true
	v.Muted = true
	v.Loop = true
	v.Styler(func(s *styles.Style) {
		s.Min.X.Dp(320)
		s.Min.Y.Dp(240)
	})
	require.NoError(t, v.Open("testdata/short.mp4"))
	b.RenderImage(func() {}, func() {
		b.AsyncLock()
		c := v.controls
		assert.True(t, c.IsShown(), "before it plays")
		require.NoError(t, v.Play(0, 0))
		assert.True(t, c.IsShown(), "when it starts")
		b.AsyncUnlock()
		time.Sleep(controlsHideDelay + 300*time.Millisecond)
		b.AsyncLock()
		assert.False(t, c.IsShown(), "soon after")
		v.Stop()
		b.AsyncUnlock()
	})
}
