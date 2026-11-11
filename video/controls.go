// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package video

import (
	"image"
	"image/color"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
)

// Player is a player of a video that [Controls] control: a [Video], or
// another widget that shows a [Playback] (as a video of a web page).
type Player interface {
	// TogglePlay plays the video if it is not playing, and pauses it if it is.
	TogglePlay()

	// IsPlaying returns whether the video is playing (not paused nor ended).
	IsPlaying() bool

	// Seek plays the video from the given time, staying paused if it is.
	Seek(t time.Duration)

	// SkipBy seeks the video by the given time from its current time.
	SkipBy(d time.Duration)

	// StepFrame pauses the video and shows its next frame, or the one
	// before it for a negative n.
	StepFrame(n int)

	// CurrentTime returns the current time of the video.
	CurrentTime() time.Duration

	// Duration returns the duration of the video, or 0 if it is not known.
	Duration() time.Duration

	// IsMuted returns whether the audio is not heard.
	IsMuted() bool

	// SetMuted sets whether the audio is not heard.
	SetMuted(muted bool)

	// Volume returns the volume of the audio, from 0 to 1.
	Volume() float32

	// SetVolume sets the volume of the audio, from 0 to 1.
	SetVolume(volume float32)
}

// controlsHideDelay is how long the controls stay shown after the mouse
// stops moving over the bottom of a playing video, and controlsLeaveDelay
// after it moves out of the bottom.
const (
	controlsHideDelay  = 2500 * time.Millisecond
	controlsLeaveDelay = 500 * time.Millisecond
)

// Controls are the standard controls of a video player, as those of the
// video elements of browsers: a bar over the bottom of the video, partly
// transparent, with the position (a slider, which seeks when it is
// changed), buttons to skip back 10 seconds, to play and pause, and to
// skip forward 10 seconds, to mute, with the volume and the time.
// They are a child of the widget of the video (their target), which must
// be their containing block (positioned, as relative), and are shown while
// the mouse moves over them (see [Controls.inZone]), until 2.5
// seconds after it stops there, or half a second after it moves out of the
// bottom or leaves, and while the video is not playing. A click on the target (not on the
// controls) plays and pauses the video, and gives it the focus, for the
// keys of players (see [Controls.handleKey]). The target must be rendered as a
// whole when the controls change (see [Video.RenderGroup]), as they are
// partly transparent over the video.
type Controls struct { //types:add
	core.Frame

	// player is the player that the controls control.
	player Player

	// target is the widget of the video that the controls are over.
	target core.Widget

	position *core.Slider
	play     *core.Button
	mute     *core.Button
	volume   *core.Slider
	time     *core.Text

	// shownAt is when the controls were last shown by the mouse moving.
	shownAt time.Time

	// second is the second of the time that is shown.
	second int
}

// SetPlayer sets the player of the controls and the widget of its video
// (their parent, as a [Video]), which shows them as the mouse moves over
// it, and plays and pauses the video when it is clicked.
func (c *Controls) SetPlayer(target core.Widget, player Player) *Controls {
	c.player, c.target = player, target
	tw := target.AsWidget()
	tw.On(events.MouseMove, func(e events.Event) {
		if c.This != nil {
			c.mouseMoved(e.Pos())
		}
	})
	tw.On(events.MouseLeave, func(e events.Event) {
		if c.This != nil {
			c.shownAt = time.Time{}
			c.SetShown(!c.player.IsPlaying())
		}
	})
	tw.OnClick(func(e events.Event) {
		if c.This == nil || e.Pos().In(c.Geom.TotalBBox) {
			return // (a click on the controls)
		}
		tw.SetFocus() // (for the keys)
		c.player.TogglePlay()
		c.Refresh()
		e.SetHandled()
	})
	// the keys of players, when the video or its controls have the focus
	// (first, before a control, as a slider, handles the arrows)
	tw.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Focusable)
	})
	tw.OnFirst(events.KeyChord, func(e events.Event) {
		if c.This != nil && c.handleKey(e) {
			e.SetHandled()
		}
	})
	c.Refresh()
	return c
}

