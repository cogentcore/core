// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package video implements a video player widget in Cogent Core.
package video

//go:generate core generate

import (
	"fmt"
	"image"
	"image/color"
	"sync"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
)

// Video is a video player widget, with the standard playback controls over
// the bottom of the video if [Video.Controls] is set.
// On desktop platforms, the playing itself is a [Playback], which other
// widgets can show too, and the controls are Cogent Core widgets
// (see controls_notjs.go); on the web, the video is a video element of the
// browser, placed at the widget, with the controls of the browser.
type Video struct { //types:add
	core.Frame

	// Media is the video media.
	Media *Media

	// degrees of rotation to apply to the video images
	// 90 = left 90, -90 = right 90
	Rotation float32

	// Muted is whether the audio is not heard (see [Video.SetMuted]).
	Muted bool `set:"-"`

	// Loop is whether to play the video again from the start at its end.
	Loop bool

	// Controls is whether to show the standard playback controls over the
	// bottom of the video when the mouse is over it, or it is paused:
	// play and pause, back and forward 10 seconds, the position (which can
	// be dragged to seek), the volume, and the time.
	Controls bool

	// Poster is the URL of an image to show before the video plays,
	// on the web (where the video is a video element of the browser).
	Poster string

	// playback is the current playing, if any.
	playback *Playback

	// playSize is the size given to Play, for playing again.
	playSize [2]float32

	// volume is the volume of the audio, from 0 to 1 (0 is full volume,
	// as the zero value; see [Video.SetVolume]).
	volume float32

	// controls are the widgets of the controls, if shown.
	controls *controls

	// mu protects frame and changed, which are set by the playback.
	mu sync.Mutex

	// frame is the last frame of the playback, to render.
	frame *image.RGBA

	// changed is whether frame has changed since it was last rendered.
	changed bool
}

func (v *Video) Init() {
	v.Frame.Init()
	v.Styler(func(s *styles.Style) {
		// the black of players around the video (as in its letterbox),
		// and the containing block of the controls
		s.Background = colors.Uniform(color.Black)
		s.Position = styles.PositionRelative
		s.Padding.Zero()
		s.Gap.Zero()
		s.Direction = styles.Column
	})
	v.initControls()
}

func (v *Video) OnAdd() {
	v.Frame.OnAdd()
	v.Scene.AddDirectRender(v)
}

func (v *Video) Destroy() {
	v.Stop()
	v.release()
	v.Scene.DeleteDirectRender(v)
	v.Frame.Destroy()
}

// Open opens the video specified by the given filepath.
func (v *Video) Open(fpath string) error {
	media, err := OpenMedia(fpath, nil)
	if err != nil {
		return err
	}
	v.Media = media
	return nil
}

// TogglePlay plays the video if it is paused, has ended, or has not
// played yet, and pauses it if it is playing.
func (v *Video) TogglePlay() {
	switch {
	case v.IsPlaying():
		v.Pause()
	case v.isPaused():
		v.Resume()
	default:
		v.Play(v.playSize[0], v.playSize[1])
	}
}

// SkipBy seeks the video by the given time from its position (back for
// a negative time), as the -10 and +10 controls do.
func (v *Video) SkipBy(d time.Duration) {
	v.Seek(v.CurrentTime() + d)
}

// formatTime returns the given time as the controls of players show it:
// m:ss, or h:mm:ss for an hour or more.
func formatTime(t time.Duration) string {
	s := int(max(t, 0).Seconds())
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
