// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package video

import (
	"image/color"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// The controls of a [Video] on desktop platforms, as those of the video
// elements of browsers: a bar over the bottom of the video, partly
// transparent, with the position (a slider, which seeks when it is
// dragged), and buttons to play and pause, to skip back and forward 10
// seconds, to mute, with the volume and the time. They are shown while
// the mouse moves over the video, until [controlsHideDelay] after it
// stops or leaves, and while the video is not playing. A click on the
// video (not on its controls) plays and pauses it.
//
// The video is drawn by its render source under the scene (see
// [Video.Render]), which draws the region of the controls of the scene
// again over it. The video is rendered as a whole when its controls
// change ([Video.RenderGroup]), so that the transparent bar is drawn over
// the cleared region, not over its previous rendering.

// controlsHideDelay is how long the controls stay shown after the mouse
// stops moving over a playing video.
const controlsHideDelay = 2500 * time.Millisecond

// controls are the widgets of the controls of a [Video].
type controls struct {
	bar      *core.Frame
	position *core.Slider
	play     *core.Button
	mute     *core.Button
	volume   *core.Slider
	time     *core.Text

	// shownAt is when the controls were last shown by the mouse moving.
	shownAt time.Time

	// second is the second of the position that is shown.
	second int
}

// RenderGroup returns whether the video is rendered as a whole when any
// widget in it needs to be rendered: with its controls (see controls.go).
func (v *Video) RenderGroup() bool {
	return v.Controls
}

// initControls adds the controls to the video, and the handling of the
// mouse that shows them, and of clicks that play and pause it.
func (v *Video) initControls() {
	v.On(events.MouseMove, func(e events.Event) {
		if v.controls != nil {
			v.showControls()
		}
	})
	v.On(events.MouseLeave, func(e events.Event) {
		if c := v.controls; c != nil {
			c.shownAt = time.Time{}
			v.setControlsShown(!v.IsPlaying())
		}
	})
	v.OnClick(func(e events.Event) {
		c := v.controls
		if c == nil || e.Pos().In(c.bar.Geom.TotalBBox) {
			return // (a click on the controls)
		}
		v.TogglePlay()
	})
	v.Maker(func(p *tree.Plan) {
		if !v.Controls {
			v.controls = nil
			return
		}
		tree.AddAt(p, "controls", func(bar *core.Frame) { v.makeControls(bar) })
	})
}

// makeControls makes the widgets of the controls in the given bar.
func (v *Video) makeControls(bar *core.Frame) {
	c := &controls{bar: bar}
	v.controls = c
	white := colors.Uniform(color.White)
	bar.Styler(func(s *styles.Style) {
		s.Position = styles.PositionAbsolute
		s.Inset.Left, s.Inset.Right, s.Inset.Bottom = units.Zero(), units.Zero(), units.Zero()
		s.InsetSet.Left, s.InsetSet.Right, s.InsetSet.Bottom = true, true, true
		s.Direction = styles.Column
		s.Background = colors.Uniform(color.RGBA{0, 0, 0, 150})
		s.Color = white
		s.Padding.Set(units.Dp(4), units.Dp(8))
		s.Gap.Zero()
	})

	// the position, which seeks when it is changed
	c.position = core.NewSlider(bar).SetMin(0).SetStep(1)
	c.position.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
	c.position.OnChange(func(e events.Event) {
		v.Seek(time.Duration(c.position.Value * float32(time.Second)))
	})

	// the buttons, volume and time
	row := core.NewFrame(bar)
	row.Styler(func(s *styles.Style) {
		s.Align.Items = styles.Center
		s.Gap.X.Dp(2)
	})
	button := func(icon, tooltip string) *core.Button {
		bt := core.NewButton(row).SetType(core.ButtonAction)
		bt.SetTooltip(tooltip)
		bt.Styler(func(s *styles.Style) {
			s.Color = white
			s.Padding.Set(units.Dp(4))
		})
		return bt
	}
	back := button("", "Back 10 seconds").SetIcon(icons.Replay10)
	back.OnClick(func(e events.Event) { v.SkipBy(-10 * time.Second) })
	c.play = button("", "Play")
	c.play.OnClick(func(e events.Event) { v.TogglePlay() })
	forward := button("", "Forward 10 seconds").SetIcon(icons.Forward10)
	forward.OnClick(func(e events.Event) { v.SkipBy(10 * time.Second) })
	c.mute = button("", "Mute")
	c.mute.OnClick(func(e events.Event) { v.SetMuted(!v.Muted) })
	c.volume = core.NewSlider(row).SetMin(0).SetMax(1).SetStep(0.05)
	c.volume.FinalStyler(func(s *styles.Style) { // (after the 20em of sliders)
		s.Min.X.Dp(80)
		s.Max.X.Dp(80)
		s.Grow.Set(0, 0)
	})
	c.volume.OnInput(func(e events.Event) { v.SetVolume(c.volume.Value) })
	c.volume.OnChange(func(e events.Event) { v.SetVolume(c.volume.Value) })
	for _, sl := range []*core.Slider{c.position, c.volume} {
		sl.Styler(func(s *styles.Style) {
			// white, as in players, over a translucent track
			sl.ValueColor, sl.ThumbColor = white, white
			s.Background = colors.Uniform(color.RGBA{255, 255, 255, 80})
			s.Padding.Set(units.Dp(6))
		})
	}
	c.time = core.NewText(row)
	c.time.Styler(func(s *styles.Style) {
		s.Color = white
		s.Margin.Left.Dp(8)
	})
	v.updateControls()
	v.setControlsShown(true)
}

// updateControls updates the controls for the state of the video.
func (v *Video) updateControls() {
	c := v.controls
	if c == nil || c.play == nil {
		return
	}
	if v.IsPlaying() {
		c.play.SetIcon(icons.Pause).SetTooltip("Pause")
	} else {
		c.play.SetIcon(icons.PlayArrow).SetTooltip("Play")
	}
	if v.Muted || v.Volume() < 1e-3 {
		c.mute.SetIcon(icons.VolumeOff).SetTooltip("Unmute")
	} else {
		c.mute.SetIcon(icons.VolumeUp).SetTooltip("Mute")
	}
	c.volume.SetValue(v.Volume())
	c.position.SetMax(float32(max(v.Duration(), time.Second).Seconds()))
	c.second = -1 // (the time is shown again)
	v.updatePosition(v.CurrentTime())
	c.play.Update()
	c.mute.Update()
	if !v.IsPlaying() {
		v.setControlsShown(true)
	}
	v.NeedsRender()
}

// updatePosition shows the given position of the video, once a second,
// unless the position is being dragged.
func (v *Video) updatePosition(t time.Duration) {
	c := v.controls
	if c == nil || c.position == nil {
		return
	}
	sec := int(t.Seconds())
	if sec == c.second {
		return
	}
	c.second = sec
	if !c.position.StateIs(states.Sliding) {
		c.position.SetValue(float32(t.Seconds()))
	}
	c.time.SetText(formatTime(t) + " / " + formatTime(v.Duration())).UpdateRender()
}

// showControls shows the controls as the mouse moves over the video, and
// hides them [controlsHideDelay] after it stops, while it plays.
func (v *Video) showControls() {
	c := v.controls
	c.shownAt = time.Now()
	v.setControlsShown(true)
	shownAt := c.shownAt
	time.AfterFunc(controlsHideDelay, func() {
		if !v.AsyncLockIfShown() {
			return
		}
		defer v.AsyncUnlock()
		if c := v.controls; c != nil && c.shownAt == shownAt && v.IsPlaying() && !c.position.StateIs(states.Sliding) {
			v.setControlsShown(false)
		}
	})
}

// setControlsShown shows or hides the controls.
func (v *Video) setControlsShown(shown bool) {
	c := v.controls
	if c == nil || c.bar.StateIs(states.Invisible) == !shown {
		return
	}
	c.bar.SetState(!shown, states.Invisible)
	v.NeedsRender()
}