func (c *Controls) Init() {
	c.Frame.Init()
	white := colors.Uniform(color.White)
	c.Styler(func(s *styles.Style) {
		s.Position = styles.PositionAbsolute
		s.Inset.Left, s.Inset.Right, s.Inset.Bottom = units.Zero(), units.Zero(), units.Zero()
		s.InsetSet.Left, s.InsetSet.Right, s.InsetSet.Bottom = true, true, true
		s.Direction = styles.Column
		// (no background of its own: see Render)
		s.Color = white
		s.Padding.Set(units.Dp(4), units.Dp(8))
		s.Gap.Zero()
	})

	// the position, which seeks when it is changed
	c.position = core.NewSlider(c).SetMin(0).SetStep(1)
	c.position.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
	c.position.OnChange(func(e events.Event) {
		c.player.Seek(time.Duration(c.position.Value * float32(time.Second)))
		c.Refresh()
	})

	// the buttons, volume and time
	row := core.NewFrame(c)
	row.Styler(func(s *styles.Style) {
		s.Align.Items = styles.Center
		s.Gap.X.Dp(2)
	})
	button := func(icon icons.Icon, tooltip string, do func()) *core.Button {
		bt := core.NewButton(row).SetType(core.ButtonAction).SetIcon(icon)
		bt.SetTooltip(tooltip)
		bt.Styler(func(s *styles.Style) {
			s.Color = white
			s.Padding.Set(units.Dp(4))
		})
		bt.OnClick(func(e events.Event) {
			do()
			c.Refresh()
		})
		return bt
	}
	button(icons.Replay10, "Back 10 seconds", func() { c.player.SkipBy(-10 * time.Second) })
	c.play = button(icons.PlayArrow, "Play", func() { c.player.TogglePlay() })
	button(icons.Forward10, "Forward 10 seconds", func() { c.player.SkipBy(10 * time.Second) })
	c.mute = button(icons.VolumeUp, "Mute", func() { c.player.SetMuted(!c.player.IsMuted()) })
	c.volume = core.NewSlider(row).SetMin(0).SetMax(1).SetStep(0.05)
	c.volume.FinalStyler(func(s *styles.Style) { // (after the 20em of sliders)
		s.Min.X.Dp(80)
		s.Max.X.Dp(80)
		s.Grow.Set(0, 0)
	})
	setVolume := func(e events.Event) {
		c.player.SetVolume(c.volume.Value)
		c.Refresh()
	}
	c.volume.OnInput(setVolume)
	c.volume.OnChange(setVolume)
	for _, sl := range []*core.Slider{c.position, c.volume} {
		sl.Styler(func(s *styles.Style) {
			// white, as in players, over a translucent track
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
}

// controlsBackground is the partly transparent background of the controls.
var controlsBackground = colors.Uniform(color.RGBA{0, 0, 0, 150})

// Render renders the partly transparent background of the controls over
// the video (see [Controls.ChildBackground]).
func (c *Controls) Render() {
	c.Frame.Render()
	c.Scene.Painter.FillBox(c.Geom.Pos.Total, c.Geom.Size.Actual.Total, controlsBackground)
}

// ChildBackground returns transparent: the widgets of the controls are over
// the video, so they do not fill their boxes with the background around
// them, which is otherwise that of the widgets they are in, with partly
// transparent ones blended over the parent (opaque gray over a white page).
func (c *Controls) ChildBackground(child core.Widget) image.Image {
	return colors.Uniform(color.Transparent)
}

// handleKey handles the keys of players, returning whether the given key
// event is one of them: space plays and pauses, the right and left arrows
// step a frame forward and back, and the up and down arrows skip 10 seconds
// forward and back. The controls are shown as for the mouse.
func (c *Controls) handleKey(e events.Event) bool {
	if e.HasAnyModifier(key.Shift, key.Control, key.Alt, key.Meta) {
		return false
	}
	switch e.KeyCode() {
	case key.CodeSpacebar:
		c.player.TogglePlay()
	case key.CodeRightArrow:
		c.player.StepFrame(1)
	case key.CodeLeftArrow:
		c.player.StepFrame(-1)
	case key.CodeUpArrow:
		c.player.SkipBy(10 * time.Second)
	case key.CodeDownArrow:
		c.player.SkipBy(-10 * time.Second)
	default:
		return false
	}
	c.Refresh()
	c.shownAt = time.Now()
	c.SetShown(true)
	c.hideAfter(controlsHideDelay, c.shownAt)
	return true
}

// Refresh updates the controls for the state of the player (with the
// lock of the widgets held).
func (c *Controls) Refresh() {
	p := c.player
	if p == nil || c.This == nil {
		return
	}
	if p.IsPlaying() {
		c.play.SetIcon(icons.Pause).SetTooltip("Pause")
	} else {
		c.play.SetIcon(icons.PlayArrow).SetTooltip("Play")
	}
	if p.IsMuted() || p.Volume() < 1e-3 {
		c.mute.SetIcon(icons.VolumeOff).SetTooltip("Unmute")
	} else {
		c.mute.SetIcon(icons.VolumeUp).SetTooltip("Mute")
	}
	c.volume.SetValue(p.Volume())
	c.position.SetMax(float32(max(p.Duration(), time.Second).Seconds()))
	c.second = -1 // (the time is shown again)
	c.SetTime(p.CurrentTime())
	c.play.Update()
	c.mute.Update()
	if !p.IsPlaying() {
		c.SetShown(true)
	}
	c.NeedsRender()
}

// SetTime shows the given current time of the video, once a second
// (unless the position is being dragged), as the video plays.
func (c *Controls) SetTime(t time.Duration) {
	if c.player == nil || c.This == nil {
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
	c.time.SetText(formatTime(t) + " / " + formatTime(c.player.Duration())).UpdateRender()
}

// mouseMoved shows the controls as the mouse moves over the bottom of the
// video (see [Controls.inZone]), and hides them [controlsHideDelay] after it
// stops there, or [controlsLeaveDelay] after it moves out of it, while the
// video plays.
func (c *Controls) mouseMoved(pos image.Point) {
	if !c.inZone(pos) {
		if c.IsShown() && !c.shownAt.IsZero() {
			c.shownAt = time.Time{}
			c.hideAfter(controlsLeaveDelay, c.shownAt)
		}
		return
	}
	c.shownAt = time.Now()
	c.SetShown(true)
	c.hideAfter(controlsHideDelay, c.shownAt)
}

// inZone returns whether the given position of the mouse is in the bottom
// of the video, where it shows the controls: the region of the controls
// (or the bottom 15% of the video before they are laid out).
func (c *Controls) inZone(pos image.Point) bool {
	tb := c.target.AsWidget().Geom.TotalBBox
	top := float32(tb.Max.Y) - 0.15*float32(tb.Dy())
	if cb := c.Geom.TotalBBox; !cb.Empty() {
		top = float32(cb.Min.Y)
	}
	return pos.In(tb) && float32(pos.Y) >= top
}

// hideAfter hides the controls after the given delay, if they have not
// been shown again since the given time, while the video plays.
func (c *Controls) hideAfter(delay time.Duration, shownAt time.Time) {
	time.AfterFunc(delay, func() {
		if c.This == nil || !c.AsyncLockIfShown() {
			return
		}
		defer c.AsyncUnlock()
		if c.This != nil && c.shownAt == shownAt && c.player.IsPlaying() && !c.position.StateIs(states.Sliding) {
			c.SetShown(false)
		}
	})
}

// SetShown shows or hides the controls.
func (c *Controls) SetShown(shown bool) {
	if c.This == nil || c.StateIs(states.Invisible) == !shown {
		return
	}
	c.SetState(!shown, states.Invisible)
	c.NeedsRender()
	if c.target != nil {
		c.target.AsWidget().NeedsRender()
	}
}

// IsShown returns whether the controls are shown.
func (c *Controls) IsShown() bool {
	return !c.StateIs(states.Invisible)
}
