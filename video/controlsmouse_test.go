// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !novideo

package video

import (
	"image"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestControlsMouse tests clicking a video with controls, which plays and
// pauses it, other than on the controls, and that the controls are shown
// when the mouse leaves a paused video.
func TestControlsMouse(t *testing.T) {
	b := core.NewBody()
	v := NewVideo(b)
	v.Controls = true
	v.Muted = true
	v.Styler(func(s *styles.Style) {
		s.Min.X.Dp(320)
		s.Min.Y.Dp(240)
	})
	require.NoError(t, v.Open("testdata/short.mp4"))
	core.RenderImage(b, core.Locked(b, func() {
		c := v.controls
		require.NotNil(t, c)
		click := func(p image.Point) {
			e := events.NewMouse(events.MouseUp, events.Left, p, 0)
			e.SetLocalOff(image.Point{}) // (as for the events of a window)
			v.Send(events.Click, e)
		}
		click(v.Geom.TotalBBox.Min.Add(image.Pt(5, 5)))
		assert.True(t, v.IsPlaying(), "a click on the video plays it")
		click(v.Geom.TotalBBox.Min.Add(image.Pt(5, 5)))
		assert.False(t, v.IsPlaying(), "and pauses it")
		cb := c.Geom.TotalBBox
		click(cb.Min.Add(cb.Max).Div(2))
		assert.False(t, v.IsPlaying(), "a click on the controls is theirs")

		c.SetShown(false)
		v.Send(events.MouseLeave)
		assert.True(t, c.IsShown(), "shown when the mouse leaves a paused video")
		v.Stop()
	}))
}
