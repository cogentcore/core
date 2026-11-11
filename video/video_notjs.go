// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package video

import (
	"image"
	"time"

	"cogentcore.org/core/base/errors"
)

// On desktop platforms, a [Video] shows the frames of a [Playback], which
// its render source draws with the GPU in its box (see render_notjs.go),
// where its region of the scene is left transparent (see [Video.Render]),
// and the scene of its controls is drawn over it again.

// CurrentFrame returns the current frame (for rendering),
// and whether it is unchanged since it was last returned.
func (v *Video) CurrentFrame() (*image.RGBA, bool) {
	if !v.IsVisible() {
		return nil, true
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	unchanged := !v.changed
	v.changed = false
	return v.frame, unchanged
}

// hasFrame returns whether the video has a frame to show.
func (v *Video) hasFrame() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.frame != nil
}

// Render renders the box of the video, whose content is left transparent
// once it has a frame, which the render source draws there, under the
// controls, which are drawn again over it.
func (v *Video) Render() {
	v.Frame.Render()
	if v.hasFrame() {
		v.Scene.Painter.BlitBox(v.Geom.Pos.Content, v.Geom.Size.Actual.Content, nil)
	}
}

// Play starts playing the video from the start, decoding its frames at the
// given size. Values of 0 indicate to use the inherent size of the video for
// that dimension (keeping its aspect ratio if only one is 0); with both 0,
// it is decoded at the size that fits in the box of the widget, if it is
// laid out, up to the inherent size.
func (v *Video) Play(width, height float32) error {
	if v.Media == nil {
		return errors.New("video: no media to play")
	}
	v.Stop()
	v.playSize = [2]float32{width, height}
	w, h := int(width), int(height)
	if w == 0 && h == 0 {
		w = v.fitWidth()
	}
	v.playback = &Playback{Media: v.Media, Width: w, Height: h, Muted: v.Muted, Loop: v.Loop, Volume: v.volume,
		OnFrame: v.showFrame,
		OnEnd: func() {
			if v.AsyncLockIfShown() {
				v.updateControls()
				v.AsyncUnlock()
			}
		}}
	err := v.playback.Play()
	v.updateControls()
	return err
}

// fitWidth returns the width at which the frames of the video fit in the
// content of the widget, keeping their aspect, up to the width of the
// video, or 0 (the width of the video) before it is laid out.
func (v *Video) fitWidth() int {
	box := v.Geom.Size.Actual.Content
	mw, mh := float32(v.Media.Width), float32(v.Media.Height)
	if box.X <= 0 || box.Y <= 0 || mw <= 0 || mh <= 0 {
		return 0
	}
	w := min(box.X, box.Y*mw/mh, mw)
	return max(int(w), 1)
}

// showFrame shows the given frame (in the goroutine of the playback).
func (v *Video) showFrame(f Frame) {
	v.mu.Lock()
	v.frame, v.changed = f.Image, true
	v.mu.Unlock()
	if v.AsyncLockIfShown() {
		v.NeedsRender()
		v.updatePosition(f.Time)
		v.AsyncUnlock()
	}
}

// Pause pauses the playing, if it is playing.
func (v *Video) Pause() {
	if v.playback != nil {
		v.playback.Pause()
		v.updateControls()
	}
}

// Resume resumes the playing, if it is paused.
func (v *Video) Resume() {
	if v.playback != nil {
		v.playback.Resume()
		v.updateControls()
	}
}

// Stop stops the playing, if any.
func (v *Video) Stop() {
	if v.playback != nil {
		v.playback.Stop()
		v.updateControls()
	}
}

// IsPlaying returns whether the video is playing, and not paused nor ended.
func (v *Video) IsPlaying() bool {
	return v.playback != nil && v.playback.IsPlaying()
}

// isPaused returns whether the video is paused.
func (v *Video) isPaused() bool {
	return v.playback != nil && v.playback.IsPaused()
}

// Seek plays the video from the given time, staying paused if it is,
// which shows the frame at the time. It plays the video if it has not
// played yet, or has ended.
func (v *Video) Seek(t time.Duration) {
	if v.playback == nil || v.playback.IsEnded() || (!v.IsPlaying() && !v.isPaused()) {
		if errors.Log(v.Play(v.playSize[0], v.playSize[1])) != nil {
			return
		}
	}
	errors.Log(v.playback.Seek(t))
	v.updateControls()
}

// CurrentTime returns the time in the video of the frame shown last.
func (v *Video) CurrentTime() time.Duration {
	if v.playback == nil {
		return 0
	}
	return v.playback.Position()
}

// Duration returns the duration of the video, or 0 if it is not known.
func (v *Video) Duration() time.Duration {
	if v.Media == nil {
		return 0
	}
	return v.Media.Duration
}

// SetMuted sets whether the audio is not heard, while it plays too.
func (v *Video) SetMuted(muted bool) {
	v.Muted = muted
	if v.playback != nil {
		v.playback.SetMuted(muted)
	}
	v.updateControls()
}

// Volume returns the volume of the audio, from 0 to 1.
func (v *Video) Volume() float32 {
	if v.volume == 0 {
		return 1
	}
	return v.volume
}

// SetVolume sets the volume of the audio, from 0 to 1, while it plays too.
func (v *Video) SetVolume(volume float32) {
	v.volume = max(min(volume, 1), 1e-6)
	if v.playback != nil {
		v.playback.SetVolume(volume)
	}
	v.updateControls()
}

// release releases the resources of the video (none on desktop).
func (v *Video) release() {}

// updateControls updates the controls for the state of the video.
func (v *Video) updateControls() {
	if c := v.controls; c != nil {
		c.Refresh()
	}
}
